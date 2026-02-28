package firehose

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"arabica/internal/metrics"

	"github.com/gorilla/websocket"
	"github.com/klauspost/compress/zstd"
	"github.com/rs/zerolog/log"
)

// JetstreamEvent represents an event from Jetstream
type JetstreamEvent struct {
	DID    string `json:"did"`
	TimeUS int64  `json:"time_us"`
	Kind   string `json:"kind"` // "commit", "identity", "account"
	Commit *struct {
		Rev        string          `json:"rev"`
		Operation  string          `json:"operation"` // "create", "update", "delete"
		Collection string          `json:"collection"`
		RKey       string          `json:"rkey"`
		Record     json.RawMessage `json:"record,omitempty"`
		CID        string          `json:"cid"`
	} `json:"commit,omitempty"`
}

// Consumer consumes events from Jetstream and indexes them
type Consumer struct {
	config *Config
	index  *FeedIndex

	// Connection state
	conn               *websocket.Conn
	connMu             sync.Mutex
	currentEndpointIdx int

	// Zstd decoder for compressed messages
	zstdDecoder *zstd.Decoder

	// Cursor for resume
	cursor atomic.Int64

	// Stats
	eventsReceived atomic.Int64
	bytesReceived  atomic.Int64

	// Control
	connected atomic.Bool
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

// NewConsumer creates a new Jetstream consumer
func NewConsumer(config *Config, index *FeedIndex) *Consumer {
	// Create zstd decoder for compressed messages
	decoder, err := zstd.NewReader(nil, zstd.WithDecoderConcurrency(1))
	if err != nil {
		log.Fatal().Err(err).Msg("firehose: failed to create zstd decoder")
	}

	c := &Consumer{
		config:      config,
		index:       index,
		stopCh:      make(chan struct{}),
		zstdDecoder: decoder,
	}

	// Load cursor from index
	if cursor, err := index.GetCursor(); err == nil && cursor > 0 {
		c.cursor.Store(cursor)
		log.Info().Int64("cursor", cursor).Msg("firehose: loaded cursor from index")
	}

	return c
}

// Start begins consuming events in a background goroutine
func (c *Consumer) Start(ctx context.Context) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.run(ctx)
	}()
}

// Stop gracefully stops the consumer
func (c *Consumer) Stop() {
	close(c.stopCh)
	c.connMu.Lock()
	if c.conn != nil {
		c.conn.Close()
	}
	c.connMu.Unlock()
	c.wg.Wait()

	// Close zstd decoder
	if c.zstdDecoder != nil {
		c.zstdDecoder.Close()
	}
}

// IsConnected returns true if currently connected to Jetstream
func (c *Consumer) IsConnected() bool {
	return c.connected.Load()
}

// Stats returns consumer statistics
func (c *Consumer) Stats() (eventsReceived, bytesReceived int64) {
	return c.eventsReceived.Load(), c.bytesReceived.Load()
}

func (c *Consumer) run(ctx context.Context) {
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("firehose: context cancelled, stopping consumer")
			return
		case <-c.stopCh:
			log.Info().Msg("firehose: stop requested, stopping consumer")
			return
		default:
		}

		endpoint := c.config.Endpoints[c.currentEndpointIdx]
		err := c.connectAndConsume(ctx, endpoint)

		if err != nil {
			c.connected.Store(false)
			log.Warn().Err(err).Str("endpoint", endpoint).Msg("firehose: connection error")

			// Rotate to next endpoint
			c.currentEndpointIdx = (c.currentEndpointIdx + 1) % len(c.config.Endpoints)

			// Backoff before retry
			select {
			case <-ctx.Done():
				return
			case <-c.stopCh:
				return
			case <-time.After(backoff):
			}

			// Increase backoff
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		} else {
			// Reset backoff on successful connection
			backoff = time.Second
		}
	}
}

