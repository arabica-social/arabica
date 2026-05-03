package firehose

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// ProfileWatcher is a dedicated Jetstream connection that subscribes to
// app.bsky.actor.profile events for known Arabica users only. Because Jetstream
// uses AND semantics when both wantedCollections and wantedDids are set, this
// must be a separate connection from the main consumer (which has no DID filter).
type ProfileWatcher struct {
	index     *FeedIndex
	endpoints []string

	conn   *websocket.Conn
	connMu sync.Mutex

	watchedDIDs   map[string]struct{}
	watchedDIDsMu sync.RWMutex

	endpointIdx int
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

type profileOptionsUpdate struct {
	Type    string `json:"type"`
	Payload struct {
		WantedCollections []string `json:"wantedCollections"`
		WantedDids        []string `json:"wantedDids"`
	} `json:"payload"`
}

// NewProfileWatcher creates a ProfileWatcher seeded with all currently known
// Arabica DIDs from the index.
func NewProfileWatcher(config *Config, index *FeedIndex) *ProfileWatcher {
	dids, _ := index.GetKnownDIDs(context.Background())
	watched := make(map[string]struct{}, len(dids))
	for _, did := range dids {
		watched[did] = struct{}{}
	}
	return &ProfileWatcher{
		index:       index,
		endpoints:   config.Endpoints,
		watchedDIDs: watched,
		stopCh:      make(chan struct{}),
	}
}

// Watch adds a DID to the subscription. If connected, an options update is sent
// immediately so Jetstream begins delivering that user's profile events.
func (pw *ProfileWatcher) Watch(did string) {
	pw.watchedDIDsMu.Lock()
	_, already := pw.watchedDIDs[did]
	pw.watchedDIDs[did] = struct{}{}
	pw.watchedDIDsMu.Unlock()

	if !already {
		pw.sendOptionsUpdate()
	}
}

// Unwatch removes a DID from the subscription. Used when an account is deleted
// so Jetstream stops sending events for that DID.
func (pw *ProfileWatcher) Unwatch(did string) {
	pw.watchedDIDsMu.Lock()
	_, present := pw.watchedDIDs[did]
	delete(pw.watchedDIDs, did)
	pw.watchedDIDsMu.Unlock()

	if present {
		pw.sendOptionsUpdate()
	}
}

// Start begins the profile watcher in a background goroutine. It will reconnect
// automatically on failure, rotating through endpoints with exponential backoff.
func (pw *ProfileWatcher) Start(ctx context.Context) {
	pw.wg.Go(func() {
		pw.run(ctx)
	})
}

// Stop gracefully shuts down the watcher.
func (pw *ProfileWatcher) Stop() {
	close(pw.stopCh)
	pw.connMu.Lock()
	if pw.conn != nil {
		pw.conn.Close()
	}
	pw.connMu.Unlock()
	pw.wg.Wait()
}

func (pw *ProfileWatcher) run(ctx context.Context) {
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		case <-pw.stopCh:
			return
		default:
		}

		// Skip connecting if we have no DIDs to watch yet — wait for the first Watch() call
		pw.watchedDIDsMu.RLock()
		n := len(pw.watchedDIDs)
		pw.watchedDIDsMu.RUnlock()
		if n == 0 {
			select {
			case <-ctx.Done():
				return
			case <-pw.stopCh:
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}

		endpoint := pw.endpoints[pw.endpointIdx]
		err := pw.connectAndConsume(ctx, endpoint)

		if err != nil {
			log.Warn().Err(err).Str("endpoint", endpoint).Msg("profile watcher: connection error")
			pw.endpointIdx = (pw.endpointIdx + 1) % len(pw.endpoints)

			select {
			case <-ctx.Done():
				return
			case <-pw.stopCh:
				return
			case <-time.After(backoff):
			}

			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		} else {
			backoff = time.Second
		}
	}
}

