package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"tangled.org/arabica.social/arabica/internal/arabica/web/pages"
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/entities/arabica"
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

// socialData holds the social interaction data shared across all entity view handlers
type socialData struct {
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
func (h *Handler) fetchSocialData(ctx context.Context, subjectURI, didStr string, isAuthenticated bool) socialData {
	var sd socialData

	if h.feedIndex != nil && subjectURI != "" {
		sd.LikeCount = h.feedIndex.GetLikeCount(ctx, subjectURI)
		sd.CommentCount = h.feedIndex.GetCommentCount(ctx, subjectURI)
		sd.Comments = h.feedIndex.GetThreadedCommentsForSubject(ctx, subjectURI, 100, didStr)
		sd.Comments = h.filterHiddenComments(ctx, sd.Comments)
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

// resolveOwnerDID resolves an owner parameter (DID or handle) to a DID string.
// Returns the DID and nil error on success, or empty string and error on failure.
func resolveOwnerDID(ctx context.Context, owner string) (string, error) {
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

// entityViewConfig captures per-entity behavior for handleEntityView.
// Construct via the h.xViewConfig() methods — closures capture h naturally.
type entityViewConfig struct {
	descriptor  *entities.Descriptor
	fromWitness func(ctx context.Context, m map[string]any, uri, rkey, ownerDID string) (any, error)
	fromPDS     func(ctx context.Context, e *atp.Record, rkey, ownerDID string) (any, error)
	fromStore   func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, map[string]any, string, string, error)
	// resolveRefs, if set, runs after a record is decoded on any of the
	// three source paths (own-store, witness, PDS). It receives the
	// typed model, the raw record map (to read ref AT-URIs), and a
	// source-bound lookup that fetches foreign records. Implementations
	// must be idempotent — for the own-store path the codec PostGet
	// may have already populated some ref fields.
	resolveRefs func(ctx context.Context, model any, raw map[string]any, lookup func(refURI string) (map[string]any, bool))
	displayName func(record any) string
	ogSubtitle  func(record any) string
	countLookup func(ctx context.Context, ownerDID, subjectURI string) int
	render      func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base pages.EntityViewBase) error
}

func (h *Handler) handleEntityView(w http.ResponseWriter, r *http.Request, cfg entityViewConfig) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	owner := r.URL.Query().Get("owner")
	didStr, _ := atpmiddleware.GetDID(r.Context())
	isAuthenticated := didStr != ""

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.getUserProfile(r.Context(), didStr)
	}

	var record any
	var subjectURI, subjectCID, entityOwnerDID string
	isOwnProfile := false

	if owner == "" {
		http.Error(w, "owner required", http.StatusBadRequest)
		return
	}

	var err error
	entityOwnerDID, err = resolveOwnerDID(r.Context(), owner)
	if err != nil {
		log.Warn().Err(err).Str("handle", owner).Msgf("Failed to resolve handle for %s view", cfg.descriptor.Noun)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	isOwnProfile = isAuthenticated && didStr == entityOwnerDID

	// If the viewer owns the record, read through the authenticated
	// AtprotoStore so locally-written records that the firehose hasn't
	// caught up to are still visible.
	if isOwnProfile {
		if store, ok := h.getAtprotoStore(r); ok {
			if atprotoStore, ok := store.(*atproto.AtprotoStore); ok {
				if rec, raw, uri, cid, err := cfg.fromStore(r.Context(), atprotoStore, rkey); err == nil {
					record, subjectURI, subjectCID = rec, uri, cid
					if cfg.resolveRefs != nil {
						cfg.resolveRefs(r.Context(), record, raw, h.witnessLookup(r.Context()))
					}
				}
			}
		}
	}

	if record == nil {
		entityURI := atp.BuildATURI(entityOwnerDID, cfg.descriptor.NSID, rkey)
		if h.witnessCache != nil {
			if wr, _ := h.witnessCache.GetWitnessRecord(r.Context(), entityURI); wr != nil {
				if m, err := atproto.WitnessRecordToMap(wr); err == nil {
					if rec, err := cfg.fromWitness(r.Context(), m, wr.URI, rkey, entityOwnerDID); err == nil {
						metrics.WitnessCacheHitsTotal.WithLabelValues(cfg.descriptor.Noun).Inc()
						record = rec
						subjectURI = wr.URI
						subjectCID = wr.CID
						if cfg.resolveRefs != nil {
							cfg.resolveRefs(r.Context(), record, m, h.witnessLookup(r.Context()))
						}
					}
				}
			}
		}
	}

	if record == nil {
		metrics.WitnessCacheMissesTotal.WithLabelValues(cfg.descriptor.Noun).Inc()
		pub := atproto.NewPublicClient()
		entry, err := pub.GetPublicRecord(r.Context(), entityOwnerDID, cfg.descriptor.NSID, rkey)
		if err != nil {
			log.Error().Err(err).Str("did", entityOwnerDID).Str("rkey", rkey).Msgf("Failed to get %s record", cfg.descriptor.Noun)
			http.Error(w, cfg.descriptor.DisplayName+" not found", http.StatusNotFound)
			return
		}
		rec, err := cfg.fromPDS(r.Context(), entry, rkey, entityOwnerDID)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to convert %s record", cfg.descriptor.Noun)
			http.Error(w, "Failed to load "+cfg.descriptor.Noun, http.StatusInternalServerError)
			return
		}
		record = rec
		subjectURI = entry.URI
		subjectCID = entry.CID
		if cfg.resolveRefs != nil {
			cfg.resolveRefs(r.Context(), record, entry.Value, publicLookup(r.Context()))
		}
	}

	var shareURL string
	if owner != "" {
		shareURL = fmt.Sprintf("/%s/%s/%s", cfg.descriptor.URLPath, owner, rkey)
	} else if userProfile != nil && userProfile.Handle != "" {
		shareURL = fmt.Sprintf("/%s/%s/%s", cfg.descriptor.URLPath, userProfile.Handle, rkey)
	}

	ownerHandle := h.resolveOwnerHandle(r.Context(), owner)
	layoutData := h.buildLayoutData(r, cfg.displayName(record), isAuthenticated, didStr, userProfile)
	populateOGFields(layoutData, cfg.ogSubtitle(record), cfg.descriptor.Noun, ownerHandle, h.publicBaseURL(r), shareURL)

	sd := h.fetchSocialData(r.Context(), subjectURI, didStr, isAuthenticated)

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
	if ap := h.getUserProfile(r.Context(), authorDID); ap != nil {
		base.AuthorHandle = ap.Handle
		base.AuthorDisplayName = ap.DisplayName
		base.AuthorAvatar = ap.Avatar
	}

	if err := cfg.render(r.Context(), w, layoutData, record, base); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msgf("Failed to render %s view", cfg.descriptor.Noun)
	}
}

