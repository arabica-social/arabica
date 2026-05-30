package coffeehandlers

import (
	"net/http"

	coffeepages "tangled.org/arabica.social/arabica/internal/arabica/web/pages"
	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/firehose"
	"tangled.org/arabica.social/arabica/internal/lexicons"
	"tangled.org/arabica.social/arabica/internal/moderation"

	"github.com/rs/zerolog/log"
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

func (h *Handlers) HandleExplore(w http.ResponseWriter, r *http.Request) {
	_, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	layoutData, viewerDID, _ := h.LayoutDataFromRequest(r, "Explore")
	if h.FeedIndex() == nil {
		http.Error(w, "Explore is unavailable", http.StatusServiceUnavailable)
		return
	}
	query := parseExploreQuery(r)
	cf := h.LoadContentFilter(r.Context())
	result, err := h.getModeratedExplore(r, query, cf)
	if err != nil {
		log.Error().Err(err).Msg("failed to query explore")
		http.Error(w, "Failed to load explore", http.StatusInternalServerError)
		return
	}
	uris := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		uris = append(uris, item.SubjectURI)
		if item.Author != nil {
			item.IsOwner = item.Author.DID == viewerDID
		}
	}
	liked := h.FeedIndex().HasUserLikedBatch(r.Context(), viewerDID, uris)
	for _, item := range result.Items {
		item.IsLikedByViewer = liked[item.SubjectURI]
	}
	if err := coffeepages.ExplorePage(layoutData, coffeepages.ExploreProps{
		Query:       query,
		Result:      result,
		Health:      h.FeedIndex().ExploreHealth(r.Context()),
		FilterNames: exploreFilterNames,
		RoutePaths:  h.exploreRoutePaths(),
	}).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("failed to render explore page")
	}
}

func (h *Handlers) exploreRoutePaths() map[lexicons.RecordType]string {
	paths := make(map[lexicons.RecordType]string)
	app := h.App()
	if app == nil {
		return paths
	}
	for _, route := range app.EntityRoutes {
		paths[route.Type] = route.Path
	}
	return paths
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
