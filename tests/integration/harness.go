package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	stdlog "log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/firehose"
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/routing"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	gormlogger "gorm.io/gorm/logger"
	"tangled.org/pdewey.com/atp"
	"tangled.org/pdewey.com/chrysalis/testpds"
)

func init() {
	// Chrysalis constructs its gorm sessions with `&gorm.Config{}` (no Logger),
	// so each session falls back to gormlogger.Default. Replace it with one
	// that ignores ErrRecordNotFound — chrysalis's preflight existence checks
	// (handle/email/seq lookups on a fresh test DB) otherwise spam yellow
	// "record not found" warnings on every test run.
	gormlogger.Default = gormlogger.New(
		stdlog.New(os.Stdout, "\r\n", stdlog.LstdFlags),
		gormlogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  gormlogger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)
}

// silenceLogs routes the noisy log outputs that show up during integration
// tests (cocoon's stdlib log + slog, arabica's zerolog) to io.Discard so
// passing runs aren't drowned in per-request access logs and handler debug
// lines.
//
// Set INTEGRATION_LOGS=1 to keep them visible — arabica's zerolog gets routed
// through a colored ConsoleWriter so its lines blend in with cocoon's
// gorm/slog output.
func silenceLogs() {
	if v := os.Getenv("INTEGRATION_LOGS"); v == "1" || v == "true" {
		zlog.Logger = zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}).With().Timestamp().Logger()
		return
	}
	stdlog.SetOutput(io.Discard)
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	zlog.Logger = zerolog.New(io.Discard)
}

// testAuthHeader is the request header used by the test harness to inject
// authentication context bypassing real OAuth cookies. The middleware in
// harnessAuthMiddleware reads this and populates the same context keys that
// OAuthManager.AuthMiddleware would set.
const (
	testAuthDIDHeader     = "X-Test-Auth-DID"
	testAuthSessionHeader = "X-Test-Auth-Session"
)

// Harness wires up a full arabica handler tree backed by an in-process test
// PDS and exposes an httptest.Server. Auth is faked via custom headers so
// tests can act as any DID without an OAuth dance.
type Harness struct {
	T            *testing.T
	PDS          *testpds.TestPDS
	Server       *httptest.Server
	Handler      *handlers.Handler
	FeedIndex      *firehose.FeedIndex
	Consumer       *firehose.Consumer
	ProfileWatcher *firehose.ProfileWatcher
	SessionCache   *atproto.SessionCache

	// PrimaryAccount is the default account created on harness setup.
	PrimaryAccount TestAccount

	// Client is an http.Client preconfigured to authenticate as PrimaryAccount.
	// Use Harness.NewClientForAccount for non-primary users.
	Client *http.Client

	// accounts maps DID -> APIClient so the ClientProvider can route XRPC calls
	// to the correct authenticated session for any registered account.
	accountsMu sync.RWMutex
	accounts   map[string]*atclient.APIClient

	// atpClients maps DID -> atp.Client for direct PDS access in tests.
	atpClients map[string]*atp.Client

	// firehoseCancel stops the firehose bridge goroutine.
	firehoseCancel context.CancelFunc

	cleanup []func()
}

// TestAccount holds credentials for a user on the test PDS.
type TestAccount struct {
	DID       string
	Handle    string
	Password  string
	AccessJwt string
}

// HarnessOptions configures harness setup.
type HarnessOptions struct {
	// PrimaryHandle is the handle of the default account. Defaults to "alice.test".
	PrimaryHandle string
	// PrimaryEmail is the email of the default account. Defaults to "alice@test.com".
	PrimaryEmail string
	// PrimaryPassword is the password for the default account. Defaults to "hunter2".
	PrimaryPassword string
	// EnableFirehose subscribes to the test PDS firehose and feeds events
	// through the Consumer → FeedIndex pipeline, so records created via the
	// PDS are automatically indexed. Use WaitForRecord to synchronise.
	EnableFirehose bool
}

