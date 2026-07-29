package handlers

import (
	"net/http"
	"time"

	"tangled.org/arabica.social/arabica/internal/firehose"
	"tangled.org/arabica.social/arabica/internal/metrics"
	"tangled.org/arabica.social/arabica/internal/moderation"
	"tangled.org/arabica.social/arabica/internal/social"
	"tangled.org/pdewey.com/atp"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/rs/zerolog/log"
)

// LikeToggleResponseJSON is the JSON response for POST /api/likes/toggle.
type LikeToggleResponseJSON struct {
	IsLiked    bool   `json:"is_liked"`
	LikeCount  int    `json:"like_count"`
	SubjectURI string `json:"subject_uri"`
}

// CommentListResponseJSON is the JSON response for GET /api/comments.
type CommentListResponseJSON struct {
	Comments        []CommentJSON `json:"comments"`
	SubjectURI      string        `json:"subject_uri"`
	IsAuthenticated bool          `json:"is_authenticated"`
}

// CommentCreateResponseJSON is the JSON response for POST /api/comments.
type CommentCreateResponseJSON struct {
	Comment  CommentJSON   `json:"comment"`
	Comments []CommentJSON `json:"comments"`
}

// CommentDeleteResponseJSON is the JSON response for DELETE /api/comments/{id}.
type CommentDeleteResponseJSON struct {
	Deleted bool `json:"deleted"`
}

// ReportResponseJSON is the JSON response for POST /api/report.
type ReportResponseJSON struct {
	ReportID  string `json:"report_id"`
	Submitted bool   `json:"submitted"`
}

// HandleLikeToggleJSON handles like toggling and returns JSON for the SvelteKit SPA.
func (h *Handler) HandleLikeToggleJSON(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getSocialStore(r)
	if !authenticated {
		WriteJSONError(w, http.StatusUnauthorized, "authentication_required", "Authentication required")
		return
	}

	didStr, _ := atpmiddleware.GetDID(r.Context())

	if err := r.ParseForm(); err != nil {
		log.Warn().Err(err).Msg("Failed to parse like toggle form")
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid form data")
		return
	}

	subjectURI := r.FormValue("subject_uri")
	subjectCID := r.FormValue("subject_cid")

	if subjectURI == "" || subjectCID == "" {
		WriteJSONError(w, http.StatusBadRequest, "validation_failed", "subject_uri and subject_cid are required")
		return
	}

	existingLike, err := store.GetUserLikeForSubject(r.Context(), subjectURI)
	if err != nil {
		log.Error().Err(err).Msg("Failed to check existing like")
		HandleStoreJSONError(w, err, "Failed to check like status")
		return
	}

	var isLiked bool
	var likeCount int

	if existingLike != nil {
		if err := store.DeleteLikeByRKey(r.Context(), existingLike.RKey); err != nil {
			log.Error().Err(err).Msg("Failed to delete like")
			HandleStoreJSONError(w, err, "Failed to unlike")
			return
		}
		isLiked = false
		metrics.LikesTotal.WithLabelValues("delete").Inc()
		if h.feedIndex != nil {
			if err := h.feedIndex.DeleteLike(r.Context(), didStr, subjectURI); err != nil {
				log.Warn().Err(err).Str("did", didStr).Str("subject_uri", subjectURI).Msg("Failed to delete like from feed index")
			}
			h.feedIndex.DeleteLikeNotification(didStr, subjectURI)
			likeCount = h.feedIndex.GetLikeCount(r.Context(), subjectURI)
		}
	} else {
		req := &social.CreateLikeRequest{
			SubjectURI: subjectURI,
			SubjectCID: subjectCID,
		}
		like, err := store.CreateLike(r.Context(), req)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create like")
			HandleStoreJSONError(w, err, "Failed to like")
			return
		}
		isLiked = true
		metrics.LikesTotal.WithLabelValues("create").Inc()
		if h.feedIndex != nil {
			if err := h.feedIndex.UpsertLike(r.Context(), didStr, like.RKey, subjectURI); err != nil {
				log.Warn().Err(err).Str("did", didStr).Str("subject_uri", subjectURI).Msg("Failed to upsert like in feed index")
			}
			likeCount = h.feedIndex.GetLikeCount(r.Context(), subjectURI)
		}
	}

	WriteJSON(w, LikeToggleResponseJSON{
		IsLiked:    isLiked,
		LikeCount:  likeCount,
		SubjectURI: subjectURI,
	}, "like-toggle")
}

