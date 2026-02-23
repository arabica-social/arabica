package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"arabica/internal/atproto"
	"arabica/internal/database/boltstore"
	"arabica/internal/middleware"
	"arabica/internal/moderation"
	"arabica/internal/web/pages"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/rs/zerolog/log"
)

// HandleJoin renders the join request page.
func (h *Handler) HandleJoin(w http.ResponseWriter, r *http.Request) {
	layoutData, _, _ := h.layoutDataFromRequest(r, "Join Arabica")

	if err := pages.Join(layoutData).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render join page")
	}
}

// HandleJoinSubmit processes a join request form submission.
func (h *Handler) HandleJoinSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Honeypot check — if the hidden field is filled, silently reject
	if r.FormValue("website") != "" {
		// Show success page anyway so bots don't know they were caught
		h.renderJoinSuccess(w, r)
		return
	}

	emailAddr := strings.TrimSpace(r.FormValue("email"))
	message := strings.TrimSpace(r.FormValue("message"))

	// Basic email validation
	if emailAddr == "" || !strings.Contains(emailAddr, "@") || !strings.Contains(emailAddr, ".") {
		http.Error(w, "A valid email address is required", http.StatusBadRequest)
		return
	}

	// Create and save the join request
	req := &boltstore.JoinRequest{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Email:     emailAddr,
		Message:   message,
		CreatedAt: time.Now().UTC(),
		IP:        middleware.GetClientIP(r),
	}

	if h.joinStore != nil {
		if err := h.joinStore.SaveRequest(req); err != nil {
			log.Error().Err(err).Str("email", emailAddr).Msg("Failed to save join request")
			http.Error(w, "Failed to save request, please try again", http.StatusInternalServerError)
			return
		}
		log.Info().Str("email", emailAddr).Msg("Join request saved")
	}

	// Send admin notification email (non-blocking)
	if h.emailSender != nil && h.emailSender.Enabled() {
		go func() {
			subject := "New Arabica Join Request"
			body := fmt.Sprintf("New account request:\n\nEmail: %s\nMessage: %s\nIP: %s\nTime: %s\n",
				req.Email, req.Message, req.IP, req.CreatedAt.Format(time.RFC3339))

			if err := h.emailSender.Send(h.emailSender.AdminEmail(), subject, body); err != nil {
				log.Error().Err(err).Str("email", emailAddr).Msg("Failed to send admin notification")
			}
		}()
	}

	h.renderJoinSuccess(w, r)
}

func (h *Handler) renderJoinSuccess(w http.ResponseWriter, r *http.Request) {
	layoutData, _, _ := h.layoutDataFromRequest(r, "Request Received")

	if err := pages.JoinSuccess(layoutData).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render join success page")
	}
}

// HandleCreateInvite creates a PDS invite code and emails it to the requester.
func (h *Handler) HandleCreateInvite(w http.ResponseWriter, r *http.Request) {
	userDID, err := atproto.GetAuthenticatedDID(r.Context())
	if err != nil || userDID == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	if h.moderationService == nil || !h.moderationService.IsAdmin(userDID) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	reqID := r.FormValue("id")
	reqEmail := r.FormValue("email")
	if reqID == "" || reqEmail == "" {
		http.Error(w, "Missing request ID or email", http.StatusBadRequest)
		return
	}

	if h.pdsAdminURL == "" || h.pdsAdminToken == "" {
		http.Error(w, "PDS admin not configured", http.StatusInternalServerError)
		return
	}

	// Create invite code via PDS admin API
	log.Info().Str("pds_url", h.pdsAdminURL).Str("email", reqEmail).Msg("Creating invite code via PDS admin API")
	client := &xrpc.Client{
		Host:       h.pdsAdminURL,
		AdminToken: &h.pdsAdminToken,
	}
	out, err := comatproto.ServerCreateInviteCode(r.Context(), client, &comatproto.ServerCreateInviteCode_Input{
		UseCount: 1,
	})
	if err != nil {
		logEvent := log.Error().Err(err).Str("email", reqEmail).Str("pds_url", h.pdsAdminURL)
		var xrpcErr *xrpc.Error
		if errors.As(err, &xrpcErr) {
			logEvent = logEvent.Int("status_code", xrpcErr.StatusCode)
			var inner *xrpc.XRPCError
			if errors.As(xrpcErr.Wrapped, &inner) {
				logEvent = logEvent.Str("xrpc_error", inner.ErrStr).Str("xrpc_message", inner.Message)
			}
		}
		logEvent.Msg("Failed to create invite code")
		http.Error(w, "Failed to create invite code", http.StatusInternalServerError)
		return
	}

	log.Info().Str("email", reqEmail).Str("code", out.Code).Str("by", userDID).Msg("Invite code created")

	// Email the invite code to the requester
	if h.emailSender != nil && h.emailSender.Enabled() {
		log.Info().Str("email", reqEmail).Str("code", out.Code).Msg("Sending invite code email")
		subject := "Your Arabica Invite Code"
		// TODO: this should probably use the env var rather than hard coded (for name/url)
		// TODO: also this could be a template file
		createURL := "https://arabica.social/join"

		body := fmt.Sprintf("Welcome to Arabica!\n\nHere is your invite code to create an account on the arabica.systems PDS:\n\n    %s\n\nVisit %s to sign up with this code.\n\nHappy brewing!\n", out.Code, createURL)
		if err := h.emailSender.Send(reqEmail, subject, body); err != nil {
			log.Error().Err(err).Str("email", reqEmail).Str("code", out.Code).Msg("Failed to send invite email")
			http.Error(w, "Invite created but failed to send email. Code: "+out.Code, http.StatusInternalServerError)
			return
		}
		log.Info().Str("email", reqEmail).Msg("Invite code emailed successfully")
	} else {
		reason := "nil"
		if h.emailSender != nil {
			reason = "disabled (no SMTP host)"
		}
		log.Warn().Str("email", reqEmail).Str("code", out.Code).Str("reason", reason).Msg("Email sender not available, invite code not emailed")
	}

	// Log the action
	if h.moderationStore != nil {
		details := map[string]string{"email": reqEmail}
		if h.joinStore != nil {
			if joinReq, err := h.joinStore.GetRequest(reqID); err == nil {
				details["ip"] = joinReq.IP
				details["message"] = joinReq.Message
			}
		}
		auditEntry := moderation.AuditEntry{
			ID:        generateTID(),
			Action:    moderation.AuditActionCreateInvite,
			ActorDID:  userDID,
			Details:   details,
			Timestamp: time.Now(),
		}
		if err := h.moderationStore.LogAction(r.Context(), auditEntry); err != nil {
			log.Error().Err(err).Msg("Failed to log create invite action")
		}
	}

	// Remove the join request
	if h.joinStore != nil {
		if err := h.joinStore.DeleteRequest(reqID); err != nil {
			log.Error().Err(err).Str("id", reqID).Msg("Failed to delete join request")
		}
	}

	w.Header().Set("HX-Trigger", "mod-action")
	w.WriteHeader(http.StatusOK)
}