func (h *Handler) roasterViewConfig() entityViewConfig {
	fromWitness, fromPDS, fromStore := standardViewTriple(
		arabica.NSIDRoaster, arabica.RecordToRoaster,
		func(r *arabica.Roaster, k string) { r.RKey = k },
	)
	return entityViewConfig{
		descriptor:  entities.Get(lexicons.RecordTypeRoaster),
		fromWitness: fromWitness,
		fromPDS:     fromPDS,
		fromStore:   fromStore,
		displayName: func(record any) string { return record.(*arabica.Roaster).Name },
		ogSubtitle:  func(record any) string { return record.(*arabica.Roaster).Name },
		countLookup: func(ctx context.Context, ownerDID, subjectURI string) int {
			if h.feedIndex == nil || subjectURI == "" {
				return 0
			}
			return h.feedIndex.BeanCountsByRoasterURI(ctx, ownerDID)[subjectURI]
		},
		render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base pages.EntityViewBase) error {
			roaster := record.(*arabica.Roaster)
			props := coffeepages.RoasterViewProps{
				Roaster:        roaster,
				EntityViewBase: base,
			}
			if h.feedIndex != nil && base.SubjectURI != "" {
				ownerDID := base.AuthorDID
				if ownerDID == "" {
					ownerDID = base.CurrentUserDID
				}
				props.BeanCount = h.feedIndex.BeanCountsByRoasterURI(ctx, ownerDID)[base.SubjectURI]
			}
			return coffeepages.RoasterView(layoutData, props).Render(ctx, w)
		},
	}
}

