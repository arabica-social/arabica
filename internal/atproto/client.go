package atproto

import (
	"context"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"tangled.org/pdewey.com/atp"
)

// userAgent is prepended to the User-Agent header on all PDS requests so
// arabica traffic is identifiable in server logs.
const userAgent = "Arabica (+https://alpha.arabica.social; abuse@mail.arabica.systems)"

// userAgentTransport wraps an http.RoundTripper and prepends a custom
// User-Agent string to every outgoing request.
type userAgentTransport struct {
	base http.RoundTripper
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	if existing := r.Header.Get("User-Agent"); existing != "" {
		r.Header.Set("User-Agent", userAgent+" "+existing)
	} else {
		r.Header.Set("User-Agent", userAgent)
	}
	return t.base.RoundTrip(r)
}

// ErrSessionExpired is returned when the OAuth session cannot be resumed.
var ErrSessionExpired = atp.ErrSessionExpired

// Profile is a user's public profile. Type alias for backward compatibility.
type Profile = atp.PublicProfile

// NewPublicClient creates a public client with OTel-instrumented transport and
// the arabica User-Agent header.
func NewPublicClient() *atp.PublicClient {
	hc := &http.Client{
		Timeout:   30 * time.Second,
		Transport: &userAgentTransport{base: otelhttp.NewTransport(http.DefaultTransport)},
	}
	return atp.NewPublicClientWithHTTP(hc)
}

// ClientProvider returns an authenticated atp.Client for the given DID and session.
type ClientProvider func(ctx context.Context, did syntax.DID, sessionID string) (*atp.Client, error)

// Client wraps the atproto API client for making authenticated requests to a PDS.
type Client struct {
	getClient ClientProvider
}

// NewClient creates a new atproto client that authenticates via OAuth.
func NewClient(oauth *atp.OAuthApp) *Client {
	return &Client{getClient: oauthProvider(oauth)}
}

// NewClientWithProvider creates a client with a custom authentication provider.
// This is useful for testing with password-auth or pre-authenticated clients.
func NewClientWithProvider(provider ClientProvider) *Client {
	return &Client{getClient: provider}
}

// oauthProvider returns a ClientProvider that resumes OAuth sessions with OTel-instrumented transport.
func oauthProvider(oauth *atp.OAuthApp) ClientProvider {
	return func(ctx context.Context, did syntax.DID, sessionID string) (*atp.Client, error) {
		atpClient, err := oauth.ResumeSession(ctx, did, sessionID)
		if err != nil {
			return nil, err
		}

		apiClient := atpClient.APIClient()

		// Wrap transport with OTel instrumentation.
		baseTransport := apiClient.Client.Transport
		if baseTransport == nil {
			baseTransport = http.DefaultTransport
		}
		apiClient.Client = &http.Client{
			Transport:     &userAgentTransport{base: otelhttp.NewTransport(baseTransport)},
			Timeout:       apiClient.Client.Timeout,
			CheckRedirect: apiClient.Client.CheckRedirect,
			Jar:           apiClient.Client.Jar,
		}

		return atpClient, nil
	}
}

// getAtpClient returns an authenticated atp.Client using the configured provider.
func (c *Client) getAtpClient(ctx context.Context, did syntax.DID, sessionID string) (*atp.Client, error) {
	return c.getClient(ctx, did, sessionID)
}

// AtpClient returns an authenticated atp.Client for the given DID and session.
// The returned client is scoped to the provided DID for PDS operations.
func (c *Client) AtpClient(ctx context.Context, did syntax.DID, sessionID string) (*atp.Client, error) {
	return c.getAtpClient(ctx, did, sessionID)
}


