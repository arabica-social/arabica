package atproto

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
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
	if err != nil {
		return nil, fmt.Errorf("failed to create record: %w", err)
	}

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
	if err != nil {
		return nil, fmt.Errorf("failed to get record: %w", err)
	}

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
	if err != nil {
		return nil, fmt.Errorf("failed to list records: %w", err)
	}

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

// PutRecordInput contains parameters for updating a record
type PutRecordInput struct {
	Collection string
	RKey       string
	Record     interface{}
}

// PutRecord updates an existing record in the user's repository
func (c *Client) PutRecord(ctx context.Context, did syntax.DID, sessionID string, input *PutRecordInput) error {
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
	if err != nil {
		return fmt.Errorf("failed to update record: %w", err)
	}

	return nil
}

// DeleteRecordInput contains parameters for deleting a record
type DeleteRecordInput struct {
	Collection string
	RKey       string
}

// DeleteRecord deletes a record from the user's repository
func (c *Client) DeleteRecord(ctx context.Context, did syntax.DID, sessionID string, input *DeleteRecordInput) error {
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
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	return nil
}
