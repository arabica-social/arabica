package routing

import (
	"encoding/json"
	"net/http"
	"strings"

	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/firehose"
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/metrics"
	"tangled.org/arabica.social/arabica/internal/middleware"
	"tangled.org/arabica.social/arabica/internal/moderation"
	"tangled.org/arabica.social/arabica/internal/web/assets"
	"tangled.org/arabica.social/arabica/internal/web/spa"
	"tangled.org/pdewey.com/atp"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Config holds the configuration needed for setting up routes
type Config struct {
	App               *domain.App
	Handlers          *handlers.Handler
	OAuthApp          *atp.OAuthApp
	OnAuth            func(did string)
	Logger            zerolog.Logger
	ModerationService *moderation.Service
	FirehoseConsumer  *firehose.Consumer
	CSSBundle         *assets.Bundle
	JSAssets          *assets.JSAssets
	AppRoutes         AppRoutes
	DisableRateLimit  bool

	// SPAAssetDevDir, when non-empty, is a path to the SvelteKit build
	// output on disk (e.g. web/build). The /_app/ route serves from it
	// instead of the embedded filesystem, picking up `vite build --watch`
	// output on refresh without a Go restart. Empty in production.
	SPAAssetDevDir string

	// SPAHandler, when non-nil, serves the SvelteKit SPA shell for unmatched
	// page routes explicitly owned by AppRoutes.SPAOwnedRoutes. During
	// migration this is nil unless explicitly enabled, so existing templ
	// pages work unchanged.
	SPAHandler http.Handler
}

// AppRoutes is implemented by app-owned packages that register routes whose
// handlers or page models are not shared platform concerns.
type AppRoutes interface {
	RegisterAppRoutes(mux *http.ServeMux, ctx AppRouteContext)
	SPAOwnedRoutes() []string
}

// AppRouteContext exposes the shared dependencies app route registrars need
// without making routing import either app package.
type AppRouteContext struct {
	App      *domain.App
	Handlers *handlers.Handler
	CSRF     *http.CrossOriginProtection
	Pages    PageRoutes
}

// PageRoutes registers each page pattern with either its legacy handler or
// the SPA shell according to an explicit app-owned allowlist. API, mutation,
// auth, asset, and unknown routes do not participate in page ownership.
type PageRoutes struct {
	spaHandler http.Handler
	spaOwned   map[string]struct{}
}

// NewPageRoutes creates a page registrar for the supplied SPA-owned patterns.
func NewPageRoutes(spaHandler http.Handler, spaOwned []string) PageRoutes {
	owned := make(map[string]struct{}, len(spaOwned))
	for _, pattern := range spaOwned {
		owned[pattern] = struct{}{}
	}
	return PageRoutes{spaHandler: spaHandler, spaOwned: owned}
}

// Register assigns one page route to its explicit owner.
func (p PageRoutes) Register(mux *http.ServeMux, pattern string, legacy http.Handler) {
	if p.IsSPA(pattern) {
		mux.Handle(pattern, p.spaHandler)
		return
	}
	mux.Handle(pattern, legacy)
}

// IsSPA reports whether a page pattern is explicitly owned by the SPA.
func (p PageRoutes) IsSPA(pattern string) bool {
	if p.spaHandler == nil {
		return false
	}
	_, ok := p.spaOwned[pattern]
	return ok
}

// SetupRouter creates and configures the HTTP router with all routes and middleware
func SetupRouter(cfg Config) http.Handler {
	h := cfg.Handlers
	mux := http.NewServeMux()
	var spaOwned []string
	if cfg.AppRoutes != nil {
		spaOwned = cfg.AppRoutes.SPAOwnedRoutes()
	}
	pages := NewPageRoutes(cfg.SPAHandler, spaOwned)

	// Create CrossOriginProtection for CSRF protection
	cop := http.NewCrossOriginProtection()

	// OAuth routes (no CSRF protection needed for GET and callback)
	mux.HandleFunc("GET /login", h.HandleLogin)
	mux.Handle("POST /auth/login", cop.Handler(http.HandlerFunc(h.HandleLoginSubmit)))
	mux.HandleFunc("GET /oauth/callback", h.HandleOAuthCallback)
	mux.Handle("POST /logout", cop.Handler(http.HandlerFunc(h.HandleLogout)))
	mux.Handle("POST /reauth", cop.Handler(http.HandlerFunc(h.HandleReauth)))
	mux.HandleFunc("GET /.well-known/oauth-client-metadata.json", h.HandleClientMetadata)
	mux.HandleFunc("GET /.well-known/oauth-protected-resource/.well-known/oauth-client-metadata.json", h.HandleClientMetadata)
	mux.HandleFunc("GET /.well-known/client-metadata.json", h.HandleClientMetadata)
	mux.HandleFunc("GET /.well-known/client-metadata", h.HandleClientMetadata)
	mux.HandleFunc("GET /client-metadata.json", h.HandleClientMetadata)
	mux.HandleFunc("GET /.well-known/security.txt", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/.well-known/security.txt")
	})
	mux.HandleFunc("GET /robots.txt", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/robots.txt")
	})
	mux.HandleFunc("GET /healthz", handleHealthz(h, cfg.FirehoseConsumer))
	mux.HandleFunc("GET /api/session", h.HandleSessionJSON)
	mux.HandleFunc("GET /api/session/status", h.HandleSessionStatusJSON)

	// API routes for handle resolution (used by login autocomplete)
	// These are intentionally public and don't require HTMX headers
	mux.HandleFunc("GET /api/resolve-handle", h.HandleResolveHandle)
	mux.HandleFunc("GET /api/search-actors", h.HandleSearchActors)

	// Suggestion routes for entity typeahead (auth-protected, read-only GET)
	mux.HandleFunc("GET /api/suggestions/{entity}", h.HandleEntitySuggestions)

	// Feed is JSON-only for the SvelteKit SPA.
	mux.HandleFunc("GET /api/feed", h.HandleFeed)

	// Page routes (must come before static files). These patterns are all
	// SPA-owned (see each app's SPAOwnedRoutes), so pages.Register routes
	// them to the SPA shell; the legacy handler arg is a nil-safe 404
	// fallback for a non-SPA build rather than a templ renderer.
	notFound := http.HandlerFunc(h.HandleNotFound)
	pages.Register(mux, "GET /{$}", notFound) // {$} means exact match
	pages.Register(mux, "GET /about", notFound)
	pages.Register(mux, "GET /terms", notFound)
	pages.Register(mux, "GET /join/create", notFound)
	pages.Register(mux, "GET /atproto", notFound)
	pages.Register(mux, "GET /notifications", notFound)
	pages.Register(mux, "GET /settings", notFound)
	pages.Register(mux, "GET /_mod", notFound)
	mux.HandleFunc("GET /og-image", h.HandleSiteOGImage)
	mux.Handle("POST /join/create", cop.Handler(http.HandlerFunc(h.HandleCreateAccountSubmit)))
	mux.HandleFunc("GET /api/signup/categories", h.HandleSignupCategories)

	if cfg.AppRoutes != nil {
		cfg.AppRoutes.RegisterAppRoutes(mux, AppRouteContext{
			App:      cfg.App,
			Handlers: h,
			CSRF:     cop,
			Pages:    pages,
		})
	}

	mux.Handle("POST /api/likes/toggle", cop.Handler(http.HandlerFunc(h.HandleLikeToggle)))
	mux.Handle("POST /api/report", cop.Handler(http.HandlerFunc(h.HandleReport)))

	// AT-URI shaped redirect: /at/{nsid}/{actor}/{rkey} -> /{slug}/{actor}/{rkey}.
	// Lets power users paste the lexicon-shaped URL and land on the canonical
	// friendly-slug page.
	mux.HandleFunc("GET /at/{nsid}/{actor}/{id}", func(w http.ResponseWriter, r *http.Request) {
		nsid := r.PathValue("nsid")
		actor := r.PathValue("actor")
		rkey := r.PathValue("id")
		if cfg.App == nil {
			http.NotFound(w, r)
			return
		}
		route, ok := cfg.App.EntityRouteByNSID(nsid)
		if !ok || route.Path == "" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/"+route.Path+"/"+actor+"/"+rkey, http.StatusFound)
	})

	// Comment routes
	mux.Handle("GET /api/comments", http.HandlerFunc(h.HandleCommentList))
	mux.Handle("POST /api/comments", cop.Handler(http.HandlerFunc(h.HandleCommentCreate)))
	mux.Handle("DELETE /api/comments/{id}", cop.Handler(http.HandlerFunc(h.HandleCommentDelete)))

	// Notification routes
	mux.HandleFunc("GET /api/notifications", h.HandleNotificationsJSON)
	mux.Handle("POST /api/notifications/read", cop.Handler(http.HandlerFunc(h.HandleNotificationsMarkRead)))

	// Settings
	mux.HandleFunc("GET /api/settings", h.HandleSettingsJSON)
	mux.Handle("POST /api/settings/preferences", cop.Handler(http.HandlerFunc(h.HandleSettingsPreferences)))
	mux.Handle("POST /api/settings/profile-visibility", cop.Handler(http.HandlerFunc(h.HandleSettingsProfileVisibility)))
	mux.Handle("POST /api/settings/bluesky-profile", cop.Handler(http.HandlerFunc(h.HandleUpdateBlueskyProfile)))
	mux.Handle("POST /settings/bluesky-profile/upgrade-scopes", cop.Handler(http.HandlerFunc(h.HandleScopeUpgrade)))

	// Moderation routes
	// HandleAdmin keeps its own auth check (redirects to / instead of 401)
	modSvc := cfg.ModerationService
	mux.Handle("GET /api/_mod", middleware.RequireModerator(modSvc,
		http.HandlerFunc(h.HandleAdminJSON)))
	mux.Handle("POST /_mod/hide", cop.Handler(
		middleware.RequirePermission(modSvc, moderation.PermissionHideRecord, http.HandlerFunc(h.HandleHideRecord))))
	mux.Handle("POST /_mod/unhide", cop.Handler(
		middleware.RequirePermission(modSvc, moderation.PermissionUnhideRecord, http.HandlerFunc(h.HandleUnhideRecord))))
	mux.Handle("POST /_mod/dismiss-report", cop.Handler(
		middleware.RequirePermission(modSvc, moderation.PermissionDismissReport, http.HandlerFunc(h.HandleDismissReport))))
	mux.Handle("POST /_mod/reset-autohide", cop.Handler(
		middleware.RequirePermission(modSvc, moderation.PermissionResetAutoHide, http.HandlerFunc(h.HandleResetAutoHide))))
	mux.Handle("POST /_mod/block", cop.Handler(
		middleware.RequirePermission(modSvc, moderation.PermissionBlacklistUser, http.HandlerFunc(h.HandleBlockUser))))
	mux.Handle("POST /_mod/unblock", cop.Handler(
		middleware.RequirePermission(modSvc, moderation.PermissionUnblacklistUser, http.HandlerFunc(h.HandleUnblockUser))))
	mux.Handle("POST /_mod/label/add", cop.Handler(
		middleware.RequirePermission(modSvc, moderation.PermissionManageLabels, http.HandlerFunc(h.HandleAddLabel))))
	mux.Handle("POST /_mod/label/remove", cop.Handler(
		middleware.RequirePermission(modSvc, moderation.PermissionManageLabels, http.HandlerFunc(h.HandleRemoveLabel))))
	mux.Handle("GET /api/_mod/stats", middleware.RequireAdmin(modSvc,
		http.HandlerFunc(h.HandleAdminStatsJSON)))
	mux.Handle("GET /_mod/export", middleware.RequireAdmin(modSvc,
		http.HandlerFunc(h.HandleAdminExportDID)))
	mux.Handle("POST /_mod/purge", cop.Handler(
		middleware.RequireAdmin(modSvc, http.HandlerFunc(h.HandleAdminPurgeDID))))
	mux.Handle("POST /_mod/rebuild", cop.Handler(
		middleware.RequireAdmin(modSvc, http.HandlerFunc(h.HandleAdminRebuildDID))))
	mux.Handle("POST /_mod/refresh-handles", cop.Handler(
		middleware.RequireAdmin(modSvc, http.HandlerFunc(h.HandleAdminRefreshHandles))))
	mux.Handle("GET /_mod/pds-records", middleware.RequireModerator(modSvc,
		http.HandlerFunc(h.HandleAdminFetchPDSRecords)))

	// CSS bundle + JS assets: serve from in-memory caches at specific paths
	// so the catch-all FileServer below never sees these requests. The URLs
	// are what HrefFor / JSHrefFor return to the templ layout helper.
	if cfg.CSSBundle != nil {
		mux.Handle("GET "+cfg.CSSBundle.URLPath(), cfg.CSSBundle.Handler())
	}
	if cfg.JSAssets != nil {
		mux.Handle("GET /static/js/{name}", cfg.JSAssets.Handler())
	}

	// Static files (must come after specific routes)
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))
	// Serve favicon.ico for pdsls
	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		http.ServeFile(w, r, "static/favicon.ico")
	})

	// SvelteKit SPA assets (_app/immutable/**) — versioned JS chunks from
	// the SvelteKit build. Served from the embedded build filesystem, or
	// from disk when SPAAssetDevDir is set so dev-mode `vite build --watch`
	// output appears on the next refresh without a Go restart.
	assetHandler := spa.AssetHandler()
	if cfg.SPAAssetDevDir != "" {
		assetHandler = spa.AssetHandlerWithDevDir(cfg.SPAAssetDevDir)
	}
	mux.Handle("GET /_app/", assetHandler)

	// Catch-all 404 handler. The SPA is registered only for explicit page
	// patterns above, so unknown direct loads remain real 404 responses.
	mux.HandleFunc("/", h.HandleNotFound)

	// Apply middleware in order (outermost first, innermost last)
	var handler http.Handler = mux

	// 1. Limit request body size (innermost - runs first on request)
	handler = middleware.LimitBodyMiddleware(handler)

	// 2. Add authenticated user attributes to the active HTTP span. This must
	// sit inside CookieAuth so the request context already contains the DID.
	handler = middleware.UserDIDSpanMiddleware(handler)

	// 3. Apply OAuth middleware to add auth context
	if cfg.OAuthApp != nil {
		didCookieName, sessCookieName := handlers.CookieNames(cfg.App)
		handler = atpmiddleware.CookieAuth(atpmiddleware.CookieAuthConfig{
			OAuthApp:       cfg.OAuthApp,
			DIDCookieName:  didCookieName,
			SessCookieName: sessCookieName,
			OnAuth:         cfg.OnAuth,
		})(handler)
	}

	// 4. Apply rate limiting
	if !cfg.DisableRateLimit {
		rateLimitConfig := middleware.NewDefaultRateLimitConfig()
		handler = middleware.RateLimitMiddleware(rateLimitConfig)(handler)
	}

	// 5. Apply security headers
	handler = middleware.SecurityHeadersMiddleware(handler)

	// 6. Apply logging middleware
	handler = middleware.LoggingMiddleware(cfg.Logger, metrics.HTTPRequestObserver{})(handler)

	// 7. Inject trace_id into zerolog context (runs after otelhttp creates the span)
	handler = middleware.RequestIDMiddleware(cfg.Logger)(handler)

	// 8. Enrich trace spans with client page context (runs inside otelhttp span)
	handler = pageContextMiddleware(handler)

	// 9. Apply OpenTelemetry HTTP instrumentation (outermost - wraps everything)
	spanName := "arabica"
	if cfg.App != nil && cfg.App.Name != "" {
		spanName = cfg.App.Name
	}
	handler = otelhttp.NewHandler(handler, spanName,
		otelhttp.WithFilter(func(r *http.Request) bool {
			return !strings.HasPrefix(r.URL.Path, "/static/") && r.URL.Path != "/favicon.ico"
		}),
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method + " " + r.URL.Path
		}),
	)

	return handler
}

