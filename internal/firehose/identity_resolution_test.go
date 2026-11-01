package firehose

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/pdewey.com/atp"
)

func TestIdentityResolution_RefreshesCachedPreviousOwner(t *testing.T) {
	ctx := context.Background()
	profile := &atproto.Profile{DID: "did:plc:new", Handle: "shared.example"}
	previous := &atproto.Profile{DID: "did:plc:previous", Handle: profile.Handle}
	idx := identityTestIndex(t, profile, profile.DID)
	require.NoError(t, idx.Register(profile.DID))
	idx.StoreProfile(ctx, previous.DID, previous)
	idx.StoreProfile(ctx, profile.DID, &atproto.Profile{DID: profile.DID, Handle: "old.example"})
	resolvedDID := previous.DID
	requests := map[string]int{}
	idx.publicClient = atp.NewPublicClientWithHTTP(&http.Client{Transport: identityTransport(func(r *http.Request) (*http.Response, error) {
		response := httptest.NewRecorder()
		switch r.URL.Path {
		case "/xrpc/app.bsky.actor.getProfile":
			require.NoError(t, json.NewEncoder(response).Encode(profile))
		case "/xrpc/com.atproto.identity.resolveHandle":
			handle := r.URL.Query().Get("handle")
			requests[handle]++
			did := resolvedDID
			if handle == "unrelated.example" {
				did = "did:plc:unrelated"
			}
			require.NoError(t, json.NewEncoder(response).Encode(map[string]string{"did": did}))
		default:
			t.Errorf("unexpected public request: %s", r.URL)
			response.WriteHeader(http.StatusNotFound)
		}
		return response.Result(), nil
	})})
	got, err := idx.publicClient.ResolveHandle(ctx, profile.Handle)
	require.NoError(t, err)
	require.Equal(t, previous.DID, got)
	_, err = idx.publicClient.ResolveHandle(ctx, "unrelated.example")
	require.NoError(t, err)
	resolvedDID = profile.DID
	require.True(t, idx.verifyIdentityEvent(ctx, profile.DID, profile.Handle))
	assert.Equal(t, 2, requests[profile.Handle], "verification must bypass the previous owner's cached resolution")
	got, ok := idx.GetDIDByHandle(ctx, profile.Handle)
	require.True(t, ok)
	assert.Equal(t, profile.DID, got)
	_, ok = idx.GetDIDByHandle(ctx, "old.example")
	assert.False(t, ok)
	assert.False(t, idx.ProfileCachedInMemory(previous.DID))
	_, ok = idx.profileStorage.loadProfile(ctx, previous.DID)
	assert.False(t, ok)
	cached, ok := idx.profileStorage.loadProfile(ctx, profile.DID)
	require.True(t, ok)
	assert.Equal(t, profile.Handle, cached.Profile.Handle)
	_, err = idx.publicClient.ResolveHandle(ctx, "unrelated.example")
	require.NoError(t, err)
	assert.Equal(t, 1, requests["unrelated.example"], "verification must retain unrelated resolver cache entries")
}

func TestIdentityResolution_UnverifiedKeepsTrustedOwnership(t *testing.T) {
	for _, mode := range []string{"profile-lag", "profile-failure", "resolver-lag", "resolver-failure", "profile-did-mismatch", "empty-handle"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			profile := &atproto.Profile{DID: "did:plc:new", Handle: "shared.example"}
			previous := &atproto.Profile{DID: "did:plc:previous", Handle: profile.Handle}
			old := &atproto.Profile{DID: profile.DID, Handle: "old.example"}
			idx := identityTestIndex(t, profile, profile.DID)
			require.NoError(t, idx.Register(profile.DID))
			idx.StoreProfile(ctx, previous.DID, previous)
			idx.StoreProfile(ctx, old.DID, old)
			warming := true
			resolverCalls := 0
			idx.publicClient = atp.NewPublicClientWithHTTP(&http.Client{Transport: identityTransport(func(r *http.Request) (*http.Response, error) {
				response := httptest.NewRecorder()
				switch r.URL.Path {
				case "/xrpc/app.bsky.actor.getProfile":
					if mode == "profile-failure" {
						return nil, errors.New("AppView unavailable")
					}
					fetched := *profile
					switch mode {
					case "profile-lag":
						fetched.Handle = old.Handle
					case "profile-did-mismatch":
						fetched.DID = previous.DID
					case "empty-handle":
						fetched.Handle = ""
					}
					require.NoError(t, json.NewEncoder(response).Encode(&fetched))
				case "/xrpc/com.atproto.identity.resolveHandle":
					resolverCalls++
					if !warming && mode == "resolver-failure" {
						return nil, errors.New("resolver unavailable")
					}
					did := profile.DID
					if !warming && mode == "resolver-lag" {
						did = previous.DID
					}
					require.NoError(t, json.NewEncoder(response).Encode(map[string]string{"did": did}))
				default:
					t.Errorf("unexpected public request: %s", r.URL)
					response.WriteHeader(http.StatusNotFound)
				}
				return response.Result(), nil
			})})
			_, err := idx.publicClient.ResolveHandle(ctx, profile.Handle)
			require.NoError(t, err)
			warming = false
			assert.False(t, idx.verifyIdentityEvent(ctx, profile.DID, profile.Handle))
			if mode == "resolver-lag" || mode == "resolver-failure" {
				assert.Equal(t, 2, resolverCalls, "cached agreement cannot authorize an update")
			}
			for _, trusted := range []*atproto.Profile{previous, old} {
				got, ok := idx.GetDIDByHandle(ctx, trusted.Handle)
				require.True(t, ok)
				assert.Equal(t, trusted.DID, got)
				assert.True(t, idx.ProfileCachedInMemory(trusted.DID))
				cached, ok := idx.profileStorage.loadProfile(ctx, trusted.DID)
				require.True(t, ok)
				assert.Equal(t, trusted.Handle, cached.Profile.Handle)
			}
		})
	}
}

func TestIdentityResolution_CompletionResult(t *testing.T) {
	ctx := context.Background()
	profile := &atproto.Profile{DID: "did:plc:unknown", Handle: "unknown.example"}
	idx := identityTestIndex(t, profile, profile.DID)
	idx.publicClient = atp.NewPublicClientWithHTTP(&http.Client{Transport: identityTransport(func(r *http.Request) (*http.Response, error) {
		t.Errorf("unknown DID must not trigger verification: %s", r.URL)
		return nil, errors.New("unexpected request")
	})})
	assert.True(t, idx.verifyIdentityEvent(ctx, profile.DID, profile.Handle), "unknown DIDs need no retry")
	assert.False(t, idx.ProfileCachedInMemory(profile.DID))
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	assert.False(t, idx.verifyIdentityEvent(cancelled, profile.DID, profile.Handle), "query failure is not an ignored unknown DID")
}
