package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/backlinks"
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/firehose"
	"tangled.org/arabica.social/arabica/internal/lexicons"
	"tangled.org/arabica.social/arabica/internal/metrics"
	"tangled.org/arabica.social/arabica/internal/moderation"
	"tangled.org/arabica.social/arabica/internal/ogcard"
	"tangled.org/arabica.social/arabica/internal/web/bff"
	"tangled.org/arabica.social/arabica/internal/web/components"
	"tangled.org/arabica.social/arabica/internal/web/pages"
	"tangled.org/pdewey.com/atp"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/rs/zerolog/log"
)

// SocialData holds the social interaction data shared across all entity view handlers
type SocialData struct {
	IsLiked        bool
	LikeCount      int
	CommentCount   int
	Comments       []firehose.IndexedComment
	IsModerator    bool
	CanHideRecord  bool
	CanBlockUser   bool
	IsRecordHidden bool
}

// fetchSocialData retrieves likes, comments, and moderation state for a record
func (h *Handler) FetchSocialData(ctx context.Context, subjectURI, didStr string, isAuthenticated bool) SocialData {
	var sd SocialData

	if h.feedIndex != nil && subjectURI != "" {
		sd.LikeCount = h.feedIndex.GetLikeCount(ctx, subjectURI)
		sd.CommentCount = h.feedIndex.GetCommentCount(ctx, subjectURI)
		sd.Comments = h.feedIndex.GetThreadedCommentsForSubject(ctx, subjectURI, 100, didStr)
		sd.Comments = h.FilterHiddenComments(ctx, sd.Comments)
		if isAuthenticated {
			sd.IsLiked = h.feedIndex.HasUserLiked(ctx, didStr, subjectURI)
		}
	}

	if h.moderationService != nil && isAuthenticated {
		sd.IsModerator = h.moderationService.IsModerator(didStr)
		sd.CanHideRecord = h.moderationService.HasPermission(didStr, moderation.PermissionHideRecord)
		sd.CanBlockUser = h.moderationService.HasPermission(didStr, moderation.PermissionBlacklistUser)
	}
	if h.moderationStore != nil && sd.IsModerator && subjectURI != "" {
		sd.IsRecordHidden = h.moderationStore.IsRecordHidden(ctx, subjectURI)
	}

	return sd
}

// ResolveOwnerDID resolves an owner parameter (DID or handle) to a DID string.
// Returns the DID and nil error on success, or empty string and error on failure.
func ResolveOwnerDID(ctx context.Context, owner string) (string, error) {
	if strings.HasPrefix(owner, "did:") {
		return owner, nil
	}
	publicClient := atproto.NewPublicClient()
	resolved, err := publicClient.ResolveHandle(ctx, owner)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

// EntityViewConfig captures per-entity behavior for RenderEntityView.
// Construct via the h.xViewConfig() methods — closures capture h naturally.
// Fields are exported so per-app handler packages (coffeehandlers,
// teahandlers) can populate this struct directly.
type EntityViewConfig struct {
	Descriptor  *entities.Descriptor
	FromWitness func(ctx context.Context, m map[string]any, uri, rkey, ownerDID string) (any, error)
	FromPDS     func(ctx context.Context, e *atp.Record, rkey, ownerDID string) (any, error)
	FromStore   func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, map[string]any, string, string, error)
	ResolveRefs func(ctx context.Context, model any, raw map[string]any, lookup func(refURI string) (map[string]any, bool))

	DisplayName func(record any) string
	OGSubtitle  func(record any) string
	CountLookup func(ctx context.Context, ownerDID, subjectURI string) int
	Render      func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base pages.EntityViewBase) error
}

func (cfg EntityViewConfig) loadConfig() EntityLoadConfig {
	return EntityLoadConfig{
		Descriptor:  cfg.Descriptor,
		FromWitness: cfg.FromWitness,
		FromPDS:     cfg.FromPDS,
		FromStore:   cfg.FromStore,
		ResolveRefs: cfg.ResolveRefs,
	}
}