// StartHarness boots a test PDS, creates a primary account, builds the full
// handler tree, and exposes everything as an httptest server.
func StartHarness(t *testing.T, opts *HarnessOptions) *Harness {
	t.Helper()

	silenceLogs()

	if opts == nil {
		opts = &HarnessOptions{}
	}
	if opts.PrimaryHandle == "" {
		opts.PrimaryHandle = "alice.test"
	}
	if opts.PrimaryEmail == "" {
		opts.PrimaryEmail = "alice@test.com"
	}
	if opts.PrimaryPassword == "" {
		opts.PrimaryPassword = "hunter2"
	}

	pds := testpds.StartT(t, nil)

	// Build an in-process FeedIndex (SQLite, temp dir) to back the witness
	// cache and the suggestion endpoint. This is the same type production
	// uses; the firehose consumer is not wired up, so the index is populated
	// purely via write-through from store CRUD operations.
	feedIndex, err := firehose.NewFeedIndex(t.TempDir()+"/feed-index.db", 1*time.Hour)
	require.NoError(t, err)

	sessionCache := atproto.NewSessionCache()

	harness := &Harness{
		T:            t,
		PDS:          pds,
		FeedIndex:    feedIndex,
		SessionCache: sessionCache,
		accounts:     make(map[string]*atclient.APIClient),
		atpClients:   make(map[string]*atp.Client),
	}

	// Provider routes XRPC calls based on the DID in the request context. The
	// harness pre-registers each account's APIClient so the right session is
	// used per-DID for multi-user scenarios.
	atprotoClient := atproto.NewClientWithProvider(func(ctx context.Context, d syntax.DID, _ string) (*atp.Client, error) {
		harness.accountsMu.RLock()
		api, ok := harness.accounts[d.String()]
		harness.accountsMu.RUnlock()
		if !ok {
			return nil, fmt.Errorf("no APIClient registered for DID %s", d)
		}
		return atp.NewClient(api, d), nil
	})

	// Construct dependencies the same way main.go does, minus the persistent
	// stores. The OAuth manager is built but never exercised — its
	// AuthMiddleware short-circuits when no cookies are present, leaving the
	// context the harness middleware installed in place.
	oauthMgr, err := atproto.NewOAuthManager("", "http://localhost/oauth/callback", []string{"atproto"}, nil)
	require.NoError(t, err)

	feedRegistry := feed.NewRegistry()
	feedService := feed.NewService(feedRegistry)

	h := handlers.NewHandler(
		oauthMgr,
		atprotoClient,
		sessionCache,
		feedService,
		feedRegistry,
		handlers.Config{
			SecureCookies: false,
			PublicURL:     "http://localhost",
		},
	)
	h.SetFeedIndex(feedIndex)
	h.SetWitnessCache(feedIndex)

	// Build the router with no moderation service (most tests don't need it).
	logger := zerolog.Nop()
	router := routing.SetupRouter(routing.Config{
		Handlers:     h,
		OAuthManager: oauthMgr,
		Logger:       logger,
	})

	// Wrap the router with the harness auth middleware so tests can pose as
	// any DID via the X-Test-Auth-* headers.
	authedRouter := harnessAuthMiddleware(router)

	server := httptest.NewServer(authedRouter)

	harness.Server = server
	harness.Handler = h
	harness.cleanup = append(harness.cleanup, server.Close, func() { _ = feedIndex.Close() })

	// Wire up the firehose consumer before creating accounts so events are
	// indexed as they happen.
	if opts.EnableFirehose {
		consumer := firehose.NewConsumer(&firehose.Config{
			WantedCollections: firehose.ArabicaCollections,
		}, feedIndex)
		harness.Consumer = consumer
		harness.ProfileWatcher = firehose.NewProfileWatcher(&firehose.Config{}, feedIndex)

		ctx, cancel := context.WithCancel(context.Background())
		harness.firehoseCancel = cancel
		harness.cleanup = append(harness.cleanup, cancel)

		ch, err := pds.Subscribe(ctx, 0)
		require.NoError(t, err)

		go harness.firehoseBridge(ctx, ch)
	}

	// Create the primary account and register it.
	harness.PrimaryAccount = harness.CreateAccount(opts.PrimaryEmail, opts.PrimaryHandle, opts.PrimaryPassword)
	harness.Client = harness.NewClientForAccount(harness.PrimaryAccount)

	t.Cleanup(func() {
		for _, fn := range harness.cleanup {
			fn()
		}
	})

	return harness
}