func (pw *ProfileWatcher) connectAndConsume(ctx context.Context, endpoint string) error {
	wsURL := pw.buildURL(endpoint)
	log.Info().Str("url", wsURL).Msg("profile watcher: connecting")

	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	pw.connMu.Lock()
	pw.conn = conn
	pw.connMu.Unlock()

	log.Info().Str("endpoint", endpoint).Msg("profile watcher: connected")

	defer func() {
		pw.connMu.Lock()
		if pw.conn != nil {
			pw.conn.Close()
			pw.conn = nil
		}
		pw.connMu.Unlock()
	}()

	const pingInterval = 30 * time.Second
	const readTimeout = 90 * time.Second

	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(readTimeout))
		return nil
	})

	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	go func() {
		for {
			select {
			case <-pingTicker.C:
				pw.connMu.Lock()
				c := pw.conn
				pw.connMu.Unlock()
				if c == nil {
					return
				}
				c.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := c.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			case <-ctx.Done():
				return
			case <-pw.stopCh:
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-pw.stopCh:
			return nil
		default:
		}

		conn.SetReadDeadline(time.Now().Add(readTimeout))
		_, message, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read error: %w", err)
		}

		pw.processMessage(message)
	}
}

func (pw *ProfileWatcher) buildURL(endpoint string) string {
	u, _ := url.Parse(endpoint)
	q := u.Query()
	q.Set("wantedCollections", NSIDBlueskyProfile)

	pw.watchedDIDsMu.RLock()
	for did := range pw.watchedDIDs {
		q.Add("wantedDids", did)
	}
	pw.watchedDIDsMu.RUnlock()

	u.RawQuery = q.Encode()
	return u.String()
}

func (pw *ProfileWatcher) sendOptionsUpdate() {
	pw.connMu.Lock()
	conn := pw.conn
	pw.connMu.Unlock()

	if conn == nil {
		return // will be applied via URL on next reconnect
	}

	pw.watchedDIDsMu.RLock()
	dids := make([]string, 0, len(pw.watchedDIDs))
	for did := range pw.watchedDIDs {
		dids = append(dids, did)
	}
	pw.watchedDIDsMu.RUnlock()

	var msg profileOptionsUpdate
	msg.Type = "options_update"
	msg.Payload.WantedCollections = []string{NSIDBlueskyProfile}
	msg.Payload.WantedDids = dids

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	pw.connMu.Lock()
	defer pw.connMu.Unlock()
	if pw.conn != nil {
		if err := pw.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Warn().Err(err).Msg("profile watcher: failed to send options update")
		}
	}
}

func (pw *ProfileWatcher) processMessage(data []byte) {
	var event JetstreamEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return
	}

	switch event.Kind {
	case "commit":
		if event.Commit == nil || event.Commit.Collection != NSIDBlueskyProfile {
			return
		}
		if event.Commit.Operation == "create" || event.Commit.Operation == "update" {
			pw.index.RefreshProfile(context.Background(), event.DID)
			log.Debug().Str("did", event.DID).Msg("profile watcher: refreshed profile cache")
		}

	case "identity":
		// Handle change or PDS migration — refresh the cached profile so handle
		// resolution stays accurate. Profile-commit events don't fire on handle
		// changes, so this is the only signal we get.
		pw.index.RefreshProfile(context.Background(), event.DID)
		handle := ""
		if event.Identity != nil {
			handle = event.Identity.Handle
		}
		log.Info().Str("did", event.DID).Str("handle", handle).Msg("profile watcher: identity update, refreshed profile")

	case "account":
		if event.Account == nil {
			return
		}
		status := event.Account.Status
		log.Info().
			Str("did", event.DID).
			Str("status", status).
			Bool("active", event.Account.Active).
			Msg("profile watcher: account event")

		// Only act on terminal states. deactivated/suspended are reversible —
		// we keep the data so it reappears if the account comes back.
		if status == "deleted" || status == "takendown" {
			if err := pw.index.DeleteAllByDID(context.Background(), event.DID); err != nil {
				log.Error().Err(err).Str("did", event.DID).Str("status", status).Msg("profile watcher: failed to delete user data")
				return
			}
			log.Warn().Str("did", event.DID).Str("status", status).Msg("profile watcher: purged all data for account")
			pw.Unwatch(event.DID)
		}
	}
}
