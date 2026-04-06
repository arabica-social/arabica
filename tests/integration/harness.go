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

	"arabica/internal/atproto"
	"arabica/internal/feed"
	"arabica/internal/firehose"
	"arabica/internal/handlers"
	"arabica/internal/routing"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/haileyok/cocoon/testpds"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"tangled.org/pdewey.com/atp"
)

// silenceLogs redirects all the noisy log outputs that show up during
// integration tests (cocoon's stdlib log + slog + GORM, arabica's zerolog) to
// io.Discard. This keeps `go test -v` output focused on actual test results.
//
// Without this, even passing runs scroll dozens of lines of GORM
// "record not found" warnings, arabica request debug logs, and cocoon's
// per-request access logs.
//
// Set INTEGRATION_VERBOSE=1 to keep the logs visible — useful when a test is
// failing and you want to see what arabica or cocoon were doing at the time.
//
// Note: without -v, `go test` already captures per-test output and only prints
// it on failure, so this silencing is mainly relevant for `go test -v`.
func silenceLogs() {
	if v := os.Getenv("INTEGRATION_VERBOSE"); v == "1" || v == "true" {
		return
	}
	// stdlib log (some GORM logger configurations write here)
	stdlog.SetOutput(io.Discard)
	// log/slog (cocoon's slogecho middleware, server lifecycle logs)
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	// zerolog (arabica's handler debug/info logs go through the global logger)
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
	T         *testing.T
	PDS       *testpds.TestPDS
	Server    *httptest.Server
	Handler   *handlers.Handler
	FeedIndex *firehose.FeedIndex

	// PrimaryAccount is the default account created on harness setup.
	PrimaryAccount TestAccount

	// Client is an http.Client preconfigured to authenticate as PrimaryAccount.
	// Use Harness.NewClientForAccount for non-primary users.
	Client *http.Client

	// accounts maps DID -> APIClient so the ClientProvider can route XRPC calls
	// to the correct authenticated session for any registered account.
	accountsMu sync.RWMutex
	accounts   map[string]*atclient.APIClient

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

	pds := testpds.Start(t, nil)

	// Build an in-process FeedIndex (SQLite, temp dir) to back the witness
	// cache and the suggestion endpoint. This is the same type production
	// uses; the firehose consumer is not wired up, so the index is populated
	// purely via write-through from store CRUD operations.
	feedIndex, err := firehose.NewFeedIndex(t.TempDir()+"/feed-index.db", 1*time.Hour)
	require.NoError(t, err)

	harness := &Harness{
		T:         t,
		PDS:       pds,
		FeedIndex: feedIndex,
		accounts:  make(map[string]*atclient.APIClient),
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
	oauthMgr, err := atproto.NewOAuthManager("", "http://localhost/oauth/callback", nil)
	require.NoError(t, err)

	sessionCache := atproto.NewSessionCache()
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
