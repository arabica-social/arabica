package coffeehandlers

import (
	"net/http"

	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/firehose"
	"tangled.org/arabica.social/arabica/internal/lexicons"
	"tangled.org/arabica.social/arabica/internal/moderation"
)

var exploreFilterNames = []string{
	"origin",
	"variety",
	"process",
	"roast_level",
	"roaster",
	"min_rating",
	"closed",
	"location",
	"grinder_type",
	"burr_type",
	"brewer_type",
	"ratio_min",
	"ratio_max",
}

func (h *Handlers) getModeratedExplore(r *http.Request, query firehose.ExploreQuery, cf *moderation.ContentFilter) (*firehose.ExploreResult, error) {
	requested := query.Limit
	if requested <= 0 {
		requested = 20
	}
	if requested > 50 {
		requested = 50
	}
	query.Limit = min(requested*3, 50)

	out := &firehose.ExploreResult{Documents: make(map[string]firehose.ExploreDocument)}
	cursor := query.Cursor
	for attempts := 0; attempts < 4 && len(out.Items) < requested; attempts++ {
		query.Cursor = cursor
		page, err := h.FeedIndex().GetExplore(r.Context(), query)
		if err != nil {
			return nil, err
		}
		if attempts == 0 {
			out.FacetCounts = page.FacetCounts
		}
		visible := moderation.FilterSlice(cf, page.Items, func(item *feed.FeedItem) (string, string) {
			if item == nil || item.Author == nil {
				return "", ""
			}
			return item.SubjectURI, item.Author.DID
		})
		for _, item := range visible {
			if len(out.Items) >= requested {
				break
			}
			out.Items = append(out.Items, item)
			if doc, ok := page.Documents[item.SubjectURI]; ok {
				out.Documents[item.SubjectURI] = doc
			}
		}
		out.NextCursor = page.NextCursor
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	if len(out.Items) >= requested {
		last := out.Items[requested-1]
		if doc, ok := out.Documents[last.SubjectURI]; ok {
			out.NextCursor = firehose.ExploreCursor(query.Sort, doc)
		}
	}
	return out, nil
}

func parseExploreQuery(r *http.Request) firehose.ExploreQuery {
	v := r.URL.Query()
	rt := lexicons.ParseRecordType(v.Get("type"))
	q := firehose.ExploreQuery{
		App:     "arabica",
		Type:    rt,
		Q:       v.Get("q"),
		Sort:    v.Get("sort"),
		Cursor:  v.Get("cursor"),
		Limit:   20,
		Filters: make(map[string]string),
	}
	for _, name := range exploreFilterNames {
		if val := v.Get(name); val != "" {
			q.Filters[name] = val
		}
	}
	return q
}
