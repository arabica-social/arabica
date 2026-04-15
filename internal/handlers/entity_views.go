package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/firehose"
	"tangled.org/arabica.social/arabica/internal/metrics"
	"tangled.org/arabica.social/arabica/internal/models"
	"tangled.org/arabica.social/arabica/internal/moderation"
	"tangled.org/arabica.social/arabica/internal/ogcard"
	"tangled.org/arabica.social/arabica/internal/web/bff"
	"tangled.org/arabica.social/arabica/internal/web/components"
	"tangled.org/arabica.social/arabica/internal/web/pages"

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

// HandleBeanView shows a bean detail page with social features
func (h *Handler) HandleBeanView(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	owner := r.URL.Query().Get("owner")
	didStr, err := atproto.GetAuthenticatedDID(r.Context())
	isAuthenticated := err == nil && didStr != ""

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.getUserProfile(r.Context(), didStr)
	}

	var beanViewProps pages.BeanViewProps
	var subjectURI, subjectCID, entityOwnerDID string

	if owner != "" {
		entityOwnerDID, err = resolveOwnerDID(r.Context(), owner)
		if err != nil {
			log.Warn().Err(err).Str("handle", owner).Msg("Failed to resolve handle for bean view")
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Try witness cache first
		beanURI := atproto.BuildATURI(entityOwnerDID, atproto.NSIDBean, rkey)
		if h.witnessCache != nil {
			if wr, _ := h.witnessCache.GetWitnessRecord(r.Context(), beanURI); wr != nil {
				if m, err := atproto.WitnessRecordToMap(wr); err == nil {
					if bean, err := atproto.RecordToBean(m, wr.URI); err == nil {
						metrics.WitnessCacheHitsTotal.WithLabelValues("bean").Inc()
						bean.RKey = rkey
						subjectURI = wr.URI
						subjectCID = wr.CID
						// Resolve roaster from witness
						if roasterRef, ok := m["roasterRef"].(string); ok && roasterRef != "" {
							if c, err := atproto.ResolveATURI(roasterRef); err == nil {
								bean.RoasterRKey = c.RKey
							}
							if rwr, _ := h.witnessCache.GetWitnessRecord(r.Context(), roasterRef); rwr != nil {
								if rm, err := atproto.WitnessRecordToMap(rwr); err == nil {
									if roaster, err := atproto.RecordToRoaster(rm, rwr.URI); err == nil {
										roaster.RKey = rwr.RKey
										bean.Roaster = roaster
									}
								}
							}
						}
						beanViewProps.Bean = bean
						beanViewProps.IsOwnProfile = isAuthenticated && didStr == entityOwnerDID
					}
				}
			}
		}

		if beanViewProps.Bean == nil {
			// PDS fallback
			metrics.WitnessCacheMissesTotal.WithLabelValues("bean").Inc()
			publicClient := atproto.NewPublicClient()
			record, err := publicClient.GetRecord(r.Context(), entityOwnerDID, atproto.NSIDBean, rkey)
			if err != nil {
				log.Error().Err(err).Str("did", entityOwnerDID).Str("rkey", rkey).Msg("Failed to get bean record")
				http.Error(w, "Bean not found", http.StatusNotFound)
				return
			}

			subjectURI = record.URI
			subjectCID = record.CID

			bean, err := atproto.RecordToBean(record.Value, record.URI)
			if err != nil {
				log.Error().Err(err).Msg("Failed to convert bean record")
				http.Error(w, "Failed to load bean", http.StatusInternalServerError)
				return
			}
			bean.RKey = rkey

			// Resolve roaster reference
			if roasterRef, ok := record.Value["roasterRef"].(string); ok && roasterRef != "" {
				if c, err := atproto.ResolveATURI(roasterRef); err == nil {
					bean.RoasterRKey = c.RKey
				}
				roasterRKey := atproto.ExtractRKeyFromURI(roasterRef)
				if roasterRKey != "" {
					roasterRecord, err := publicClient.GetRecord(r.Context(), entityOwnerDID, atproto.NSIDRoaster, roasterRKey)
					if err == nil {
						if roaster, err := atproto.RecordToRoaster(roasterRecord.Value, roasterRecord.URI); err == nil {
							roaster.RKey = roasterRKey
							bean.Roaster = roaster
						}
					}
				}
			}

			beanViewProps.Bean = bean
			beanViewProps.IsOwnProfile = isAuthenticated && didStr == entityOwnerDID
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

		beanRecord, err := atprotoStore.GetBeanRecordByRKey(r.Context(), rkey)
		if err != nil {
			http.Error(w, "Bean not found", http.StatusNotFound)
			log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get bean for view")
			return
		}

		beanViewProps.Bean = beanRecord.Bean
		subjectURI = beanRecord.URI
		subjectCID = beanRecord.CID
		beanViewProps.IsOwnProfile = true
	}

	// Construct share URL
	var shareURL string
	if owner != "" {
		shareURL = fmt.Sprintf("/beans/%s?owner=%s", rkey, owner)
	} else if userProfile != nil && userProfile.Handle != "" {
		shareURL = fmt.Sprintf("/beans/%s?owner=%s", rkey, userProfile.Handle)
	}

	layoutData := h.buildLayoutData(r, beanViewProps.Bean.Name, isAuthenticated, didStr, userProfile)
	h.populateBeanOGMetadata(layoutData, beanViewProps.Bean, h.resolveOwnerHandle(r.Context(), owner), h.publicBaseURL(r), shareURL)

	sd := h.fetchSocialData(r.Context(), subjectURI, didStr, isAuthenticated)

	beanViewProps.IsAuthenticated = isAuthenticated
	beanViewProps.SubjectURI = subjectURI
	beanViewProps.SubjectCID = subjectCID
	beanViewProps.IsLiked = sd.IsLiked
	beanViewProps.LikeCount = sd.LikeCount
	beanViewProps.CommentCount = sd.CommentCount
	beanViewProps.Comments = sd.Comments
	beanViewProps.CurrentUserDID = didStr
	beanViewProps.ShareURL = shareURL
	beanViewProps.IsModerator = sd.IsModerator
	beanViewProps.CanHideRecord = sd.CanHideRecord
	beanViewProps.CanBlockUser = sd.CanBlockUser
	beanViewProps.IsRecordHidden = sd.IsRecordHidden
	beanViewProps.AuthorDID = entityOwnerDID

	// Fetch author profile for display
	var authorProfile *bff.UserProfile
	authorDIDForProfile := entityOwnerDID
	if authorDIDForProfile == "" {
		authorDIDForProfile = didStr
	}
	authorProfile = h.getUserProfile(r.Context(), authorDIDForProfile)
	if authorProfile != nil {
		beanViewProps.AuthorHandle = authorProfile.Handle
		beanViewProps.AuthorDisplayName = authorProfile.DisplayName
		beanViewProps.AuthorAvatar = authorProfile.Avatar
	}

	if h.feedIndex != nil && subjectURI != "" {
		ownerDID := entityOwnerDID
		if ownerDID == "" {
			ownerDID = didStr
		}
		counts := h.feedIndex.BrewCountsByBeanURI(r.Context(), ownerDID)
		beanViewProps.BrewCount = counts[subjectURI]
	}

	if err := pages.BeanView(layoutData, beanViewProps).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render bean view")
	}
}