func (h *Handler) grinderViewConfig() entityViewConfig {
	fromWitness, fromPDS, fromStore := standardViewTriple(
		arabica.NSIDGrinder, arabica.RecordToGrinder,
		func(g *arabica.Grinder, k string) { g.RKey = k },
	)
	return entityViewConfig{
		descriptor:  entities.Get(lexicons.RecordTypeGrinder),
		fromWitness: fromWitness,
		fromPDS:     fromPDS,
		fromStore:   fromStore,
		displayName: func(record any) string { return record.(*arabica.Grinder).Name },
		ogSubtitle:  func(record any) string { return record.(*arabica.Grinder).Name },
		render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base pages.EntityViewBase) error {
			grinder := record.(*arabica.Grinder)
			props := coffeepages.GrinderViewProps{
				Grinder:        grinder,
				EntityViewBase: base,
			}
			if h.feedIndex != nil && base.SubjectURI != "" {
				ownerDID := base.AuthorDID
				if ownerDID == "" {
					ownerDID = base.CurrentUserDID
				}
				props.BrewCount = h.feedIndex.BrewCountsByGrinderURI(ctx, ownerDID)[base.SubjectURI]
			}
			return coffeepages.GrinderView(layoutData, props).Render(ctx, w)
		},
	}
}

func (h *Handler) brewerViewConfig() entityViewConfig {
	fromWitness, fromPDS, fromStore := standardViewTriple(
		arabica.NSIDBrewer, arabica.RecordToBrewer,
		func(b *arabica.Brewer, k string) { b.RKey = k },
	)
	return entityViewConfig{
		descriptor:  entities.Get(lexicons.RecordTypeBrewer),
		fromWitness: fromWitness,
		fromPDS:     fromPDS,
		fromStore:   fromStore,
		displayName: func(record any) string { return record.(*arabica.Brewer).Name },
		ogSubtitle:  func(record any) string { return record.(*arabica.Brewer).Name },
		render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base pages.EntityViewBase) error {
			brewer := record.(*arabica.Brewer)
			props := coffeepages.BrewerViewProps{
				Brewer:         brewer,
				EntityViewBase: base,
			}
			if h.feedIndex != nil && base.SubjectURI != "" {
				ownerDID := base.AuthorDID
				if ownerDID == "" {
					ownerDID = base.CurrentUserDID
				}
				props.BrewCount = h.feedIndex.BrewCountsByBrewerURI(ctx, ownerDID)[base.SubjectURI]
			}
			return coffeepages.BrewerView(layoutData, props).Render(ctx, w)
		},
	}
}

