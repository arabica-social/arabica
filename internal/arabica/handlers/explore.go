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

var exploreFilterNames = []string{"origin", "variety", "process", "roast_level", "roaster", "min_rating", "closed", "location", "grinder_type", "burr_type", "brewer_type", "ratio_min", "ratio_max"}

func (h *Handlers) HandleExplore(w http.ResponseWriter, r *http.Request) {
	_, authenticated := h.GetAtprotoStore(r)
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
	result, err := h.FeedIndex().GetExplore(r.Context(), query)
	if err != nil {
		log.Error().Err(err).Msg("failed to query explore")
		http.Error(w, "Failed to load explore", http.StatusInternalServerError)
		return
	}
	cf := h.LoadContentFilter(r.Context())
	result.Items = moderation.FilterSlice(cf, result.Items, func(item *feed.FeedItem) (string, string) {
		if item == nil || item.Author == nil {
			return "", ""
		}
		return item.SubjectURI, item.Author.DID
	})
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
	if err := coffeepages.ExplorePage(layoutData, coffeepages.ExploreProps{Query: query, Result: result, Health: h.FeedIndex().ExploreHealth(r.Context()), FilterNames: exploreFilterNames}).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("failed to render explore page")
	}
}

func parseExploreQuery(r *http.Request) firehose.ExploreQuery {
	v := r.URL.Query()
	rt := lexicons.ParseRecordType(v.Get("type"))
	q := firehose.ExploreQuery{App: "arabica", Type: rt, Q: v.Get("q"), Sort: v.Get("sort"), Cursor: v.Get("cursor"), Limit: 20, Filters: make(map[string]string)}
	for _, name := range exploreFilterNames {
		if val := v.Get(name); val != "" {
			q.Filters[name] = val
		}
	}
	return q
}