// HandleRoasterView shows a roaster detail page with social features
func (h *Handler) HandleRoasterView(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	owner := r.URL.Query().Get("owner")
	didStr, err := atproto.GetAuthenticatedDID(r.Context())
	isAuthenticated := err == nil && didStr != ""

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.getUserProfile(r.Context(), didStr)
	}

	var props pages.RoasterViewProps
	var subjectURI, subjectCID, entityOwnerDID string

	if owner != "" {
		entityOwnerDID, err = resolveOwnerDID(r.Context(), owner)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Try witness cache first
		roasterURI := atproto.BuildATURI(entityOwnerDID, atproto.NSIDRoaster, rkey)
		if h.witnessCache != nil {
			if wr, _ := h.witnessCache.GetWitnessRecord(r.Context(), roasterURI); wr != nil {
				if m, err := atproto.WitnessRecordToMap(wr); err == nil {
					if roaster, err := atproto.RecordToRoaster(m, wr.URI); err == nil {
						metrics.WitnessCacheHitsTotal.WithLabelValues("roaster").Inc()
						roaster.RKey = rkey
						subjectURI = wr.URI
						subjectCID = wr.CID
						props.Roaster = roaster
						props.IsOwnProfile = isAuthenticated && didStr == entityOwnerDID
					}
				}
			}
		}

		if props.Roaster == nil {
			// PDS fallback
			metrics.WitnessCacheMissesTotal.WithLabelValues("roaster").Inc()
			publicClient := atproto.NewPublicClient()
			record, err := publicClient.GetRecord(r.Context(), entityOwnerDID, atproto.NSIDRoaster, rkey)
			if err != nil {
				http.Error(w, "Roaster not found", http.StatusNotFound)
				return
			}

			subjectURI = record.URI
			subjectCID = record.CID

			roaster, err := atproto.RecordToRoaster(record.Value, record.URI)
			if err != nil {
				http.Error(w, "Failed to load roaster", http.StatusInternalServerError)
				return
			}
			roaster.RKey = rkey
			props.Roaster = roaster
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

		roasterRecord, err := atprotoStore.GetRoasterRecordByRKey(r.Context(), rkey)
		if err != nil {
			http.Error(w, "Roaster not found", http.StatusNotFound)
			return
		}

		props.Roaster = roasterRecord.Roaster
		subjectURI = roasterRecord.URI
		subjectCID = roasterRecord.CID
		props.IsOwnProfile = true
	}

	var shareURL string
	if owner != "" {
		shareURL = fmt.Sprintf("/roasters/%s?owner=%s", rkey, owner)
	} else if userProfile != nil && userProfile.Handle != "" {
		shareURL = fmt.Sprintf("/roasters/%s?owner=%s", rkey, userProfile.Handle)
	}

	layoutData := h.buildLayoutData(r, props.Roaster.Name, isAuthenticated, didStr, userProfile)
	h.populateRoasterOGMetadata(layoutData, props.Roaster, h.resolveOwnerHandle(r.Context(), owner), h.publicBaseURL(r), shareURL)

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

	if h.feedIndex != nil && subjectURI != "" {
		ownerDID := entityOwnerDID
		if ownerDID == "" {
			ownerDID = didStr
		}
		counts := h.feedIndex.BeanCountsByRoasterURI(r.Context(), ownerDID)
		props.BeanCount = counts[subjectURI]
	}

	if err := pages.RoasterView(layoutData, props).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render roaster view")
	}
}