func (h *Handler) RenderEntityView(w http.ResponseWriter, r *http.Request, cfg EntityViewConfig) {
	rkey := ValidateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	owner := r.URL.Query().Get("owner")
	didStr, _ := atpmiddleware.GetDID(r.Context())
	isAuthenticated := didStr != ""

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.GetUserProfile(r.Context(), didStr)
	}

	loaded, err := h.EntityViewLoader().Load(r, rkey, cfg.loadConfig())
	if err != nil {
		if loadErr, ok := err.(*EntityLoadError); ok {
			http.Error(w, loadErr.Msg, loadErr.HTTPStatus())
		} else {
			http.Error(w, "Failed to load record", http.StatusInternalServerError)
		}
		return
	}

	var shareURL string
	if owner != "" && loaded.Route.Path != "" {
		shareURL = fmt.Sprintf("/%s/%s/%s", loaded.Route.Path, owner, rkey)
	} else if userProfile != nil && userProfile.Handle != "" && loaded.Route.Path != "" {
		shareURL = fmt.Sprintf("/%s/%s/%s", loaded.Route.Path, userProfile.Handle, rkey)
	}

	ownerHandle := h.ResolveOwnerHandle(r.Context(), owner)
	layoutData := h.BuildLayoutData(r, cfg.DisplayName(loaded.Record), isAuthenticated, didStr, userProfile)
	PopulateOGFields(layoutData, cfg.OGSubtitle(loaded.Record), loaded.EntityNoun, ownerHandle, h.PublicBaseURL(r), shareURL)

	sd := h.FetchSocialData(r.Context(), loaded.SubjectURI, didStr, isAuthenticated)
	bl, blDetailURL := h.fetchBacklinks(r.Context(), loaded.SubjectURI, loaded.Route.Path, rkey, ownerSegment(owner, userProfile, didStr))

	authorDID := loaded.OwnerDID
	if authorDID == "" {
		authorDID = didStr
	}
	base := pages.EntityViewBase{
		IsOwnProfile:       loaded.IsOwnProfile,
		IsAuthenticated:    isAuthenticated,
		SubjectURI:         loaded.SubjectURI,
		SubjectCID:         loaded.SubjectCID,
		IsLiked:            sd.IsLiked,
		LikeCount:          sd.LikeCount,
		CommentCount:       sd.CommentCount,
		Comments:           sd.Comments,
		CurrentUserDID:     didStr,
		ShareURL:           shareURL,
		IsModerator:        sd.IsModerator,
		CanHideRecord:      sd.CanHideRecord,
		CanBlockUser:       sd.CanBlockUser,
		IsRecordHidden:     sd.IsRecordHidden,
		AuthorDID:          loaded.OwnerDID,
		Backlinks:          bl,
		BacklinksDetailURL: blDetailURL,
	}
	if ap := h.GetUserProfile(r.Context(), authorDID); ap != nil {
		base.AuthorHandle = ap.Handle
		base.AuthorDisplayName = ap.DisplayName
		base.AuthorAvatar = ap.Avatar
	}

	if err := cfg.Render(r.Context(), w, layoutData, loaded.Record, base); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msgf("Failed to render %s view", loaded.EntityNoun)
	}
}

func ownerSegment(owner string, profile *bff.UserProfile, did string) string {
	if owner != "" {
		return owner
	}
	if profile != nil && profile.Handle != "" {
		return profile.Handle
	}
	return did
}

func (h *Handler) fetchBacklinks(ctx context.Context, subjectURI, routePath, rkey, owner string) (*backlinks.Result, string) {
	return h.fetchBacklinksWithOptions(ctx, subjectURI, routePath, rkey, owner, backlinks.LookupOptions{})
}

func (h *Handler) fetchBacklinksWithOptions(ctx context.Context, subjectURI, routePath, rkey, owner string, opts backlinks.LookupOptions) (*backlinks.Result, string) {
	detailURL := ""
	if routePath != "" && rkey != "" && owner != "" {
		detailURL = fmt.Sprintf("/%s/%s/%s/backlinks", routePath, owner, rkey)
	}
	if h.feedIndex == nil || subjectURI == "" {
		return nil, detailURL
	}
	src := backlinkIndexSource{idx: h.feedIndex}
	svc := backlinks.NewService(src, src)
	res, err := svc.LookupWithOptions(ctx, subjectURI, opts)
	if err != nil {
		log.Warn().Err(err).Str("uri", subjectURI).Msg("backlinks lookup failed")
		return nil, detailURL
	}
	return res, detailURL
}

