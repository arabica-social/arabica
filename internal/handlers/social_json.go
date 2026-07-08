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
	Comments       []CommentJSON `json:"comments"`
	SubjectURI     string        `json:"subject_uri"`
	IsAuthenticated bool         `json:"is_authenticated"`
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
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	didStr, _ := atpmiddleware.GetDID(r.Context())

	if err := r.ParseForm(); err != nil {
		log.Warn().Err(err).Msg("Failed to parse like toggle form")
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	subjectURI := r.FormValue("subject_uri")
	subjectCID := r.FormValue("subject_cid")

	if subjectURI == "" || subjectCID == "" {
		http.Error(w, "subject_uri and subject_cid are required", http.StatusBadRequest)
		return
	}

	existingLike, err := store.GetUserLikeForSubject(r.Context(), subjectURI)
	if err != nil {
		log.Error().Err(err).Msg("Failed to check existing like")
		HandleStoreError(w, err, "Failed to check like status")
		return
	}

	var isLiked bool
	var likeCount int

	if existingLike != nil {
		if err := store.DeleteLikeByRKey(r.Context(), existingLike.RKey); err != nil {
			log.Error().Err(err).Msg("Failed to delete like")
			HandleStoreError(w, err, "Failed to unlike")
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
			HandleStoreError(w, err, "Failed to like")
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
		http.Error(w, "subject_uri is required", http.StatusBadRequest)
		return
	}

	didStr, isAuthenticated := atpmiddleware.GetDID(r.Context())

	var comments []firehose.IndexedComment
	if h.feedIndex != nil {
		comments = h.feedIndex.GetThreadedCommentsForSubject(r.Context(), subjectURI, 100, didStr)
		comments = h.FilterHiddenComments(r.Context(), comments)
	}

	WriteJSON(w, CommentListResponseJSON{
		Comments:       NewCommentsJSON(comments),
		SubjectURI:     subjectURI,
		IsAuthenticated: isAuthenticated,
	}, "comments")
}

// HandleCommentCreateJSON creates a comment and returns JSON for the SvelteKit SPA.
func (h *Handler) HandleCommentCreateJSON(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getSocialStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	didStr, _ := atpmiddleware.GetDID(r.Context())

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	subjectURI := r.FormValue("subject_uri")
	subjectCID := r.FormValue("subject_cid")
	text := r.FormValue("text")
	parentURI := r.FormValue("parent_uri")
	parentCID := r.FormValue("parent_cid")

	if subjectURI == "" || subjectCID == "" {
		http.Error(w, "subject_uri and subject_cid are required", http.StatusBadRequest)
		return
	}
	if text == "" {
		http.Error(w, "comment text is required", http.StatusBadRequest)
		return
	}
	if len(text) > social.MaxCommentLength {
		http.Error(w, "comment text is too long", http.StatusBadRequest)
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
		HandleStoreError(w, err, "Failed to create comment")
		return
	}

	metrics.CommentsTotal.WithLabelValues("create").Inc()

	if h.feedIndex != nil {
		if err := h.feedIndex.UpsertComment(r.Context(), didStr, comment.RKey, subjectURI, parentURI, comment.CID, text, comment.CreatedAt); err != nil {
			log.Warn().Err(err).Str("did", didStr).Str("rkey", comment.RKey).Str("subject_uri", subjectURI).Msg("Failed to upsert comment in feed index")
		}
		h.feedIndex.CreateCommentNotification(didStr, subjectURI, parentURI)
	}

	// Return the updated comment section
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

// HandleCommentDeleteJSON deletes a comment and returns JSON for the SvelteKit SPA.
func (h *Handler) HandleCommentDeleteJSON(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getSocialStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	didStr, _ := atpmiddleware.GetDID(r.Context())

	rkey := r.PathValue("id")
	if rkey == "" {
		http.Error(w, "Comment ID is required", http.StatusBadRequest)
		return
	}

	if err := store.DeleteCommentByRKey(r.Context(), rkey); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Str("did", didStr).Msg("Failed to delete comment from PDS")
		HandleStoreError(w, err, "Failed to delete comment")
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
		WriteJSON(w, map[string]string{"error": "Invalid form data"}, "report-error")
		return
	}
	subjectURI := r.FormValue("subject_uri")
	rawReason := r.FormValue("reason")

	reporterDID, ok := atpmiddleware.GetDID(ctx)
	if !ok {
		WriteJSON(w, map[string]string{"error": "Authentication required"}, "report-error")
		return
	}

	if h.moderationStore == nil {
		log.Error().Msg("moderation: store not configured")
		WriteJSON(w, map[string]string{"error": "Reports are not enabled"}, "report-error")
		return
	}

	if subjectURI == "" {
		WriteJSON(w, map[string]string{"error": "subject_uri is required"}, "report-error")
		return
	}

	uriParts, err := atp.ParseATURI(subjectURI)
	if err != nil {
		WriteJSON(w, map[string]string{"error": "Invalid subject_uri format"}, "report-error")
		return
	}
	subjectDID := uriParts.DID

	if subjectDID == reporterDID {
		WriteJSON(w, map[string]string{"error": "You cannot report your own content"}, "report-error")
		return
	}

	reason := rawReason
	if reason == "" {
		reason = "No reason provided"
	}
	if len(reason) > MaxReportReasonLength {
		reason = reason[:MaxReportReasonLength]
	}

	// Rate limit check
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	recentCount, err := h.moderationStore.CountReportsFromUserSince(ctx, reporterDID, oneHourAgo)
	if err != nil {
		log.Error().Err(err).Str("reporter", reporterDID).Msg("moderation: failed to check rate limit")
		WriteJSON(w, map[string]string{"error": "Failed to process report"}, "report-error")
		return
	}
	if recentCount >= ReportRateLimitPerHour {
		WriteJSON(w, map[string]string{"error": "Rate limit exceeded. Please try again later."}, "report-error")
		return
	}

	alreadyReported, err := h.moderationStore.HasReportedURI(ctx, reporterDID, subjectURI)
	if err != nil {
		log.Error().Err(err).Str("reporter", reporterDID).Msg("moderation: failed to check duplicate")
		WriteJSON(w, map[string]string{"error": "Failed to process report"}, "report-error")
		return
	}
	if alreadyReported {
		WriteJSON(w, map[string]string{"error": "You have already reported this content"}, "report-error")
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
		WriteJSON(w, map[string]string{"error": "Failed to save report"}, "report-error")
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
