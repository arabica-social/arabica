package atproto

import (
	"context"
	"net/http"
	"sync"
	"time"

	"tangled.org/pdewey.com/atp"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Profile is a user's public profile. It is a type alias for atp.PublicProfile
// so existing callers continue to work without changes.
type Profile = atp.PublicProfile

const resolverCacheTTL = time.Hour

type cachedValue struct {
	value  string
	expiry time.Time
}

// PublicClient wraps atp.PublicClient and exposes the same method signatures
// that arabica callers already use (GetRecord, ListRecords, etc.).
type PublicClient struct {
	inner *atp.PublicClient

	pdsMu   sync.RWMutex
	pdsCache map[string]cachedValue // DID → PDS URL

	handleMu    sync.RWMutex
	handleCache map[string]cachedValue // handle → DID
}

// NewPublicClient creates a PublicClient with OTel-instrumented HTTP transport.
func NewPublicClient() *PublicClient {
	hc := &http.Client{
		Timeout:   30 * time.Second,
		Transport: &userAgentTransport{base: otelhttp.NewTransport(http.DefaultTransport)},
	}
	return &PublicClient{
		inner:       atp.NewPublicClientWithHTTP(hc),
		pdsCache:    make(map[string]cachedValue),
		handleCache: make(map[string]cachedValue),
	}
}

// GetPDSEndpoint resolves a DID to the user's PDS base URL.
func (c *PublicClient) GetPDSEndpoint(ctx context.Context, did string) (string, error) {
	c.pdsMu.RLock()
	if v, ok := c.pdsCache[did]; ok && time.Now().Before(v.expiry) {
		c.pdsMu.RUnlock()
		return v.value, nil
	}
	c.pdsMu.RUnlock()

	url, err := c.inner.GetPDSEndpoint(ctx, did)
	if err != nil {
		return "", err
	}

	c.pdsMu.Lock()
	c.pdsCache[did] = cachedValue{value: url, expiry: time.Now().Add(resolverCacheTTL)}
	c.pdsMu.Unlock()
	return url, nil
}

// GetProfile fetches a user's public profile by DID or handle.
func (c *PublicClient) GetProfile(ctx context.Context, actor string) (*Profile, error) {
	return c.inner.GetProfile(ctx, actor)
}

// ResolveHandle resolves an AT Protocol handle to a DID.
func (c *PublicClient) ResolveHandle(ctx context.Context, handle string) (string, error) {
	c.handleMu.RLock()
	if v, ok := c.handleCache[handle]; ok && time.Now().Before(v.expiry) {
		c.handleMu.RUnlock()
		return v.value, nil
	}
	c.handleMu.RUnlock()

	did, err := c.inner.ResolveHandle(ctx, handle)
	if err != nil {
		return "", err
	}

	c.handleMu.Lock()
	c.handleCache[handle] = cachedValue{value: did, expiry: time.Now().Add(resolverCacheTTL)}
	c.handleMu.Unlock()
	return did, nil
}

// InvalidateHandle removes a handle from the resolver cache so the next
// ResolveHandle call refetches from the directory. Called when a firehose
// identity event signals that a handle's DID mapping has changed.
func (c *PublicClient) InvalidateHandle(handle string) {
	if handle == "" {
		return
	}
	c.handleMu.Lock()
	delete(c.handleCache, handle)
	c.handleMu.Unlock()
}

// InvalidateDID drops any cached entries pointing at this DID — both the
// PDS endpoint cache and any handle→DID mappings whose resolved DID is the
// given one. Used when a DID's repo is gone (account deleted/takendown) or
// when a handle has been reassigned away from this DID.
func (c *PublicClient) InvalidateDID(did string) {
	if did == "" {
		return
	}
	c.pdsMu.Lock()
	delete(c.pdsCache, did)
	c.pdsMu.Unlock()

	c.handleMu.Lock()
	for h, v := range c.handleCache {
		if v.value == did {
			delete(c.handleCache, h)
		}
	}
	c.handleMu.Unlock()
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

// ListAllRecords paginates through every record in a collection on the user's
// PDS and returns them all, newest-first. Use for moderation tools that need
// the full repo state for a user; for normal feed paths prefer the witness
// cache or a single-page ListRecords.
func (c *PublicClient) ListAllRecords(ctx context.Context, did, collection string) ([]PublicRecordEntry, error) {
	const pageSize = 100
	const maxPages = 100 // hard ceiling against runaway loops; 10k records per collection is plenty.

	var all []PublicRecordEntry
	cursor := ""
	for page := 0; page < maxPages; page++ {
		records, next, err := c.inner.ListPublicRecords(ctx, did, collection, atp.ListPublicRecordsOpts{
			Limit:   pageSize,
			Cursor:  cursor,
			Reverse: true,
		})
		if err != nil {
			return nil, err
		}
		for _, r := range records {
			all = append(all, PublicRecordEntry{
				URI:   r.URI,
				CID:   r.CID,
				Value: r.Value,
			})
		}
		if next == "" || len(records) == 0 {
			return all, nil
		}
		cursor = next
	}
	return all, nil
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
