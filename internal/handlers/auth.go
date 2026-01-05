package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// HandleLogin shows the login page
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	// TODO: Render login template
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
	<title>Login - Arabica</title>
</head>
<body>
	<h1>Login to Arabica</h1>
	<form method="POST" action="/auth/login">
		<label for="handle">Your ATProto Handle:</label>
		<input type="text" id="handle" name="handle" placeholder="alice.bsky.social" required>
		<button type="submit">Login</button>
	</form>
</body>
</html>
	`))
}

// HandleLoginSubmit initiates the OAuth flow
func (h *Handler) HandleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	if h.oauth == nil {
		http.Error(w, "OAuth not configured", http.StatusInternalServerError)
		return
	}

	handle := r.FormValue("handle")
	if handle == "" {
		http.Error(w, "Handle is required", http.StatusBadRequest)
		return
	}

	// Initiate OAuth flow
	authURL, err := h.oauth.InitiateLogin(r.Context(), handle)
	if err != nil {
		log.Printf("Error initiating login: %v", err)
		http.Error(w, "Failed to initiate login", http.StatusInternalServerError)
		return
	}

	// Redirect to PDS authorization endpoint
	// State and PKCE are handled automatically by the OAuth client
	http.Redirect(w, r, authURL, http.StatusFound)
}

// HandleOAuthCallback handles the OAuth callback from the PDS
func (h *Handler) HandleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	if h.oauth == nil {
		http.Error(w, "OAuth not configured", http.StatusInternalServerError)
		return
	}

	// Process the callback with all query parameters
	sessData, err := h.oauth.HandleCallback(r.Context(), r.URL.Query())
	if err != nil {
		log.Printf("Error completing OAuth flow: %v", err)
		http.Error(w, "Failed to complete login", http.StatusInternalServerError)
		return
	}

	// Set session cookies
	http.SetCookie(w, &http.Cookie{
		Name:     "account_did",
		Value:    sessData.AccountDID.String(),
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // TODO: Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 30, // 30 days
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessData.SessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // TODO: Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 30, // 30 days
	})

	log.Printf("User logged in: DID=%s, SessionID=%s", sessData.AccountDID.String(), sessData.SessionID)

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusFound)
}

// HandleLogout logs out the user
func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if h.oauth == nil {
		http.Error(w, "OAuth not configured", http.StatusInternalServerError)
		return
	}

	// Get session cookies
	didCookie, err1 := r.Cookie("account_did")
	sessionCookie, err2 := r.Cookie("session_id")

	if err1 == nil && err2 == nil {
		// Parse DID
		did, err := syntax.ParseDID(didCookie.Value)
		if err == nil {
			// Delete session from store
			err = h.oauth.DeleteSession(r.Context(), did, sessionCookie.Value)
			if err != nil {
				log.Printf("Error deleting session: %v", err)
			}
		}
	}

	// Clear session cookies
	http.SetCookie(w, &http.Cookie{
		Name:     "account_did",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusFound)
}

// HandleClientMetadata serves the OAuth client metadata
func (h *Handler) HandleClientMetadata(w http.ResponseWriter, r *http.Request) {
	if h.oauth == nil {
		http.Error(w, "OAuth not configured", http.StatusInternalServerError)
		return
	}

	metadata := h.oauth.ClientMetadata()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandleWellKnownOAuth serves the OAuth client metadata at /.well-known/oauth-client-metadata
func (h *Handler) HandleWellKnownOAuth(w http.ResponseWriter, r *http.Request) {
	h.HandleClientMetadata(w, r)
}
