package atproto

import (
	"context"
	"fmt"
	"time"

	"arabica/internal/metrics"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/rs/zerolog/log"
)

// Client wraps the atproto API client for making authenticated requests to a PDS
type Client struct {
	oauth *OAuthManager
}

// NewClient creates a new atproto client
func NewClient(oauth *OAuthManager) *Client {
	return &Client{
		oauth: oauth,
	}
}

// getAuthenticatedAPIClient creates an authenticated API client for a specific session
// This properly handles DPOP token signing and refresh
func (c *Client) getAuthenticatedAPIClient(ctx context.Context, did syntax.DID, sessionID string) (*atclient.APIClient, error) {
	// Resume the OAuth session - this returns a ClientSession that handles DPOP
	session, err := c.oauth.app.ResumeSession(ctx, did, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to resume session: %w", err)
	}

	// Get the authenticated API client from the session
	// This client automatically handles DPOP signing and token refresh
	apiClient := session.APIClient()

	return apiClient, nil
}

// CreateRecordInput contains parameters for creating a record
type CreateRecordInput struct {
	Collection string
	Record     interface{}
	RKey       *string // Optional, if nil a TID will be generated
}

// CreateRecordOutput contains the result of creating a record
type CreateRecordOutput struct {
	URI string // AT-URI of the created record
	CID string // Content ID
}

// CreateRecord creates a new record in the user's repository
func (c *Client) CreateRecord(ctx context.Context, did syntax.DID, sessionID string, input *CreateRecordInput) (*CreateRecordOutput, error) {
	start := time.Now()

	apiClient, err := c.getAuthenticatedAPIClient(ctx, did, sessionID)
	if err != nil {
		return nil, err
	}

	// Build the request body
	body := map[string]interface{}{
		"repo":       did.String(),
		"collection": input.Collection,
		"record":     input.Record,
	}

	if input.RKey != nil {
		body["rkey"] = *input.RKey
	}

	// Use the API client's Post method to call com.atproto.repo.createRecord
	var result struct {
		URI string `json:"uri"`
		CID string `json:"cid"`
	}

	err = apiClient.Post(ctx, "com.atproto.repo.createRecord", body, &result)

	duration := time.Since(start)
	metrics.PDSRequestDuration.WithLabelValues("createRecord").Observe(duration.Seconds())
	metrics.PDSRequestsTotal.WithLabelValues("createRecord", input.Collection).Inc()

	if err != nil {
		metrics.PDSErrorsTotal.WithLabelValues("createRecord").Inc()
		log.Error().
			Err(err).
			Str("method", "createRecord").
			Str("collection", input.Collection).
			Str("did", did.String()).
			Dur("duration", duration).
			Msg("PDS request failed")
		return nil, fmt.Errorf("failed to create record: %w", err)
	}

	log.Debug().
		Str("method", "createRecord").
		Str("collection", input.Collection).
		Str("did", did.String()).
		Str("uri", result.URI).
		Str("cid", result.CID).
		Dur("duration", duration).
		Msg("PDS request completed")

	return &CreateRecordOutput{
		URI: result.URI,
		CID: result.CID,
	}, nil
}

// GetRecordInput contains parameters for getting a record
type GetRecordInput struct {
	Collection string
	RKey       string
}

// GetRecordOutput contains the result of getting a record
type GetRecordOutput struct {
	URI   string
	CID   string
	Value map[string]interface{}
}

// GetRecord retrieves a single record by its rkey
func (c *Client) GetRecord(ctx context.Context, did syntax.DID, sessionID string, input *GetRecordInput) (*GetRecordOutput, error) {
	start := time.Now()

	apiClient, err := c.getAuthenticatedAPIClient(ctx, did, sessionID)
	if err != nil {
		return nil, err
	}

	// Build query parameters
	params := map[string]any{
		"repo":       did.String(),
		"collection": input.Collection,
		"rkey":       input.RKey,
	}

	// Use the API client's Get method to call com.atproto.repo.getRecord
	var result struct {
		URI   string                 `json:"uri"`
		CID   string                 `json:"cid"`
		Value map[string]interface{} `json:"value"`
	}

	err = apiClient.Get(ctx, "com.atproto.repo.getRecord", params, &result)

	duration := time.Since(start)
	metrics.PDSRequestDuration.WithLabelValues("getRecord").Observe(duration.Seconds())
	metrics.PDSRequestsTotal.WithLabelValues("getRecord", input.Collection).Inc()

	if err != nil {
		metrics.PDSErrorsTotal.WithLabelValues("getRecord").Inc()
		log.Error().
			Err(err).
			Str("method", "getRecord").
			Str("collection", input.Collection).
			Str("rkey", input.RKey).
			Str("did", did.String()).
			Dur("duration", duration).
			Msg("PDS request failed")
		return nil, fmt.Errorf("failed to get record: %w", err)
	}

	log.Debug().
		Str("method", "getRecord").
		Str("collection", input.Collection).
		Str("rkey", input.RKey).
		Str("did", did.String()).
		Str("uri", result.URI).
		Str("cid", result.CID).
		Dur("duration", duration).
		Msg("PDS request completed")

	return &GetRecordOutput{
		URI:   result.URI,
		CID:   result.CID,
		Value: result.Value,
	}, nil
}

