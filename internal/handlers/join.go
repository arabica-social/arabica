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
	client := &xrpc.Client{
		Host:       h.pdsAdminURL,
		AdminToken: &h.pdsAdminToken,
	}
	out, err := comatproto.ServerCreateInviteCode(r.Context(), client, &comatproto.ServerCreateInviteCode_Input{
		UseCount: 1,
	})
	if err != nil {
		log.Error().Err(err).Str("email", reqEmail).Msg("Failed to create invite code")
		http.Error(w, "Failed to create invite code", http.StatusInternalServerError)
		return
	}

	log.Info().Str("email", reqEmail).Str("code", out.Code).Str("by", userDID).Msg("Invite code created")

	// Email the invite code to the requester
	if h.emailSender != nil && h.emailSender.Enabled() {
		subject := "Your Arabica Invite Code"
		// TODO: this should probably use the env var rather than hard coded (for name/url)
		// TODO: also this could be a template file
		body := fmt.Sprintf("Welcome to Arabica!\n\nHere is your invite code to create an account on the arabica.systems PDS:\n\n    %s\n\nVisit https://arabica.social/join to sign up with this code.\n\nHappy brewing!\n", out.Code)
		if err := h.emailSender.Send(reqEmail, subject, body); err != nil {
			log.Error().Err(err).Str("email", reqEmail).Msg("Failed to send invite email")
			http.Error(w, "Invite created but failed to send email. Code: "+out.Code, http.StatusInternalServerError)
			return
		}
		log.Info().Str("email", reqEmail).Msg("Invite code emailed")
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

// HandleCreateAccount renders the account creation form (GET /join/create).
func (h *Handler) HandleCreateAccount(w http.ResponseWriter, r *http.Request) {
	layoutData, _, _ := h.layoutDataFromRequest(r, "Create Account")

	props := pages.CreateAccountProps{
		InviteCode:   r.URL.Query().Get("code"),
		HandleDomain: "arabica.systems",
	}

	if err := pages.CreateAccount(layoutData, props).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render create account page")
	}
}

// HandleCreateAccountSubmit processes the account creation form (POST /join/create).
func (h *Handler) HandleCreateAccountSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	inviteCode := strings.TrimSpace(r.FormValue("invite_code"))
	handle := strings.TrimSpace(r.FormValue("handle"))
	emailAddr := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	passwordConfirm := r.FormValue("password_confirm")
	honeypot := r.FormValue("website")

	// Honeypot check — bots fill hidden fields; show fake success
	if honeypot != "" {
		layoutData, _, _ := h.layoutDataFromRequest(r, "Account Created")
		_ = pages.CreateAccountSuccess(layoutData, pages.CreateAccountSuccessProps{Handle: "user.arabica.systems"}).Render(r.Context(), w)
		return
	}

	handleDomain := "arabica.systems"

	// Render form with error helper
	renderError := func(msg string) {
		layoutData, _, _ := h.layoutDataFromRequest(r, "Create Account")
		props := pages.CreateAccountProps{
			Error:        msg,
			InviteCode:   inviteCode,
			Handle:       handle,
			Email:        emailAddr,
			HandleDomain: handleDomain,
		}
		if err := pages.CreateAccount(layoutData, props).Render(r.Context(), w); err != nil {
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
		}
	}

	// Validate required fields
	if inviteCode == "" || handle == "" || emailAddr == "" || password == "" {
		renderError("All fields are required.")
		return
	}
	if password != passwordConfirm {
		renderError("Passwords do not match.")
		return
	}

	// Build full handle
	fullHandle := handle + "." + handleDomain

	if h.pdsAdminURL == "" {
		renderError("Account creation is not available at this time.")
		log.Error().Msg("PDS admin URL not configured for account creation")
		return
	}

	// Call PDS createAccount (public endpoint, no admin token needed)
	client := &xrpc.Client{Host: h.pdsAdminURL}
	out, err := comatproto.ServerCreateAccount(r.Context(), client, &comatproto.ServerCreateAccount_Input{
		Handle:     fullHandle,
		Email:      &emailAddr,
		Password:   &password,
		InviteCode: &inviteCode,
	})
	if err != nil {
		errMsg := "Account creation failed. Please try again."
		var xrpcErr *xrpc.Error
		if errors.As(err, &xrpcErr) {
			var inner *xrpc.XRPCError
			if errors.As(xrpcErr.Wrapped, &inner) {
				switch inner.ErrStr {
				case "InvalidInviteCode":
					errMsg = "Invalid or expired invite code."
				case "HandleNotAvailable":
					errMsg = "This handle is already taken."
				case "InvalidHandle":
					errMsg = "Invalid handle format. Use only letters, numbers, and hyphens."
				default:
					if inner.Message != "" {
						errMsg = inner.Message
					}
				}
			}
		}
		log.Error().Err(err).Str("handle", fullHandle).Msg("Failed to create account")
		renderError(errMsg)
		return
	}

	log.Info().Str("handle", out.Handle).Str("did", out.Did).Msg("Account created")

	layoutData, _, _ := h.layoutDataFromRequest(r, "Account Created")
	if err := pages.CreateAccountSuccess(layoutData, pages.CreateAccountSuccessProps{Handle: out.Handle}).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render create account success page")
	}
}