type backlinkIndexSource struct{ idx *firehose.FeedIndex }

func (s backlinkIndexSource) ListSourceRefChain(ctx context.Context, uri string, maxDepth, maxRecords int) ([]backlinks.IndexedRecord, error) {
	recs, err := s.idx.ListSourceRefChain(ctx, uri, maxDepth, maxRecords)
	return convertBacklinkRecords(recs), err
}

func (s backlinkIndexSource) ListRecordsByCollectionOldest(ctx context.Context, collection string) ([]backlinks.IndexedRecord, error) {
	recs, err := s.idx.ListRecordsByCollectionOldest(ctx, collection)
	return convertBacklinkRecords(recs), err
}

func (s backlinkIndexSource) ListUsageBacklinks(ctx context.Context, uri, fromCollection, fieldName string) ([]backlinks.IndexedRecord, error) {
	recs, err := s.idx.ListUsageBacklinks(ctx, uri, fromCollection, fieldName)
	return convertBacklinkRecords(recs), err
}

func (s backlinkIndexSource) ListUsageBacklinksPage(ctx context.Context, uri, fromCollection, fieldName string, limit, offset int) ([]backlinks.IndexedRecord, int, error) {
	recs, count, err := s.idx.ListUsageBacklinksPage(ctx, uri, fromCollection, fieldName, limit, offset)
	return convertBacklinkRecords(recs), count, err
}

func (s backlinkIndexSource) GetRecord(ctx context.Context, uri string) (backlinks.IndexedRecord, bool) {
	rec, err := s.idx.GetRecord(ctx, uri)
	if err != nil || rec == nil {
		return backlinks.IndexedRecord{}, false
	}
	converted := convertBacklinkRecords([]firehose.IndexedRecord{*rec})
	return converted[0], true
}

func convertBacklinkRecords(recs []firehose.IndexedRecord) []backlinks.IndexedRecord {
	out := make([]backlinks.IndexedRecord, 0, len(recs))
	for _, rec := range recs {
		out = append(out, backlinks.IndexedRecord{
			URI:        rec.URI,
			DID:        rec.DID,
			Collection: rec.Collection,
			RKey:       rec.RKey,
			Record:     rec.Record,
			CreatedAt:  rec.CreatedAt,
		})
	}
	return out
}

func (s backlinkIndexSource) GetProfile(ctx context.Context, did string) (*backlinks.Profile, error) {
	p, err := s.idx.GetProfile(ctx, did)
	if err != nil || p == nil {
		return nil, err
	}
	out := &backlinks.Profile{Handle: p.Handle}
	if p.DisplayName != nil {
		out.DisplayName = *p.DisplayName
	}
	if p.Avatar != nil {
		out.AvatarURL = *p.Avatar
	}
	return out, nil
}

func (h *Handler) RenderBacklinksView(w http.ResponseWriter, r *http.Request, cfg EntityViewConfig) {
	rkey := ValidateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	owner := r.URL.Query().Get("owner")
	didStr, _ := atpmiddleware.GetDID(r.Context())
	isAuthenticated := didStr != ""
	if owner == "" && !isAuthenticated {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.GetUserProfile(r.Context(), didStr)
	}

	loaded, err := h.EntityViewLoader().Load(r, rkey, cfg.loadConfig())
	if err != nil {
		if loadErr, ok := err.(*EntityLoadError); ok {
			http.Error(w, loadErr.Msg, loadErr.HTTPStatus())
		} else {
			http.Error(w, "Failed to load record", http.StatusInternalServerError)
		}
		return
	}

	name := cfg.DisplayName(loaded.Record)
	if name == "" && cfg.Descriptor != nil {
		name = cfg.Descriptor.DisplayName
	}
	ownerID := ownerSegment(owner, userProfile, didStr)
	backURL := fmt.Sprintf("/%s/%s/%s", loaded.Route.Path, ownerID, rkey)
	usageKey := r.URL.Query().Get("usage")
	usagePage, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if usagePage <= 0 {
		usagePage = 1
	}
	result, detailURL := h.fetchBacklinksWithOptions(r.Context(), loaded.SubjectURI, loaded.Route.Path, rkey, ownerID, backlinks.LookupOptions{UsageKey: usageKey, UsagePage: usagePage, UsagePerPage: 25})

	layoutData := h.BuildLayoutData(r, "Community · "+name, isAuthenticated, didStr, userProfile)
	props := pages.BacklinksViewProps{
		EntityNoun: strings.ToLower(loaded.EntityNoun),
		EntityName: name,
		BackURL:    backURL,
		DetailURL:  detailURL,
		Result:     result,
		RoutePaths: h.entityRoutePaths(),
	}
	if err := pages.BacklinksView(layoutData, props).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render", http.StatusInternalServerError)
		log.Error().Err(err).Msg("failed to render backlinks page")
	}
}

