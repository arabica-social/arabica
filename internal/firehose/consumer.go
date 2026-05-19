package firehose

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"tangled.org/arabica.social/arabica/internal/metrics"
	atpjetstream "tangled.org/pdewey.com/atp/jetstream"

	"github.com/rs/zerolog/log"
)

// JetstreamCommit represents the commit data within a Jetstream event.
type JetstreamCommit struct {
	Rev        string          `json:"rev"`
	Operation  string          `json:"operation"` // "create", "update", "delete"
	Collection string          `json:"collection"`
	RKey       string          `json:"rkey"`
	Record     json.RawMessage `json:"record,omitempty"`
	CID        string          `json:"cid"`
}

// JetstreamIdentity represents the identity payload of a Jetstream event.
// Emitted when a DID's handle or PDS endpoint changes.
type JetstreamIdentity struct {
	DID    string `json:"did"`
	Handle string `json:"handle,omitempty"`
	Seq    int64  `json:"seq"`
	Time   string `json:"time"`
}

// JetstreamAccount represents the account payload of a Jetstream event.
// Status is one of "active", "deleted", "deactivated", "suspended", "takendown".
type JetstreamAccount struct {
	Active bool   `json:"active"`
	DID    string `json:"did"`
	Seq    int64  `json:"seq"`
	Status string `json:"status,omitempty"`
	Time   string `json:"time"`
}

// JetstreamEvent represents an event from Jetstream. Retained as a package-local
// type so integration tests and ProfileWatcher can pass events through
// ProcessEvent without depending on the upstream atp/jetstream Event shape.
type JetstreamEvent struct {
	DID      string             `json:"did"`
	TimeUS   int64              `json:"time_us"`
	Kind     string             `json:"kind"` // "commit", "identity", "account"
	Commit   *JetstreamCommit   `json:"commit,omitempty"`
	Identity *JetstreamIdentity `json:"identity,omitempty"`
	Account  *JetstreamAccount  `json:"account,omitempty"`
}

// Consumer consumes events from Jetstream and indexes them. Connection
// lifecycle, decompression, and cursor persistence are delegated to the
// atp/jetstream package; this type owns the arabica-specific indexing
// pipeline and metrics.
type Consumer struct {
	config    *Config
	index     *FeedIndex
	wantedSet map[string]struct{} // membership lookup over config.WantedCollections
	upstream  *atpjetstream.Consumer
}

// NewConsumer creates a new Jetstream consumer
func NewConsumer(config *Config, index *FeedIndex) *Consumer {
	wantedSet := make(map[string]struct{}, len(config.WantedCollections))
	for _, coll := range config.WantedCollections {
		wantedSet[coll] = struct{}{}
	}

	c := &Consumer{
		config:    config,
		index:     index,
		wantedSet: wantedSet,
	}

	if cursor, err := index.GetCursor(context.Background()); err == nil && cursor > 0 {
		log.Info().Int64("cursor", cursor).Msg("firehose: loaded cursor from index")
	}

	c.upstream = atpjetstream.New(&atpjetstream.Config{
		Endpoints:         config.Endpoints,
		WantedCollections: config.WantedCollections,
		Compress:          config.Compress,
		CursorStore:       index,
		OnConnect: func() {
			metrics.FirehoseConnectionState.Set(1)
			log.Info().Str("endpoint", c.upstream.CurrentEndpoint()).Msg("firehose: connected to Jetstream")
			c.index.SetReady(true)
		},
		OnDisconnect: func() {
			metrics.FirehoseConnectionState.Set(0)
			log.Warn().Str("endpoint", c.upstream.CurrentEndpoint()).Msg("firehose: disconnected from Jetstream")
		},
		OnError: func(err error, _ *atpjetstream.Event) {
			metrics.FirehoseErrorsTotal.Inc()
			log.Warn().Err(err).Msg("firehose: event processing error")
		},
	}, c.handleEvent)

	return c
}

