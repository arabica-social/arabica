package routing

import (
	"net/http"

	coffeehandlers "tangled.org/arabica.social/arabica/internal/arabica/handlers"
	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/middleware"
	teahandlers "tangled.org/arabica.social/arabica/internal/oolong/handlers"
)

func registerAppDataRoute(mux *http.ServeMux, app *domain.App, coffee *coffeehandlers.Handlers, tea *teahandlers.Handlers) {
	if app.Name == "oolong" {
		mux.HandleFunc("GET /api/data", tea.HandleOolongAPIListAll)
		return
	}
	mux.HandleFunc("GET /api/data", coffee.HandleAPIListAll)
}

func registerAppPartials(mux *http.ServeMux, cop *http.CrossOriginProtection, app *domain.App, coffee *coffeehandlers.Handlers, tea *teahandlers.Handlers) {
	switch app.Name {
	case "arabica":
		mux.Handle("GET /api/brews", middleware.RequireHTMXMiddleware(http.HandlerFunc(coffee.HandleBrewListPartial)))
		mux.Handle("GET /api/manage", middleware.RequireHTMXMiddleware(http.HandlerFunc(coffee.HandleManagePartial)))
		mux.Handle("GET /api/incomplete-records", middleware.RequireHTMXMiddleware(http.HandlerFunc(coffee.HandleIncompleteRecordsPartial)))
		mux.Handle("GET /api/get-started-card", middleware.RequireHTMXMiddleware(http.HandlerFunc(coffee.HandleGetStartedCard)))
		mux.Handle("GET /api/onboarding/station-form/{kind}", middleware.RequireHTMXMiddleware(http.HandlerFunc(coffee.HandleOnboardingStationForm)))
		mux.Handle("GET /api/popular-recipes", middleware.RequireHTMXMiddleware(http.HandlerFunc(coffee.HandlePopularRecipesPartial)))
		mux.Handle("POST /api/manage/refresh", cop.Handler(http.HandlerFunc(coffee.HandleManageRefresh)))
	case "oolong":
		mux.Handle("GET /api/get-started-card", middleware.RequireHTMXMiddleware(http.HandlerFunc(tea.HandleOolongGetStartedCard)))
		mux.Handle("GET /api/onboarding/station-form/{kind}", middleware.RequireHTMXMiddleware(http.HandlerFunc(tea.HandleOolongOnboardingStationForm)))
	}
}

func registerAppPages(mux *http.ServeMux, cop *http.CrossOriginProtection, app *domain.App, coffee *coffeehandlers.Handlers, tea *teahandlers.Handlers) {
	switch app.Name {
	case "arabica":
		registerArabicaPages(mux, cop, coffee)
	case "oolong":
		registerOolongPages(mux, cop, tea)
	}
}

func registerArabicaPages(mux *http.ServeMux, cop *http.CrossOriginProtection, coffee *coffeehandlers.Handlers) {
	mux.HandleFunc("GET /onboarding", coffee.HandleOnboarding)
	mux.HandleFunc("GET /add", coffee.HandleAddRecords)
	mux.HandleFunc("GET /my-coffee", coffee.HandleMyCoffee)
	mux.HandleFunc("GET /manage", coffee.HandleManage)
	mux.HandleFunc("GET /brews", coffee.HandleBrewList)
	mux.HandleFunc("GET /brews/new", coffee.HandleBrewNew)
	mux.HandleFunc("GET /brews/{id}/edit", coffee.HandleBrewEdit)
	mux.HandleFunc("GET /brews/{actor}/{id}/og-image", rewriteActorToOwner(coffee.HandleBrewOGImage))
	mux.HandleFunc("GET /brews/{actor}/{id}", rewriteActorToOwner(coffee.HandleBrewView))
	mux.Handle("POST /brews", cop.Handler(http.HandlerFunc(coffee.HandleBrewCreate)))
	mux.Handle("PUT /brews/{id}", cop.Handler(http.HandlerFunc(coffee.HandleBrewUpdate)))
	mux.Handle("DELETE /brews/{id}", cop.Handler(http.HandlerFunc(coffee.HandleBrewDelete)))
	mux.HandleFunc("GET /brews/export", coffee.HandleBrewExport)

	mux.HandleFunc("GET /recipes", coffee.HandleRecipeExplore)
	mux.HandleFunc("GET /recipes/{actor}/{id}/og-image", rewriteActorToOwner(coffee.HandleRecipeOGImage))
	mux.HandleFunc("GET /recipes/{actor}/{id}", rewriteActorToOwner(coffee.HandleRecipeView))
	mux.HandleFunc("GET /api/recipes", coffee.HandleRecipeList)
	mux.HandleFunc("GET /api/recipes/suggestions", coffee.HandleRecipeSuggestions)
	mux.HandleFunc("GET /api/recipes/{id}", coffee.HandleRecipeGet)
	mux.Handle("POST /api/recipes", cop.Handler(http.HandlerFunc(coffee.HandleRecipeCreate)))
	mux.Handle("PUT /api/recipes/{id}", cop.Handler(http.HandlerFunc(coffee.HandleRecipeUpdate)))
	mux.Handle("DELETE /api/recipes/{id}", cop.Handler(http.HandlerFunc(coffee.HandleRecipeDelete)))
	mux.Handle("POST /api/recipes/from-brew/{id}", cop.Handler(http.HandlerFunc(coffee.HandleRecipeCreateFromBrew)))
	mux.Handle("POST /api/recipes/fork/{id}", cop.Handler(http.HandlerFunc(coffee.HandleRecipeFork)))
	mux.HandleFunc("GET /api/modals/recipe/new", coffee.HandleRecipeModalNew)
	mux.HandleFunc("GET /api/modals/recipe/{id}", coffee.HandleRecipeModalEdit)
}

func registerOolongPages(mux *http.ServeMux, cop *http.CrossOriginProtection, tea *teahandlers.Handlers) {
	mux.HandleFunc("GET /onboarding", tea.HandleOolongOnboarding)
	mux.HandleFunc("GET /my-tea", tea.HandleMyTea)
	mux.Handle("POST /api/tea/refresh", cop.Handler(http.HandlerFunc(tea.HandleTeaRefresh)))
	mux.HandleFunc("GET /brews/new", tea.HandleOolongSteepNew)
	mux.HandleFunc("GET /brews/{id}/edit", tea.HandleOolongSteepEdit)
	mux.HandleFunc("GET /teas/new", tea.HandleOolongTeaNew)
	mux.HandleFunc("GET /teas/{id}/edit", tea.HandleOolongTeaEdit)
}

func registerProfileRoute(mux *http.ServeMux, app *domain.App, h *handlers.Handler, tea *teahandlers.Handlers) {
	if app.Name == "oolong" {
		mux.HandleFunc("GET /profile/{actor}", tea.HandleOolongProfile)
		return
	}
	mux.HandleFunc("GET /profile/{actor}", h.HandleProfile)
}