// HandleCommentListJSON returns comments as JSON for the SvelteKit SPA.
func (h *Handler) HandleCommentListJSON(w http.ResponseWriter, r *http.Request) {
	subjectURI := r.URL.Query().Get("subject_uri")
	if subjectURI == "" {
		WriteJSONError(w, http.StatusBadRequest, "validation_failed", "subject_uri is required")
		return
	}

	didStr, isAuthenticated := atpmiddleware.GetDID(r.Context())

	var comments []firehose.IndexedComment
	if h.feedIndex != nil {
		comments = h.feedIndex.GetThreadedCommentsForSubject(r.Context(), subjectURI, 100, didStr)
		comments = h.FilterHiddenComments(r.Context(), comments)
	}

	WriteJSON(w, CommentListResponseJSON{
		Comments:        NewCommentsJSON(comments),
		SubjectURI:      subjectURI,
		IsAuthenticated: isAuthenticated,
	}, "comments")
}

// HandleCommentCreateJSON creates a comment and returns JSON for the SvelteKit SPA.
func (h *Handler) HandleCommentCreateJSON(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getSocialStore(r)
	if !authenticated {
		WriteJSONError(w, http.StatusUnauthorized, "authentication_required", "Authentication required")
		return
	}

	didStr, _ := atpmiddleware.GetDID(r.Context())

	if err := r.ParseForm(); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid form data")
		return
	}

	subjectURI := r.FormValue("subject_uri")
	subjectCID := r.FormValue("subject_cid")
	text := r.FormValue("text")
	parentURI := r.FormValue("parent_uri")
	parentCID := r.FormValue("parent_cid")

	if subjectURI == "" || subjectCID == "" {
		WriteJSONError(w, http.StatusBadRequest, "validation_failed", "subject_uri and subject_cid are required")
		return
	}
	if text == "" {
		WriteJSONError(w, http.StatusBadRequest, "validation_failed", "comment text is required")
		return
	}
	if len(text) > social.MaxCommentLength {
		WriteJSONError(w, http.StatusBadRequest, "validation_failed", "comment text is too long")
		return
	}

	// Validate that parent fields are either both present or both absent.
	if (parentURI != "" && parentCID == "") || (parentURI == "" && parentCID != "") {
		WriteJSONError(w, http.StatusBadRequest, "validation_failed", "both parent_uri and parent_cid must be provided together")
		return
	}

	req := &social.CreateCommentRequest{
		SubjectURI: subjectURI,
		SubjectCID: subjectCID,
		Text:       text,
		ParentURI:  parentURI,
		ParentCID:  parentCID,
	}

	comment, err := store.CreateComment(r.Context(), req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create comment")
		HandleStoreJSONError(w, err, "Failed to create comment")
		return
	}

	metrics.CommentsTotal.WithLabelValues("create").Inc()

	if h.feedIndex != nil {
		if err := h.feedIndex.UpsertComment(r.Context(), didStr, comment.RKey, subjectURI, parentURI, comment.CID, text, comment.CreatedAt); err != nil {
			log.Warn().Err(err).Str("did", didStr).Str("rkey", comment.RKey).Str("subject_uri", subjectURI).Msg("Failed to upsert comment in feed index")
		}
		h.feedIndex.CreateCommentNotification(didStr, subjectURI, parentURI)
	}

	var comments []firehose.IndexedComment
	if h.feedIndex != nil {
		comments = h.feedIndex.GetThreadedCommentsForSubject(r.Context(), subjectURI, 100, didStr)
		comments = h.FilterHiddenComments(r.Context(), comments)
	}

	WriteJSON(w, CommentCreateResponseJSON{
		Comment:  NewCommentJSON(toIndexedComment(comment, didStr)),
		Comments: NewCommentsJSON(comments),
	}, "comment-create")
}

// HandleCommentDeleteJSON deletes a comment.
func (h *Handler) HandleCommentDeleteJSON(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getSocialStore(r)
	if !authenticated {
		WriteJSONError(w, http.StatusUnauthorized, "authentication_required", "Authentication required")
		return
	}

	didStr, _ := atpmiddleware.GetDID(r.Context())

	rkey := r.PathValue("id")
	if rkey == "" {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Comment ID is required")
		return
	}

	if err := store.DeleteCommentByRKey(r.Context(), rkey); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Str("did", didStr).Msg("Failed to delete comment from PDS")
		HandleStoreJSONError(w, err, "Failed to delete comment")
		return
	}

	metrics.CommentsTotal.WithLabelValues("delete").Inc()

	if h.feedIndex != nil {
		subjectURI := h.feedIndex.GetCommentSubjectURI(didStr, rkey)
		if err := h.feedIndex.DeleteComment(r.Context(), didStr, rkey, ""); err != nil {
			log.Warn().Err(err).Str("did", didStr).Str("rkey", rkey).Msg("Failed to delete comment from feed index")
		}
		if subjectURI != "" {
			h.feedIndex.DeleteCommentNotification(didStr, subjectURI, "")
		}
	}

	WriteJSON(w, CommentDeleteResponseJSON{Deleted: true}, "comment-delete")
}