// ListRecordsInput contains parameters for listing records
type ListRecordsInput struct {
	Collection string
	Limit      *int64
	Cursor     *string
}

// ListRecordsOutput contains the result of listing records
type ListRecordsOutput struct {
	Records []Record
	Cursor  *string
}

// Record represents a single record in a list
type Record struct {
	URI   string
	CID   string
	Value map[string]interface{}
}

// ListRecords retrieves a list of records from a collection
func (c *Client) ListRecords(ctx context.Context, did syntax.DID, sessionID string, input *ListRecordsInput) (*ListRecordsOutput, error) {
	start := time.Now()

	apiClient, err := c.getAuthenticatedAPIClient(ctx, did, sessionID)
	if err != nil {
		return nil, err
	}

	// Build query parameters
	params := map[string]any{
		"repo":       did.String(),
		"collection": input.Collection,
	}

	if input.Limit != nil {
		params["limit"] = *input.Limit
	}
	if input.Cursor != nil {
		params["cursor"] = *input.Cursor
	}

	// Use the API client's Get method to call com.atproto.repo.listRecords
	var result struct {
		Records []struct {
			URI   string                 `json:"uri"`
			CID   string                 `json:"cid"`
			Value map[string]interface{} `json:"value"`
		} `json:"records"`
		Cursor *string `json:"cursor,omitempty"`
	}

	err = apiClient.Get(ctx, "com.atproto.repo.listRecords", params, &result)

	duration := time.Since(start)
	recordCount := len(result.Records)
	metrics.PDSRequestDuration.WithLabelValues("listRecords").Observe(duration.Seconds())
	metrics.PDSRequestsTotal.WithLabelValues("listRecords", input.Collection).Inc()

	if err != nil {
		metrics.PDSErrorsTotal.WithLabelValues("listRecords").Inc()
		log.Error().
			Err(err).
			Str("method", "listRecords").
			Str("collection", input.Collection).
			Str("did", did.String()).
			Dur("duration", duration).
			Msg("PDS request failed")
		return nil, fmt.Errorf("failed to list records: %w", err)
	}

	logEvent := log.Debug().
		Str("method", "listRecords").
		Str("collection", input.Collection).
		Str("did", did.String()).
		Int("record_count", recordCount).
		Dur("duration", duration)

	if result.Cursor != nil && *result.Cursor != "" {
		logEvent.Str("cursor", *result.Cursor).Bool("has_more", true)
	} else {
		logEvent.Bool("has_more", false)
	}

	logEvent.Msg("PDS request completed")

	// Convert to our output format
	records := make([]Record, len(result.Records))
	for i, r := range result.Records {
		records[i] = Record{
			URI:   r.URI,
			CID:   r.CID,
			Value: r.Value,
		}
	}

	return &ListRecordsOutput{
		Records: records,
		Cursor:  result.Cursor,
	}, nil
}