// CreateAccount registers a new account on the test PDS, logs in via password
// auth, and registers the resulting APIClient with the harness so the handler
// tree can act as that user. Use this for multi-user test scenarios.
func (h *Harness) CreateAccount(email, handle, password string) TestAccount {
	h.T.Helper()

	acct := createAccountOnPDS(h.T, h.PDS.URL, email, handle, password)

	apiClient, err := atclient.LoginWithPasswordHost(
		context.Background(), h.PDS.URL, acct.Handle, password, "", nil,
	)
	require.NoError(h.T, err)

	h.accountsMu.Lock()
	h.accounts[acct.DID] = apiClient
	h.atpClients[acct.DID] = atp.NewClient(apiClient, syntax.DID(acct.DID))
	h.accountsMu.Unlock()

	return acct
}

// NewClientForAccount returns an http.Client that automatically attaches the
// test auth headers for the given account. Use this when a test needs to act
// as a non-primary user.
func (h *Harness) NewClientForAccount(acct TestAccount) *http.Client {
	return &http.Client{
		Transport: &authInjectingTransport{
			did:       acct.DID,
			sessionID: "test-session-" + acct.DID,
			base:      http.DefaultTransport,
		},
	}
}

// URL returns the test server's base URL with the given path appended.
func (h *Harness) URL(path string) string {
	return h.Server.URL + path
}

// PostForm posts a urlencoded form as the primary account and returns the response.
func (h *Harness) PostForm(path string, form url.Values) *http.Response {
	h.T.Helper()
	req, err := http.NewRequest("POST", h.URL(path), strings.NewReader(form.Encode()))
	require.NoError(h.T, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := h.Client.Do(req)
	require.NoError(h.T, err)
	return resp
}

// PostJSON posts a JSON body as the primary account and returns the response.
func (h *Harness) PostJSON(path string, body any) *http.Response {
	h.T.Helper()
	buf, err := json.Marshal(body)
	require.NoError(h.T, err)
	req, err := http.NewRequest("POST", h.URL(path), bytes.NewReader(buf))
	require.NoError(h.T, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.Client.Do(req)
	require.NoError(h.T, err)
	return resp
}

// Get fetches a path as the primary account.
func (h *Harness) Get(path string) *http.Response {
	h.T.Helper()
	req, err := http.NewRequest("GET", h.URL(path), nil)
	require.NoError(h.T, err)
	resp, err := h.Client.Do(req)
	require.NoError(h.T, err)
	return resp
}

// GetHTMX fetches a path as the primary account with the HX-Request header set,
// for endpoints behind RequireHTMXMiddleware.
func (h *Harness) GetHTMX(path string) *http.Response {
	h.T.Helper()
	req, err := http.NewRequest("GET", h.URL(path), nil)
	require.NoError(h.T, err)
	req.Header.Set("HX-Request", "true")
	resp, err := h.Client.Do(req)
	require.NoError(h.T, err)
	return resp
}

// PutForm sends a urlencoded form via PUT as the primary account.
func (h *Harness) PutForm(path string, form url.Values) *http.Response {
	h.T.Helper()
	req, err := http.NewRequest("PUT", h.URL(path), strings.NewReader(form.Encode()))
	require.NoError(h.T, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := h.Client.Do(req)
	require.NoError(h.T, err)
	return resp
}

// SessionIDFor returns the test session ID assigned to an account by
// authInjectingTransport. Tests use it when reaching into the session cache
// directly (e.g. to evict an entry to force a witness/PDS read).
func (h *Harness) SessionIDFor(acct TestAccount) string {
	return "test-session-" + acct.DID
}

// EvictWitnessRecord deletes a single record from the witness cache by
// (collection, rkey). Tests call this to force a witness-cache miss without
// going through the delete handler (which would also delete from the PDS).
// Combined with InvalidateSessionCache, this exercises the PDS fallback path.
func (h *Harness) EvictWitnessRecord(acct TestAccount, collection, rkey string) {
	h.T.Helper()
	require.NoError(h.T, h.FeedIndex.DeleteWitnessRecord(context.Background(), acct.DID, collection, rkey))
}

// InvalidateSessionCache wipes the per-session in-memory cache for an account.
// Tests use it together with EvictWitnessRecord to force the store to fall
// through both cache layers down to a real PDS read.
func (h *Harness) InvalidateSessionCache(acct TestAccount) {
	h.SessionCache.Invalidate(h.SessionIDFor(acct))
}

// PDSGetRecord fetches a single record directly from the PDS via XRPC,
// bypassing all arabica caching and conversion layers. Returns the raw
// record value as stored in the user's repo.
func (h *Harness) PDSGetRecord(acct TestAccount, collection, rkey string) map[string]any {
	h.T.Helper()
	h.accountsMu.RLock()
	client := h.atpClients[acct.DID]
	h.accountsMu.RUnlock()
	require.NotNil(h.T, client, "no atp client for DID %s", acct.DID)

	rec, err := client.GetRecord(context.Background(), collection, rkey)
	require.NoError(h.T, err)
	return rec.Value
}

// PDSListRecords fetches all records in a collection directly from the PDS
// via XRPC. Returns raw record values as stored in the user's repo.
func (h *Harness) PDSListRecords(acct TestAccount, collection string) []map[string]any {
	h.T.Helper()
	h.accountsMu.RLock()
	client := h.atpClients[acct.DID]
	h.accountsMu.RUnlock()
	require.NotNil(h.T, client, "no atp client for DID %s", acct.DID)

	records, err := client.ListAllRecords(context.Background(), collection)
	require.NoError(h.T, err)

	values := make([]map[string]any, len(records))
	for i, r := range records {
		values[i] = r.Value
	}
	return values
}

// Delete sends a DELETE request as the primary account.
func (h *Harness) Delete(path string) *http.Response {
	h.T.Helper()
	req, err := http.NewRequest("DELETE", h.URL(path), nil)
	require.NoError(h.T, err)
	resp, err := h.Client.Do(req)
	require.NoError(h.T, err)
	return resp
}

// ReadBody drains and returns the response body, closing it.
func ReadBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(body)
}

// --- internals ---

// harnessAuthMiddleware injects authentication context based on test headers.
// Runs before the real OAuth middleware. When the OAuth middleware sees no
// cookies, it passes the request through unchanged, leaving the harness
// context intact.
func harnessAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		did := r.Header.Get(testAuthDIDHeader)
		sessionID := r.Header.Get(testAuthSessionHeader)
		if did != "" && sessionID != "" {
			r = r.WithContext(atproto.ContextWithAuth(r.Context(), did, sessionID))
		}
		next.ServeHTTP(w, r)
	})
}

