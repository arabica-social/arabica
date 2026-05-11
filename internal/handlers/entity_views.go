package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

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
	"tangled.org/arabica.social/arabica/internal/arabica/web/pages"
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
	fromStore   func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, string, string, error)
	displayName func(record any) string
	ogSubtitle  func(record any) string
	countLookup func(ctx context.Context, ownerDID, subjectURI string) int
	render      func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base coffeepages.EntityViewBase) error
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

	if owner != "" {
		var err error
		entityOwnerDID, err = resolveOwnerDID(r.Context(), owner)
		if err != nil {
			log.Warn().Err(err).Str("handle", owner).Msgf("Failed to resolve handle for %s view", cfg.descriptor.Noun)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		entityURI := atp.BuildATURI(entityOwnerDID, cfg.descriptor.NSID, rkey)
		if h.witnessCache != nil {
			if wr, _ := h.witnessCache.GetWitnessRecord(r.Context(), entityURI); wr != nil {
				if m, err := atproto.WitnessRecordToMap(wr); err == nil {
					if rec, err := cfg.fromWitness(r.Context(), m, wr.URI, rkey, entityOwnerDID); err == nil {
						metrics.WitnessCacheHitsTotal.WithLabelValues(cfg.descriptor.Noun).Inc()
						record = rec
						subjectURI = wr.URI
						subjectCID = wr.CID
						isOwnProfile = isAuthenticated && didStr == entityOwnerDID
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
			isOwnProfile = isAuthenticated && didStr == entityOwnerDID
		}
	} else {
		store, authenticated := h.getAtprotoStore(r)
		if !authenticated {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		atprotoStore, ok := store.(*atproto.AtprotoStore)
		if !ok {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		rec, uri, cid, err := cfg.fromStore(r.Context(), atprotoStore, rkey)
		if err != nil {
			http.Error(w, cfg.descriptor.DisplayName+" not found", http.StatusNotFound)
			log.Error().Err(err).Str("rkey", rkey).Msgf("Failed to get %s for view", cfg.descriptor.Noun)
			return
		}
		record, subjectURI, subjectCID = rec, uri, cid
		isOwnProfile = true
	}

	var shareURL string
	if owner != "" {
		shareURL = fmt.Sprintf("/%s/%s?owner=%s", cfg.descriptor.URLPath, rkey, owner)
	} else if userProfile != nil && userProfile.Handle != "" {
		shareURL = fmt.Sprintf("/%s/%s?owner=%s", cfg.descriptor.URLPath, rkey, userProfile.Handle)
	}

	ownerHandle := h.resolveOwnerHandle(r.Context(), owner)
	layoutData := h.buildLayoutData(r, cfg.displayName(record), isAuthenticated, didStr, userProfile)
	populateOGFields(layoutData, cfg.ogSubtitle(record), cfg.descriptor.Noun, ownerHandle, h.publicBaseURL(r), shareURL)

	sd := h.fetchSocialData(r.Context(), subjectURI, didStr, isAuthenticated)

	authorDID := entityOwnerDID
	if authorDID == "" {
		authorDID = didStr
	}
	base := coffeepages.EntityViewBase{
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
	return entityViewConfig{
		descriptor: entities.Get(lexicons.RecordTypeRoaster),
		fromWitness: func(_ context.Context, m map[string]any, uri, rkey, _ string) (any, error) {
			r, err := arabica.RecordToRoaster(m, uri)
			if err != nil {
				return nil, err
			}
			r.RKey = rkey
			return r, nil
		},
		fromPDS: func(_ context.Context, e *atp.Record, rkey, _ string) (any, error) {
			r, err := arabica.RecordToRoaster(e.Value, e.URI)
			if err != nil {
				return nil, err
			}
			r.RKey = rkey
			return r, nil
		},
		fromStore: func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, string, string, error) {
			rec, err := s.GetRoasterRecordByRKey(ctx, rkey)
			if err != nil {
				return nil, "", "", err
			}
			return rec.Roaster, rec.URI, rec.CID, nil
		},
		displayName: func(record any) string { return record.(*arabica.Roaster).Name },
		ogSubtitle:  func(record any) string { return record.(*arabica.Roaster).Name },
		countLookup: func(ctx context.Context, ownerDID, subjectURI string) int {
			if h.feedIndex == nil || subjectURI == "" {
				return 0
			}
			return h.feedIndex.BeanCountsByRoasterURI(ctx, ownerDID)[subjectURI]
		},
		render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base coffeepages.EntityViewBase) error {
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
	return entityViewConfig{
		descriptor: entities.Get(lexicons.RecordTypeGrinder),
		fromWitness: func(_ context.Context, m map[string]any, uri, rkey, _ string) (any, error) {
			g, err := arabica.RecordToGrinder(m, uri)
			if err != nil {
				return nil, err
			}
			g.RKey = rkey
			return g, nil
		},
		fromPDS: func(_ context.Context, e *atp.Record, rkey, _ string) (any, error) {
			g, err := arabica.RecordToGrinder(e.Value, e.URI)
			if err != nil {
				return nil, err
			}
			g.RKey = rkey
			return g, nil
		},
		fromStore: func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, string, string, error) {
			rec, err := s.GetGrinderRecordByRKey(ctx, rkey)
			if err != nil {
				return nil, "", "", err
			}
			return rec.Grinder, rec.URI, rec.CID, nil
		},
		displayName: func(record any) string { return record.(*arabica.Grinder).Name },
		ogSubtitle:  func(record any) string { return record.(*arabica.Grinder).Name },
		render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base coffeepages.EntityViewBase) error {
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
	return entityViewConfig{
		descriptor: entities.Get(lexicons.RecordTypeBrewer),
		fromWitness: func(_ context.Context, m map[string]any, uri, rkey, _ string) (any, error) {
			b, err := arabica.RecordToBrewer(m, uri)
			if err != nil {
				return nil, err
			}
			b.RKey = rkey
			return b, nil
		},
		fromPDS: func(_ context.Context, e *atp.Record, rkey, _ string) (any, error) {
			b, err := arabica.RecordToBrewer(e.Value, e.URI)
			if err != nil {
				return nil, err
			}
			b.RKey = rkey
			return b, nil
		},
		fromStore: func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, string, string, error) {
			rec, err := s.GetBrewerRecordByRKey(ctx, rkey)
			if err != nil {
				return nil, "", "", err
			}
			return rec.Brewer, rec.URI, rec.CID, nil
		},
		displayName: func(record any) string { return record.(*arabica.Brewer).Name },
		ogSubtitle:  func(record any) string { return record.(*arabica.Brewer).Name },
		render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base coffeepages.EntityViewBase) error {
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
	return entityViewConfig{
		descriptor: entities.Get(lexicons.RecordTypeBean),
		fromWitness: func(ctx context.Context, m map[string]any, uri, rkey, _ string) (any, error) {
			bean, err := arabica.RecordToBean(m, uri)
			if err != nil {
				return nil, err
			}
			bean.RKey = rkey
			if roasterRef, ok := m["roasterRef"].(string); ok && roasterRef != "" {
				if rkey := atp.RKeyFromURI(roasterRef); rkey != "" {
					bean.RoasterRKey = rkey
				}
				if h.witnessCache != nil {
					if rwr, _ := h.witnessCache.GetWitnessRecord(ctx, roasterRef); rwr != nil {
						if rm, err := atproto.WitnessRecordToMap(rwr); err == nil {
							if roaster, err := arabica.RecordToRoaster(rm, rwr.URI); err == nil {
								roaster.RKey = rwr.RKey
								bean.Roaster = roaster
							}
						}
					}
				}
			}
			return bean, nil
		},
		fromPDS: func(ctx context.Context, e *atp.Record, rkey, ownerDID string) (any, error) {
			bean, err := arabica.RecordToBean(e.Value, e.URI)
			if err != nil {
				return nil, err
			}
			bean.RKey = rkey
			if roasterRef, ok := e.Value["roasterRef"].(string); ok && roasterRef != "" {
				if roasterRKey := atp.RKeyFromURI(roasterRef); roasterRKey != "" {
					bean.RoasterRKey = roasterRKey
					pub := atproto.NewPublicClient()
					if rr, err := pub.GetPublicRecord(ctx, ownerDID, arabica.NSIDRoaster, roasterRKey); err == nil {
						if roaster, err := arabica.RecordToRoaster(rr.Value, rr.URI); err == nil {
							roaster.RKey = roasterRKey
							bean.Roaster = roaster
						}
					}
				}
			}
			return bean, nil
		},
		fromStore: func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, string, string, error) {
			rec, err := s.GetBeanRecordByRKey(ctx, rkey)
			if err != nil {
				return nil, "", "", err
			}
			return rec.Bean, rec.URI, rec.CID, nil
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
		render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base coffeepages.EntityViewBase) error {
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

// HandleRecipeView displays a recipe detail page
func (h *Handler) HandleRecipeView(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	owner := r.URL.Query().Get("owner")
	didStr, isAuthenticated := atpmiddleware.GetDID(r.Context())

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.getUserProfile(r.Context(), didStr)
	}

	var props coffeepages.RecipeViewProps
	var subjectURI, subjectCID, entityOwnerDID string

	if owner != "" {
		entityOwnerDID, err := resolveOwnerDID(r.Context(), owner)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Try witness cache first
		recipeURI := atp.BuildATURI(entityOwnerDID, arabica.NSIDRecipe, rkey)
		if h.witnessCache != nil {
			if wr, _ := h.witnessCache.GetWitnessRecord(r.Context(), recipeURI); wr != nil {
				if m, err := atproto.WitnessRecordToMap(wr); err == nil {
					if recipe, err := arabica.RecordToRecipe(m, wr.URI); err == nil {
						metrics.WitnessCacheHitsTotal.WithLabelValues("recipe").Inc()
						recipe.RKey = rkey
						subjectURI = wr.URI
						subjectCID = wr.CID
						// Resolve brewer from witness
						if brewerRef, ok := m["brewerRef"].(string); ok && brewerRef != "" {
							if rkey := atp.RKeyFromURI(brewerRef); rkey != "" {
								recipe.BrewerRKey = rkey
							}
							if bwr, _ := h.witnessCache.GetWitnessRecord(r.Context(), brewerRef); bwr != nil {
								if bm, err := atproto.WitnessRecordToMap(bwr); err == nil {
									if brewer, err := arabica.RecordToBrewer(bm, bwr.URI); err == nil {
										brewer.RKey = bwr.RKey
										recipe.BrewerObj = brewer
									}
								}
							}
						}
						recipe.Interpolate()
						props.Recipe = recipe
						props.IsOwnProfile = isAuthenticated && didStr == entityOwnerDID
					}
				}
			}
		}

		if props.Recipe == nil {
			// PDS fallback
			metrics.WitnessCacheMissesTotal.WithLabelValues("recipe").Inc()
			publicClient := atproto.NewPublicClient()
			record, err := publicClient.GetPublicRecord(r.Context(), entityOwnerDID, arabica.NSIDRecipe, rkey)
			if err != nil {
				http.Error(w, "Recipe not found", http.StatusNotFound)
				return
			}

			subjectURI = record.URI
			subjectCID = record.CID

			recipe, err := arabica.RecordToRecipe(record.Value, record.URI)
			if err != nil {
				http.Error(w, "Failed to load recipe", http.StatusInternalServerError)
				return
			}
			recipe.RKey = rkey

			// Resolve brewer reference if present
			if brewerRef, ok := record.Value["brewerRef"].(string); ok && brewerRef != "" {
				if brewerRKey := atp.RKeyFromURI(brewerRef); brewerRKey != "" {
					recipe.BrewerRKey = brewerRKey
					brewerRecord, err := publicClient.GetPublicRecord(r.Context(), entityOwnerDID, arabica.NSIDBrewer, brewerRKey)
					if err == nil {
						if brewer, err := arabica.RecordToBrewer(brewerRecord.Value, brewerRecord.URI); err == nil {
							brewer.RKey = brewerRKey
							recipe.BrewerObj = brewer
						}
					}
				}
			}

			recipe.Interpolate()
			props.Recipe = recipe
			props.IsOwnProfile = isAuthenticated && didStr == entityOwnerDID
		}
	} else {
		store, authenticated := h.getAtprotoStore(r)
		if !authenticated {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		atprotoStore, ok := store.(*atproto.AtprotoStore)
		if !ok {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		recipeRecord, err := atprotoStore.GetRecipeRecordByRKey(r.Context(), rkey)
		if err != nil {
			http.Error(w, "Recipe not found", http.StatusNotFound)
			return
		}

		recipeRecord.Recipe.Interpolate()
		props.Recipe = recipeRecord.Recipe
		subjectURI = recipeRecord.URI
		subjectCID = recipeRecord.CID
		props.IsOwnProfile = true
	}

	var shareURL string
	if owner != "" {
		shareURL = fmt.Sprintf("/recipes/%s?owner=%s", rkey, owner)
	} else if userProfile != nil && userProfile.Handle != "" {
		shareURL = fmt.Sprintf("/recipes/%s?owner=%s", rkey, userProfile.Handle)
	}

	layoutData := h.buildLayoutData(r, props.Recipe.Name, isAuthenticated, didStr, userProfile)
	h.populateRecipeOGMetadata(layoutData, props.Recipe, h.resolveOwnerHandle(r.Context(), owner), h.publicBaseURL(r), shareURL)

	sd := h.fetchSocialData(r.Context(), subjectURI, didStr, isAuthenticated)

	props.IsAuthenticated = isAuthenticated
	props.SubjectURI = subjectURI
	props.SubjectCID = subjectCID
	props.IsLiked = sd.IsLiked
	props.LikeCount = sd.LikeCount
	props.CommentCount = sd.CommentCount
	props.Comments = sd.Comments
	props.CurrentUserDID = didStr
	props.ShareURL = shareURL
	props.IsModerator = sd.IsModerator
	props.CanHideRecord = sd.CanHideRecord
	props.CanBlockUser = sd.CanBlockUser
	props.IsRecordHidden = sd.IsRecordHidden
	props.AuthorDID = entityOwnerDID

	// Fetch author profile for display
	{
		var authorProfile *bff.UserProfile
		authorDIDForProfile := entityOwnerDID
		if authorDIDForProfile == "" {
			authorDIDForProfile = didStr
		}
		authorProfile = h.getUserProfile(r.Context(), authorDIDForProfile)
		if authorProfile != nil {
			props.AuthorHandle = authorProfile.Handle
			props.AuthorDisplayName = authorProfile.DisplayName
			props.AuthorAvatar = authorProfile.Avatar
		}
	}

	// Resolve source recipe provenance if this is a fork
	if props.Recipe.SourceRef != "" {
		if srcURI, err := atp.ParseATURI(props.Recipe.SourceRef); err == nil {
			// Build a view URL for the source recipe
			sourceOwner := srcURI.DID
			if profile, err := h.feedIndex.GetProfile(r.Context(), srcURI.DID); err == nil && profile != nil {
				sourceOwner = profile.Handle
				if profile.DisplayName != nil && *profile.DisplayName != "" {
					props.SourceRecipeAuthor = *profile.DisplayName
				} else {
					props.SourceRecipeAuthor = profile.Handle
				}
			}
			props.SourceRecipeURL = fmt.Sprintf("/recipes/%s?owner=%s", srcURI.RKey, sourceOwner)
		}
	}

	if err := coffeepages.RecipeView(layoutData, props).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render recipe view")
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

func (h *Handler) populateRecipeOGMetadata(layoutData *components.LayoutData, recipe *arabica.Recipe, owner, baseURL, shareURL string) {
	if recipe == nil {
		return
	}
	populateOGFields(layoutData, recipe.Name, "recipe", owner, baseURL, shareURL)
}
