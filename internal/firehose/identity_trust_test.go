package firehose

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/pdewey.com/atp"
	atpjetstream "tangled.org/pdewey.com/atp/jetstream"
)

type identityTransport func(*http.Request) (*http.Response, error)

func (f identityTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func identityTestIndex(t *testing.T, profile *atproto.Profile, resolvedDID string) *FeedIndex {
	t.Helper()
	idx, err := NewFeedIndex(t.TempDir()+"/index.db", time.Hour)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, idx.Close()) })
	idx.publicClient = atp.NewPublicClientWithHTTP(&http.Client{Transport: identityTransport(func(r *http.Request) (*http.Response, error) {
		response := httptest.NewRecorder()
		switch r.URL.Path {
		case "/xrpc/app.bsky.actor.getProfile":
			require.Equal(t, profile.DID, r.URL.Query().Get("actor"))
			require.NoError(t, json.NewEncoder(response).Encode(profile))
		case "/xrpc/com.atproto.identity.resolveHandle":
			require.NoError(t, json.NewEncoder(response).Encode(map[string]string{"did": resolvedDID}))
		default:
			t.Errorf("unexpected public request: %s", r.URL)
			response.WriteHeader(http.StatusNotFound)
		}
		return response.Result(), nil
	})})
	return idx
}

func TestIdentityTrust_ForeignHandleCannotEvictOwner(t *testing.T) {
	ctx := context.Background()
	victim := &atproto.Profile{DID: "did:plc:victim", Handle: "famous.example"}
	attacker := &atproto.Profile{DID: "did:plc:attacker", Handle: "attacker.example"}
	idx := identityTestIndex(t, attacker, victim.DID)
	require.NoError(t, idx.Register(attacker.DID))
	idx.StoreProfile(ctx, victim.DID, victim)
	idx.StoreProfile(ctx, attacker.DID, attacker)
	pw := NewProfileWatcher(&Config{}, idx)
	pw.ProcessEvent(JetstreamEvent{Kind: "identity", DID: attacker.DID, Identity: &JetstreamIdentity{DID: attacker.DID, Handle: victim.Handle}})
	got, ok := idx.GetDIDByHandle(ctx, victim.Handle)
	assert.True(t, ok, "victim's handle row must survive")
	assert.Equal(t, victim.DID, got, "a frame must not transfer a foreign handle")
	cached, ok := idx.profileStorage.loadProfile(ctx, victim.DID)
	assert.True(t, ok, "rejected claim must not evict victim's persistent profile")
	if ok {
		assert.Equal(t, victim.Handle, cached.Profile.Handle)
	}
	assert.True(t, idx.ProfileCachedInMemory(victim.DID), "rejected claim must not evict victim's memory cache")
}

func TestIdentityTrust_AccountFrameCannotPurge(t *testing.T) {
	for _, status := range []string{"deleted", "takendown"} {
		t.Run(status, func(t *testing.T) {
			ctx := context.Background()
			victim := &atproto.Profile{DID: "did:plc:victim", Handle: "victim.example"}
			idx := identityTestIndex(t, victim, victim.DID)
			require.NoError(t, idx.Register(victim.DID))
			require.NoError(t, idx.MarkBackfilled(ctx, victim.DID))
			require.NoError(t, idx.UpsertRecord(ctx, victim.DID, "social.arabica.alpha.bean", "b1", "cid1", []byte(`{"name":"Bean","createdAt":"2025-01-01T00:00:00Z"}`), time.Now().UnixMicro()))
			idx.StoreProfile(ctx, victim.DID, victim)
			require.NoError(t, idx.profileStorage.setUserPreferences(ctx, victim.DID, `{"private":true}`))
			pw := NewProfileWatcher(&Config{}, idx)
			pw.accountClient = accountResponseClient(t, http.StatusOK, `{}`)
			pw.ProcessEvent(accountEvent(victim.DID, status))
			assert.Equal(t, 1, idx.RecordCount(), "a relay frame cannot authorize destruction")
			assert.True(t, idx.IsRegistered(victim.DID))
			assert.True(t, idx.IsBackfilled(ctx, victim.DID))
			preferences, ok := idx.profileStorage.userPreferences(ctx, victim.DID)
			assert.True(t, ok, "user preferences must survive")
			assert.Equal(t, `{"private":true}`, preferences)
			did, ok := idx.GetDIDByHandle(ctx, victim.Handle)
			assert.True(t, ok)
			assert.Equal(t, victim.DID, did)
			_, ok = idx.profileStorage.loadProfile(ctx, victim.DID)
			assert.True(t, ok)
			assert.Contains(t, pw.snapshotDIDs(), victim.DID)
		})
	}
}

