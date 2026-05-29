package feedviews

import (
	"github.com/a-h/templ"

	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

type Renderer func(*feed.FeedItem) templ.Component

type View struct {
	Render  Renderer
	Compact bool
}

type Registry map[lexicons.RecordType]View

func (r Registry) Render(item *feed.FeedItem) templ.Component {
	if item == nil {
		return nil
	}
	view, ok := r[item.RecordType]
	if !ok || view.Render == nil {
		return nil
	}
	return view.Render(item)
}

func (r Registry) Compact(rt lexicons.RecordType) bool {
	view, ok := r[rt]
	return ok && view.Compact
}
