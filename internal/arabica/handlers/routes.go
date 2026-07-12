package coffeehandlers

import (
	"net/http"

	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/lexicons"
	"tangled.org/arabica.social/arabica/internal/middleware"
	"tangled.org/arabica.social/arabica/internal/routing"
)

// Routes owns Arabica-specific HTTP route registration. Keeping this in the
// app package prevents the shared router from importing coffee handlers.
type Routes struct{}

// SPAOwnedRoutes is the explicit Arabica page-cutover inventory. A route is
// added only after its SvelteKit direct-load path exists; unlisted routes keep
// their legacy handlers during migration.
func (Routes) SPAOwnedRoutes() []string {
	return []string{
		"GET /{$}",
		"GET /about",
		"GET /manage", // redirects to /my-coffee via SPA
		"GET /brews",  // brew list page
		"GET /terms",
		"GET /join/create",
		"GET /atproto",
		"GET /notifications",
		"GET /settings",
		"GET /_mod",
		"GET /onboarding",
		"GET /add",
		"GET /my-coffee",
		"GET /explore",
		"GET /brews/new",
		"GET /brews/{id}/edit",
		"GET /beans/new",
		"GET /beans/{id}/edit",
		"GET /grinders/new",
		"GET /grinders/{id}/edit",
		"GET /brewers/new",
		"GET /brewers/{id}/edit",
		"GET /recipes",
		"GET /recipes/new",
		"GET /recipes/{id}/edit",
		"GET /brews/{actor}/{id}",
		"GET /recipes/{actor}/{id}",
		"GET /profile/{actor}",
		"GET /beans/{actor}/{id}",
		"GET /beans/{actor}/{id}/backlinks",
		"GET /roasters/{actor}/{id}",
		"GET /roasters/{actor}/{id}/backlinks",
		"GET /roasters/new",
		"GET /roasters/{id}/edit",
		"GET /grinders/{actor}/{id}",
		"GET /grinders/{actor}/{id}/backlinks",
		"GET /brewers/{actor}/{id}",
		"GET /brewers/{actor}/{id}/backlinks",
		"GET /recipes/{actor}/{id}/backlinks",
	}
}