func (h *Handler) beanViewConfig() entityViewConfig {
	fromWitness, fromPDS, fromStore := standardViewTriple(
		arabica.NSIDBean, arabica.RecordToBean,
		func(b *arabica.Bean, k string) { b.RKey = k },
	)
	return entityViewConfig{
		descriptor:  entities.Get(lexicons.RecordTypeBean),
		fromWitness: fromWitness,
		fromPDS:     fromPDS,
		fromStore:   fromStore,
		resolveRefs: func(_ context.Context, model any, raw map[string]any, lookup func(string) (map[string]any, bool)) {
			bean := model.(*arabica.Bean)
			roasterRef, _ := raw["roasterRef"].(string)
			if roasterRef == "" {
				return
			}
			if bean.RoasterRKey == "" {
				if rk := atp.RKeyFromURI(roasterRef); rk != "" {
					bean.RoasterRKey = rk
				}
			}
			if bean.Roaster != nil {
				return
			}
			m, ok := lookup(roasterRef)
			if !ok {
				return
			}
			if roaster, err := arabica.RecordToRoaster(m, roasterRef); err == nil {
				roaster.RKey = bean.RoasterRKey
				bean.Roaster = roaster
			}
		},
		displayName: func(record any) string { return record.(*arabica.Bean).Name },
		ogSubtitle: func(record any) string {
			bean := record.(*arabica.Bean)
			sub := bean.Name
			if sub == "" {
				sub = bean.Origin
			}
			if bean.Roaster != nil && bean.Roaster.Name != "" {
				sub += " from " + bean.Roaster.Name
			}
			return sub
		},
		render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base pages.EntityViewBase) error {
			bean := record.(*arabica.Bean)
			props := coffeepages.BeanViewProps{
				Bean:           bean,
				EntityViewBase: base,
			}
			if h.feedIndex != nil && base.SubjectURI != "" {
				ownerDID := base.AuthorDID
				if ownerDID == "" {
					ownerDID = base.CurrentUserDID
				}
				props.BrewCount = h.feedIndex.BrewCountsByBeanURI(ctx, ownerDID)[base.SubjectURI]
			}
			return coffeepages.BeanView(layoutData, props).Render(ctx, w)
		},
	}
}

// HandleBeanView shows a bean detail page with social features
func (h *Handler) HandleBeanView(w http.ResponseWriter, r *http.Request) {
	h.handleEntityView(w, r, h.beanViewConfig())
}

// HandleRoasterView shows a roaster detail page with social features
func (h *Handler) HandleRoasterView(w http.ResponseWriter, r *http.Request) {
	h.handleEntityView(w, r, h.roasterViewConfig())
}

// HandleGrinderView shows a grinder detail page with social features
func (h *Handler) HandleGrinderView(w http.ResponseWriter, r *http.Request) {
	h.handleEntityView(w, r, h.grinderViewConfig())
}

// HandleBrewerView shows a brewer detail page with social features
func (h *Handler) HandleBrewerView(w http.ResponseWriter, r *http.Request) {
	h.handleEntityView(w, r, h.brewerViewConfig())
}