func accountResponseClient(t *testing.T, code int, body string) *http.Client {
	t.Helper()
	return &http.Client{Transport: identityTransport(func(r *http.Request) (*http.Response, error) {
		if code == 0 {
			return nil, context.DeadlineExceeded
		}
		assert.Equal(t, "https", r.URL.Scheme)
		assert.Equal(t, "public.api.bsky.app", r.URL.Host)
		response := httptest.NewRecorder()
		response.WriteHeader(code)
		_, _ = response.WriteString(body)
		return response.Result(), nil
	})}
}

func TestIdentityTrust_GenuineHandleChange(t *testing.T) {
	for _, registered := range []bool{false, true} {
		t.Run(fmt.Sprint("registered=", registered), func(t *testing.T) {
			ctx := context.Background()
			profile := &atproto.Profile{DID: "did:plc:known", Handle: "new.example"}
			idx := identityTestIndex(t, profile, profile.DID)
			if registered {
				require.NoError(t, idx.Register(profile.DID))
			} else {
				require.NoError(t, idx.UpsertRecord(ctx, profile.DID, "social.arabica.alpha.bean", "b1", "cid1", []byte(`{"name":"Bean"}`), time.Now().UnixMicro()))
			}
			// Simulate a profile poisoned by the previous relay-trusting implementation.
			idx.StoreProfile(ctx, profile.DID, &atproto.Profile{DID: profile.DID, Handle: "poisoned.example"})
			idx.OnIdentityEvent(ctx, profile.DID, profile.Handle)
			did, ok := idx.GetDIDByHandle(ctx, profile.Handle)
			assert.True(t, ok)
			assert.Equal(t, profile.DID, did)
			_, ok = idx.GetDIDByHandle(ctx, "poisoned.example")
			assert.False(t, ok, "verified refresh repairs the old poisoned mapping")
			got, err := idx.GetProfile(ctx, profile.DID)
			require.NoError(t, err)
			assert.Equal(t, profile.Handle, got.Handle)
		})
	}
}

func TestIdentityTrust_UnknownDIDCannotWrite(t *testing.T) {
	profile := &atproto.Profile{DID: "did:plc:unknown", Handle: "unknown.example"}
	idx := identityTestIndex(t, profile, profile.DID)
	idx.OnIdentityEvent(context.Background(), profile.DID, profile.Handle)
	_, ok := idx.GetDIDByHandle(context.Background(), profile.Handle)
	assert.False(t, ok)
	assert.False(t, idx.ProfileCachedInMemory(profile.DID))
}

func TestIdentityTrust_ReverseResolutionMustAgree(t *testing.T) {
	ctx := context.Background()
	profile := &atproto.Profile{DID: "did:plc:attacker", Handle: "famous.example"}
	idx := identityTestIndex(t, profile, "did:plc:victim")
	require.NoError(t, idx.Register(profile.DID))
	idx.StoreProfile(ctx, "did:plc:victim", &atproto.Profile{DID: "did:plc:victim", Handle: profile.Handle})
	idx.OnIdentityEvent(ctx, profile.DID, profile.Handle)
	got, ok := idx.GetDIDByHandle(ctx, profile.Handle)
	assert.True(t, ok)
	assert.Equal(t, "did:plc:victim", got)
	assert.True(t, idx.ProfileCachedInMemory("did:plc:victim"))
	_, ok = idx.profileStorage.loadProfile(ctx, "did:plc:victim")
	assert.True(t, ok)
}