// HandleReportJSON handles content report submissions and returns JSON for the
// SvelteKit SPA. Reuses the same validation and automod logic as the HTML handler.
func (h *Handler) HandleReportJSON(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid form data")
		return
	}
	subjectURI := r.FormValue("subject_uri")
	rawReason := r.FormValue("reason")

	reporterDID, ok := atpmiddleware.GetDID(ctx)
	if !ok {
		WriteJSONError(w, http.StatusUnauthorized, "authentication_required", "Authentication required")
		return
	}

	if h.moderationStore == nil {
		log.Error().Msg("moderation: store not configured")
		WriteJSONError(w, http.StatusServiceUnavailable, "service_unavailable", "Reports are not enabled")
		return
	}

	if subjectURI == "" {
		WriteJSONError(w, http.StatusBadRequest, "validation_failed", "subject_uri is required")
		return
	}

	uriParts, err := atp.ParseATURI(subjectURI)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid subject_uri format")
		return
	}
	subjectDID := uriParts.DID

	if subjectDID == reporterDID {
		WriteJSONError(w, http.StatusBadRequest, "validation_failed", "You cannot report your own content")
		return
	}

	reason := rawReason
	if reason == "" {
		reason = "No reason provided"
	}
	if len(reason) > MaxReportReasonLength {
		reason = reason[:MaxReportReasonLength]
	}

	oneHourAgo := time.Now().Add(-1 * time.Hour)
	recentCount, err := h.moderationStore.CountReportsFromUserSince(ctx, reporterDID, oneHourAgo)
	if err != nil {
		log.Error().Err(err).Str("reporter", reporterDID).Msg("moderation: failed to check rate limit")
		WriteJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to process report")
		return
	}
	if recentCount >= ReportRateLimitPerHour {
		WriteJSONError(w, http.StatusTooManyRequests, "rate_limited", "Rate limit exceeded. Please try again later.")
		return
	}

	alreadyReported, err := h.moderationStore.HasReportedURI(ctx, reporterDID, subjectURI)
	if err != nil {
		log.Error().Err(err).Str("reporter", reporterDID).Msg("moderation: failed to check duplicate")
		WriteJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to process report")
		return
	}
	if alreadyReported {
		WriteJSONError(w, http.StatusConflict, "conflict", "You have already reported this content")
		return
	}

	report := moderation.Report{
		ID:          generateTID(),
		SubjectURI:  subjectURI,
		SubjectDID:  subjectDID,
		ReporterDID: reporterDID,
		Reason:      reason,
		CreatedAt:   time.Now(),
		Status:      moderation.ReportStatusPending,
	}

	if err := h.moderationStore.CreateReport(ctx, report); err != nil {
		log.Error().Err(err).Str("reporter", reporterDID).Msg("moderation: failed to create report")
		WriteJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to save report")
		return
	}

	metrics.ReportsTotal.Inc()

	log.Info().
		Str("report_id", report.ID).
		Str("subject_uri", report.SubjectURI).
		Str("subject_did", report.SubjectDID).
		Str("reporter_did", report.ReporterDID).
		Str("reason", report.Reason).
		Msg("moderation: report created")

	h.checkAutomod(ctx, report)

	WriteJSON(w, ReportResponseJSON{
		ReportID:  report.ID,
		Submitted: true,
	}, "report")
}

// toIndexedComment converts a social.Comment to an IndexedComment for JSON
// serialization. The comment was just created so it has no replies, likes, or
// depth yet.
func toIndexedComment(c *social.Comment, actorDID string) firehose.IndexedComment {
	return firehose.IndexedComment{
		RKey:       c.RKey,
		SubjectURI: c.SubjectURI,
		Text:       c.Text,
		ActorDID:   actorDID,
		CreatedAt:  c.CreatedAt,
		CID:        c.CID,
	}
}