// Start begins consuming events in a background goroutine
func (c *Consumer) Start(ctx context.Context) {
	c.upstream.Start(ctx)
}

// Stop gracefully stops the consumer
func (c *Consumer) Stop() {
	c.upstream.Stop()
}

// IsConnected returns true if currently connected to Jetstream
func (c *Consumer) IsConnected() bool {
	return c.upstream.IsConnected()
}

// Stats returns consumer statistics
func (c *Consumer) Stats() (eventsReceived, bytesReceived int64) {
	return c.upstream.Stats()
}

// handleEvent bridges atp/jetstream events into the arabica indexing pipeline.
func (c *Consumer) handleEvent(_ context.Context, evt *atpjetstream.Event) error {
	if evt == nil || evt.Kind != "commit" || evt.Commit == nil {
		return nil
	}
	if !c.isWantedCollection(evt.Commit.Collection) {
		return nil
	}

	metrics.FirehoseEventsTotal.WithLabelValues(evt.Commit.Collection, evt.Commit.Operation).Inc()

	log.Debug().
		Str("did", evt.DID).
		Str("collection", evt.Commit.Collection).
		Str("operation", evt.Commit.Operation).
		Str("rkey", evt.Commit.RKey).
		Msg("firehose: processing event")

	return c.processCommit(JetstreamEvent{
		DID:    evt.DID,
		TimeUS: evt.TimeUS,
		Kind:   evt.Kind,
		Commit: &JetstreamCommit{
			Rev:        evt.Commit.Rev,
			Operation:  evt.Commit.Operation,
			Collection: evt.Commit.Collection,
			RKey:       evt.Commit.RKey,
			Record:     evt.Commit.Record,
			CID:        evt.Commit.CID,
		},
	})
}

// ProcessEvent processes a single Jetstream event through the indexing pipeline.
// Exported for use in integration tests where events are fed from a test PDS
// firehose rather than a live Jetstream connection.
func (c *Consumer) ProcessEvent(event JetstreamEvent) error {
	if event.Kind != "commit" || event.Commit == nil {
		return nil
	}
	if !c.isWantedCollection(event.Commit.Collection) {
		return nil
	}
	return c.processCommit(event)
}

// isWantedCollection reports whether the collection NSID is in this
// consumer's WantedCollections set. When the set is empty (e.g. test
// fixtures that didn't configure it), everything is accepted so the
// existing behavior of those tests is preserved.
func (c *Consumer) isWantedCollection(collection string) bool {
	if len(c.wantedSet) == 0 {
		return true
	}
	_, ok := c.wantedSet[collection]
	return ok
}