// authInjectingTransport adds the test auth headers to every request.
type authInjectingTransport struct {
	did       string
	sessionID string
	base      http.RoundTripper
}

func (t *authInjectingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Set(testAuthDIDHeader, t.did)
	clone.Header.Set(testAuthSessionHeader, t.sessionID)
	// Set Origin to match the host so Go's CrossOriginProtection allows the request.
	if clone.Header.Get("Origin") == "" {
		clone.Header.Set("Origin", "http://"+req.Host)
	}
	return t.base.RoundTrip(clone)
}

// createAccountOnPDS registers a new account on the test PDS and returns its credentials.
func createAccountOnPDS(t *testing.T, pdsURL, email, handle, password string) TestAccount {
	t.Helper()

	body, err := json.Marshal(map[string]string{
		"email":    email,
		"handle":   handle,
		"password": password,
	})
	require.NoError(t, err)

	resp, err := http.Post(pdsURL+"/xrpc/com.atproto.server.createAccount", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode, "createAccount failed: %s", string(respBody))

	var result struct {
		AccessJwt string `json:"accessJwt"`
		Handle    string `json:"handle"`
		Did       string `json:"did"`
	}
	require.NoError(t, json.Unmarshal(respBody, &result))

	return TestAccount{
		DID:       result.Did,
		Handle:    result.Handle,
		Password:  password,
		AccessJwt: result.AccessJwt,
	}
}

// --- firehose bridge ---