func (h *Handler) recipeViewConfig() entityViewConfig {
	fromWitness, fromPDS, fromStore := standardViewTriple(
		arabica.NSIDRecipe, arabica.RecordToRecipe,
		func(r *arabica.Recipe, k string) { r.RKey = k },
	)
	return entityViewConfig{
		descriptor:  entities.Get(lexicons.RecordTypeRecipe),
		fromWitness: fromWitness,
		fromPDS:     fromPDS,
		fromStore:   fromStore,
		resolveRefs: func(_ context.Context, model any, raw map[string]any, lookup func(string) (map[string]any, bool)) {
			recipe := model.(*arabica.Recipe)
			brewerRef, _ := raw["brewerRef"].(string)
			if brewerRef != "" {
				if recipe.BrewerRKey == "" {
					if rk := atp.RKeyFromURI(brewerRef); rk != "" {
						recipe.BrewerRKey = rk
					}
				}
				if recipe.BrewerObj == nil {
					if m, ok := lookup(brewerRef); ok {
						if brewer, err := arabica.RecordToBrewer(m, brewerRef); err == nil {
							brewer.RKey = recipe.BrewerRKey
							recipe.BrewerObj = brewer
						}
					}
				}
			}
			recipe.Interpolate()
		},
		displayName: func(record any) string { return record.(*arabica.Recipe).Name },
		ogSubtitle:  func(record any) string { return record.(*arabica.Recipe).Name },
		render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base pages.EntityViewBase) error {
			recipe := record.(*arabica.Recipe)
			props := coffeepages.RecipeViewProps{
				Recipe:            recipe,
				IsOwnProfile:      base.IsOwnProfile,
				IsAuthenticated:   base.IsAuthenticated,
				SubjectURI:        base.SubjectURI,
				SubjectCID:        base.SubjectCID,
				IsLiked:           base.IsLiked,
				LikeCount:         base.LikeCount,
				CommentCount:      base.CommentCount,
				Comments:          base.Comments,
				CurrentUserDID:    base.CurrentUserDID,
				ShareURL:          base.ShareURL,
				IsModerator:       base.IsModerator,
				CanHideRecord:     base.CanHideRecord,
				CanBlockUser:      base.CanBlockUser,
				IsRecordHidden:    base.IsRecordHidden,
				AuthorDID:         base.AuthorDID,
				AuthorHandle:      base.AuthorHandle,
				AuthorDisplayName: base.AuthorDisplayName,
				AuthorAvatar:      base.AuthorAvatar,
			}
			if recipe.SourceRef != "" {
				if srcURI, err := atp.ParseATURI(recipe.SourceRef); err == nil {
					sourceOwner := srcURI.DID
					if h.feedIndex != nil {
						if profile, err := h.feedIndex.GetProfile(ctx, srcURI.DID); err == nil && profile != nil {
							sourceOwner = profile.Handle
							if profile.DisplayName != nil && *profile.DisplayName != "" {
								props.SourceRecipeAuthor = *profile.DisplayName
							} else {
								props.SourceRecipeAuthor = profile.Handle
							}
						}
					}
					props.SourceRecipeURL = fmt.Sprintf("/recipes/%s/%s", sourceOwner, srcURI.RKey)
				}
			}
			return coffeepages.RecipeView(layoutData, props).Render(ctx, w)
		},
	}
}

// HandleRecipeView displays a recipe detail page
func (h *Handler) HandleRecipeView(w http.ResponseWriter, r *http.Request) {
	h.handleEntityView(w, r, h.recipeViewConfig())
}

func (h *Handler) brewViewConfig() entityViewConfig {
	fromWitness, fromPDS, _ := standardViewTriple(
		arabica.NSIDBrew, arabica.RecordToBrew,
		func(b *arabica.Brew, k string) { b.RKey = k },
	)
	// Custom fromStore that also extracts the rkeys from ref AT-URIs.
	// We deliberately bypass GetBrewRecordByRKey here: that path resolves
	// refs via the authenticated atpClient as a fallback for session-cache
	// hits that aren't in witness yet, but in practice users select refs
	// from existing entities, so witness lookup via resolveRefs covers it.
	// Worst case: a fresh brew whose refs are also fresh-writes shows
	// unresolved refs until the next firehose catchup.
	fromStore := func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, map[string]any, string, string, error) {
		raw, uri, cid, err := s.FetchRecord(ctx, arabica.NSIDBrew, rkey)
		if err != nil {
			return nil, nil, "", "", err
		}
		brew, err := arabica.RecordToBrew(raw, uri)
		if err != nil {
			return nil, nil, "", "", err
		}
		brew.RKey = rkey
		atproto.ExtractBrewRefRKeys(brew, raw)
		return brew, raw, uri, cid, nil
	}
	return entityViewConfig{
		descriptor:  entities.Get(lexicons.RecordTypeBrew),
		fromWitness: fromWitness,
		fromPDS:     fromPDS,
		fromStore:   fromStore,
		resolveRefs: func(_ context.Context, model any, raw map[string]any, lookup func(string) (map[string]any, bool)) {
			brew := model.(*arabica.Brew)
			atproto.ExtractBrewRefRKeys(brew, raw)
			resolveBrewRefsViaLookup(brew, raw, lookup)
		},
		displayName: func(any) string { return "Brew Details" },
		ogSubtitle: func(record any) string {
			brew := record.(*arabica.Brew)
			var sub string
			if brew.Bean != nil {
				sub = brew.Bean.Name
				if brew.Bean.Roaster != nil && brew.Bean.Roaster.Name != "" {
					sub += " from " + brew.Bean.Roaster.Name
				}
			}
			return sub
		},
		render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base pages.EntityViewBase) error {
			brew := record.(*arabica.Brew)
			props := coffeepages.BrewViewProps{
				Brew:              brew,
				IsOwnProfile:      base.IsOwnProfile,
				IsAuthenticated:   base.IsAuthenticated,
				SubjectURI:        base.SubjectURI,
				SubjectCID:        base.SubjectCID,
				IsLiked:           base.IsLiked,
				LikeCount:         base.LikeCount,
				CommentCount:      base.CommentCount,
				Comments:          base.Comments,
				CurrentUserDID:    base.CurrentUserDID,
				ShareURL:          base.ShareURL,
				IsModerator:       base.IsModerator,
				CanHideRecord:     base.CanHideRecord,
				CanBlockUser:      base.CanBlockUser,
				IsRecordHidden:    base.IsRecordHidden,
				AuthorDID:         base.AuthorDID,
				AuthorHandle:      base.AuthorHandle,
				AuthorDisplayName: base.AuthorDisplayName,
				AuthorAvatar:      base.AuthorAvatar,
			}
			return coffeepages.BrewView(layoutData, props).Render(ctx, w)
		},
	}
}

