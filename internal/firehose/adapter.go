package firehose

import (
	"context"

	"tangled.org/arabica.social/arabica/internal/feed"
)

// FeedIndexAdapter wraps FeedIndex to implement feed.FirehoseIndex.
//
// Items flow through unchanged (both sides use *feed.FeedItem); the
// adapter exists only to translate feed.FirehoseFeedQuery into
// firehose.FeedQuery (different Sort field type) and to satisfy the
// interface contract.
type FeedIndexAdapter struct {
	index *FeedIndex
}

// NewFeedIndexAdapter creates a new adapter for the FeedIndex
func NewFeedIndexAdapter(index *FeedIndex) *FeedIndexAdapter {
	return &FeedIndexAdapter{index: index}
}

// IsReady returns true if the index is ready to serve queries
func (a *FeedIndexAdapter) IsReady() bool {
	return a.index.IsReady()
}

// GetRecentFeed returns recent feed items from the index.
func (a *FeedIndexAdapter) GetRecentFeed(ctx context.Context, limit int) ([]*feed.FeedItem, error) {
	return a.index.GetRecentFeed(ctx, limit)
}

// GetFeedWithQuery returns feed items matching query parameters
func (a *FeedIndexAdapter) GetFeedWithQuery(ctx context.Context, q feed.FirehoseFeedQuery) (*feed.FirehoseFeedResult, error) {
	result, err := a.index.GetFeedWithQuery(ctx, FeedQuery{
		Limit:       q.Limit,
		Cursor:      q.Cursor,
		TypeFilter:  q.TypeFilter,
		TypeFilters: q.TypeFilters,
		Sort:        FeedSort(q.Sort),
	})
	if err != nil {
		return nil, err
	}

	return &feed.FirehoseFeedResult{
		Items:      result.Items,
		NextCursor: result.NextCursor,
	}, nil
}