// HandleGrinderView shows a grinder detail page with social features
func (h *Handler) HandleGrinderView(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	owner := r.URL.Query().Get("owner")
	didStr, err := atproto.GetAuthenticatedDID(r.Context())
	isAuthenticated := err == nil && didStr != ""

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.getUserProfile(r.Context(), didStr)
	}

	var props pages.GrinderViewProps
	var subjectURI, subjectCID, entityOwnerDID string

	if owner != "" {
		entityOwnerDID, err = resolveOwnerDID(r.Context(), owner)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Try witness cache first
		grinderURI := atproto.BuildATURI(entityOwnerDID, atproto.NSIDGrinder, rkey)
		if h.witnessCache != nil {
			if wr, _ := h.witnessCache.GetWitnessRecord(r.Context(), grinderURI); wr != nil {
				if m, err := atproto.WitnessRecordToMap(wr); err == nil {
					if grinder, err := atproto.RecordToGrinder(m, wr.URI); err == nil {
						metrics.WitnessCacheHitsTotal.WithLabelValues("grinder").Inc()
						grinder.RKey = rkey
						subjectURI = wr.URI
						subjectCID = wr.CID
						props.Grinder = grinder
						props.IsOwnProfile = isAuthenticated && didStr == entityOwnerDID
					}
				}
			}
		}

		if props.Grinder == nil {
			// PDS fallback
			metrics.WitnessCacheMissesTotal.WithLabelValues("grinder").Inc()
			publicClient := atproto.NewPublicClient()
			record, err := publicClient.GetRecord(r.Context(), entityOwnerDID, atproto.NSIDGrinder, rkey)
			if err != nil {
				http.Error(w, "Grinder not found", http.StatusNotFound)
				return
			}

			subjectURI = record.URI
			subjectCID = record.CID

			grinder, err := atproto.RecordToGrinder(record.Value, record.URI)
			if err != nil {
				http.Error(w, "Failed to load grinder", http.StatusInternalServerError)
				return
			}
			grinder.RKey = rkey
			props.Grinder = grinder
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

		grinderRecord, err := atprotoStore.GetGrinderRecordByRKey(r.Context(), rkey)
		if err != nil {
			http.Error(w, "Grinder not found", http.StatusNotFound)
			return
		}

		props.Grinder = grinderRecord.Grinder
		subjectURI = grinderRecord.URI
		subjectCID = grinderRecord.CID
		props.IsOwnProfile = true
	}

	var shareURL string
	if owner != "" {
		shareURL = fmt.Sprintf("/grinders/%s?owner=%s", rkey, owner)
	} else if userProfile != nil && userProfile.Handle != "" {
		shareURL = fmt.Sprintf("/grinders/%s?owner=%s", rkey, userProfile.Handle)
	}

	layoutData := h.buildLayoutData(r, props.Grinder.Name, isAuthenticated, didStr, userProfile)
	h.populateGrinderOGMetadata(layoutData, props.Grinder, h.resolveOwnerHandle(r.Context(), owner), h.publicBaseURL(r), shareURL)

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

	if h.feedIndex != nil && subjectURI != "" {
		ownerDID := entityOwnerDID
		if ownerDID == "" {
			ownerDID = didStr
		}
		counts := h.feedIndex.BrewCountsByGrinderURI(r.Context(), ownerDID)
		props.BrewCount = counts[subjectURI]
	}

	if err := pages.GrinderView(layoutData, props).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render grinder view")
	}
}

