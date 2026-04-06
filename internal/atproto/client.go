package atproto

import (
	"context"
	"fmt"
	"net/http"

	"arabica/internal/metrics"
	"arabica/internal/tracing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"tangled.org/pdewey.com/atp"
)

// ErrSessionExpired is returned when the OAuth session cannot be resumed.
var ErrSessionExpired = atp.ErrSessionExpired

// wrapPDSError checks whether an error indicates an expired OAuth grant.
var wrapPDSError = atp.WrapPDSError

// Record represents a single record from a PDS.
type Record = atp.Record

// Client wraps the atproto API client for making authenticated requests to a PDS.
type Client struct {
	oauth *OAuthManager
}

// NewClient creates a new atproto client.
func NewClient(oauth *OAuthManager) *Client {
	return &Client{oauth: oauth}
}

// getAtpClient resumes an OAuth session and returns an atp.Client with OTel-instrumented transport.
func (c *Client) getAtpClient(ctx context.Context, did syntax.DID, sessionID string) (*atp.Client, error) {
	session, err := c.oauth.app.ResumeSession(ctx, did, sessionID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSessionExpired, err)
	}

	apiClient := session.APIClient()

	// Wrap transport with OTel instrumentation.
	baseTransport := apiClient.Client.Transport
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}
	apiClient.Client = &http.Client{
		Transport:     otelhttp.NewTransport(baseTransport),
		Timeout:       apiClient.Client.Timeout,
		CheckRedirect: apiClient.Client.CheckRedirect,
		Jar:           apiClient.Client.Jar,
	}

	return atp.NewClient(apiClient, did), nil
}

// --- Input/Output types (kept for caller compatibility) ---

type CreateRecordInput struct {
	Collection string
	Record     any
	RKey       *string
}

type CreateRecordOutput struct {
	URI string
	CID string
}

type GetRecordInput struct {
	Collection string
	RKey       string
}

type GetRecordOutput struct {
	URI   string
	CID   string
	Value map[string]any
}

type ListRecordsInput struct {
	Collection string
	Limit      *int64
	Cursor     *string
}

type ListRecordsOutput struct {
	Records []Record
	Cursor  *string
}

type PutRecordInput struct {
	Collection string
	RKey       string
	Record     any
}

type DeleteRecordInput struct {
	Collection string
	RKey       string
}

// --- CRUD methods ---

func (c *Client) CreateRecord(ctx context.Context, did syntax.DID, sessionID string, input *CreateRecordInput) (*CreateRecordOutput, error) {
	ctx, span := tracing.PdsSpan(ctx, "createRecord", input.Collection, did.String())
	defer span.End()

	atpClient, err := c.getAtpClient(ctx, did, sessionID)
	if err != nil {
		tracing.EndWithError(span, err)
		return nil, err
	}

	var uri, cid string
	if input.RKey != nil {
		uri, cid, err = atpClient.CreateRecordWithRKey(ctx, input.Collection, *input.RKey, input.Record)
	} else {
		uri, cid, err = atpClient.CreateRecord(ctx, input.Collection, input.Record)
	}
	metrics.PDSRequestsTotal.WithLabelValues("createRecord", input.Collection).Inc()

	if err != nil {
		tracing.EndWithError(span, err)
		log.Error().Err(err).Str("method", "createRecord").Str("collection", input.Collection).Str("did", did.String()).Msg("PDS request failed")
		return nil, fmt.Errorf("failed to create record: %w", err)
	}

	log.Debug().Str("method", "createRecord").Str("collection", input.Collection).Str("did", did.String()).Str("uri", uri).Str("cid", cid).Msg("PDS request completed")
	return &CreateRecordOutput{URI: uri, CID: cid}, nil
}

func (c *Client) GetRecord(ctx context.Context, did syntax.DID, sessionID string, input *GetRecordInput) (*GetRecordOutput, error) {
	ctx, span := tracing.PdsSpan(ctx, "getRecord", input.Collection, did.String())
	defer span.End()

	atpClient, err := c.getAtpClient(ctx, did, sessionID)
	if err != nil {
		tracing.EndWithError(span, err)
		return nil, err
	}

	rec, err := atpClient.GetRecord(ctx, input.Collection, input.RKey)
	metrics.PDSRequestsTotal.WithLabelValues("getRecord", input.Collection).Inc()

	if err != nil {
		tracing.EndWithError(span, err)
		log.Error().Err(err).Str("method", "getRecord").Str("collection", input.Collection).Str("rkey", input.RKey).Str("did", did.String()).Msg("PDS request failed")
		return nil, fmt.Errorf("failed to get record: %w", err)
	}

	log.Debug().Str("method", "getRecord").Str("collection", input.Collection).Str("rkey", input.RKey).Str("did", did.String()).Str("uri", rec.URI).Str("cid", rec.CID).Msg("PDS request completed")
	return &GetRecordOutput{URI: rec.URI, CID: rec.CID, Value: rec.Value}, nil
}

