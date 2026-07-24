package feedviews

import (
	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

// ActionURL computes a per-item action URL (share, delete, edit) from a feed
// item. Kept as a typed function so the registry can be assembled declaratively
// per app.
type ActionURL func(*feed.FeedItem) string

// View describes how a record type appears in the feed. Only the metadata
// needed by the JSON feed path (filter labels, action URLs, compactness) is
// retained; the legacy templ Render/RenderPrefs fields were removed with the
// templ stack.
type View struct {
	Compact       bool
	CardClassNoun string
	ActionNoun    string
	FilterLabel   string
	FilterValue   string
	ShareURL      ActionURL
	DeleteURL     ActionURL
	EditURL       ActionURL
	EditModalURL  ActionURL
}

type Registry map[lexicons.RecordType]View

func (r Registry) Compact(rt lexicons.RecordType) bool {
	view, ok := r[rt]
	return ok && view.Compact
}

func (r Registry) CardClassNoun(rt lexicons.RecordType) string {
	view, ok := r[rt]
	if !ok {
		return ""
	}
	return view.CardClassNoun
}

func (r Registry) ActionNoun(rt lexicons.RecordType) string {
	view, ok := r[rt]
	if !ok {
		return ""
	}
	if view.ActionNoun != "" {
		return view.ActionNoun
	}
	return view.CardClassNoun
}

func (r Registry) FilterLabel(rt lexicons.RecordType) string {
	view, ok := r[rt]
	if !ok {
		return ""
	}
	return view.FilterLabel
}

func (r Registry) FilterValue(rt lexicons.RecordType) string {
	view, ok := r[rt]
	if !ok {
		return ""
	}
	if view.FilterValue != "" {
		return view.FilterValue
	}
	return view.CardClassNoun
}

func (r Registry) ShareURL(item *feed.FeedItem) string {
	if item == nil {
		return ""
	}
	view, ok := r[item.RecordType]
	if ok && view.ShareURL != nil {
		if url := view.ShareURL(item); url != "" {
			return url
		}
	}
	if item.Author == nil {
		return ""
	}
	return "/profile/" + item.Author.DID
}

func (r Registry) DeleteURL(item *feed.FeedItem) string {
	if item == nil {
		return ""
	}
	view, ok := r[item.RecordType]
	if !ok || view.DeleteURL == nil {
		return ""
	}
	return view.DeleteURL(item)
}

func (r Registry) EditURL(item *feed.FeedItem) string {
	if item == nil {
		return ""
	}
	view, ok := r[item.RecordType]
	if !ok || view.EditURL == nil {
		return ""
	}
	return view.EditURL(item)
}

func (r Registry) EditModalURL(item *feed.FeedItem) string {
	if item == nil {
		return ""
	}
	view, ok := r[item.RecordType]
	if !ok || view.EditModalURL == nil {
		return ""
	}
	return view.EditModalURL(item)
}

func Actor(item *feed.FeedItem) string {
	if item == nil || item.Author == nil {
		return ""
	}
	if item.Author.Handle != "" {
		return item.Author.Handle
	}
	return item.Author.DID
}
