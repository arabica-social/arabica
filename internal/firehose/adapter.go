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

	return convertFeedItems(items), nil
}

// GetFeedWithQuery returns feed items matching query parameters
func (a *FeedIndexAdapter) GetFeedWithQuery(ctx context.Context, q feed.FirehoseFeedQuery) (*feed.FirehoseFeedResult, error) {
	result, err := a.index.GetFeedWithQuery(ctx, FeedQuery{
		Limit:      q.Limit,
		Cursor:     q.Cursor,
		TypeFilter: q.TypeFilter,
		Sort:       FeedSort(q.Sort),
	})
	if err != nil {
		return nil, err
	}

	return &feed.FirehoseFeedResult{
		Items:      convertFeedItems(result.Items),
		NextCursor: result.NextCursor,
	}, nil
}

func convertFeedItems(items []*FeedItem) []*feed.FirehoseFeedItem {
	result := make([]*feed.FirehoseFeedItem, len(items))
	for i, item := range items {
		result[i] = &feed.FirehoseFeedItem{
			RecordType:   item.RecordType,
			Action:       item.Action,
			Brew:         item.Brew,
			Bean:         item.Bean,
			Roaster:      item.Roaster,
			Grinder:      item.Grinder,
			Brewer:       item.Brewer,
			Author:       item.Author,
			Timestamp:    item.Timestamp,
			TimeAgo:      item.TimeAgo,
			LikeCount:    item.LikeCount,
			CommentCount: item.CommentCount,
			SubjectURI:   item.SubjectURI,
			SubjectCID:   item.SubjectCID,
		}
	}
	return result
}