func (c *Client) ListRecords(ctx context.Context, did syntax.DID, sessionID string, input *ListRecordsInput) (*ListRecordsOutput, error) {
	ctx, span := tracing.PdsSpan(ctx, "listRecords", input.Collection, did.String())
	defer span.End()

	atpClient, err := c.getAtpClient(ctx, did, sessionID)
	if err != nil {
		tracing.EndWithError(span, err)
		return nil, err
	}

	var limit int
	if input.Limit != nil {
		limit = int(*input.Limit)
	}
	var cursor string
	if input.Cursor != nil {
		cursor = *input.Cursor
	}

	result, err := atpClient.ListRecords(ctx, input.Collection, limit, cursor)
	metrics.PDSRequestsTotal.WithLabelValues("listRecords", input.Collection).Inc()

	if err != nil {
		tracing.EndWithError(span, err)
		log.Error().Err(err).Str("method", "listRecords").Str("collection", input.Collection).Str("did", did.String()).Msg("PDS request failed")
		return nil, fmt.Errorf("failed to list records: %w", err)
	}

	logEvent := log.Debug().Str("method", "listRecords").Str("collection", input.Collection).Str("did", did.String()).Int("record_count", len(result.Records))
	if result.Cursor != "" {
		logEvent.Str("cursor", result.Cursor).Bool("has_more", true)
	} else {
		logEvent.Bool("has_more", false)
	}
	logEvent.Msg("PDS request completed")

	var cursorPtr *string
	if result.Cursor != "" {
		cursorPtr = &result.Cursor
	}

	return &ListRecordsOutput{
		Records: result.Records,
		Cursor:  cursorPtr,
	}, nil
}

func (c *Client) ListAllRecords(ctx context.Context, did syntax.DID, sessionID string, collection string) (*ListRecordsOutput, error) {
	ctx, span := tracing.PdsSpan(ctx, "listAllRecords", collection, did.String())
	defer span.End()

	atpClient, err := c.getAtpClient(ctx, did, sessionID)
	if err != nil {
		tracing.EndWithError(span, err)
		return nil, err
	}

	records, err := atpClient.ListAllRecords(ctx, collection)
	if err != nil {
		tracing.EndWithError(span, err)
		return nil, err
	}

	log.Info().Str("method", "listAllRecords").Str("collection", collection).Str("did", did.String()).Int("total_records", len(records)).Msg("PDS pagination completed")

	return &ListRecordsOutput{
		Records: records,
		Cursor:  nil,
	}, nil
}

func (c *Client) PutRecord(ctx context.Context, did syntax.DID, sessionID string, input *PutRecordInput) error {
	ctx, span := tracing.PdsSpan(ctx, "putRecord", input.Collection, did.String())
	defer span.End()

	atpClient, err := c.getAtpClient(ctx, did, sessionID)
	if err != nil {
		tracing.EndWithError(span, err)
		return err
	}

	_, _, err = atpClient.PutRecord(ctx, input.Collection, input.RKey, input.Record)
	metrics.PDSRequestsTotal.WithLabelValues("putRecord", input.Collection).Inc()

	if err != nil {
		tracing.EndWithError(span, err)
		log.Error().Err(err).Str("method", "putRecord").Str("collection", input.Collection).Str("rkey", input.RKey).Str("did", did.String()).Msg("PDS request failed")
		return fmt.Errorf("failed to update record: %w", err)
	}

	log.Debug().Str("method", "putRecord").Str("collection", input.Collection).Str("rkey", input.RKey).Str("did", did.String()).Msg("PDS request completed")
	return nil
}

func (c *Client) DeleteRecord(ctx context.Context, did syntax.DID, sessionID string, input *DeleteRecordInput) error {
	ctx, span := tracing.PdsSpan(ctx, "deleteRecord", input.Collection, did.String())
	defer span.End()

	atpClient, err := c.getAtpClient(ctx, did, sessionID)
	if err != nil {
		tracing.EndWithError(span, err)
		return err
	}

	err = atpClient.DeleteRecord(ctx, input.Collection, input.RKey)
	metrics.PDSRequestsTotal.WithLabelValues("deleteRecord", input.Collection).Inc()

	if err != nil {
		tracing.EndWithError(span, err)
		log.Error().Err(err).Str("method", "deleteRecord").Str("collection", input.Collection).Str("rkey", input.RKey).Str("did", did.String()).Msg("PDS request failed")
		return fmt.Errorf("failed to delete record: %w", err)
	}

	log.Debug().Str("method", "deleteRecord").Str("collection", input.Collection).Str("rkey", input.RKey).Str("did", did.String()).Msg("PDS request completed")
	return nil
}
