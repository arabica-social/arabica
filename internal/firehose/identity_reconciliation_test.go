package firehose

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/pdewey.com/atp"
)

func reconciliationWatcher(t *testing.T, idx *FeedIndex) (*ProfileWatcher, context.Context) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	pw := NewProfileWatcher(&Config{}, idx)
	pw.verificationRetryDelays = []time.Duration{5 * time.Millisecond, 10 * time.Millisecond}
	t.Cleanup(func() {
		cancel()
		if pw.verificationDone != nil {
			select {
			case <-pw.verificationDone:
			case <-time.After(time.Second):
				t.Error("verification worker did not stop")
			}
		}
	})
	return pw, ctx
}

func identityHint(did, handle string) JetstreamEvent {
	return JetstreamEvent{Kind: "identity", DID: did, Identity: &JetstreamIdentity{DID: did, Handle: handle}}
}

func TestIdentityReconciliation_RetriesUntilAuthorityConverges(t *testing.T) {
	for _, failure := range []string{"profile-lag", "resolver-lag", "unavailable"} {
		t.Run(failure, func(t *testing.T) {
			profile := &atproto.Profile{DID: "did:plc:known", Handle: "new.example"}
			idx := identityTestIndex(t, profile, profile.DID)
			require.NoError(t, idx.Register(profile.DID))
			idx.StoreProfile(context.Background(), profile.DID, &atproto.Profile{DID: profile.DID, Handle: "old.example"})
			var profiles, resolutions atomic.Int32
			idx.publicClient = atp.NewPublicClientWithHTTP(&http.Client{Transport: identityTransport(func(r *http.Request) (*http.Response, error) {
				response := httptest.NewRecorder()
				switch r.URL.Path {
				case "/xrpc/app.bsky.actor.getProfile":
					attempt := profiles.Add(1)
					if attempt == 1 && failure == "unavailable" {
						return nil, context.DeadlineExceeded
					}
					result := *profile
					if attempt == 1 && failure == "profile-lag" {
						result.Handle = "old.example"
					}
					_ = json.NewEncoder(response).Encode(result)
				case "/xrpc/com.atproto.identity.resolveHandle":
					did := profile.DID
					if resolutions.Add(1) == 1 && failure == "resolver-lag" {
						did = "did:plc:previous"
					}
					_ = json.NewEncoder(response).Encode(map[string]string{"did": did})
				default:
					t.Errorf("unexpected request %s", r.URL)
				}
				return response.Result(), nil
			})})
			pw, ctx := reconciliationWatcher(t, idx)
			pw.enqueueVerification(ctx, identityHint(profile.DID, profile.Handle))
			require.Eventually(t, func() bool {
				did, ok := idx.GetDIDByHandle(ctx, profile.Handle)
				return ok && did == profile.DID
			}, time.Second, time.Millisecond)
			assert.EqualValues(t, 2, profiles.Load())
			_, ok := idx.GetDIDByHandle(ctx, "old.example")
			assert.False(t, ok)
		})
	}
}

func TestIdentityReconciliation_ChangedInFlightHintIsRetained(t *testing.T) {
	profile := &atproto.Profile{DID: "did:plc:known", Handle: "latest.example"}
	idx := identityTestIndex(t, profile, profile.DID)
	require.NoError(t, idx.Register(profile.DID))
	idx.StoreProfile(context.Background(), profile.DID, &atproto.Profile{DID: profile.DID, Handle: "old.example"})
	started, release := make(chan struct{}), make(chan struct{})
	var requests atomic.Int32
	idx.publicClient = atp.NewPublicClientWithHTTP(&http.Client{Transport: identityTransport(func(r *http.Request) (*http.Response, error) {
		response := httptest.NewRecorder()
		if r.URL.Path == "/xrpc/app.bsky.actor.getProfile" {
			if requests.Add(1) == 1 {
				close(started)
				select {
				case <-release:
				case <-r.Context().Done():
					return nil, r.Context().Err()
				}
			}
			_ = json.NewEncoder(response).Encode(profile)
		} else {
			_ = json.NewEncoder(response).Encode(map[string]string{"did": profile.DID})
		}
		return response.Result(), nil
	})})
	pw, ctx := reconciliationWatcher(t, idx)
	pw.enqueueVerification(ctx, identityHint(profile.DID, "intermediate.example"))
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("verification never started")
	}
	pw.enqueueVerification(ctx, identityHint(profile.DID, "second.example"))
	pw.enqueueVerification(ctx, identityHint(profile.DID, profile.Handle))
	close(release)
	require.Eventually(t, func() bool {
		did, ok := idx.GetDIDByHandle(ctx, profile.Handle)
		return ok && did == profile.DID
	}, time.Second, time.Millisecond)
	assert.EqualValues(t, 2, requests.Load())
}