// HandleBrewerView shows a brewer detail page with social features
func (h *Handler) HandleBrewerView(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	owner := r.URL.Query().Get("owner")
	didStr, err := atproto.GetAuthenticatedDID(r.Context())
	isAuthenticated := err == nil && didStr != ""

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.getUserProfile(r.Context(), didStr)
	}

	var props pages.BrewerViewProps
	var subjectURI, subjectCID, entityOwnerDID string

	if owner != "" {
		entityOwnerDID, err = resolveOwnerDID(r.Context(), owner)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Try witness cache first
		brewerURI := atproto.BuildATURI(entityOwnerDID, atproto.NSIDBrewer, rkey)
		if h.witnessCache != nil {
			if wr, _ := h.witnessCache.GetWitnessRecord(r.Context(), brewerURI); wr != nil {
				if m, err := atproto.WitnessRecordToMap(wr); err == nil {
					if brewer, err := atproto.RecordToBrewer(m, wr.URI); err == nil {
						metrics.WitnessCacheHitsTotal.WithLabelValues("brewer").Inc()
						brewer.RKey = rkey
						subjectURI = wr.URI
						subjectCID = wr.CID
						props.Brewer = brewer
						props.IsOwnProfile = isAuthenticated && didStr == entityOwnerDID
					}
				}
			}
		}

		if props.Brewer == nil {
			// PDS fallback
			metrics.WitnessCacheMissesTotal.WithLabelValues("brewer").Inc()
			publicClient := atproto.NewPublicClient()
			record, err := publicClient.GetRecord(r.Context(), entityOwnerDID, atproto.NSIDBrewer, rkey)
			if err != nil {
				http.Error(w, "Brewer not found", http.StatusNotFound)
				return
			}

			subjectURI = record.URI
			subjectCID = record.CID

			brewer, err := atproto.RecordToBrewer(record.Value, record.URI)
			if err != nil {
				http.Error(w, "Failed to load brewer", http.StatusInternalServerError)
				return
			}
			brewer.RKey = rkey
			props.Brewer = brewer
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

		brewerRecord, err := atprotoStore.GetBrewerRecordByRKey(r.Context(), rkey)
		if err != nil {
			http.Error(w, "Brewer not found", http.StatusNotFound)
			return
		}

		props.Brewer = brewerRecord.Brewer
		subjectURI = brewerRecord.URI
		subjectCID = brewerRecord.CID
		props.IsOwnProfile = true
	}

	var shareURL string
	if owner != "" {
		shareURL = fmt.Sprintf("/brewers/%s?owner=%s", rkey, owner)
	} else if userProfile != nil && userProfile.Handle != "" {
		shareURL = fmt.Sprintf("/brewers/%s?owner=%s", rkey, userProfile.Handle)
	}

	layoutData := h.buildLayoutData(r, props.Brewer.Name, isAuthenticated, didStr, userProfile)
	h.populateBrewerOGMetadata(layoutData, props.Brewer, h.resolveOwnerHandle(r.Context(), owner), h.publicBaseURL(r), shareURL)

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

	if h.feedIndex != nil && subjectURI != "" {
		ownerDID := entityOwnerDID
		if ownerDID == "" {
			ownerDID = didStr
		}
		counts := h.feedIndex.BrewCountsByBrewerURI(r.Context(), ownerDID)
		props.BrewCount = counts[subjectURI]
	}

	if err := pages.BrewerView(layoutData, props).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render brewer view")
	}
}

