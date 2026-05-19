package firehose

import (
	"context"
	"encoding/json"
	"sync"

	atpjetstream "tangled.org/pdewey.com/atp/jetstream"

	"github.com/rs/zerolog/log"
)

// ProfileWatcher is a dedicated Jetstream connection that subscribes to
// app.bsky.actor.profile events for known Arabica users only. Because Jetstream
// uses AND semantics when both wantedCollections and wantedDids are set, this
// must be a separate connection from the main consumer (which has no DID filter).
//
// The upstream consumer is created lazily on the first Watch() so an instance
// with no watched DIDs does not connect — connecting with an empty wantedDids
// would cause the relay to deliver every profile event globally.
type ProfileWatcher struct {
	index     *FeedIndex
	endpoints []string
	ctx       context.Context

	watchedDIDs   map[string]struct{}
	watchedDIDsMu sync.RWMutex

	upstreamMu sync.Mutex
	upstream   *atpjetstream.Consumer

	stopCh chan struct{}
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

// Watch adds a DID to the subscription. If a connection is already open, an
// options_update frame is sent immediately so Jetstream begins delivering that
// user's profile events.
func (pw *ProfileWatcher) Watch(did string) {
	pw.watchedDIDsMu.Lock()
	_, already := pw.watchedDIDs[did]
	pw.watchedDIDs[did] = struct{}{}
	pw.watchedDIDsMu.Unlock()

	if already {
		return
	}
	if u := pw.ensureUpstream(); u != nil {
		if err := u.UpdateOptions([]string{NSIDBlueskyProfile}, pw.snapshotDIDs()); err != nil {
			log.Warn().Err(err).Msg("profile watcher: options_update failed")
		}
	}
}

// Unwatch removes a DID from the subscription. Used when an account is deleted
// so Jetstream stops sending events for that DID.
func (pw *ProfileWatcher) Unwatch(did string) {
	pw.watchedDIDsMu.Lock()
	_, present := pw.watchedDIDs[did]
	delete(pw.watchedDIDs, did)
	pw.watchedDIDsMu.Unlock()

	if !present {
		return
	}
	pw.upstreamMu.Lock()
	u := pw.upstream
	pw.upstreamMu.Unlock()
	if u == nil {
		return
	}
	if err := u.UpdateOptions([]string{NSIDBlueskyProfile}, pw.snapshotDIDs()); err != nil {
		log.Warn().Err(err).Msg("profile watcher: options_update failed")
	}
}

// Start begins the profile watcher. If the index already seeded watchedDIDs,
// the upstream consumer connects immediately; otherwise the first Watch() call
// will spin it up.
func (pw *ProfileWatcher) Start(ctx context.Context) {
	pw.ctx = ctx
	pw.watchedDIDsMu.RLock()
	n := len(pw.watchedDIDs)
	pw.watchedDIDsMu.RUnlock()
	if n > 0 {
		pw.ensureUpstream()
	}
}

// Stop gracefully shuts down the watcher.
func (pw *ProfileWatcher) Stop() {
	close(pw.stopCh)
	pw.upstreamMu.Lock()
	u := pw.upstream
	pw.upstreamMu.Unlock()
	if u != nil {
		u.Stop()
	}
}

func (pw *ProfileWatcher) snapshotDIDs() []string {
	pw.watchedDIDsMu.RLock()
	defer pw.watchedDIDsMu.RUnlock()
	out := make([]string, 0, len(pw.watchedDIDs))
	for did := range pw.watchedDIDs {
		out = append(out, did)
	}
	return out
}

// ensureUpstream lazily creates and starts the upstream consumer on the first
// call after Start. Returns nil if Start has not yet been invoked (the consumer
// will be created when Start is eventually called).
func (pw *ProfileWatcher) ensureUpstream() *atpjetstream.Consumer {
	pw.upstreamMu.Lock()
	defer pw.upstreamMu.Unlock()
	if pw.upstream != nil {
		return pw.upstream
	}
	if pw.ctx == nil {
		return nil
	}

	pw.upstream = atpjetstream.New(&atpjetstream.Config{
		Endpoints:         pw.endpoints,
		WantedCollections: []string{NSIDBlueskyProfile},
		WantedDIDs:        pw.snapshotDIDs(),
		OnConnect: func() {
			log.Info().Msg("profile watcher: connected")
		},
		OnDisconnect: func() {
			log.Warn().Msg("profile watcher: disconnected")
		},
		OnError: func(err error, _ *atpjetstream.Event) {
			log.Warn().Err(err).Msg("profile watcher: event processing error")
		},
	}, pw.handleEvent)

	pw.upstream.Start(pw.ctx)
	return pw.upstream
}

func (pw *ProfileWatcher) handleEvent(_ context.Context, evt *atpjetstream.Event) error {
	if evt == nil {
		return nil
	}
	pw.dispatch(toLegacyEvent(evt))
	return nil
}

// toLegacyEvent converts an atp/jetstream Event into the package-local
// JetstreamEvent so dispatch() and ProcessEvent() share a single signature.
func toLegacyEvent(evt *atpjetstream.Event) JetstreamEvent {
	out := JetstreamEvent{DID: evt.DID, TimeUS: evt.TimeUS, Kind: evt.Kind}
	if evt.Commit != nil {
		out.Commit = &JetstreamCommit{
			Rev:        evt.Commit.Rev,
			Operation:  evt.Commit.Operation,
			Collection: evt.Commit.Collection,
			RKey:       evt.Commit.RKey,
			Record:     evt.Commit.Record,
			CID:        evt.Commit.CID,
		}
	}
	if evt.Identity != nil {
		out.Identity = &JetstreamIdentity{
			DID:    evt.Identity.DID,
			Handle: evt.Identity.Handle,
			Seq:    evt.Identity.Seq,
			Time:   evt.Identity.Time,
		}
	}
	if evt.Account != nil {
		out.Account = &JetstreamAccount{
			Active: evt.Account.Active,
			DID:    evt.Account.DID,
			Seq:    evt.Account.Seq,
			Status: evt.Account.Status,
			Time:   evt.Account.Time,
		}
	}
	return out
}

// ProcessEvent processes a single Jetstream event through the profile-watcher
// pipeline. Exported for use in integration tests where events are fed from a
// test PDS firehose rather than a live Jetstream connection.
func (pw *ProfileWatcher) ProcessEvent(event JetstreamEvent) {
	pw.dispatch(event)
}

// processMessage parses a raw Jetstream frame and dispatches it. Retained
// because profile_watcher_test.go exercises this path with raw JSON fixtures.
func (pw *ProfileWatcher) processMessage(data []byte) {
	var event JetstreamEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return
	}
	pw.dispatch(event)
}

func (pw *ProfileWatcher) dispatch(event JetstreamEvent) {
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
		// Handle change, handle reassignment, or PDS migration. OnIdentityEvent
		// reconciles the profile cache, did_by_handle index, and resolver caches
		// before refreshing — this is also the only signal we get when a handle
		// moves from an abandoned DID to a new one.
		handle := ""
		if event.Identity != nil {
			handle = event.Identity.Handle
		}
		pw.index.OnIdentityEvent(context.Background(), event.DID, handle)
		log.Info().Str("did", event.DID).Str("handle", handle).Msg("profile watcher: identity update reconciled")

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
			pw.index.InvalidatePublicCachesForDID(event.DID)
			log.Warn().Str("did", event.DID).Str("status", status).Msg("profile watcher: purged all data for account")
			pw.Unwatch(event.DID)
		}
	}
}
