package routing

import (
	"net/http"

	"arabica/internal/atproto"
	"arabica/internal/handlers"
	"arabica/internal/middleware"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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

	// Suggestion routes for entity typeahead (auth-protected, read-only GET)
	mux.HandleFunc("GET /api/suggestions/{entity}", h.HandleEntitySuggestions)

	// HTMX partials (loaded async via HTMX)
	// These return HTML fragments and should only be accessed via HTMX
	mux.Handle("GET /api/feed", middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandleFeedPartial)))
	mux.Handle("GET /api/brews", middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandleBrewListPartial)))
	mux.Handle("GET /api/manage", middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandleManagePartial)))
	mux.Handle("GET /api/profile/{actor}", middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandleProfilePartial)))

	// Page routes (must come before static files)
	mux.HandleFunc("GET /{$}", h.HandleHome) // {$} means exact match
	mux.HandleFunc("GET /about", h.HandleAbout)
	mux.HandleFunc("GET /terms", h.HandleTerms)
	mux.HandleFunc("GET /join", h.HandleJoin)
	mux.Handle("POST /join", cop.Handler(http.HandlerFunc(h.HandleJoinSubmit)))
	mux.HandleFunc("GET /join/create", h.HandleCreateAccount)
	mux.Handle("POST /join/create", cop.Handler(http.HandlerFunc(h.HandleCreateAccountSubmit)))
	mux.HandleFunc("GET /atproto", h.HandleATProto)
	mux.HandleFunc("GET /manage", h.HandleManage)
	mux.HandleFunc("GET /brews", h.HandleBrewList)
	mux.HandleFunc("GET /brews/new", h.HandleBrewNew)
	mux.HandleFunc("GET /brews/{id}", h.HandleBrewView)
	mux.HandleFunc("GET /beans/{id}", h.HandleBeanView)
	mux.HandleFunc("GET /roasters/{id}", h.HandleRoasterView)
	mux.HandleFunc("GET /grinders/{id}", h.HandleGrinderView)
	mux.HandleFunc("GET /brewers/{id}", h.HandleBrewerView)
	mux.HandleFunc("GET /brews/{id}/edit", h.HandleBrewEdit)
	mux.Handle("POST /brews", cop.Handler(http.HandlerFunc(h.HandleBrewCreate)))
	mux.Handle("PUT /brews/{id}", cop.Handler(http.HandlerFunc(h.HandleBrewUpdate)))
	mux.Handle("DELETE /brews/{id}", cop.Handler(http.HandlerFunc(h.HandleBrewDelete)))
	mux.HandleFunc("GET /brews/export", h.HandleBrewExport)

	// API routes for CRUD operations
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

	mux.Handle("POST /api/likes/toggle", cop.Handler(http.HandlerFunc(h.HandleLikeToggle)))
	mux.Handle("POST /api/report", cop.Handler(http.HandlerFunc(h.HandleReport)))

	// Comment routes
	mux.Handle("GET /api/comments", middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandleCommentList)))
	mux.Handle("POST /api/comments", cop.Handler(http.HandlerFunc(h.HandleCommentCreate)))
	mux.Handle("DELETE /api/comments/{id}", cop.Handler(http.HandlerFunc(h.HandleCommentDelete)))

	// Modal routes for entity management (return dialog HTML)
	mux.HandleFunc("GET /api/modals/bean/new", h.HandleBeanModalNew)
	mux.HandleFunc("GET /api/modals/bean/{id}", h.HandleBeanModalEdit)
	mux.HandleFunc("GET /api/modals/grinder/new", h.HandleGrinderModalNew)
	mux.HandleFunc("GET /api/modals/grinder/{id}", h.HandleGrinderModalEdit)
	mux.HandleFunc("GET /api/modals/brewer/new", h.HandleBrewerModalNew)
	mux.HandleFunc("GET /api/modals/brewer/{id}", h.HandleBrewerModalEdit)
	mux.HandleFunc("GET /api/modals/roaster/new", h.HandleRoasterModalNew)
	mux.HandleFunc("GET /api/modals/roaster/{id}", h.HandleRoasterModalEdit)

	// Notification routes
	mux.HandleFunc("GET /notifications", h.HandleNotifications)
	mux.Handle("POST /api/notifications/read", cop.Handler(http.HandlerFunc(h.HandleNotificationsMarkRead)))

	// Profile routes (public user profiles)
	mux.HandleFunc("GET /profile/{actor}", h.HandleProfile)

	// Moderation routes
	mux.HandleFunc("GET /_mod", h.HandleAdmin)
	mux.Handle("GET /_mod/content", middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandleAdminPartial)))
	mux.Handle("POST /_mod/hide", cop.Handler(http.HandlerFunc(h.HandleHideRecord)))
	mux.Handle("POST /_mod/unhide", cop.Handler(http.HandlerFunc(h.HandleUnhideRecord)))
	mux.Handle("POST /_mod/dismiss-report", cop.Handler(http.HandlerFunc(h.HandleDismissReport)))
	mux.Handle("POST /_mod/reset-autohide", cop.Handler(http.HandlerFunc(h.HandleResetAutoHide)))
	mux.Handle("POST /_mod/block", cop.Handler(http.HandlerFunc(h.HandleBlockUser)))
	mux.Handle("POST /_mod/unblock", cop.Handler(http.HandlerFunc(h.HandleUnblockUser)))
	mux.Handle("POST /_mod/invite", cop.Handler(http.HandlerFunc(h.HandleCreateInvite)))
	mux.Handle("POST /_mod/dismiss-join", cop.Handler(http.HandlerFunc(h.HandleDismissJoinRequest)))
	mux.Handle("GET /_mod/stats", middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandleAdminStats)))

	// Static files (must come after specific routes)
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))
	// Serve favicon.ico for pdsls
	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		http.ServeFile(w, r, "static/favicon.ico")
	})

	// Catch-all 404 handler - must be last, catches any unmatched routes
	mux.HandleFunc("/", h.HandleNotFound)

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

	// 5. Apply logging middleware
	handler = middleware.LoggingMiddleware(cfg.Logger)(handler)

	// 6. Apply OpenTelemetry HTTP instrumentation (outermost - wraps everything)
	handler = otelhttp.NewHandler(handler, "arabica",
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method + " " + r.URL.Path
		}),
	)

	return handler
}