// HandleRecipeView displays a recipe detail page
func (h *Handler) HandleRecipeView(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	owner := r.URL.Query().Get("owner")
	didStr, err := atproto.GetAuthenticatedDID(r.Context())
	isAuthenticated := err == nil && didStr != ""

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.getUserProfile(r.Context(), didStr)
	}

	var props pages.RecipeViewProps
	var subjectURI, subjectCID, entityOwnerDID string

	if owner != "" {
		entityOwnerDID, err = resolveOwnerDID(r.Context(), owner)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Try witness cache first
		recipeURI := atproto.BuildATURI(entityOwnerDID, atproto.NSIDRecipe, rkey)
		if h.witnessCache != nil {
			if wr, _ := h.witnessCache.GetWitnessRecord(r.Context(), recipeURI); wr != nil {
				if m, err := atproto.WitnessRecordToMap(wr); err == nil {
					if recipe, err := atproto.RecordToRecipe(m, wr.URI); err == nil {
						metrics.WitnessCacheHitsTotal.WithLabelValues("recipe").Inc()
						recipe.RKey = rkey
						subjectURI = wr.URI
						subjectCID = wr.CID
						// Resolve brewer from witness
						if brewerRef, ok := m["brewerRef"].(string); ok && brewerRef != "" {
							if c, err := atproto.ResolveATURI(brewerRef); err == nil {
								recipe.BrewerRKey = c.RKey
							}
							if bwr, _ := h.witnessCache.GetWitnessRecord(r.Context(), brewerRef); bwr != nil {
								if bm, err := atproto.WitnessRecordToMap(bwr); err == nil {
									if brewer, err := atproto.RecordToBrewer(bm, bwr.URI); err == nil {
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
			record, err := publicClient.GetRecord(r.Context(), entityOwnerDID, atproto.NSIDRecipe, rkey)
			if err != nil {
				http.Error(w, "Recipe not found", http.StatusNotFound)
				return
			}

			subjectURI = record.URI
			subjectCID = record.CID

			recipe, err := atproto.RecordToRecipe(record.Value, record.URI)
			if err != nil {
				http.Error(w, "Failed to load recipe", http.StatusInternalServerError)
				return
			}
			recipe.RKey = rkey

			// Resolve brewer reference if present
			if brewerRef, ok := record.Value["brewerRef"].(string); ok && brewerRef != "" {
				if c, err := atproto.ResolveATURI(brewerRef); err == nil {
					recipe.BrewerRKey = c.RKey
				}
				brewerRKey := atproto.ExtractRKeyFromURI(brewerRef)
				if brewerRKey != "" {
					brewerRecord, err := publicClient.GetRecord(r.Context(), entityOwnerDID, atproto.NSIDBrewer, brewerRKey)
					if err == nil {
						if brewer, err := atproto.RecordToBrewer(brewerRecord.Value, brewerRecord.URI); err == nil {
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
		if sourceComponents, err := atproto.ResolveATURI(props.Recipe.SourceRef); err == nil {
			// Build a view URL for the source recipe
			sourceOwner := sourceComponents.DID
			if profile, err := h.feedIndex.GetProfile(r.Context(), sourceComponents.DID); err == nil && profile != nil {
				sourceOwner = profile.Handle
				if profile.DisplayName != nil && *profile.DisplayName != "" {
					props.SourceRecipeAuthor = *profile.DisplayName
				} else {
					props.SourceRecipeAuthor = profile.Handle
				}
			}
			props.SourceRecipeURL = fmt.Sprintf("/recipes/%s?owner=%s", sourceComponents.RKey, sourceOwner)
		}
	}

	if err := pages.RecipeView(layoutData, props).Render(r.Context(), w); err != nil {
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

	var bean *models.Bean
	beanURI := atproto.BuildATURI(ownerDID, atproto.NSIDBean, rkey)
	if h.witnessCache != nil {
		if wr, _ := h.witnessCache.GetWitnessRecord(r.Context(), beanURI); wr != nil {
			if m, err := atproto.WitnessRecordToMap(wr); err == nil {
				if b, err := atproto.RecordToBean(m, wr.URI); err == nil {
					metrics.WitnessCacheHitsTotal.WithLabelValues("bean_og").Inc()
					bean = b
					bean.RKey = rkey
					// Resolve roaster
					if roasterRef, ok := m["roasterRef"].(string); ok && roasterRef != "" {
						if rwr, _ := h.witnessCache.GetWitnessRecord(r.Context(), roasterRef); rwr != nil {
							if rm, err := atproto.WitnessRecordToMap(rwr); err == nil {
								if roaster, err := atproto.RecordToRoaster(rm, rwr.URI); err == nil {
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
		record, err := publicClient.GetRecord(r.Context(), ownerDID, atproto.NSIDBean, rkey)
		if err != nil {
			http.Error(w, "Bean not found", http.StatusNotFound)
			return
		}
		bean, err = atproto.RecordToBean(record.Value, record.URI)
		if err != nil {
			http.Error(w, "Failed to load bean", http.StatusInternalServerError)
			return
		}
		// Resolve roaster reference
		if roasterRef, ok := record.Value["roasterRef"].(string); ok && roasterRef != "" {
			roasterRKey := atproto.ExtractRKeyFromURI(roasterRef)
			if roasterRKey != "" {
				if rr, err := publicClient.GetRecord(r.Context(), ownerDID, atproto.NSIDRoaster, roasterRKey); err == nil {
					if roaster, err := atproto.RecordToRoaster(rr.Value, rr.URI); err == nil {
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

// HandleRoasterOGImage generates a 1200x630 PNG preview card for a roaster.
func (h *Handler) HandleRoasterOGImage(w http.ResponseWriter, r *http.Request) {
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

	var roaster *models.Roaster
	roasterURI := atproto.BuildATURI(ownerDID, atproto.NSIDRoaster, rkey)
	if h.witnessCache != nil {
		if wr, _ := h.witnessCache.GetWitnessRecord(r.Context(), roasterURI); wr != nil {
			if m, err := atproto.WitnessRecordToMap(wr); err == nil {
				if r, err := atproto.RecordToRoaster(m, wr.URI); err == nil {
					metrics.WitnessCacheHitsTotal.WithLabelValues("roaster_og").Inc()
					roaster = r
					roaster.RKey = rkey
				}
			}
		}
	}
	if roaster == nil {
		metrics.WitnessCacheMissesTotal.WithLabelValues("roaster_og").Inc()
		publicClient := atproto.NewPublicClient()
		record, err := publicClient.GetRecord(r.Context(), ownerDID, atproto.NSIDRoaster, rkey)
		if err != nil {
			http.Error(w, "Roaster not found", http.StatusNotFound)
			return
		}
		roaster, err = atproto.RecordToRoaster(record.Value, record.URI)
		if err != nil {
			http.Error(w, "Failed to load roaster", http.StatusInternalServerError)
			return
		}
	}

	card, err := ogcard.DrawRoasterCard(roaster)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate roaster OG image")
		http.Error(w, "Failed to generate image", http.StatusInternalServerError)
		return
	}
	writeOGImage(w, card)
}

// HandleGrinderOGImage generates a 1200x630 PNG preview card for a grinder.
func (h *Handler) HandleGrinderOGImage(w http.ResponseWriter, r *http.Request) {
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

	var grinder *models.Grinder
	grinderURI := atproto.BuildATURI(ownerDID, atproto.NSIDGrinder, rkey)
	if h.witnessCache != nil {
		if wr, _ := h.witnessCache.GetWitnessRecord(r.Context(), grinderURI); wr != nil {
			if m, err := atproto.WitnessRecordToMap(wr); err == nil {
				if g, err := atproto.RecordToGrinder(m, wr.URI); err == nil {
					metrics.WitnessCacheHitsTotal.WithLabelValues("grinder_og").Inc()
					grinder = g
					grinder.RKey = rkey
				}
			}
		}
	}
	if grinder == nil {
		metrics.WitnessCacheMissesTotal.WithLabelValues("grinder_og").Inc()
		publicClient := atproto.NewPublicClient()
		record, err := publicClient.GetRecord(r.Context(), ownerDID, atproto.NSIDGrinder, rkey)
		if err != nil {
			http.Error(w, "Grinder not found", http.StatusNotFound)
			return
		}
		grinder, err = atproto.RecordToGrinder(record.Value, record.URI)
		if err != nil {
			http.Error(w, "Failed to load grinder", http.StatusInternalServerError)
			return
		}
	}

	card, err := ogcard.DrawGrinderCard(grinder)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate grinder OG image")
		http.Error(w, "Failed to generate image", http.StatusInternalServerError)
		return
	}
	writeOGImage(w, card)
}

// HandleBrewerOGImage generates a 1200x630 PNG preview card for a brewer.
func (h *Handler) HandleBrewerOGImage(w http.ResponseWriter, r *http.Request) {
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

	var brewer *models.Brewer
	brewerURI := atproto.BuildATURI(ownerDID, atproto.NSIDBrewer, rkey)
	if h.witnessCache != nil {
		if wr, _ := h.witnessCache.GetWitnessRecord(r.Context(), brewerURI); wr != nil {
			if m, err := atproto.WitnessRecordToMap(wr); err == nil {
				if b, err := atproto.RecordToBrewer(m, wr.URI); err == nil {
					metrics.WitnessCacheHitsTotal.WithLabelValues("brewer_og").Inc()
					brewer = b
					brewer.RKey = rkey
				}
			}
		}
	}
	if brewer == nil {
		metrics.WitnessCacheMissesTotal.WithLabelValues("brewer_og").Inc()
		publicClient := atproto.NewPublicClient()
		record, err := publicClient.GetRecord(r.Context(), ownerDID, atproto.NSIDBrewer, rkey)
		if err != nil {
			http.Error(w, "Brewer not found", http.StatusNotFound)
			return
		}
		brewer, err = atproto.RecordToBrewer(record.Value, record.URI)
		if err != nil {
			http.Error(w, "Failed to load brewer", http.StatusInternalServerError)
			return
		}
	}

	card, err := ogcard.DrawBrewerCard(brewer)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate brewer OG image")
		http.Error(w, "Failed to generate image", http.StatusInternalServerError)
		return
	}
	writeOGImage(w, card)
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

	var recipe *models.Recipe
	recipeURI := atproto.BuildATURI(ownerDID, atproto.NSIDRecipe, rkey)
	if h.witnessCache != nil {
		if wr, _ := h.witnessCache.GetWitnessRecord(r.Context(), recipeURI); wr != nil {
			if m, err := atproto.WitnessRecordToMap(wr); err == nil {
				if rec, err := atproto.RecordToRecipe(m, wr.URI); err == nil {
					metrics.WitnessCacheHitsTotal.WithLabelValues("recipe_og").Inc()
					recipe = rec
					recipe.RKey = rkey
					// Resolve brewer from witness
					if brewerRef, ok := m["brewerRef"].(string); ok && brewerRef != "" {
						if bwr, _ := h.witnessCache.GetWitnessRecord(r.Context(), brewerRef); bwr != nil {
							if bm, err := atproto.WitnessRecordToMap(bwr); err == nil {
								if brewer, err := atproto.RecordToBrewer(bm, bwr.URI); err == nil {
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
		record, err := publicClient.GetRecord(r.Context(), ownerDID, atproto.NSIDRecipe, rkey)
		if err != nil {
			http.Error(w, "Recipe not found", http.StatusNotFound)
			return
		}
		recipe, err = atproto.RecordToRecipe(record.Value, record.URI)
		if err != nil {
			http.Error(w, "Failed to load recipe", http.StatusInternalServerError)
			return
		}
		// Resolve brewer reference
		if brewerRef, ok := record.Value["brewerRef"].(string); ok && brewerRef != "" {
			brewerRKey := atproto.ExtractRKeyFromURI(brewerRef)
			if brewerRKey != "" {
				if br, err := publicClient.GetRecord(r.Context(), ownerDID, atproto.NSIDBrewer, brewerRKey); err == nil {
					if brewer, err := atproto.RecordToBrewer(br.Value, br.URI); err == nil {
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

// OG metadata helpers for entity types

func (h *Handler) populateBeanOGMetadata(layoutData *components.LayoutData, bean *models.Bean, owner, baseURL, shareURL string) {
	if bean == nil {
		return
	}
	subtitle := bean.Name
	if subtitle == "" {
		subtitle = bean.Origin
	}
	if bean.Roaster != nil && bean.Roaster.Name != "" {
		subtitle += " from " + bean.Roaster.Name
	}
	populateOGFields(layoutData, subtitle, "bean", owner, baseURL, shareURL)
}

func (h *Handler) populateRoasterOGMetadata(layoutData *components.LayoutData, roaster *models.Roaster, owner, baseURL, shareURL string) {
	if roaster == nil {
		return
	}
	populateOGFields(layoutData, roaster.Name, "roaster", owner, baseURL, shareURL)
}

func (h *Handler) populateGrinderOGMetadata(layoutData *components.LayoutData, grinder *models.Grinder, owner, baseURL, shareURL string) {
	if grinder == nil {
		return
	}
	populateOGFields(layoutData, grinder.Name, "grinder", owner, baseURL, shareURL)
}

func (h *Handler) populateBrewerOGMetadata(layoutData *components.LayoutData, brewer *models.Brewer, owner, baseURL, shareURL string) {
	if brewer == nil {
		return
	}
	populateOGFields(layoutData, brewer.Name, "brewer", owner, baseURL, shareURL)
}

func (h *Handler) populateRecipeOGMetadata(layoutData *components.LayoutData, recipe *models.Recipe, owner, baseURL, shareURL string) {
	if recipe == nil {
		return
	}
	populateOGFields(layoutData, recipe.Name, "recipe", owner, baseURL, shareURL)
}