func (Routes) RegisterAppRoutes(mux *http.ServeMux, ctx routing.AppRouteContext) {
	h := New(ctx.Handlers)
	cop := ctx.CSRF

	// API routes used by both the templ stack and the SvelteKit SPA.
	mux.HandleFunc("GET /api/data", h.HandleAPIListAll)
	mux.HandleFunc("GET /api/brews", h.HandleBrewList)
	mux.HandleFunc("GET /api/manage", h.HandleManageAPI)
	mux.HandleFunc("GET /api/incomplete-records", h.HandleIncompleteRecordsPartial)
	mux.HandleFunc("GET /api/profile/{actor}", h.HandleProfileAPI)
	mux.HandleFunc("GET /api/onboarding", h.HandleOnboardingJSON)
	mux.HandleFunc("GET /api/popular-recipes", h.HandlePopularRecipesPartial)
	mux.HandleFunc("GET /api/explore", h.HandleExploreJSON)
	mux.Handle("POST /api/manage/refresh", cop.Handler(http.HandlerFunc(h.HandleManageRefresh)))

	// Brew CRUD + OG image.
	mux.HandleFunc("GET /brews/{actor}/{id}/og-image", routing.RewriteActorToOwner(h.HandleBrewOGImage))
	mux.HandleFunc("GET /api/brews/{actor}/{id}", routing.RewriteActorToOwner(h.HandleBrewViewJSON))
	mux.Handle("POST /brews", cop.Handler(http.HandlerFunc(h.HandleBrewCreate)))
	mux.Handle("PUT /brews/{id}", cop.Handler(http.HandlerFunc(h.HandleBrewUpdate)))
	mux.Handle("DELETE /brews/{id}", cop.Handler(http.HandlerFunc(h.HandleBrewDelete)))
	mux.HandleFunc("GET /brews/export", h.HandleBrewExport)

	// Recipe CRUD + OG image + JSON backlinks.
	mux.HandleFunc("GET /recipes/{actor}/{id}/og-image", routing.RewriteActorToOwner(h.HandleRecipeOGImage))
	mux.HandleFunc("GET /api/recipes/{actor}/{id}/backlinks", routing.RewriteActorToOwner(h.HandleRecipeBacklinksJSON))
	mux.HandleFunc("GET /api/recipes/{actor}/{id}", routing.RewriteActorToOwner(h.HandleRecipeViewJSON))
	mux.HandleFunc("GET /api/recipes", h.HandleRecipeList)
	mux.HandleFunc("GET /api/recipes/suggestions", h.HandleRecipeSuggestions)
	mux.HandleFunc("GET /api/recipes/{id}", h.HandleRecipeGet)
	mux.Handle("POST /api/recipes", cop.Handler(http.HandlerFunc(h.HandleRecipeCreate)))
	mux.Handle("PUT /api/recipes/{id}", cop.Handler(http.HandlerFunc(h.HandleRecipeUpdate)))
	mux.Handle("DELETE /api/recipes/{id}", cop.Handler(http.HandlerFunc(h.HandleRecipeDelete)))
	mux.Handle("POST /api/recipes/from-brew/{id}", cop.Handler(http.HandlerFunc(h.HandleRecipeCreateFromBrew)))
	mux.Handle("POST /api/recipes/fork/{id}", cop.Handler(http.HandlerFunc(h.HandleRecipeFork)))

	// Entity page routes use explicit ownership; JSON, OG, mutation, and modal
	// routes remain independent of the frontend owner.
	routing.RegisterEntityRoutes(mux, cop, ctx.App, h.EntityRouteBundles(), ctx.Pages)

	// HTML/HTMX routes that have not been ported to SvelteKit yet. They are
	// registered in both modes so users always have a working fallback.
	mux.Handle("GET /api/get-started-card", middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandleGetStartedCard)))
	mux.Handle("GET /api/onboarding/station-form/{kind}", middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandleOnboardingStationForm)))

	ctx.Pages.Register(mux, "GET /manage", http.HandlerFunc(h.HandleManage))
	ctx.Pages.Register(mux, "GET /brews", http.HandlerFunc(h.HandleBrewList))

	ctx.Pages.Register(mux, "GET /recipes", http.HandlerFunc(h.HandleRecipeExplore))
	ctx.Pages.Register(mux, "GET /recipes/{actor}/{id}/backlinks", http.HandlerFunc(routing.RewriteActorToOwner(h.HandleRecipeBacklinks)))

	// Recipe modal partials remain for the legacy stack; the SvelteKit SPA
	// uses /recipes/new and /recipes/{id}/edit pages (SPAOwnedRoutes).
	mux.HandleFunc("GET /api/modals/recipe/new", h.HandleRecipeModalNew)
	mux.HandleFunc("GET /api/modals/recipe/{id}", h.HandleRecipeModalEdit)

	ctx.Pages.Register(mux, "GET /onboarding", http.HandlerFunc(h.HandleOnboarding))
	ctx.Pages.Register(mux, "GET /add", http.HandlerFunc(h.HandleAddRecords))
	ctx.Pages.Register(mux, "GET /my-coffee", http.HandlerFunc(h.HandleMyCoffee))
	ctx.Pages.Register(mux, "GET /explore", http.HandlerFunc(h.HandleExplore))
	ctx.Pages.Register(mux, "GET /brews/new", http.HandlerFunc(h.HandleBrewNew))
	ctx.Pages.Register(mux, "GET /brews/{id}/edit", http.HandlerFunc(h.HandleBrewEdit))
	ctx.Pages.Register(mux, "GET /brews/{actor}/{id}", http.HandlerFunc(routing.RewriteActorToOwner(h.HandleBrewView)))
	ctx.Pages.Register(mux, "GET /recipes/{actor}/{id}", http.HandlerFunc(routing.RewriteActorToOwner(h.HandleRecipeView)))
	ctx.Pages.Register(mux, "GET /profile/{actor}", http.HandlerFunc(h.HandleProfile))

	// Simple-entity create/edit pages (bean, roaster, grinder, brewer,
	// recipe) are SPA-owned (see SPAOwnedRoutes). They had no legacy
	// full-page handlers — the templ stack used modal partials for these —
	// so register them here purely to claim the SPA shell. pages.Register
	// routes SPA-owned patterns to the SPA handler regardless of the legacy
	// handler argument; the legacy arg is only a fallback for non-SPA mode,
	// which is not exercised for these routes. A nil-safe 404 handler is
	// passed as the legacy fallback so a non-SPA build still responds
	// predictably instead of panicking on a nil handler.
	notFound := http.HandlerFunc(h.HandleNotFound)
	for _, pattern := range []string{
		"GET /beans/new",
		"GET /beans/{id}/edit",
		"GET /roasters/new",
		"GET /roasters/{id}/edit",
		"GET /grinders/new",
		"GET /grinders/{id}/edit",
		"GET /brewers/new",
		"GET /brewers/{id}/edit",
		"GET /recipes/new",
		"GET /recipes/{id}/edit",
	} {
		ctx.Pages.Register(mux, pattern, notFound)
	}
}

