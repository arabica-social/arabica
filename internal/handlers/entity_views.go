package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/firehose"
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
	// ResolveRefs, if set, runs after a record is decoded on any of the
	// three source paths (own-store, witness, PDS). It receives the
	// typed model, the raw record map (to read ref AT-URIs), and a
	// source-bound lookup that fetches foreign records. Implementations
	// must be idempotent — for the own-store path the codec PostGet
	// may have already populated some ref fields.
	ResolveRefs func(ctx context.Context, model any, raw map[string]any, lookup func(refURI string) (map[string]any, bool))
	DisplayName func(record any) string
	OGSubtitle  func(record any) string
	CountLookup func(ctx context.Context, ownerDID, subjectURI string) int
	Render      func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base pages.EntityViewBase) error
}

func (h *Handler) RenderEntityView(w http.ResponseWriter, r *http.Request, cfg EntityViewConfig) {
	rkey := ValidateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	route := h.entityRouteFor(cfg.Descriptor)
	entityNoun := route.Noun
	if entityNoun == "" {
		entityNoun = "content"
	}

	owner := r.URL.Query().Get("owner")
	didStr, _ := atpmiddleware.GetDID(r.Context())
	isAuthenticated := didStr != ""

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.GetUserProfile(r.Context(), didStr)
	}

	var record any
	var subjectURI, subjectCID, entityOwnerDID string
	isOwnProfile := false

	if owner == "" {
		http.Error(w, "owner required", http.StatusBadRequest)
		return
	}

	var err error
	entityOwnerDID, err = ResolveOwnerDID(r.Context(), owner)
	if err != nil {
		log.Warn().Err(err).Str("handle", owner).Msgf("Failed to resolve handle for %s view", entityNoun)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	isOwnProfile = isAuthenticated && didStr == entityOwnerDID

	// If the viewer owns the record, read through the authenticated
	// AtprotoStore so locally-written records that the firehose hasn't
	// caught up to are still visible.
	if isOwnProfile {
		if store, ok := h.GetRecordStore(r); ok {
			if atprotoStore, ok := store.(*atproto.AtprotoStore); ok {
				if rec, raw, uri, cid, err := cfg.FromStore(r.Context(), atprotoStore, rkey); err == nil {
					record, subjectURI, subjectCID = rec, uri, cid
					if cfg.ResolveRefs != nil {
						cfg.ResolveRefs(r.Context(), record, raw, h.WitnessLookup(r.Context()))
					}
				}
			}
		}
	}

	if record == nil {
		entityURI := atp.BuildATURI(entityOwnerDID, cfg.Descriptor.NSID, rkey)
		if h.witnessCache != nil {
			if wr, _ := h.witnessCache.GetWitnessRecord(r.Context(), entityURI); wr != nil {
				if m, err := atproto.WitnessRecordToMap(wr); err == nil {
					if rec, err := cfg.FromWitness(r.Context(), m, wr.URI, rkey, entityOwnerDID); err == nil {
						metrics.WitnessCacheHitsTotal.WithLabelValues(entityNoun).Inc()
						record = rec
						subjectURI = wr.URI
						subjectCID = wr.CID
						if cfg.ResolveRefs != nil {
							cfg.ResolveRefs(r.Context(), record, m, h.WitnessLookup(r.Context()))
						}
					}
				}
			}
		}
	}

	if record == nil {
		metrics.WitnessCacheMissesTotal.WithLabelValues(entityNoun).Inc()
		pub := atproto.NewPublicClient()
		entry, err := pub.GetPublicRecord(r.Context(), entityOwnerDID, cfg.Descriptor.NSID, rkey)
		if err != nil {
			log.Error().Err(err).Str("did", entityOwnerDID).Str("rkey", rkey).Msgf("Failed to get %s record", entityNoun)
			http.Error(w, cfg.Descriptor.DisplayName+" not found", http.StatusNotFound)
			return
		}
		rec, err := cfg.FromPDS(r.Context(), entry, rkey, entityOwnerDID)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to convert %s record", entityNoun)
			http.Error(w, "Failed to load "+entityNoun, http.StatusInternalServerError)
			return
		}
		record = rec
		subjectURI = entry.URI
		subjectCID = entry.CID
		if cfg.ResolveRefs != nil {
			cfg.ResolveRefs(r.Context(), record, entry.Value, PublicLookup(r.Context()))
		}
	}

	var shareURL string
	if owner != "" && route.Path != "" {
		shareURL = fmt.Sprintf("/%s/%s/%s", route.Path, owner, rkey)
	} else if userProfile != nil && userProfile.Handle != "" && route.Path != "" {
		shareURL = fmt.Sprintf("/%s/%s/%s", route.Path, userProfile.Handle, rkey)
	}

	ownerHandle := h.ResolveOwnerHandle(r.Context(), owner)
	layoutData := h.BuildLayoutData(r, cfg.DisplayName(record), isAuthenticated, didStr, userProfile)
	PopulateOGFields(layoutData, cfg.OGSubtitle(record), entityNoun, ownerHandle, h.PublicBaseURL(r), shareURL)

	sd := h.FetchSocialData(r.Context(), subjectURI, didStr, isAuthenticated)

	authorDID := entityOwnerDID
	if authorDID == "" {
		authorDID = didStr
	}
	base := pages.EntityViewBase{
		IsOwnProfile:    isOwnProfile,
		IsAuthenticated: isAuthenticated,
		SubjectURI:      subjectURI,
		SubjectCID:      subjectCID,
		IsLiked:         sd.IsLiked,
		LikeCount:       sd.LikeCount,
		CommentCount:    sd.CommentCount,
		Comments:        sd.Comments,
		CurrentUserDID:  didStr,
		ShareURL:        shareURL,
		IsModerator:     sd.IsModerator,
		CanHideRecord:   sd.CanHideRecord,
		CanBlockUser:    sd.CanBlockUser,
		IsRecordHidden:  sd.IsRecordHidden,
		AuthorDID:       entityOwnerDID,
	}
	if ap := h.GetUserProfile(r.Context(), authorDID); ap != nil {
		base.AuthorHandle = ap.Handle
		base.AuthorDisplayName = ap.DisplayName
		base.AuthorAvatar = ap.Avatar
	}

	if err := cfg.Render(r.Context(), w, layoutData, record, base); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msgf("Failed to render %s view", entityNoun)
	}
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