// firehoseBridge reads testpds firehose events, fetches records via XRPC, and
// feeds them through the Consumer's event processing pipeline.
func (h *Harness) firehoseBridge(ctx context.Context, ch <-chan testpds.FirehoseEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}

			if evt.Identity != nil && h.ProfileWatcher != nil {
				handle := ""
				if evt.Identity.Handle != nil {
					handle = *evt.Identity.Handle
				}
				h.ProfileWatcher.ProcessEvent(firehose.JetstreamEvent{
					DID:    evt.Identity.Did,
					TimeUS: time.Now().UnixMicro(),
					Kind:   "identity",
					Identity: &firehose.JetstreamIdentity{
						DID:    evt.Identity.Did,
						Handle: handle,
						Seq:    evt.Identity.Seq,
						Time:   evt.Identity.Time,
					},
				})
				continue
			}

			if evt.Account != nil && h.ProfileWatcher != nil {
				status := ""
				if evt.Account.Status != nil {
					status = *evt.Account.Status
				}
				h.ProfileWatcher.ProcessEvent(firehose.JetstreamEvent{
					DID:    evt.Account.Did,
					TimeUS: time.Now().UnixMicro(),
					Kind:   "account",
					Account: &firehose.JetstreamAccount{
						Active: evt.Account.Active,
						DID:    evt.Account.Did,
						Seq:    evt.Account.Seq,
						Status: status,
						Time:   evt.Account.Time,
					},
				})
				continue
			}

			if evt.Commit == nil {
				continue
			}
			for _, op := range evt.Commit.Ops {
				parts := strings.SplitN(op.Path, "/", 2)
				if len(parts) != 2 {
					continue
				}
				collection, rkey := parts[0], parts[1]
				if !strings.HasPrefix(collection, "social.arabica.alpha.") {
					continue
				}

				event := firehose.JetstreamEvent{
					DID:    evt.Commit.Repo,
					TimeUS: time.Now().UnixMicro(),
					Kind:   "commit",
				}

				switch op.Action {
				case "create", "update":
					record, cid, err := h.fetchRecordJSON(evt.Commit.Repo, collection, rkey)
					if err != nil {
						continue
					}
					event.Commit = &firehose.JetstreamCommit{
						Operation:  op.Action,
						Collection: collection,
						RKey:       rkey,
						Record:     record,
						CID:        cid,
					}
				case "delete":
					event.Commit = &firehose.JetstreamCommit{
						Operation:  "delete",
						Collection: collection,
						RKey:       rkey,
					}
				default:
					continue
				}

				_ = h.Consumer.ProcessEvent(event)
			}
		}
	}
}

// fetchRecordJSON fetches a record from the test PDS as raw JSON.
func (h *Harness) fetchRecordJSON(did, collection, rkey string) (json.RawMessage, string, error) {
	u := fmt.Sprintf("%s/xrpc/com.atproto.repo.getRecord?repo=%s&collection=%s&rkey=%s",
		h.PDS.URL, url.QueryEscape(did), url.QueryEscape(collection), url.QueryEscape(rkey))
	resp, err := http.Get(u)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("getRecord %d", resp.StatusCode)
	}
	var result struct {
		CID   string          `json:"cid"`
		Value json.RawMessage `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", err
	}
	return result.Value, result.CID, nil
}

// WaitForRecord polls the FeedIndex until the record with the given AT-URI is
// indexed, or fails the test after timeout.
func (h *Harness) WaitForRecord(uri string, timeout time.Duration) {
	h.T.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		rec, _ := h.FeedIndex.GetRecord(context.Background(), uri)
		if rec != nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	h.T.Fatalf("timed out waiting for record %s to be indexed", uri)
}

// WaitForRecordAbsent polls the FeedIndex until the record with the given
// AT-URI is no longer present, or fails the test after timeout.
func (h *Harness) WaitForRecordAbsent(uri string, timeout time.Duration) {
	h.T.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		rec, _ := h.FeedIndex.GetRecord(context.Background(), uri)
		if rec == nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	h.T.Fatalf("timed out waiting for record %s to be removed from index", uri)
}

// statusErr is a small helper for assertions that print useful diagnostics.
// Bodies larger than 512 chars are truncated so a 404 HTML page doesn't drown
// the test output.
func statusErr(resp *http.Response, body string) string {
	const maxBody = 512
	if len(body) > maxBody {
		body = body[:maxBody] + "... [truncated, " + fmt.Sprintf("%d", len(body)) + " bytes total]"
	}
	return fmt.Sprintf("status %d: %s", resp.StatusCode, body)
}