func (h *Handler) entityRoutePaths() map[lexicons.RecordType]string {
	paths := map[lexicons.RecordType]string{}
	if h.app == nil {
		return paths
	}
	for _, route := range h.app.EntityRoutes {
		paths[route.Type] = route.Path
	}
	return paths
}

func (h *Handler) entityRouteFor(desc *entities.Descriptor) domain.EntityRoute {
	if desc == nil {
		return domain.EntityRoute{}
	}
	if route, ok := h.app.EntityRouteByType(desc.Type); ok {
		return route
	}
	return domain.EntityRoute{Type: desc.Type, Noun: strings.ToLower(desc.DisplayName)}
}

// OGImageConfig captures per-entity behavior for HandleSimpleOGImage.
// Fields exported so per-app packages can populate this struct directly.
type OGImageConfig struct {
	NSID        string
	MetricLabel string
	Convert     func(m map[string]any, uri, rkey string) (any, error)
	DrawCard    func(record any) (*ogcard.Card, error)
}

// handleSimpleOGImage serves a simple entity OG image (no nested ref resolution).
// Bean and Recipe have bespoke handlers due to nested ref resolution.
func (h *Handler) HandleSimpleOGImage(w http.ResponseWriter, r *http.Request, cfg OGImageConfig) {
	rkey := ValidateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	owner := r.URL.Query().Get("owner")
	if owner == "" {
		http.Error(w, "owner parameter required", http.StatusBadRequest)
		return
	}
	ownerDID, err := ResolveOwnerDID(r.Context(), owner)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	var record any
	entityURI := atp.BuildATURI(ownerDID, cfg.NSID, rkey)
	if h.witnessCache != nil {
		if wr, _ := h.witnessCache.GetWitnessRecord(r.Context(), entityURI); wr != nil {
			if m, err := atproto.WitnessRecordToMap(wr); err == nil {
				if rec, err := cfg.Convert(m, wr.URI, rkey); err == nil {
					metrics.WitnessCacheHitsTotal.WithLabelValues(cfg.MetricLabel).Inc()
					record = rec
				}
			}
		}
	}
	if record == nil {
		metrics.WitnessCacheMissesTotal.WithLabelValues(cfg.MetricLabel).Inc()
		pub := atproto.NewPublicClient()
		pr, err := pub.GetPublicRecord(r.Context(), ownerDID, cfg.NSID, rkey)
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		rec, err := cfg.Convert(pr.Value, pr.URI, rkey)
		if err != nil {
			http.Error(w, "Failed to load record", http.StatusInternalServerError)
			return
		}
		record = rec
	}
	card, err := cfg.DrawCard(record)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to generate %s OG image", cfg.MetricLabel)
		http.Error(w, "Failed to generate image", http.StatusInternalServerError)
		return
	}
	WriteOGImage(w, card)
}

// WriteOGImage encodes a card as PNG with appropriate cache headers.
func WriteOGImage(w http.ResponseWriter, card *ogcard.Card) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	if err := card.EncodePNG(w); err != nil {
		log.Error().Err(err).Msg("Failed to encode OG image")
	}
}