func TestIdentityTrust_AccountCorroboration(t *testing.T) {
	for _, tt := range []struct {
		name, status, body string
		code               int
		purged             bool
	}{
		{"takedown", "takendown", `{"error":"AccountTakedown"}`, 400, true},
		{"deleted-hint-takedown", "deleted", `{"error":"AccountTakedown"}`, 400, true},
		{"rate-limit", "deleted", `{"error":"AccountTakedown"}`, 429, false},
		{"HTML", "takendown", `<html>outage</html>`, 400, false},
		{"absent-is-not-deleted", "deleted", `{"error":"InvalidRequest","message":"Profile not found"}`, 400, false},
		{"active", "deleted", `{}`, 200, false},
		{"timeout", "deleted", ``, 0, false},
		{"outage", "deleted", `{"error":"AccountTakedown"}`, 503, false},
		{"not-found-http", "deleted", `{}`, 404, false},
		{"invalid", "deleted", `{"error":"InvalidRequest","message":"bad request"}`, 400, false},
		{"deactivated", "deleted", `{"error":"AccountDeactivated"}`, 400, false},
		{"malformed", "deleted", `not JSON`, 400, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			profile := &atproto.Profile{DID: "did:plc:known", Handle: "known.example"}
			idx := identityTestIndex(t, profile, profile.DID)
			require.NoError(t, idx.Register(profile.DID))
			require.NoError(t, idx.UpsertRecord(ctx, profile.DID, "social.arabica.alpha.bean", "b1", "cid1", []byte(`{"name":"Bean"}`), time.Now().UnixMicro()))
			require.NoError(t, idx.profileStorage.setUserPreferences(ctx, profile.DID, `{"private":true}`))
			idx.StoreProfile(ctx, profile.DID, profile)
			pw := NewProfileWatcher(&Config{}, idx)
			pw.accountClient = accountResponseClient(t, tt.code, tt.body)
			pw.ProcessEvent(accountEvent(profile.DID, tt.status))
			assert.True(t, idx.IsRegistered(profile.DID), "relay cleanup must retain registration")
			prefs, ok := idx.profileStorage.userPreferences(ctx, profile.DID)
			assert.True(t, ok)
			assert.Equal(t, `{"private":true}`, prefs)
			if tt.purged {
				assert.Equal(t, 0, idx.RecordCount())
				assert.NotContains(t, pw.snapshotDIDs(), profile.DID)
				_, ok := idx.GetDIDByHandle(ctx, profile.Handle)
				assert.False(t, ok)
			} else {
				assert.Equal(t, 1, idx.RecordCount())
				assert.Contains(t, pw.snapshotDIDs(), profile.DID)
				got, ok := idx.GetDIDByHandle(ctx, profile.Handle)
				assert.True(t, ok)
				assert.Equal(t, profile.DID, got)
			}
		})
	}
}

func TestIdentityTrust_VerificationFailurePreservesOwner(t *testing.T) {
	ctx := context.Background()
	profile := &atproto.Profile{DID: "did:plc:attacker", Handle: "famous.example"}
	idx := identityTestIndex(t, profile, profile.DID)
	require.NoError(t, idx.Register(profile.DID))
	idx.StoreProfile(ctx, "did:plc:victim", &atproto.Profile{DID: "did:plc:victim", Handle: profile.Handle})
	idx.publicClient = atp.NewPublicClientWithHTTP(&http.Client{Transport: identityTransport(func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("AppView unavailable")
	})})
	idx.OnIdentityEvent(ctx, profile.DID, profile.Handle)
	got, ok := idx.GetDIDByHandle(ctx, profile.Handle)
	assert.True(t, ok)
	assert.Equal(t, "did:plc:victim", got)
	assert.True(t, idx.ProfileCachedInMemory("did:plc:victim"))
	_, ok = idx.profileStorage.loadProfile(ctx, "did:plc:victim")
	assert.True(t, ok)
}