// HandleDismissJoinRequest removes a join request without sending an invite.
func (h *Handler) HandleDismissJoinRequest(w http.ResponseWriter, r *http.Request) {
	userDID, err := atproto.GetAuthenticatedDID(r.Context())
	if err != nil || userDID == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	if h.moderationService == nil || !h.moderationService.IsAdmin(userDID) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	reqID := r.FormValue("id")
	if reqID == "" {
		http.Error(w, "Missing request ID", http.StatusBadRequest)
		return
	}

	// Fetch request details before deleting so we can log them
	var details map[string]string
	if h.joinStore != nil {
		if joinReq, err := h.joinStore.GetRequest(reqID); err == nil {
			details = map[string]string{
				"email":   joinReq.Email,
				"ip":      joinReq.IP,
				"message": joinReq.Message,
			}
		}

		if err := h.joinStore.DeleteRequest(reqID); err != nil {
			log.Error().Err(err).Str("id", reqID).Msg("Failed to delete join request")
			http.Error(w, "Failed to dismiss request", http.StatusInternalServerError)
			return
		}
	}

	// Log the action
	if h.moderationStore != nil {
		auditEntry := moderation.AuditEntry{
			ID:        generateTID(),
			Action:    moderation.AuditActionDismissJoinRequest,
			ActorDID:  userDID,
			Details:   details,
			Timestamp: time.Now(),
		}
		if err := h.moderationStore.LogAction(r.Context(), auditEntry); err != nil {
			log.Error().Err(err).Msg("Failed to log dismiss join request action")
		}
	}

	log.Info().Str("id", reqID).Str("by", userDID).Msg("Join request dismissed")

	w.Header().Set("HX-Trigger", "mod-action")
	w.WriteHeader(http.StatusOK)
}

// signupAllowedPDSURLs is the set of PDS URLs that are allowed for signup.
// This must match the URLs hardcoded in create_account.templ.
var signupAllowedPDSURLs = map[string]bool{
	"https://arabica.systems":   true,
	"https://selfhosted.social": true,
	"https://bsky.social":       true,
}

// HandleCreateAccount renders the account creation page (GET /join/create).
// PDS server options are defined in create_account.templ.
func (h *Handler) HandleCreateAccount(w http.ResponseWriter, r *http.Request) {
	layoutData, _, _ := h.layoutDataFromRequest(r, "Create Account")

	props := pages.CreateAccountProps{
		Error: r.URL.Query().Get("error"),
	}

	if err := pages.CreateAccount(layoutData, props).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render create account page")
	}
}

// HandleCreateAccountSubmit initiates the OAuth prompt=create flow (POST /join/create).
func (h *Handler) HandleCreateAccountSubmit(w http.ResponseWriter, r *http.Request) {
	if h.oauth == nil {
		http.Error(w, "OAuth not configured", http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	pdsURL := r.FormValue("pds_url")
	if pdsURL == "" {
		http.Redirect(w, r, "/join/create?error=Please+select+a+server", http.StatusSeeOther)
		return
	}

	if !signupAllowedPDSURLs[pdsURL] {
		log.Warn().Str("pds_url", pdsURL).Msg("Signup attempt with unlisted PDS URL")
		http.Redirect(w, r, "/join/create?error=Invalid+server+selection", http.StatusSeeOther)
		return
	}

	// Initiate OAuth flow with prompt=create
	authURL, err := h.oauth.InitiateSignup(r.Context(), pdsURL)
	if err != nil {
		log.Error().Err(err).Str("pds_url", pdsURL).Msg("Failed to initiate signup flow")
		http.Redirect(w, r, "/join/create?error=Failed+to+connect+to+server.+Please+try+again.", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, authURL, http.StatusFound)
}