func TestIdentityReconciliation_AccountHintCannotBeOverwritten(t *testing.T) {
	profile := &atproto.Profile{DID: "did:plc:known", Handle: "known.example"}
	idx := identityTestIndex(t, profile, profile.DID)
	require.NoError(t, idx.Register(profile.DID))
	started := make(chan struct{})
	release := make(chan struct{})
	accountProbed := make(chan struct{}, 1)
	idx.publicClient = atp.NewPublicClientWithHTTP(&http.Client{Transport: identityTransport(func(r *http.Request) (*http.Response, error) {
		select {
		case <-started:
		default:
			close(started)
		}

		select {
		case <-release:
			return nil, context.DeadlineExceeded
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}
	})})
	pw, ctx := reconciliationWatcher(t, idx)
	pw.accountClient = &http.Client{Transport: identityTransport(func(r *http.Request) (*http.Response, error) {
		accountProbed <- struct{}{}
		response := httptest.NewRecorder()
		_, _ = response.WriteString(`{}`)
		return response.Result(), nil
	})}
	pw.enqueueVerification(ctx, identityHint(profile.DID, profile.Handle))
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("verification never started")
	}
	pw.enqueueVerification(ctx, accountEvent(profile.DID, "deleted"))
	pw.enqueueVerification(ctx, identityHint(profile.DID, "changed.example"))
	pw.verificationMu.Lock()
	assert.Len(t, pw.verificationPending, 2)

	assert.Equal(t, "deleted", pw.verificationPending[verificationKey{did: profile.DID, kind: "account"}].event.Account.Status)
	pw.verificationMu.Unlock()
	close(release)
	select {
	case <-accountProbed:
	case <-time.After(time.Second):
		t.Fatal("account hint was not processed after identity verification")
	}
}

func TestIdentityReconciliation_RetriesAreBounded(t *testing.T) {
	profile := &atproto.Profile{DID: "did:plc:known", Handle: "old.example"}
	idx := identityTestIndex(t, profile, profile.DID)
	require.NoError(t, idx.Register(profile.DID))
	idx.StoreProfile(context.Background(), profile.DID, profile)
	var requests atomic.Int32
	idx.publicClient = atp.NewPublicClientWithHTTP(&http.Client{Transport: identityTransport(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		return nil, context.DeadlineExceeded
	})})
	pw, ctx := reconciliationWatcher(t, idx)
	hint := identityHint(profile.DID, "new.example")
	pw.enqueueVerification(ctx, hint)
	require.Eventually(t, func() bool {
		pw.verificationMu.Lock()
		defer pw.verificationMu.Unlock()
		return len(pw.verificationRecent) == 1 && len(pw.verificationPending) == 0
	}, time.Second, time.Millisecond)
	assert.EqualValues(t, 3, requests.Load())
	pw.enqueueVerification(ctx, hint)
	pw.verificationMu.Lock()
	assert.Empty(t, pw.verificationPending)
	pw.verificationMu.Unlock()
	did, ok := idx.GetDIDByHandle(ctx, profile.Handle)
	assert.True(t, ok)
	assert.Equal(t, profile.DID, did)
	_, ok = idx.GetDIDByHandle(ctx, "new.example")
	assert.False(t, ok)
}
