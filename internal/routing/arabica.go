package routing

import (
	"net/http"

	arabicaai "tangled.org/arabica.social/arabica/internal/arabica/ai"

	arabicaapp "tangled.org/arabica.social/arabica/internal/arabica/app"
	"tangled.org/arabica.social/arabica/internal/handlers"
)

// SPAOwnedRoutes lists Arabica page routes with SvelteKit direct-load support.
func SPAOwnedRoutes() []string {
	return []string{
		"GET /{$}",
		"GET /about",
		"GET /chat",
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

func registerArabicaRoutes(mux *http.ServeMux, cop *http.CrossOriginProtection, app *arabicaapp.App, h *handlers.Handler, pages pageRoutes, ai *arabicaai.Service) {
	aiHandlers := handlers.AIHandlers{Handler: h, Service: ai}
	mux.HandleFunc("GET /api/ai/status", aiHandlers.HandleStatus)
	mux.Handle("POST /api/ai/chat", aiHandlers.RequireAccess(cop.Handler(http.HandlerFunc(aiHandlers.HandleChat))))
	mux.HandleFunc("GET /api/_mod/ai-access", h.HandleAIAccess)
	mux.Handle("PUT /api/_mod/ai-access", cop.Handler(http.HandlerFunc(h.HandleAIAccess)))
	mux.Handle("DELETE /api/_mod/ai-access", cop.Handler(http.HandlerFunc(h.HandleAIAccess)))
	var chatPage http.Handler = http.HandlerFunc(h.HandleNotFound)
	if pages.IsSPA("GET /chat") {
		chatPage = pages.spaHandler
	}
	mux.Handle("GET /chat", aiHandlers.RequirePageAccess(chatPage))

	mux.HandleFunc("GET /api/data", h.HandleAPIListAll)
	mux.HandleFunc("GET /api/brews", h.HandleBrewList)
	mux.HandleFunc("GET /api/manage", h.HandleManageAPI)
	mux.HandleFunc("GET /api/incomplete-records", h.HandleIncompleteRecordsJSON)
	mux.HandleFunc("GET /api/profile/{actor}", h.HandleProfileAPI)
	mux.HandleFunc("GET /api/onboarding", h.HandleOnboardingJSON)
	mux.HandleFunc("GET /api/popular-recipes", h.HandlePopularRecipesPartial)
	mux.HandleFunc("GET /api/explore", h.HandleExploreJSON)
	mux.Handle("POST /api/manage/refresh", cop.Handler(http.HandlerFunc(h.HandleManageRefresh)))

	// Brew CRUD + OG image.
	mux.HandleFunc("GET /brews/{actor}/{id}/og-image", rewriteActorToOwner(h.HandleBrewOGImage))
	mux.HandleFunc("GET /api/brews/{actor}/{id}", rewriteActorToOwner(h.HandleBrewViewJSON))
	mux.Handle("POST /brews", cop.Handler(http.HandlerFunc(h.HandleBrewCreate)))
	mux.Handle("PUT /brews/{id}", cop.Handler(http.HandlerFunc(h.HandleBrewUpdate)))
	mux.Handle("DELETE /brews/{id}", cop.Handler(http.HandlerFunc(h.HandleBrewDelete)))
	mux.HandleFunc("GET /brews/export", h.HandleBrewExport)

	// Recipe CRUD + OG image + JSON backlinks.
	mux.HandleFunc("GET /recipes/{actor}/{id}/og-image", rewriteActorToOwner(h.HandleRecipeOGImage))
	mux.HandleFunc("GET /api/recipes/{actor}/{id}/backlinks", rewriteActorToOwner(h.HandleRecipeBacklinksJSON))
	mux.HandleFunc("GET /api/recipes/{actor}/{id}", rewriteActorToOwner(h.HandleRecipeViewJSON))
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
	registerEntityRoutes(mux, cop, app, entityRouteBundles(h), pages)

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
		pages.Register(mux, pattern, notFound)
	}
}
