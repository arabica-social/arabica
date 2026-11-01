package firehose

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"
	"tangled.org/pdewey.com/atp"
)

// getProfile.ts returns AccountTakedown for a suspended account; deactivation
// is reversible and missing profiles are not an unambiguous deletion signal.
// https://github.com/bluesky-social/atproto/blob/main/packages/bsky/src/api/app/bsky/actor/getProfile.ts
const appViewAccountTakedown = "AccountTakedown"

const verificationQueueDepth = 64
const verificationCooldown = 30 * time.Second

var identityRetryDelays = []time.Duration{time.Second, 5 * time.Second, 30 * time.Second, 2 * time.Minute}

type verificationKey struct {
	did, kind string
}

type verificationHint struct {
	kind, handle, status string
	expires              time.Time
}

type verificationWork struct {
	event    JetstreamEvent
	revision uint64
	attempts int
	next     time.Time
}

func hintForEvent(event JetstreamEvent) verificationHint {
	hint := verificationHint{kind: event.Kind}
	if event.Identity != nil {
		hint.handle = event.Identity.Handle
	}
	if event.Account != nil {
		hint.status = event.Account.Status
	}
	return hint
}

// Relay hints cannot authorize writes. At most 64 DID/kind pairs share one
// network worker; changed hints replace pending work without resetting identical
// retries. Identity and account work cannot overwrite each other.
func (pw *ProfileWatcher) enqueueVerification(ctx context.Context, event JetstreamEvent) {
	pw.verificationOnce.Do(func() {
		pw.verificationWake = make(chan struct{}, 1)
		pw.verificationPending = make(map[verificationKey]*verificationWork)
		pw.verificationRecent = make(map[verificationKey]verificationHint)
		pw.verificationDone = make(chan struct{})
		go pw.verifyEvents(ctx)
	})
	pw.verificationMu.Lock()
	defer pw.verificationMu.Unlock()
	if ctx.Err() != nil {
		return
	}
	key := verificationKey{did: event.DID, kind: event.Kind}
	hint := hintForEvent(event)
	if pending, ok := pw.verificationPending[key]; ok {
		if hintForEvent(pending.event) == hint {
			return
		}
		pending.event = event
		pending.revision++
		pending.attempts = 0
		pending.next = time.Now()
	} else {
		if recent, ok := pw.verificationRecent[key]; ok && time.Now().Before(recent.expires) {
			hint.expires = recent.expires
			if recent == hint {
				return
			}
		}
		if len(pw.verificationPending) >= verificationQueueDepth {
			log.Warn().Str("did", event.DID).Msg("profile watcher: verification queue full; dropping hint")
			return
		}
		pw.verificationPending[key] = &verificationWork{event: event, next: time.Now()}
	}
	select {
	case pw.verificationWake <- struct{}{}:
	default:
	}
}

func (pw *ProfileWatcher) verifyEvents(ctx context.Context) {
	defer close(pw.verificationDone)
	for {
		if ctx.Err() != nil {
			return
		}
		key, work, ok := pw.nextVerification()
		if ok && !time.Now().Before(work.next) {
			verified := true
			if work.event.Kind == "identity" {
				handle := ""
				if work.event.Identity != nil {
					handle = work.event.Identity.Handle
				}
				verified = pw.index.verifyIdentityEvent(ctx, work.event.DID, handle)
			} else {
				pw.dispatchContext(ctx, work.event)
			}
			pw.finishVerification(key, work, verified)
			continue
		}
		var timer *time.Timer
		var ready <-chan time.Time
		if ok {
			timer = time.NewTimer(time.Until(work.next))
			ready = timer.C
		}
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return
		case <-pw.verificationWake:
		case <-ready:
		}
		if timer != nil {
			timer.Stop()
		}
	}
}

func (pw *ProfileWatcher) nextVerification() (verificationKey, verificationWork, bool) {
	pw.verificationMu.Lock()
	defer pw.verificationMu.Unlock()
	var key verificationKey
	var next verificationWork
	found := false
	for candidate, work := range pw.verificationPending {
		if !found || work.next.Before(next.next) {
			key, next, found = candidate, *work, true
		}
	}
	return key, next, found
}

func (pw *ProfileWatcher) finishVerification(key verificationKey, work verificationWork, verified bool) {
	pw.verificationMu.Lock()
	defer pw.verificationMu.Unlock()
	pending := pw.verificationPending[key]
	if pending.revision != work.revision {
		return
	}
	delays := pw.verificationRetryDelays
	if delays == nil {
		delays = identityRetryDelays
	}
	if !verified && pending.attempts < len(delays) {
		pending.next = time.Now().Add(delays[pending.attempts])
		pending.attempts++
		return
	}
	delete(pw.verificationPending, key)
	if len(pw.verificationRecent) >= verificationQueueDepth {
		var oldestKey verificationKey
		var oldest time.Time
		for candidate, recent := range pw.verificationRecent {
			if oldest.IsZero() || recent.expires.Before(oldest) {
				oldestKey, oldest = candidate, recent.expires
			}
		}
		delete(pw.verificationRecent, oldestKey)
	}
	hint := hintForEvent(work.event)
	hint.expires = time.Now().Add(verificationCooldown)
	pw.verificationRecent[key] = hint
	if !verified {
		log.Warn().Str("did", key.did).Msg("profile watcher: identity verification retries exhausted; keeping cached profile")
	}
}

// corroborateAccountStatus accepts only explicit terminal responses from the
// fixed trusted AppView. Generic HTTP errors (including 404/5xx), transport
// errors, malformed responses and deactivation are NOT evidence of deletion.
// GetProfile's generic error discards the XRPC code, so inspect that code here.
// Genuinely deleted-but-not-takendown accounts keep their indexed content,
// because the AppView exposes no unambiguous deletion code. The content-deletion
// path for these accounts is POST /_mod/purge (admin-gated HandleAdminPurgeDID,
// internal/handlers/admin.go), which retains DeleteAllByDID's full-purge behavior.
// RefreshAllProfiles repairs handle/profile mappings only and deletes nothing.
func (pw *ProfileWatcher) corroborateAccountStatus(ctx context.Context, did string) (bool, string) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		atp.PublicAPIBase+"/xrpc/app.bsky.actor.getProfile?actor="+url.QueryEscape(did), nil)
	if err != nil {
		return false, "request construction failed"
	}
	client := pw.accountClient
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	}
	response, err := client.Do(req)
	if err != nil {
		return false, "AppView request failed"
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		return false, fmt.Sprintf("AppView HTTP %d is not terminal evidence", response.StatusCode)
	}
	var result struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&result); err != nil {
		return false, "unparseable AppView response"
	}
	if result.Error != appViewAccountTakedown {
		return false, "AppView XRPC code is not terminal: " + result.Error
	}
	return true, "AppView AccountTakedown"
}
