package atproto

import (
	"context"
	"net/http"
	"time"

	"tangled.org/pdewey.com/atp"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Profile is a user's public profile. It is a type alias for atp.PublicProfile
// so existing callers continue to work without changes.
type Profile = atp.PublicProfile

// PublicClient wraps atp.PublicClient and exposes the same method signatures
// that arabica callers already use (GetRecord, ListRecords, etc.).
type PublicClient struct {
	inner *atp.PublicClient
}

// NewPublicClient creates a PublicClient with OTel-instrumented HTTP transport.
func NewPublicClient() *PublicClient {
	hc := &http.Client{
		Timeout:   30 * time.Second,
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
	return &PublicClient{inner: atp.NewPublicClientWithHTTP(hc)}
}

// GetPDSEndpoint resolves a DID to the user's PDS base URL.
func (c *PublicClient) GetPDSEndpoint(ctx context.Context, did string) (string, error) {
	return c.inner.GetPDSEndpoint(ctx, did)
}

// GetProfile fetches a user's public profile by DID or handle.
func (c *PublicClient) GetProfile(ctx context.Context, actor string) (*Profile, error) {
	return c.inner.GetProfile(ctx, actor)
}

// ResolveHandle resolves an AT Protocol handle to a DID.
func (c *PublicClient) ResolveHandle(ctx context.Context, handle string) (string, error) {
	return c.inner.ResolveHandle(ctx, handle)
}

// PublicListRecordsOutput represents the response from public listRecords API.
type PublicListRecordsOutput struct {
	Records []PublicRecordEntry `json:"records"`
	Cursor  *string             `json:"cursor,omitempty"`
}

// PublicRecordEntry represents a single record in the public listRecords response.
type PublicRecordEntry struct {
	URI   string         `json:"uri"`
	CID   string         `json:"cid"`
	Value map[string]any `json:"value"`
}

// ListRecords fetches public records from a user's repository via their PDS,
// newest-first.
func (c *PublicClient) ListRecords(ctx context.Context, did, collection string, limit int) (*PublicListRecordsOutput, error) {
	records, cursor, err := c.inner.ListPublicRecords(ctx, did, collection, atp.ListPublicRecordsOpts{
		Limit:   limit,
		Reverse: true,
	})
	if err != nil {
		return nil, err
	}

	out := &PublicListRecordsOutput{
		Records: make([]PublicRecordEntry, len(records)),
	}
	for i, r := range records {
		out.Records[i] = PublicRecordEntry{
			URI:   r.URI,
			CID:   r.CID,
			Value: r.Value,
		}
	}
	if cursor != "" {
		out.Cursor = &cursor
	}
	return out, nil
}

// GetRecord fetches a single public record from a user's PDS.
func (c *PublicClient) GetRecord(ctx context.Context, did, collection, rkey string) (*PublicRecordEntry, error) {
	r, err := c.inner.GetPublicRecord(ctx, did, collection, rkey)
	if err != nil {
		return nil, err
	}
	return &PublicRecordEntry{
		URI:   r.URI,
		CID:   r.CID,
		Value: r.Value,
	}, nil
}
