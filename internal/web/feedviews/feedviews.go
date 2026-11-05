package feedviews

import "tangled.org/arabica.social/arabica/internal/lexicons"

// View describes how a record type appears in the JSON feed filter bar.
type View struct {
	FilterLabel string
	FilterValue string
}

type Registry map[lexicons.RecordType]View

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
	return view.FilterValue
}