// ListAllRecords retrieves all records from a collection, handling pagination automatically
// This is useful when you need to fetch the complete collection without worrying about pagination
func (c *Client) ListAllRecords(ctx context.Context, did syntax.DID, sessionID string, collection string) (*ListRecordsOutput, error) {
	start := time.Now()
	var allRecords []Record
	var cursor *string
	pageCount := 0

	// ATProto typically returns up to 100 records per page by default
	// We'll request 100 at a time and paginate through all results
	limit := int64(100)

	for {
		// Check for context cancellation before each page request
		// This allows long-running pagination to be cancelled gracefully
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		output, err := c.ListRecords(ctx, did, sessionID, &ListRecordsInput{
			Collection: collection,
			Limit:      &limit,
			Cursor:     cursor,
		})
		if err != nil {
			return nil, err
		}

		allRecords = append(allRecords, output.Records...)
		pageCount++

		// If there's no cursor, we've fetched all records
		if output.Cursor == nil || *output.Cursor == "" {
			break
		}

		cursor = output.Cursor
	}

	duration := time.Since(start)

	log.Info().
		Str("method", "listAllRecords").
		Str("collection", collection).
		Str("did", did.String()).
		Int("total_records", len(allRecords)).
		Int("pages_fetched", pageCount).
		Dur("duration", duration).
		Msg("PDS pagination completed")

	return &ListRecordsOutput{
		Records: allRecords,
		Cursor:  nil, // All records fetched, no more pagination
	}, nil
}

// PutRecordInput contains parameters for updating a record
type PutRecordInput struct {
	Collection string
	RKey       string
	Record     interface{}
}

// PutRecord updates an existing record in the user's repository
func (c *Client) PutRecord(ctx context.Context, did syntax.DID, sessionID string, input *PutRecordInput) error {
	start := time.Now()

	apiClient, err := c.getAuthenticatedAPIClient(ctx, did, sessionID)
	if err != nil {
		return err
	}

	// Build the request body
	body := map[string]interface{}{
		"repo":       did.String(),
		"collection": input.Collection,
		"rkey":       input.RKey,
		"record":     input.Record,
	}

	// Use the API client's Post method to call com.atproto.repo.putRecord
	var result struct {
		URI string `json:"uri"`
		CID string `json:"cid"`
	}

	err = apiClient.Post(ctx, "com.atproto.repo.putRecord", body, &result)

	duration := time.Since(start)
	metrics.PDSRequestDuration.WithLabelValues("putRecord").Observe(duration.Seconds())
	metrics.PDSRequestsTotal.WithLabelValues("putRecord", input.Collection).Inc()

	if err != nil {
		metrics.PDSErrorsTotal.WithLabelValues("putRecord").Inc()
		log.Error().
			Err(err).
			Str("method", "putRecord").
			Str("collection", input.Collection).
			Str("rkey", input.RKey).
			Str("did", did.String()).
			Dur("duration", duration).
			Msg("PDS request failed")
		return fmt.Errorf("failed to update record: %w", err)
	}

	log.Debug().
		Str("method", "putRecord").
		Str("collection", input.Collection).
		Str("rkey", input.RKey).
		Str("did", did.String()).
		Str("uri", result.URI).
		Str("cid", result.CID).
		Dur("duration", duration).
		Msg("PDS request completed")

	return nil
}

// DeleteRecordInput contains parameters for deleting a record
type DeleteRecordInput struct {
	Collection string
	RKey       string
}

// DeleteRecord deletes a record from the user's repository
func (c *Client) DeleteRecord(ctx context.Context, did syntax.DID, sessionID string, input *DeleteRecordInput) error {
	start := time.Now()

	apiClient, err := c.getAuthenticatedAPIClient(ctx, did, sessionID)
	if err != nil {
		return err
	}

	// Build the request body
	body := map[string]interface{}{
		"repo":       did.String(),
		"collection": input.Collection,
		"rkey":       input.RKey,
	}

	// Use the API client's Post method to call com.atproto.repo.deleteRecord
	var result struct{}

	err = apiClient.Post(ctx, "com.atproto.repo.deleteRecord", body, &result)

	duration := time.Since(start)
	metrics.PDSRequestDuration.WithLabelValues("deleteRecord").Observe(duration.Seconds())
	metrics.PDSRequestsTotal.WithLabelValues("deleteRecord", input.Collection).Inc()

	if err != nil {
		metrics.PDSErrorsTotal.WithLabelValues("deleteRecord").Inc()
		log.Error().
			Err(err).
			Str("method", "deleteRecord").
			Str("collection", input.Collection).
			Str("rkey", input.RKey).
			Str("did", did.String()).
			Dur("duration", duration).
			Msg("PDS request failed")
		return fmt.Errorf("failed to delete record: %w", err)
	}

	log.Debug().
		Str("method", "deleteRecord").
		Str("collection", input.Collection).
		Str("rkey", input.RKey).
		Str("did", did.String()).
		Dur("duration", duration).
		Msg("PDS request completed")

	return nil
}