func (c *Consumer) processCommit(event JetstreamEvent) error {
	commit := event.Commit

	switch commit.Operation {
	case "create", "update":
		if commit.Record == nil {
			return nil
		}
		if err := c.index.UpsertRecord(
			context.Background(),
			event.DID,
			commit.Collection,
			commit.RKey,
			commit.CID,
			commit.Record,
			event.TimeUS,
		); err != nil {
			return fmt.Errorf("failed to upsert record: %w", err)
		}

		// Special handling for likes - index for counts. Matches any
		// app's like collection (arabica + oolong both use ".like" suffix).
		if strings.HasSuffix(commit.Collection, ".like") {
			var recordData map[string]any
			if err := json.Unmarshal(commit.Record, &recordData); err == nil {
				if subject, ok := recordData["subject"].(map[string]any); ok {
					if subjectURI, ok := subject["uri"].(string); ok {
						if err := c.index.UpsertLike(context.Background(), event.DID, commit.RKey, subjectURI); err != nil {
							log.Warn().Err(err).Str("did", event.DID).Str("subject", subjectURI).Msg("failed to index like")
						}
						// Create notification for the like
						c.index.CreateLikeNotification(event.DID, subjectURI)
					}
				}
			}
		}

		// Special handling for comments - index for counts and retrieval.
		// Matches any app's comment collection.
		if strings.HasSuffix(commit.Collection, ".comment") {
			var recordData map[string]any
			if err := json.Unmarshal(commit.Record, &recordData); err == nil {
				if subject, ok := recordData["subject"].(map[string]any); ok {
					if subjectURI, ok := subject["uri"].(string); ok {
						text, _ := recordData["text"].(string)
						var createdAt time.Time
						if createdAtStr, ok := recordData["createdAt"].(string); ok {
							if parsed, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
								createdAt = parsed
							} else {
								createdAt = time.Now()
							}
						} else {
							createdAt = time.Now()
						}
						// Extract optional parent URI for threading
						var parentURI string
						if parent, ok := recordData["parent"].(map[string]any); ok {
							parentURI, _ = parent["uri"].(string)
						}
						if err := c.index.UpsertComment(context.Background(), event.DID, commit.RKey, subjectURI, parentURI, commit.CID, text, createdAt); err != nil {
							log.Warn().Err(err).Str("did", event.DID).Str("subject", subjectURI).Msg("failed to index comment")
						}
						// Create notification for the comment
						c.index.CreateCommentNotification(event.DID, subjectURI, parentURI)
					}
				}
			}
		}

	case "delete":
		// Special handling for likes - need to look up subject URI before delete
		if strings.HasSuffix(commit.Collection, ".like") {
			// Try to get the existing record to find its subject
			if existingRecord, err := c.index.GetRecord(
				context.Background(),
				fmt.Sprintf("at://%s/%s/%s", event.DID, commit.Collection, commit.RKey),
			); err == nil && existingRecord != nil {
				var recordData map[string]any
				if err := json.Unmarshal(existingRecord.Record, &recordData); err == nil {
					if subject, ok := recordData["subject"].(map[string]any); ok {
						if subjectURI, ok := subject["uri"].(string); ok {
							if err := c.index.DeleteLike(context.Background(), event.DID, subjectURI); err != nil {
								log.Warn().Err(err).Str("did", event.DID).Str("subject", subjectURI).Msg("failed to delete like index")
							}
							c.index.DeleteLikeNotification(event.DID, subjectURI)
						}
					}
				}
			}
		}

		// Special handling for comments - need to look up subject URI before delete
		if strings.HasSuffix(commit.Collection, ".comment") {
			// Try to get the existing record to find its subject
			if existingRecord, err := c.index.GetRecord(
				context.Background(),
				fmt.Sprintf("at://%s/%s/%s", event.DID, commit.Collection, commit.RKey),
			); err == nil && existingRecord != nil {
				var recordData map[string]any
				if err := json.Unmarshal(existingRecord.Record, &recordData); err == nil {
					if subject, ok := recordData["subject"].(map[string]any); ok {
						if subjectURI, ok := subject["uri"].(string); ok {
							var parentURI string
							if parent, ok := recordData["parent"].(map[string]any); ok {
								parentURI, _ = parent["uri"].(string)
							}
							if err := c.index.DeleteComment(context.Background(), event.DID, commit.RKey, subjectURI); err != nil {
								log.Warn().Err(err).Str("did", event.DID).Str("subject", subjectURI).Msg("failed to delete comment index")
							}
							c.index.DeleteCommentNotification(event.DID, subjectURI, parentURI)
						}
					}
				}
			}
		}

		if err := c.index.DeleteRecord(
			context.Background(),
			event.DID,
			commit.Collection,
			commit.RKey,
		); err != nil {
			return fmt.Errorf("failed to delete record: %w", err)
		}
	}

	return nil
}

// BackfillDID backfills records for a specific DID using the consumer's
// configured WantedCollections (which come from app.NSIDs() at startup).
func (c *Consumer) BackfillDID(ctx context.Context, did string) error {
	return c.index.BackfillUser(ctx, did, c.config.WantedCollections)
}

// BackfilledDIDs returns the set of all DIDs that have been backfilled.
func (c *Consumer) BackfilledDIDs(ctx context.Context) (map[string]struct{}, error) {
	return c.index.BackfilledDIDs(ctx)
}