// OG image handlers for entity types

// HandleBeanOGImage generates a 1200x630 PNG preview card for a bean.
func (h *Handler) HandleBeanOGImage(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	owner := r.URL.Query().Get("owner")
	if owner == "" {
		http.Error(w, "owner parameter required", http.StatusBadRequest)
		return
	}

	ownerDID, err := resolveOwnerDID(r.Context(), owner)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var bean *arabica.Bean
	beanURI := atp.BuildATURI(ownerDID, arabica.NSIDBean, rkey)
	if h.witnessCache != nil {
		if wr, _ := h.witnessCache.GetWitnessRecord(r.Context(), beanURI); wr != nil {
			if m, err := atproto.WitnessRecordToMap(wr); err == nil {
				if b, err := arabica.RecordToBean(m, wr.URI); err == nil {
					metrics.WitnessCacheHitsTotal.WithLabelValues("bean_og").Inc()
					bean = b
					bean.RKey = rkey
					// Resolve roaster
					if roasterRef, ok := m["roasterRef"].(string); ok && roasterRef != "" {
						if rwr, _ := h.witnessCache.GetWitnessRecord(r.Context(), roasterRef); rwr != nil {
							if rm, err := atproto.WitnessRecordToMap(rwr); err == nil {
								if roaster, err := arabica.RecordToRoaster(rm, rwr.URI); err == nil {
									bean.Roaster = roaster
								}
							}
						}
					}
				}
			}
		}
	}
	if bean == nil {
		metrics.WitnessCacheMissesTotal.WithLabelValues("bean_og").Inc()
		publicClient := atproto.NewPublicClient()
		record, err := publicClient.GetPublicRecord(r.Context(), ownerDID, arabica.NSIDBean, rkey)
		if err != nil {
			http.Error(w, "Bean not found", http.StatusNotFound)
			return
		}
		bean, err = arabica.RecordToBean(record.Value, record.URI)
		if err != nil {
			http.Error(w, "Failed to load bean", http.StatusInternalServerError)
			return
		}
		// Resolve roaster reference
		if roasterRef, ok := record.Value["roasterRef"].(string); ok && roasterRef != "" {
			roasterRKey := atp.RKeyFromURI(roasterRef)
			if roasterRKey != "" {
				if rr, err := publicClient.GetPublicRecord(r.Context(), ownerDID, arabica.NSIDRoaster, roasterRKey); err == nil {
					if roaster, err := arabica.RecordToRoaster(rr.Value, rr.URI); err == nil {
						bean.Roaster = roaster
					}
				}
			}
		}
	}

	card, err := ogcard.DrawBeanCard(bean)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate bean OG image")
		http.Error(w, "Failed to generate image", http.StatusInternalServerError)
		return
	}
	writeOGImage(w, card)
}