func handleHealthz(h *handlers.Handler, consumer *firehose.Consumer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := "ok"
		httpStatus := http.StatusOK

		// Check firehose connection
		firehoseCheck := map[string]any{"connected": false}
		if consumer != nil {
			connected := consumer.IsConnected()
			firehoseCheck["connected"] = connected
			if !connected {
				status = "degraded"
			}
		}

		// Check SQLite feed index
		feedIndexCheck := map[string]any{"healthy": false, "ready": false}
		if idx := h.FeedIndex(); idx != nil {
			feedIndexCheck["ready"] = idx.IsReady()
			// feedIndexCheck["explore"] = idx.ExploreHealth(r.Context())
			if err := idx.DB().PingContext(r.Context()); err != nil {
				feedIndexCheck["healthy"] = false
				status = "error"
				httpStatus = http.StatusServiceUnavailable
			} else {
				feedIndexCheck["healthy"] = true
			}
		}

		resp := map[string]any{
			"status":     status,
			"firehose":   firehoseCheck,
			"feed_index": feedIndexCheck,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(httpStatus)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// RegisterEntityRoutes wires the per-entity public routes — view page, OG
// image, JSON CRUD, and modal partials — for every bundle whose
// RecordType has a matching descriptor on app.
//
// The app entity route path becomes the URL segment (e.g., "beans"); the
// app entity route noun becomes the modal path segment (e.g., "bean"). A
// nil handler in a bundle field skips the corresponding route, letting
// future entities omit (say) modal partials without forcing every app
// to publish stubs.
func RegisterEntityRoutes(mux *http.ServeMux, cop *http.CrossOriginProtection, app *domain.App, bundles []handlers.EntityRouteBundle, pages PageRoutes) {
	for _, b := range bundles {
		if app.DescriptorByType(b.RecordType) == nil {
			// Bundle declared a route for an entity this app doesn't run.
			// Skip silently — supports app-specific entity subsets.
			continue
		}
		route, ok := app.EntityRouteByType(b.RecordType)
		if !ok || route.Path == "" {
			continue
		}

		urlPath := route.Path
		// Entity view and backlinks page routes are SPA-owned: the SvelteKit
		// shell serves them directly. Register them through pages.Register so
		// a non-SPA build (e.g. tests without a shell handler) falls back to
		// the 404 handler instead of a nil panic.
		notFound := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "Not Found", http.StatusNotFound)
		})
		pages.Register(mux, "GET /"+urlPath+"/{actor}/{id}", notFound)
		pages.Register(mux, "GET /"+urlPath+"/{actor}/{id}/backlinks", notFound)
		if b.JSONView != nil {
			mux.HandleFunc("GET /api/"+urlPath+"/{actor}/{id}", RewriteActorToOwner(b.JSONView))
		}
		if b.JSONBacklinks != nil {
			mux.HandleFunc("GET /api/"+urlPath+"/{actor}/{id}/backlinks", RewriteActorToOwner(b.JSONBacklinks))
		}
		if b.OGImage != nil {
			mux.HandleFunc("GET /"+urlPath+"/{actor}/{id}/og-image", RewriteActorToOwner(b.OGImage))
		}
		if b.Create != nil {
			mux.Handle("POST /api/"+urlPath, cop.Handler(b.Create))
		}
		if b.Update != nil {
			mux.Handle("PUT /api/"+urlPath+"/{id}", cop.Handler(b.Update))
		}
		if b.Delete != nil {
			mux.Handle("DELETE /api/"+urlPath+"/{id}", cop.Handler(b.Delete))
		}
	}
}

// RewriteActorToOwner promotes the {actor} path segment to the ?owner= query
// param so the existing record view/og-image handlers (which key off ?owner=)
// can serve the new /{slug}/{actor}/{rkey} canonical route without changes.
// The actor segment may be either a did:* identifier or a handle; both are
// already accepted by the downstream resolver.
func RewriteActorToOwner(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if actor := r.PathValue("actor"); actor != "" {
			q := r.URL.Query()
			q.Set("owner", actor)
			r.URL.RawQuery = q.Encode()
		}
		h(w, r)
	}
}

// pageContextMiddleware reads the X-Page-Context header (set by client-side JS)
// and adds it as a span attribute so traces show which page triggered the request.
func pageContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if page := r.Header.Get("X-Page-Context"); page != "" {
			span := trace.SpanFromContext(r.Context())
			span.SetAttributes(attribute.String("http.page_context", page))
		}
		next.ServeHTTP(w, r)
	})
}
