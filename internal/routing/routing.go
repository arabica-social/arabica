package routing

import (
	"encoding/json"
	"net/http"
	"strings"

	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/firehose"
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/middleware"
	"tangled.org/arabica.social/arabica/internal/moderation"
	"tangled.org/arabica.social/arabica/internal/web/assets"
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
	mux.Handle("POST /reauth", cop.Handler(http.HandlerFunc(h.HandleReauth)))
	mux.HandleFunc("GET /.well-known/oauth-client-metadata.json", h.HandleClientMetadata)
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
	mux.Handle("GET /api/incomplete-records", middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandleIncompleteRecordsPartial)))
	mux.Handle("GET /api/popular-recipes", middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandlePopularRecipesPartial)))
	mux.Handle("POST /api/manage/refresh", cop.Handler(http.HandlerFunc(h.HandleManageRefresh)))
	mux.Handle("GET /api/profile/{actor}", middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandleProfilePartial)))

	// Page routes (must come before static files)
	mux.HandleFunc("GET /{$}", h.HandleHome) // {$} means exact match
	mux.HandleFunc("GET /og-image", h.HandleSiteOGImage)
	mux.HandleFunc("GET /about", h.HandleAbout)
	mux.HandleFunc("GET /terms", h.HandleTerms)
	mux.HandleFunc("GET /join", h.HandleJoin)
	mux.Handle("POST /join", cop.Handler(http.HandlerFunc(h.HandleJoinSubmit)))
	mux.HandleFunc("GET /join/create", h.HandleCreateAccount)
	mux.Handle("POST /join/create", cop.Handler(http.HandlerFunc(h.HandleCreateAccountSubmit)))
	mux.HandleFunc("GET /atproto", h.HandleATProto)
	mux.HandleFunc("GET /my-coffee", h.HandleMyCoffee)
	mux.HandleFunc("GET /manage", h.HandleManage)
	mux.HandleFunc("GET /brews", h.HandleBrewList)
	mux.HandleFunc("GET /brews/new", h.HandleBrewNew)
	// Brew is registered explicitly: edit page and export endpoint don't
	// fit the simple-entity route shape.
	mux.HandleFunc("GET /brews/{id}/og-image", h.HandleBrewOGImage)
	mux.HandleFunc("GET /brews/{id}", h.HandleBrewView)
	mux.HandleFunc("GET /brews/{id}/edit", h.HandleBrewEdit)
	mux.Handle("POST /brews", cop.Handler(http.HandlerFunc(h.HandleBrewCreate)))
	mux.Handle("PUT /brews/{id}", cop.Handler(http.HandlerFunc(h.HandleBrewUpdate)))
	mux.Handle("DELETE /brews/{id}", cop.Handler(http.HandlerFunc(h.HandleBrewDelete)))
	mux.HandleFunc("GET /brews/export", h.HandleBrewExport)

	// Recipe view + OG image are registered here; the API CRUD ops below
	// have additional endpoints (from-brew, fork) that don't fit the
	// simple-entity bundle.
	mux.HandleFunc("GET /recipes", h.HandleRecipeExplore)
	mux.HandleFunc("GET /recipes/{id}/og-image", h.HandleRecipeOGImage)
	mux.HandleFunc("GET /recipes/{id}", h.HandleRecipeView)

	// Per-entity routes for the simple entities (bean, roaster, grinder,
	// brewer): view pages, OG images, JSON CRUD, modal partials. Driven
	// by the route bundles + the App's descriptor list so a sister app
	// (oolong) reuses the loop without duplicating wiring.
	registerEntityRoutes(mux, cop, cfg.App, h.EntityRouteBundles())

	mux.HandleFunc("GET /api/recipes", h.HandleRecipeList)
	mux.HandleFunc("GET /api/recipes/suggestions", h.HandleRecipeSuggestions)
	mux.HandleFunc("GET /api/recipes/{id}", h.HandleRecipeGet)
	mux.Handle("POST /api/recipes", cop.Handler(http.HandlerFunc(h.HandleRecipeCreate)))
	mux.Handle("PUT /api/recipes/{id}", cop.Handler(http.HandlerFunc(h.HandleRecipeUpdate)))
	mux.Handle("DELETE /api/recipes/{id}", cop.Handler(http.HandlerFunc(h.HandleRecipeDelete)))
	mux.Handle("POST /api/recipes/from-brew/{id}", cop.Handler(http.HandlerFunc(h.HandleRecipeCreateFromBrew)))
	mux.Handle("POST /api/recipes/fork/{id}", cop.Handler(http.HandlerFunc(h.HandleRecipeFork)))

	mux.Handle("POST /api/likes/toggle", cop.Handler(http.HandlerFunc(h.HandleLikeToggle)))
	mux.Handle("POST /api/report", cop.Handler(http.HandlerFunc(h.HandleReport)))

	// Comment routes
	mux.Handle("GET /api/comments", middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandleCommentList)))
	mux.Handle("POST /api/comments", cop.Handler(http.HandlerFunc(h.HandleCommentCreate)))
	mux.Handle("DELETE /api/comments/{id}", cop.Handler(http.HandlerFunc(h.HandleCommentDelete)))

	// Recipe modal stays explicit (no JSON CRUD bundle).
	mux.HandleFunc("GET /api/modals/recipe/new", h.HandleRecipeModalNew)
	mux.HandleFunc("GET /api/modals/recipe/{id}", h.HandleRecipeModalEdit)

	// Notification routes
	mux.HandleFunc("GET /notifications", h.HandleNotifications)
	mux.Handle("POST /api/notifications/read", cop.Handler(http.HandlerFunc(h.HandleNotificationsMarkRead)))

	// Settings
	mux.HandleFunc("GET /settings", h.HandleSettings)
	mux.Handle("POST /api/settings/profile-visibility", cop.Handler(http.HandlerFunc(h.HandleSettingsProfileVisibility)))

	// Profile routes (public user profiles)
	mux.HandleFunc("GET /profile/{actor}", h.HandleProfile)

	// Moderation routes
	// HandleAdmin keeps its own auth check (redirects to / instead of 401)
	modSvc := cfg.ModerationService
	mux.HandleFunc("GET /_mod", h.HandleAdmin)
	mux.Handle("GET /_mod/content", middleware.RequireModerator(modSvc,
		middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandleAdminPartial))))
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
	mux.Handle("POST /_mod/invite", cop.Handler(
		middleware.RequireAdmin(modSvc, http.HandlerFunc(h.HandleCreateInvite))))
	mux.Handle("POST /_mod/dismiss-join", cop.Handler(
		middleware.RequireAdmin(modSvc, http.HandlerFunc(h.HandleDismissJoinRequest))))
	mux.Handle("GET /_mod/stats", middleware.RequireAdmin(modSvc,
		middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandleAdminStats))))
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

	// Catch-all 404 handler - must be last, catches any unmatched routes
	mux.HandleFunc("/", h.HandleNotFound)

	// Apply middleware in order (outermost first, innermost last)
	var handler http.Handler = mux

	// 1. Limit request body size (innermost - runs first on request)
	handler = middleware.LimitBodyMiddleware(handler)

	// 2. Apply OAuth middleware to add auth context
	if cfg.OAuthApp != nil {
		handler = atpmiddleware.CookieAuth(atpmiddleware.CookieAuthConfig{
			OAuthApp: cfg.OAuthApp,
			OnAuth:   cfg.OnAuth,
		})(handler)
	}

	// 3. Apply rate limiting
	rateLimitConfig := middleware.NewDefaultRateLimitConfig()
	handler = middleware.RateLimitMiddleware(rateLimitConfig)(handler)

	// 4. Apply security headers
	handler = middleware.SecurityHeadersMiddleware(handler)

	// 5. Apply logging middleware
	handler = middleware.LoggingMiddleware(cfg.Logger)(handler)

	// 6. Inject trace_id into zerolog context (runs after otelhttp creates the span)
	handler = middleware.RequestIDMiddleware(cfg.Logger)(handler)

	// 7. Enrich trace spans with client page context (runs inside otelhttp span)
	handler = pageContextMiddleware(handler)

	// 8. Apply OpenTelemetry HTTP instrumentation (outermost - wraps everything)
	handler = otelhttp.NewHandler(handler, "arabica",
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
		json.NewEncoder(w).Encode(resp)
	}
}