// ogImageConfig captures per-entity behavior for handleSimpleOGImage.
type ogImageConfig struct {
	nsid        string
	metricLabel string
	convert     func(m map[string]any, uri, rkey string) (any, error)
	drawCard    func(record any) (*ogcard.Card, error)
}

// handleSimpleOGImage serves a simple entity OG image (no nested ref resolution).
// Bean and Recipe have bespoke handlers due to nested ref resolution.
func (h *Handler) handleSimpleOGImage(w http.ResponseWriter, r *http.Request, cfg ogImageConfig) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	owner := r.URL.Query().Get("owner")
	if owner == "" {
		http.Error(w, "owner parameter required", http.StatusBadRequest)
		return
	}
	ownerDID, err := resolveOwnerDID(r.Context(), owner)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	var record any
	entityURI := atp.BuildATURI(ownerDID, cfg.nsid, rkey)
	if h.witnessCache != nil {
		if wr, _ := h.witnessCache.GetWitnessRecord(r.Context(), entityURI); wr != nil {
			if m, err := atproto.WitnessRecordToMap(wr); err == nil {
				if rec, err := cfg.convert(m, wr.URI, rkey); err == nil {
					metrics.WitnessCacheHitsTotal.WithLabelValues(cfg.metricLabel).Inc()
					record = rec
				}
			}
		}
	}
	if record == nil {
		metrics.WitnessCacheMissesTotal.WithLabelValues(cfg.metricLabel).Inc()
		pub := atproto.NewPublicClient()
		pr, err := pub.GetPublicRecord(r.Context(), ownerDID, cfg.nsid, rkey)
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		rec, err := cfg.convert(pr.Value, pr.URI, rkey)
		if err != nil {
			http.Error(w, "Failed to load record", http.StatusInternalServerError)
			return
		}
		record = rec
	}
	card, err := cfg.drawCard(record)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to generate %s OG image", cfg.metricLabel)
		http.Error(w, "Failed to generate image", http.StatusInternalServerError)
		return
	}
	writeOGImage(w, card)
}

// HandleRoasterOGImage generates a 1200x630 PNG preview card for a roaster.
func (h *Handler) HandleRoasterOGImage(w http.ResponseWriter, r *http.Request) {
	h.handleSimpleOGImage(w, r, ogImageConfig{
		nsid: arabica.NSIDRoaster, metricLabel: "roaster_og",
		convert: func(m map[string]any, uri, rkey string) (any, error) {
			rec, err := arabica.RecordToRoaster(m, uri)
			if err != nil {
				return nil, err
			}
			rec.RKey = rkey
			return rec, nil
		},
		drawCard: func(rec any) (*ogcard.Card, error) { return ogcard.DrawRoasterCard(rec.(*arabica.Roaster)) },
	})
}

// HandleGrinderOGImage generates a 1200x630 PNG preview card for a grinder.
func (h *Handler) HandleGrinderOGImage(w http.ResponseWriter, r *http.Request) {
	h.handleSimpleOGImage(w, r, ogImageConfig{
		nsid: arabica.NSIDGrinder, metricLabel: "grinder_og",
		convert: func(m map[string]any, uri, rkey string) (any, error) {
			rec, err := arabica.RecordToGrinder(m, uri)
			if err != nil {
				return nil, err
			}
			rec.RKey = rkey
			return rec, nil
		},
		drawCard: func(rec any) (*ogcard.Card, error) { return ogcard.DrawGrinderCard(rec.(*arabica.Grinder)) },
	})
}

