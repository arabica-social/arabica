package firehose

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/pdewey.com/atp"
)

func TestIdentityReconciliation_DelayedRetryDoesNotBlockReadyWork(t *testing.T) {
	profile := &atproto.Profile{DID: "did:plc:known", Handle: "known.example"}
	idx := identityTestIndex(t, profile, profile.DID)
	require.NoError(t, idx.Register(profile.DID))
	require.NoError(t, idx.Register("did:plc:other"))
	idx.publicClient = atp.NewPublicClientWithHTTP(&http.Client{Transport: identityTransport(func(r *http.Request) (*http.Response, error) {
		return nil, context.DeadlineExceeded
	})})
	pw, ctx := reconciliationWatcher(t, idx)
	pw.verificationRetryDelays = []time.Duration{time.Hour}
	probed := make(chan struct{}, 1)
	pw.accountClient = &http.Client{Transport: identityTransport(func(r *http.Request) (*http.Response, error) {
		probed <- struct{}{}
		return nil, context.DeadlineExceeded
	})}
	pw.enqueueVerification(ctx, identityHint(profile.DID, "new.example"))
	require.Eventually(t, func() bool {
		pw.verificationMu.Lock()
		defer pw.verificationMu.Unlock()
		work := pw.verificationPending[verificationKey{did: profile.DID, kind: "identity"}]
		return work != nil && work.attempts == 1
	}, time.Second, time.Millisecond)
	pw.enqueueVerification(ctx, accountEvent("did:plc:other", "deleted"))
	select {
	case <-probed:
	case <-time.After(time.Second):
		t.Fatal("delayed identity retry blocked ready account verification")
	}
}

func TestIdentityReconciliation_CancelDuringBackoff(t *testing.T) {
	profile := &atproto.Profile{DID: "did:plc:known", Handle: "known.example"}
	idx := identityTestIndex(t, profile, profile.DID)
	require.NoError(t, idx.Register(profile.DID))
	var requests atomic.Int32
	idx.publicClient = atp.NewPublicClientWithHTTP(&http.Client{Transport: identityTransport(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		return nil, context.DeadlineExceeded
	})})
	pw, parent := reconciliationWatcher(t, idx)
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	pw.verificationRetryDelays = []time.Duration{time.Hour}
	hint := identityHint(profile.DID, "new.example")
	pw.enqueueVerification(ctx, hint)
	require.Eventually(t, func() bool {
		pw.verificationMu.Lock()
		defer pw.verificationMu.Unlock()
		work := pw.verificationPending[verificationKey{did: profile.DID, kind: "identity"}]
		return work != nil && work.attempts == 1
	}, time.Second, time.Millisecond)
	for i := 0; i < 100; i++ {
		pw.enqueueVerification(ctx, hint)
	}
	pw.verificationMu.Lock()
	work := *pw.verificationPending[verificationKey{did: profile.DID, kind: "identity"}]
	pw.verificationMu.Unlock()
	assert.Equal(t, 1, work.attempts, "duplicate hints must not reset retries")
	cancel()
	select {
	case <-pw.verificationDone:
	case <-time.After(time.Second):
		t.Fatal("worker waited for backoff instead of cancellation")
	}
	assert.EqualValues(t, 1, requests.Load())
}

func TestIdentityReconciliation_ChangedHintWakesDelayedRetry(t *testing.T) {
	profile := &atproto.Profile{DID: "did:plc:known", Handle: "latest.example"}
	idx := identityTestIndex(t, profile, profile.DID)
	require.NoError(t, idx.Register(profile.DID))
	pw, ctx := reconciliationWatcher(t, idx)
	pw.verificationRetryDelays = []time.Duration{time.Hour}
	pw.enqueueVerification(ctx, identityHint(profile.DID, "intermediate.example"))
	require.Eventually(t, func() bool {
		pw.verificationMu.Lock()
		defer pw.verificationMu.Unlock()
		work := pw.verificationPending[verificationKey{did: profile.DID, kind: "identity"}]
		return work != nil && work.attempts == 1
	}, time.Second, time.Millisecond)
	pw.enqueueVerification(ctx, identityHint(profile.DID, profile.Handle))
	require.Eventually(t, func() bool {
		did, ok := idx.GetDIDByHandle(ctx, profile.Handle)
		return ok && did == profile.DID
	}, time.Second, time.Millisecond)
}