func TestIdentityTrust_RelayVerificationBoundedAndNonblocking(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	profile := &atproto.Profile{DID: "did:plc:known", Handle: "known.example"}
	idx := identityTestIndex(t, profile, profile.DID)
	require.NoError(t, idx.Register(profile.DID))
	started := make(chan struct{})
	idx.publicClient = atp.NewPublicClientWithHTTP(&http.Client{Transport: identityTransport(func(r *http.Request) (*http.Response, error) {
		close(started)
		<-r.Context().Done()
		return nil, r.Context().Err()
	})})
	pw := NewProfileWatcher(&Config{}, idx)
	returned := make(chan error, 1)
	go func() { returned <- pw.handleEvent(ctx, &atpjetstream.Event{Kind: "identity", DID: profile.DID}) }()
	select {
	case err := <-returned:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("relay handler blocked on verification")
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("verification never started")
	}
	// Repeated hints for the in-flight DID consume no queue space.
	for i := 0; i < 100; i++ {
		require.NoError(t, pw.handleEvent(ctx, &atpjetstream.Event{Kind: "identity", DID: profile.DID}))
	}
	pw.verificationMu.Lock()
	assert.Len(t, pw.verificationPending, 1)
	pw.verificationMu.Unlock()
	for i := 0; i < verificationQueueDepth+10; i++ {
		require.NoError(t, pw.handleEvent(ctx, &atpjetstream.Event{Kind: "identity", DID: fmt.Sprintf("did:plc:queued%d", i)}))
	}
	pw.verificationMu.Lock()
	assert.Len(t, pw.verificationPending, verificationQueueDepth)
	pw.verificationMu.Unlock()
	cancel()
	select {
	case <-pw.verificationDone:
	case <-time.After(time.Second):
		t.Fatal("verification worker failed to stop")
	}
	assert.False(t, idx.ProfileCachedInMemory(profile.DID), "failed/cancelled verification must not write")
}

func TestIdentityTrust_AccountNegativeCache(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	profile := &atproto.Profile{DID: "did:plc:known", Handle: "known.example"}
	idx := identityTestIndex(t, profile, profile.DID)
	require.NoError(t, idx.Register(profile.DID))
	require.NoError(t, idx.Register("did:plc:barrier"))
	pw := NewProfileWatcher(&Config{}, idx)
	firstProbed := make(chan struct{})
	barrierStarted := make(chan struct{})
	pw.accountClient = &http.Client{Transport: identityTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Query().Get("actor") == profile.DID {
			close(firstProbed)
			response := httptest.NewRecorder()
			response.WriteHeader(http.StatusServiceUnavailable)
			return response.Result(), nil
		}
		close(barrierStarted)
		<-r.Context().Done()
		return nil, r.Context().Err()
	})}
	first := &atpjetstream.Event{Kind: "account", DID: profile.DID, Account: &atpjetstream.Account{DID: profile.DID, Status: "deleted"}}
	require.NoError(t, pw.handleEvent(ctx, first))
	select {
	case <-firstProbed:
	case <-time.After(time.Second):
		t.Fatal("account was not probed")
	}
	// The single worker enters the barrier only after caching the first result.
	require.NoError(t, pw.handleEvent(ctx, &atpjetstream.Event{Kind: "account", DID: "did:plc:barrier", Account: &atpjetstream.Account{Status: "deleted"}}))
	select {
	case <-barrierStarted:
	case <-time.After(time.Second):
		t.Fatal("verification barrier was not reached")
	}
	for i := 0; i < 100; i++ {
		require.NoError(t, pw.handleEvent(ctx, first))
	}
	pw.verificationMu.Lock()
	assert.Len(t, pw.verificationPending, 1, "negative cache must suppress repeated probes")
	pw.verificationMu.Unlock()
	changed := &atpjetstream.Event{Kind: "account", DID: profile.DID, Account: &atpjetstream.Account{DID: profile.DID, Status: "takendown"}}
	require.NoError(t, pw.handleEvent(ctx, changed))
	pw.verificationMu.Lock()
	assert.Len(t, pw.verificationPending, 2, "changed status must bypass the cached hint")
	pw.verificationMu.Unlock()
	cancel()
	select {
	case <-pw.verificationDone:
	case <-time.After(time.Second):
		t.Fatal("verification worker failed to stop")
	}
}
