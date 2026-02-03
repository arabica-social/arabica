package firehose

import (
	"context"

	"arabica/internal/feed"
)

// FeedIndexAdapter wraps FeedIndex to implement feed.FirehoseIndex interface
// This avoids import cycles between feed and firehose packages
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

// GetRecentFeed returns recent feed items from the index
// Converts FeedItem to feed.FirehoseFeedItem to satisfy the interface
func (a *FeedIndexAdapter) GetRecentFeed(ctx context.Context, limit int) ([]*feed.FirehoseFeedItem, error) {
	items, err := a.index.GetRecentFeed(ctx, limit)
	if err != nil {
		return nil, err
	}

	// Convert to the type expected by feed.Service
	result := make([]*feed.FirehoseFeedItem, len(items))
	for i, item := range items {
		result[i] = &feed.FirehoseFeedItem{
			RecordType: item.RecordType,
			Action:     item.Action,
			Brew:       item.Brew,
			Bean:       item.Bean,
			Roaster:    item.Roaster,
			Grinder:    item.Grinder,
			Brewer:     item.Brewer,
			Author:     item.Author,
			Timestamp:  item.Timestamp,
			TimeAgo:    item.TimeAgo,
			LikeCount:  item.LikeCount,
			SubjectURI: item.SubjectURI,
			SubjectCID: item.SubjectCID,
		}
	}

	return result, nil
}
