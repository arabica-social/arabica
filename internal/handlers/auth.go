package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// HandleLogin redirects to the home page
// The login form is now integrated into the home page
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/", http.StatusFound)
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

// HandleResolveHandle resolves an AT Protocol handle and returns basic profile info
// This is used for the autocomplete login feature
func (h *Handler) HandleResolveHandle(w http.ResponseWriter, r *http.Request) {
	handle := r.URL.Query().Get("handle")
	if handle == "" {
		http.Error(w, "Handle parameter is required", http.StatusBadRequest)
		return
	}

	// Use a public API client to resolve the handle
	// We don't need authentication for this
	apiClient := &http.Client{}

	// First resolve the handle to a DID
	// The URL package will properly encode the query parameter
	resolveURL := fmt.Sprintf("https://bsky.social/xrpc/com.atproto.identity.resolveHandle?handle=%s", handle)
	resp, err := apiClient.Get(resolveURL)
	if err != nil {
		log.Printf("Error resolving handle %q: %v", handle, err)
		http.Error(w, "Failed to resolve handle", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Handle not found"})
		return
	}

	if resp.StatusCode != 200 {
		// Read the error body for better debugging
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("Unexpected status resolving handle %q: %d, body: %s", handle, resp.StatusCode, string(bodyBytes))

		// Return a more informative error for 400s
		if resp.StatusCode == 400 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid handle format"})
			return
		}

		http.Error(w, "Failed to resolve handle", http.StatusInternalServerError)
		return
	}

	var resolveResult struct {
		DID string `json:"did"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&resolveResult); err != nil {
		log.Printf("Error decoding resolve response: %v", err)
		http.Error(w, "Failed to parse resolve response", http.StatusInternalServerError)
		return
	}

	// Now fetch the profile for this DID
	profileURL := fmt.Sprintf("https://bsky.social/xrpc/app.bsky.actor.getProfile?actor=%s", resolveResult.DID)
	profileResp, err := apiClient.Get(profileURL)
	if err != nil {
		log.Printf("Error fetching profile: %v", err)
		// Return just the DID if we can't get the profile
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"did":    resolveResult.DID,
			"handle": handle,
		})
		return
	}
	defer profileResp.Body.Close()

	if profileResp.StatusCode != 200 {
		// Return just the DID if we can't get the profile
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"did":    resolveResult.DID,
			"handle": handle,
		})
		return
	}

	var profile struct {
		DID         string  `json:"did"`
		Handle      string  `json:"handle"`
		DisplayName *string `json:"displayName,omitempty"`
		Avatar      *string `json:"avatar,omitempty"`
	}
	if err := json.NewDecoder(profileResp.Body).Decode(&profile); err != nil {
		log.Printf("Error decoding profile: %v", err)
		// Return just the DID if we can't parse the profile
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"did":    resolveResult.DID,
			"handle": handle,
		})
		return
	}

	// Return the profile info
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// HandleSearchActors searches for actors by handle or display name
// This is used for the autocomplete login feature
func (h *Handler) HandleSearchActors(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	// Use a public API client
	apiClient := &http.Client{}

	// Try using the public API endpoint with typeahead parameter
	// Some PDS instances support public search
	searchURL := fmt.Sprintf("https://public.api.bsky.app/xrpc/app.bsky.actor.searchActorsTypeahead?q=%s&limit=5", query)
	resp, err := apiClient.Get(searchURL)
	if err != nil {
		log.Printf("Error searching actors for %q: %v", query, err)
		// Return empty results instead of error
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"actors": []interface{}{}})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("Unexpected status searching actors %q: %d, body: %s", query, resp.StatusCode, string(bodyBytes))
		// Return empty results instead of error
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"actors": []interface{}{}})
		return
	}

	var searchResult struct {
		Actors []struct {
			DID         string  `json:"did"`
			Handle      string  `json:"handle"`
			DisplayName *string `json:"displayName,omitempty"`
			Avatar      *string `json:"avatar,omitempty"`
		} `json:"actors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		log.Printf("Error decoding search response: %v", err)
		// Return empty results instead of error
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"actors": []interface{}{}})
		return
	}

	// Return the actors
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(searchResult)
}