func (c *Consumer) connectAndConsume(ctx context.Context, endpoint string) error {
	// Build WebSocket URL with query parameters
	wsURL, err := c.buildWebSocketURL(endpoint)
	if err != nil {
		return fmt.Errorf("failed to build WebSocket URL: %w", err)
	}

	log.Info().Str("url", wsURL).Msg("firehose: connecting to Jetstream")

	// Connect
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()

	c.connected.Store(true)
	metrics.FirehoseConnectionState.Set(1)
	log.Info().Str("endpoint", endpoint).Msg("firehose: connected to Jetstream")

	// Mark index as ready once connected
	c.index.SetReady(true)

	defer func() {
		c.connMu.Lock()
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		c.connMu.Unlock()
		c.connected.Store(false)
		metrics.FirehoseConnectionState.Set(0)
	}()

	// Read events
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopCh:
			return nil
		default:
		}

		// Set read deadline
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		_, message, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read error: %w", err)
		}

		c.bytesReceived.Add(int64(len(message)))

		if err := c.processMessage(message); err != nil {
			metrics.FirehoseErrorsTotal.Inc()
			log.Warn().Err(err).Msg("firehose: failed to process message")
		}
	}
}

func (c *Consumer) buildWebSocketURL(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}

	q := u.Query()

	// Add wanted collections
	for _, coll := range c.config.WantedCollections {
		q.Add("wantedCollections", coll)
	}

	// Add compression
	if c.config.Compress {
		q.Set("compress", "true")
	}

	// Add cursor if we have one (rewind slightly for safety)
	cursor := c.cursor.Load()
	if cursor > 0 {
		// Rewind by 5 seconds to handle any gaps
		cursor -= 5 * time.Second.Microseconds()
		q.Set("cursor", fmt.Sprintf("%d", cursor))
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (c *Consumer) processMessage(data []byte) error {
	// Try to decompress if compression is enabled and data looks compressed
	// Zstd compressed data starts with magic number 0x28 0xB5 0x2F 0xFD
	if c.config.Compress && len(data) >= 4 && data[0] == 0x28 && data[1] == 0xB5 && data[2] == 0x2F && data[3] == 0xFD {
		decompressed, err := c.zstdDecoder.DecodeAll(data, nil)
		if err != nil {
			return fmt.Errorf("failed to decompress message: %w", err)
		}
		data = decompressed
	} else if c.config.Compress && len(data) > 0 && data[0] != '{' {
		// Try decompression anyway if it doesn't look like JSON
		decompressed, err := c.zstdDecoder.DecodeAll(data, nil)
		if err == nil {
			data = decompressed
		}
		// If decompression fails, try parsing as-is (maybe it's uncompressed)
	}

	var event JetstreamEvent
	if err := json.Unmarshal(data, &event); err != nil {
		// Log the first few bytes for debugging
		preview := data
		if len(preview) > 50 {
			preview = preview[:50]
		}
		return fmt.Errorf("failed to unmarshal event (first bytes: %q): %w", preview, err)
	}

	c.eventsReceived.Add(1)

	// Update cursor
	if event.TimeUS > 0 {
		c.cursor.Store(event.TimeUS)

		// Persist cursor periodically (every 1000 events)
		if c.eventsReceived.Load()%1000 == 0 {
			if err := c.index.SetCursor(event.TimeUS); err != nil {
				log.Warn().Err(err).Msg("firehose: failed to persist cursor")
			}
		}
	}

	// Only process commit events
	if event.Kind != "commit" || event.Commit == nil {
		return nil
	}

	commit := event.Commit

	// Verify it's an Arabica collection
	if !strings.HasPrefix(commit.Collection, "social.arabica.alpha.") {
		return nil
	}

	metrics.FirehoseEventsTotal.WithLabelValues(commit.Collection, commit.Operation).Inc()

	log.Debug().
		Str("did", event.DID).
		Str("collection", commit.Collection).
		Str("operation", commit.Operation).
		Str("rkey", commit.RKey).
		Msg("firehose: processing event")

	switch commit.Operation {
	case "create", "update":
		if commit.Record == nil {
			return nil
		}
		if err := c.index.UpsertRecord(
			event.DID,
			commit.Collection,
			commit.RKey,
			commit.CID,
			commit.Record,
			event.TimeUS,
		); err != nil {
			return fmt.Errorf("failed to upsert record: %w", err)
		}

		// Special handling for likes - index for counts
		if commit.Collection == "social.arabica.alpha.like" {
			var recordData map[string]interface{}
			if err := json.Unmarshal(commit.Record, &recordData); err == nil {
				if subject, ok := recordData["subject"].(map[string]interface{}); ok {
					if subjectURI, ok := subject["uri"].(string); ok {
						if err := c.index.UpsertLike(event.DID, commit.RKey, subjectURI); err != nil {
							log.Warn().Err(err).Str("did", event.DID).Str("subject", subjectURI).Msg("failed to index like")
						}
						// Create notification for the like
						c.index.CreateLikeNotification(event.DID, subjectURI)
					}
				}
			}
		}

		// Special handling for comments - index for counts and retrieval
		if commit.Collection == "social.arabica.alpha.comment" {
			var recordData map[string]interface{}
			if err := json.Unmarshal(commit.Record, &recordData); err == nil {
				if subject, ok := recordData["subject"].(map[string]interface{}); ok {
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
						if parent, ok := recordData["parent"].(map[string]interface{}); ok {
							parentURI, _ = parent["uri"].(string)
						}
						if err := c.index.UpsertComment(event.DID, commit.RKey, subjectURI, parentURI, commit.CID, text, createdAt); err != nil {
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
		if commit.Collection == "social.arabica.alpha.like" {
			// Try to get the existing record to find its subject
			if existingRecord, err := c.index.GetRecord(
				fmt.Sprintf("at://%s/%s/%s", event.DID, commit.Collection, commit.RKey),
			); err == nil && existingRecord != nil {
				var recordData map[string]interface{}
				if err := json.Unmarshal(existingRecord.Record, &recordData); err == nil {
					if subject, ok := recordData["subject"].(map[string]interface{}); ok {
						if subjectURI, ok := subject["uri"].(string); ok {
							if err := c.index.DeleteLike(event.DID, subjectURI); err != nil {
								log.Warn().Err(err).Str("did", event.DID).Str("subject", subjectURI).Msg("failed to delete like index")
							}
							c.index.DeleteLikeNotification(event.DID, subjectURI)
						}
					}
				}
			}
		}

		// Special handling for comments - need to look up subject URI before delete
		if commit.Collection == "social.arabica.alpha.comment" {
			// Try to get the existing record to find its subject
			if existingRecord, err := c.index.GetRecord(
				fmt.Sprintf("at://%s/%s/%s", event.DID, commit.Collection, commit.RKey),
			); err == nil && existingRecord != nil {
				var recordData map[string]interface{}
				if err := json.Unmarshal(existingRecord.Record, &recordData); err == nil {
					if subject, ok := recordData["subject"].(map[string]interface{}); ok {
						if subjectURI, ok := subject["uri"].(string); ok {
							var parentURI string
							if parent, ok := recordData["parent"].(map[string]interface{}); ok {
								parentURI, _ = parent["uri"].(string)
							}
							if err := c.index.DeleteComment(event.DID, commit.RKey, subjectURI); err != nil {
								log.Warn().Err(err).Str("did", event.DID).Str("subject", subjectURI).Msg("failed to delete comment index")
							}
							c.index.DeleteCommentNotification(event.DID, subjectURI, parentURI)
						}
					}
				}
			}
		}

		if err := c.index.DeleteRecord(
			event.DID,
			commit.Collection,
			commit.RKey,
		); err != nil {
			return fmt.Errorf("failed to delete record: %w", err)
		}
	}

	return nil
}

// BackfillDID backfills records for a specific DID
func (c *Consumer) BackfillDID(ctx context.Context, did string) error {
	return c.index.BackfillUser(ctx, did)
}
