package handlers

import (
	"net/http"

	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/lexicons"
	"tangled.org/arabica.social/arabica/internal/routing"
)

// Routes owns Arabica-specific HTTP route registration. Keeping this in the
// app package prevents the shared router from importing coffee handlers.
type Routes struct{}

// SPAOwnedRoutes lists Arabica page routes with SvelteKit direct-load support.
func (Routes) SPAOwnedRoutes() []string {
	return []string{
		"GET /{$}",
		"GET /about",
		"GET /feedback",
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
	// Feedback is an SPA-only page. It opens a mail draft locally rather than
	// storing operator correspondence in a user's PDS.
	ctx.Pages.Register(mux, "GET /feedback", http.HandlerFunc(h.HandleNotFound))

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

	// Keep the multipart routes above for compatibility with existing clients.
	mux.Handle("POST /api/brews", cop.Handler(http.HandlerFunc(h.HandleBrewCreateJSON)))
	mux.Handle("PUT /api/brews/{id}", cop.Handler(http.HandlerFunc(h.HandleBrewUpdateJSON)))

	// Entity page routes use explicit ownership; JSON, OG, mutation, and modal
	// routes remain independent of the frontend owner.
	routing.RegisterEntityRoutes(mux, cop, ctx.App, h.EntityRouteBundles(), ctx.Pages)

	// Preserve retired modal URLs as explicit 404s rather than SPA fallbacks.
	mux.HandleFunc("GET /api/modals/recipe/new", h.HandleNotFound)
	mux.HandleFunc("GET /api/modals/recipe/{id}", h.HandleNotFound)

	// Non-SPA builds return 404 for routes owned by the SPA.
	notFound := http.HandlerFunc(h.HandleNotFound)
	for _, pattern := range []string{
		"GET /manage",
		"GET /brews",
		"GET /recipes",
		"GET /recipes/{actor}/{id}/backlinks",
		"GET /onboarding",
		"GET /add",
		"GET /my-coffee",
		"GET /explore",
		"GET /brews/new",
		"GET /brews/{id}/edit",
		"GET /brews/{actor}/{id}",
		"GET /recipes/{actor}/{id}",
		"GET /profile/{actor}",
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
			JSONView:      h.HandleBeanViewJSON,
			JSONBacklinks: h.HandleBeanBacklinksJSON,
			OGImage:       h.HandleBeanOGImage,
		},
		{
			RecordType:    lexicons.RecordTypeRoaster,
			Create:        h.HandleRoasterCreate,
			Update:        h.HandleRoasterUpdate,
			Delete:        h.HandleRoasterDelete,
			JSONView:      h.HandleRoasterViewJSON,
			JSONBacklinks: h.HandleRoasterBacklinksJSON,
			OGImage:       h.HandleRoasterOGImage,
		},
		{
			RecordType:    lexicons.RecordTypeGrinder,
			Create:        h.HandleGrinderCreate,
			Update:        h.HandleGrinderUpdate,
			Delete:        h.HandleGrinderDelete,
			JSONView:      h.HandleGrinderViewJSON,
			JSONBacklinks: h.HandleGrinderBacklinksJSON,
			OGImage:       h.HandleGrinderOGImage,
		},
		{
			RecordType:    lexicons.RecordTypeBrewer,
			Create:        h.HandleBrewerCreate,
			Update:        h.HandleBrewerUpdate,
			Delete:        h.HandleBrewerDelete,
			JSONView:      h.HandleBrewerViewJSON,
			JSONBacklinks: h.HandleBrewerBacklinksJSON,
			OGImage:       h.HandleBrewerOGImage,
		},
	}
}
