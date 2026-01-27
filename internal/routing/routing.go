package routing

import (
	"net/http"

	"arabica/internal/atproto"
	"arabica/internal/handlers"
	"arabica/internal/middleware"

	"github.com/rs/zerolog"
)

// Config holds the configuration needed for setting up routes
type Config struct {
	Handlers     *handlers.Handler
	OAuthManager *atproto.OAuthManager
	Logger       zerolog.Logger
}

// SetupRouter creates and configures the HTTP router with all routes and middleware
func SetupRouter(cfg Config) http.Handler {
	h := cfg.Handlers
	mux := http.NewServeMux()

	// Create CrossOriginProtection for CSRF protection
	cop := http.NewCrossOriginProtection()

	// OAuth routes (no CSRF protection needed for GET and callback)
	mux.HandleFunc("GET /login", h.HandleLogin)
	mux.Handle("POST /auth/login", cop.Handler(http.HandlerFunc(h.HandleLoginSubmit)))
	mux.HandleFunc("GET /oauth/callback", h.HandleOAuthCallback)
	mux.Handle("POST /logout", cop.Handler(http.HandlerFunc(h.HandleLogout)))
	mux.HandleFunc("GET /client-metadata.json", h.HandleClientMetadata)
	mux.HandleFunc("GET /.well-known/oauth-client-metadata", h.HandleWellKnownOAuth)

	// API routes for handle resolution (used by login autocomplete)
	// These are intentionally public and don't require HTMX headers
	mux.HandleFunc("GET /api/resolve-handle", h.HandleResolveHandle)
	mux.HandleFunc("GET /api/search-actors", h.HandleSearchActors)

	// API route for fetching all user data (used by client-side cache via fetch())
	// Auth-protected but accessible without HTMX header (called from JavaScript)
	mux.HandleFunc("GET /api/data", h.HandleAPIListAll)

	// API route for current user info (used by Svelte auth store)
	mux.HandleFunc("GET /api/me", h.HandleAPIMe)

	// API endpoint for feed (JSON)
	mux.HandleFunc("GET /api/feed-json", h.HandleFeedAPI)

	// API endpoint for fetching public brew by AT-URI
	mux.HandleFunc("GET /api/brew", h.HandleBrewGetPublic)

	// API endpoint for profile data (JSON for Svelte)
	mux.HandleFunc("GET /api/profile-json/{actor}", h.HandleProfileAPI)

	// Brew CRUD API routes (used by Svelte SPA)
	mux.Handle("POST /brews", cop.Handler(http.HandlerFunc(h.HandleBrewCreate)))
	mux.Handle("PUT /brews/{id}", cop.Handler(http.HandlerFunc(h.HandleBrewUpdate)))
	mux.Handle("DELETE /brews/{id}", cop.Handler(http.HandlerFunc(h.HandleBrewDelete)))

	// Entity CRUD API routes (used by Svelte SPA)
	mux.Handle("POST /api/beans", cop.Handler(http.HandlerFunc(h.HandleBeanCreate)))
	mux.Handle("PUT /api/beans/{id}", cop.Handler(http.HandlerFunc(h.HandleBeanUpdate)))
	mux.Handle("DELETE /api/beans/{id}", cop.Handler(http.HandlerFunc(h.HandleBeanDelete)))

	mux.Handle("POST /api/roasters", cop.Handler(http.HandlerFunc(h.HandleRoasterCreate)))
	mux.Handle("PUT /api/roasters/{id}", cop.Handler(http.HandlerFunc(h.HandleRoasterUpdate)))
	mux.Handle("DELETE /api/roasters/{id}", cop.Handler(http.HandlerFunc(h.HandleRoasterDelete)))

	mux.Handle("POST /api/grinders", cop.Handler(http.HandlerFunc(h.HandleGrinderCreate)))
	mux.Handle("PUT /api/grinders/{id}", cop.Handler(http.HandlerFunc(h.HandleGrinderUpdate)))
	mux.Handle("DELETE /api/grinders/{id}", cop.Handler(http.HandlerFunc(h.HandleGrinderDelete)))

	mux.Handle("POST /api/brewers", cop.Handler(http.HandlerFunc(h.HandleBrewerCreate)))
	mux.Handle("PUT /api/brewers/{id}", cop.Handler(http.HandlerFunc(h.HandleBrewerUpdate)))
	mux.Handle("DELETE /api/brewers/{id}", cop.Handler(http.HandlerFunc(h.HandleBrewerDelete)))

	// Static files (must come after specific routes)
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	// SPA fallback - serve index.html for all unmatched routes (client-side routing)
	// This must be after all API routes and static files
	mux.HandleFunc("GET /{path...}", h.HandleSPAFallback)

	// Apply middleware in order (outermost first, innermost last)
	var handler http.Handler = mux

	// 1. Limit request body size (innermost - runs first on request)
	handler = middleware.LimitBodyMiddleware(handler)

	// 2. Apply OAuth middleware to add auth context
	handler = cfg.OAuthManager.AuthMiddleware(handler)

	// 3. Apply rate limiting
	rateLimitConfig := middleware.NewDefaultRateLimitConfig()
	handler = middleware.RateLimitMiddleware(rateLimitConfig)(handler)

	// 4. Apply security headers
	handler = middleware.SecurityHeadersMiddleware(handler)

	// 5. Apply logging middleware (outermost - wraps everything)
	handler = middleware.LoggingMiddleware(cfg.Logger)(handler)

	return handler
}
