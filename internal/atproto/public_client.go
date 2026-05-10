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

// PublicClient wraps atp.PublicClient with arabica-specific HTTP transport
// (OTel instrumentation and custom User-Agent). All caching now lives upstream.
type PublicClient struct {
	*atp.PublicClient
}

// NewPublicClient creates a PublicClient with OTel-instrumented transport and
// the arabica User-Agent header.
func NewPublicClient() *PublicClient {
	hc := &http.Client{
		Timeout:   30 * time.Second,
		Transport: &userAgentTransport{base: otelhttp.NewTransport(http.DefaultTransport)},
	}
	return &PublicClient{PublicClient: atp.NewPublicClientWithHTTP(hc)}
}

// ListRecords fetches public records from a user's PDS, newest-first.
// Delegates to atp.PublicClient.ListPublicRecords.
func (c *PublicClient) ListRecords(ctx context.Context, did, collection string, limit int) (*PublicListRecordsOutput, error) {
	records, cursor, err := c.ListPublicRecords(ctx, did, collection, atp.ListPublicRecordsOpts{
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
// Delegates to atp.PublicClient.GetPublicRecord.
func (c *PublicClient) GetRecord(ctx context.Context, did, collection, rkey string) (*PublicRecordEntry, error) {
	r, err := c.GetPublicRecord(ctx, did, collection, rkey)
	if err != nil {
		return nil, err
	}
	return &PublicRecordEntry{
		URI:   r.URI,
		CID:   r.CID,
		Value: r.Value,
	}, nil
}

// ListAllRecords paginates through every record and returns them all.
// Delegates to atp.PublicClient.ListAllRecords.
func (c *PublicClient) ListAllRecords(ctx context.Context, did, collection string) ([]PublicRecordEntry, error) {
	records, err := c.PublicClient.ListAllRecords(ctx, did, collection)
	if err != nil {
		return nil, err
	}
	out := make([]PublicRecordEntry, len(records))
	for i, r := range records {
		out[i] = PublicRecordEntry{
			URI:   r.URI,
			CID:   r.CID,
			Value: r.Value,
		}
	}
	return out, nil
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