// registerEntityRoutes wires the per-entity public routes — view page, OG
// image, JSON CRUD, and modal partials — for every bundle whose
// RecordType has a matching descriptor on app.
//
// The descriptor's URLPath becomes the URL segment (e.g., "beans"); the
// descriptor's Noun becomes the modal path segment (e.g., "bean"). A
// nil handler in a bundle field skips the corresponding route, letting
// future entities omit (say) modal partials without forcing every app
// to publish stubs.
func registerEntityRoutes(mux *http.ServeMux, cop *http.CrossOriginProtection, app *domain.App, bundles []handlers.EntityRouteBundle) {
	for _, b := range bundles {
		desc := app.DescriptorByType(b.RecordType)
		if desc == nil {
			// Bundle declared a route for an entity this app doesn't run.
			// Skip silently — supports app-specific entity subsets.
			continue
		}

		urlPath := desc.URLPath
		if b.View != nil {
			mux.HandleFunc("GET /"+urlPath+"/{id}", b.View)
		}
		if b.OGImage != nil {
			mux.HandleFunc("GET /"+urlPath+"/{id}/og-image", b.OGImage)
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
		if b.ModalNew != nil {
			mux.HandleFunc("GET /api/modals/"+desc.Noun+"/new", b.ModalNew)
		}
		if b.ModalEdit != nil {
			mux.HandleFunc("GET /api/modals/"+desc.Noun+"/{id}", b.ModalEdit)
		}
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
