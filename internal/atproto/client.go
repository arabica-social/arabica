package atproto

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/xrpc"
)

// Client wraps the XRPC client for making authenticated requests to a PDS
type Client struct {
	oauth *OAuthManager
}

// NewClient creates a new atproto client
func NewClient(oauth *OAuthManager) *Client {
	return &Client{
		oauth: oauth,
	}
}

// getAuthenticatedXRPCClient creates an XRPC client with authentication for a specific session
func (c *Client) getAuthenticatedXRPCClient(ctx context.Context, did syntax.DID, sessionID string) (*xrpc.Client, error) {
	// Get session data from OAuth store
	sessData, err := c.oauth.GetSession(ctx, did, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Create XRPC client pointing to the user's PDS
	client := &xrpc.Client{
		Host: sessData.HostURL,
	}

	// Set authentication using the session's access token
	// The OAuth library handles DPOP automatically
	client.Auth = &xrpc.AuthInfo{
		AccessJwt:  sessData.AccessToken,
		RefreshJwt: sessData.RefreshToken,
		Did:        sessData.AccountDID.String(),
		Handle:     "", // Optional, can be empty
	}

	return client, nil
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
	client, err := c.getAuthenticatedXRPCClient(ctx, did, sessionID)
	if err != nil {
		return nil, err
	}

	// Build the request
	params := map[string]interface{}{
		"repo":       did.String(),
		"collection": input.Collection,
		"record":     input.Record,
	}

	if input.RKey != nil {
		params["rkey"] = *input.RKey
	}

	// Make the XRPC call
	var result struct {
		URI string `json:"uri"`
		CID string `json:"cid"`
	}

	err = client.Do(ctx, xrpc.Procedure, "", "com.atproto.repo.createRecord", nil, params, &result)
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
	Value map[string]interface{} // The record data
}

// GetRecord retrieves a record from the user's repository
func (c *Client) GetRecord(ctx context.Context, did syntax.DID, sessionID string, input *GetRecordInput) (*GetRecordOutput, error) {
	client, err := c.getAuthenticatedXRPCClient(ctx, did, sessionID)
	if err != nil {
		return nil, err
	}

	// Build query parameters
	params := map[string]interface{}{
		"repo":       did.String(),
		"collection": input.Collection,
		"rkey":       input.RKey,
	}

	// Make the XRPC call
	var result struct {
		URI   string                 `json:"uri"`
		CID   string                 `json:"cid"`
		Value map[string]interface{} `json:"value"`
	}

	err = client.Do(ctx, xrpc.Query, "", "com.atproto.repo.getRecord", params, nil, &result)
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
	client, err := c.getAuthenticatedXRPCClient(ctx, did, sessionID)
	if err != nil {
		return nil, err
	}

	// Build query parameters
	params := map[string]interface{}{
		"repo":       did.String(),
		"collection": input.Collection,
	}

	if input.Limit != nil {
		params["limit"] = *input.Limit
	}
	if input.Cursor != nil {
		params["cursor"] = *input.Cursor
	}

	// Make the XRPC call
	var result struct {
		Records []struct {
			URI   string                 `json:"uri"`
			CID   string                 `json:"cid"`
			Value map[string]interface{} `json:"value"`
		} `json:"records"`
		Cursor *string `json:"cursor,omitempty"`
	}

	err = client.Do(ctx, xrpc.Query, "", "com.atproto.repo.listRecords", params, nil, &result)
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
	client, err := c.getAuthenticatedXRPCClient(ctx, did, sessionID)
	if err != nil {
		return err
	}

	// Build the request
	params := map[string]interface{}{
		"repo":       did.String(),
		"collection": input.Collection,
		"rkey":       input.RKey,
		"record":     input.Record,
	}

	// Make the XRPC call
	var result struct {
		URI string `json:"uri"`
		CID string `json:"cid"`
	}

	err = client.Do(ctx, xrpc.Procedure, "", "com.atproto.repo.putRecord", nil, params, &result)
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
	client, err := c.getAuthenticatedXRPCClient(ctx, did, sessionID)
	if err != nil {
		return err
	}

	// Build the request
	params := map[string]interface{}{
		"repo":       did.String(),
		"collection": input.Collection,
		"rkey":       input.RKey,
	}

	// Make the XRPC call
	err = client.Do(ctx, xrpc.Procedure, "", "com.atproto.repo.deleteRecord", nil, params, nil)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	return nil
}