// EntityRouteBundles returns the per-entity handler bundles for arabica's
// simple entities (bean, roaster, grinder, brewer). Recipe and brew have
// additional endpoints and stay registered explicitly in routing.go.
func (h *Handlers) EntityRouteBundles() []handlers.EntityRouteBundle {
	return []handlers.EntityRouteBundle{
		{
			RecordType:    lexicons.RecordTypeBean,
			Create:        h.HandleBeanCreate,
			Update:        h.HandleBeanUpdate,
			Delete:        h.HandleBeanDelete,
			View:          h.HandleBeanView,
			JSONView:      h.HandleBeanViewJSON,
			Backlinks:     h.HandleBeanBacklinks,
			JSONBacklinks: h.HandleBeanBacklinksJSON,
			OGImage:       h.HandleBeanOGImage,
			ModalNew:      h.HandleBeanModalNew,
			ModalEdit:     h.HandleBeanModalEdit,
		},
		{
			RecordType:    lexicons.RecordTypeRoaster,
			Create:        h.HandleRoasterCreate,
			Update:        h.HandleRoasterUpdate,
			Delete:        h.HandleRoasterDelete,
			View:          h.HandleRoasterView,
			JSONView:      h.HandleRoasterViewJSON,
			Backlinks:     h.HandleRoasterBacklinks,
			JSONBacklinks: h.HandleRoasterBacklinksJSON,
			OGImage:       h.HandleRoasterOGImage,
			ModalNew:      h.HandleRoasterModalNew,
			ModalEdit:     h.HandleRoasterModalEdit,
		},
		{
			RecordType:    lexicons.RecordTypeGrinder,
			Create:        h.HandleGrinderCreate,
			Update:        h.HandleGrinderUpdate,
			Delete:        h.HandleGrinderDelete,
			View:          h.HandleGrinderView,
			JSONView:      h.HandleGrinderViewJSON,
			Backlinks:     h.HandleGrinderBacklinks,
			JSONBacklinks: h.HandleGrinderBacklinksJSON,
			OGImage:       h.HandleGrinderOGImage,
			ModalNew:      h.HandleGrinderModalNew,
			ModalEdit:     h.HandleGrinderModalEdit,
		},
		{
			RecordType:    lexicons.RecordTypeBrewer,
			Create:        h.HandleBrewerCreate,
			Update:        h.HandleBrewerUpdate,
			Delete:        h.HandleBrewerDelete,
			View:          h.HandleBrewerView,
			JSONView:      h.HandleBrewerViewJSON,
			Backlinks:     h.HandleBrewerBacklinks,
			JSONBacklinks: h.HandleBrewerBacklinksJSON,
			OGImage:       h.HandleBrewerOGImage,
			ModalNew:      h.HandleBrewerModalNew,
			ModalEdit:     h.HandleBrewerModalEdit,
		},
	}
}