// HandleBrewerOGImage generates a 1200x630 PNG preview card for a brewer.
func (h *Handler) HandleBrewerOGImage(w http.ResponseWriter, r *http.Request) {
	h.handleSimpleOGImage(w, r, ogImageConfig{
		nsid: arabica.NSIDBrewer, metricLabel: "brewer_og",
		convert: func(m map[string]any, uri, rkey string) (any, error) {
			rec, err := arabica.RecordToBrewer(m, uri)
			if err != nil {
				return nil, err
			}
			rec.RKey = rkey
			return rec, nil
		},
		drawCard: func(rec any) (*ogcard.Card, error) { return ogcard.DrawBrewerCard(rec.(*arabica.Brewer)) },
	})
}

// HandleRecipeOGImage generates a 1200x630 PNG preview card for a recipe.
func (h *Handler) HandleRecipeOGImage(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	owner := r.URL.Query().Get("owner")
	if owner == "" {
		http.Error(w, "owner parameter required", http.StatusBadRequest)
		return
	}

	ownerDID, err := resolveOwnerDID(r.Context(), owner)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var recipe *arabica.Recipe
	recipeURI := atp.BuildATURI(ownerDID, arabica.NSIDRecipe, rkey)
	if h.witnessCache != nil {
		if wr, _ := h.witnessCache.GetWitnessRecord(r.Context(), recipeURI); wr != nil {
			if m, err := atproto.WitnessRecordToMap(wr); err == nil {
				if rec, err := arabica.RecordToRecipe(m, wr.URI); err == nil {
					metrics.WitnessCacheHitsTotal.WithLabelValues("recipe_og").Inc()
					recipe = rec
					recipe.RKey = rkey
					// Resolve brewer from witness
					if brewerRef, ok := m["brewerRef"].(string); ok && brewerRef != "" {
						if bwr, _ := h.witnessCache.GetWitnessRecord(r.Context(), brewerRef); bwr != nil {
							if bm, err := atproto.WitnessRecordToMap(bwr); err == nil {
								if brewer, err := arabica.RecordToBrewer(bm, bwr.URI); err == nil {
									recipe.BrewerObj = brewer
								}
							}
						}
					}
					recipe.Interpolate()
				}
			}
		}
	}
	if recipe == nil {
		metrics.WitnessCacheMissesTotal.WithLabelValues("recipe_og").Inc()
		publicClient := atproto.NewPublicClient()
		record, err := publicClient.GetPublicRecord(r.Context(), ownerDID, arabica.NSIDRecipe, rkey)
		if err != nil {
			http.Error(w, "Recipe not found", http.StatusNotFound)
			return
		}
		recipe, err = arabica.RecordToRecipe(record.Value, record.URI)
		if err != nil {
			http.Error(w, "Failed to load recipe", http.StatusInternalServerError)
			return
		}
		// Resolve brewer reference
		if brewerRef, ok := record.Value["brewerRef"].(string); ok && brewerRef != "" {
			brewerRKey := atp.RKeyFromURI(brewerRef)
			if brewerRKey != "" {
				if br, err := publicClient.GetPublicRecord(r.Context(), ownerDID, arabica.NSIDBrewer, brewerRKey); err == nil {
					if brewer, err := arabica.RecordToBrewer(br.Value, br.URI); err == nil {
						recipe.BrewerObj = brewer
					}
				}
			}
		}
		recipe.Interpolate()
	}

	card, err := ogcard.DrawRecipeCard(recipe)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate recipe OG image")
		http.Error(w, "Failed to generate image", http.StatusInternalServerError)
		return
	}
	writeOGImage(w, card)
}

// writeOGImage encodes a card as PNG with appropriate cache headers.
func writeOGImage(w http.ResponseWriter, card *ogcard.Card) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	if err := card.EncodePNG(w); err != nil {
		log.Error().Err(err).Msg("Failed to encode OG image")
	}
}

