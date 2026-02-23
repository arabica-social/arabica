package metrics

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

// StatsSource provides functions to retrieve current counts for gauge metrics.
// Each function returns the current count; returning -1 indicates the source is unavailable.
type StatsSource struct {
	KnownDIDCount          func() int
	RegisteredCount        func() int
	RecordCount            func() int
	PendingJoinCount       func() int
	LikeCount              func() int
	CommentCount           func() int
	RecordCountByCollection func() map[string]int
	FirehoseConnected      func() bool
}

// StartCollector launches a goroutine that periodically updates gauge metrics.
// It runs every interval until the context is cancelled.
func StartCollector(ctx context.Context, src StatsSource, interval time.Duration) {
	// Do an initial collection immediately
	collect(src)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				collect(src)
			}
		}
	}()

	log.Info().Dur("interval", interval).Msg("Metrics collector started")
}

func collect(src StatsSource) {
	if src.KnownDIDCount != nil {
		KnownUsersTotal.Set(float64(src.KnownDIDCount()))
	}
	if src.RegisteredCount != nil {
		RegisteredUsersTotal.Set(float64(src.RegisteredCount()))
	}
	if src.RecordCount != nil {
		IndexedRecordsTotal.Set(float64(src.RecordCount()))
	}
	if src.PendingJoinCount != nil {
		JoinRequestsPending.Set(float64(src.PendingJoinCount()))
	}
	if src.LikeCount != nil {
		IndexedLikesTotal.Set(float64(src.LikeCount()))
	}
	if src.CommentCount != nil {
		IndexedCommentsTotal.Set(float64(src.CommentCount()))
	}
	if src.RecordCountByCollection != nil {
		for collection, count := range src.RecordCountByCollection() {
			IndexedRecordsByCollection.WithLabelValues(collection).Set(float64(count))
		}
	}
	if src.FirehoseConnected != nil {
		if src.FirehoseConnected() {
			FirehoseConnectionState.Set(1)
		} else {
			FirehoseConnectionState.Set(0)
		}
	}
}
