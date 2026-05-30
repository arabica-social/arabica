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

func (Routes) RegisterAppRoutes(mux *http.ServeMux, ctx routing.AppRouteContext) {
	h := New(ctx.Handlers)
	cop := ctx.CSRF

	mux.HandleFunc("GET /api/data", h.HandleAPIListAll)

	mux.Handle("GET /api/brews", middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandleBrewListPartial)))
	mux.Handle("GET /api/manage", middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandleManagePartial)))
	mux.Handle("GET /api/incomplete-records", middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandleIncompleteRecordsPartial)))
	mux.Handle("GET /api/profile/{actor}", middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandleProfilePartial)))
	mux.Handle("GET /api/get-started-card", middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandleGetStartedCard)))
	mux.Handle("GET /api/onboarding/station-form/{kind}", middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandleOnboardingStationForm)))
	mux.Handle("GET /api/popular-recipes", middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandlePopularRecipesPartial)))
	mux.Handle("POST /api/manage/refresh", cop.Handler(http.HandlerFunc(h.HandleManageRefresh)))

	mux.HandleFunc("GET /onboarding", h.HandleOnboarding)
	mux.HandleFunc("GET /add", h.HandleAddRecords)
	mux.HandleFunc("GET /my-coffee", h.HandleMyCoffee)
	mux.HandleFunc("GET /explore", h.HandleExplore)
	mux.HandleFunc("GET /manage", h.HandleManage)
	mux.HandleFunc("GET /brews", h.HandleBrewList)
	mux.HandleFunc("GET /brews/new", h.HandleBrewNew)
	mux.HandleFunc("GET /brews/{id}/edit", h.HandleBrewEdit)
	mux.HandleFunc("GET /brews/{actor}/{id}/og-image", routing.RewriteActorToOwner(h.HandleBrewOGImage))
	mux.HandleFunc("GET /brews/{actor}/{id}", routing.RewriteActorToOwner(h.HandleBrewView))
	mux.Handle("POST /brews", cop.Handler(http.HandlerFunc(h.HandleBrewCreate)))
	mux.Handle("PUT /brews/{id}", cop.Handler(http.HandlerFunc(h.HandleBrewUpdate)))
	mux.Handle("DELETE /brews/{id}", cop.Handler(http.HandlerFunc(h.HandleBrewDelete)))
	mux.HandleFunc("GET /brews/export", h.HandleBrewExport)

	mux.HandleFunc("GET /recipes", h.HandleRecipeExplore)
	mux.HandleFunc("GET /recipes/{actor}/{id}/og-image", routing.RewriteActorToOwner(h.HandleRecipeOGImage))
	mux.HandleFunc("GET /recipes/{actor}/{id}/backlinks", routing.RewriteActorToOwner(h.HandleRecipeBacklinks))
	mux.HandleFunc("GET /recipes/{actor}/{id}", routing.RewriteActorToOwner(h.HandleRecipeView))

	mux.HandleFunc("GET /api/recipes", h.HandleRecipeList)
	mux.HandleFunc("GET /api/recipes/suggestions", h.HandleRecipeSuggestions)
	mux.HandleFunc("GET /api/recipes/{id}", h.HandleRecipeGet)
	mux.Handle("POST /api/recipes", cop.Handler(http.HandlerFunc(h.HandleRecipeCreate)))
	mux.Handle("PUT /api/recipes/{id}", cop.Handler(http.HandlerFunc(h.HandleRecipeUpdate)))
	mux.Handle("DELETE /api/recipes/{id}", cop.Handler(http.HandlerFunc(h.HandleRecipeDelete)))
	mux.Handle("POST /api/recipes/from-brew/{id}", cop.Handler(http.HandlerFunc(h.HandleRecipeCreateFromBrew)))
	mux.Handle("POST /api/recipes/fork/{id}", cop.Handler(http.HandlerFunc(h.HandleRecipeFork)))

	mux.HandleFunc("GET /api/modals/recipe/new", h.HandleRecipeModalNew)
	mux.HandleFunc("GET /api/modals/recipe/{id}", h.HandleRecipeModalEdit)

	routing.RegisterEntityRoutes(mux, cop, ctx.App, h.EntityRouteBundles())
	mux.HandleFunc("GET /profile/{actor}", h.HandleProfile)
}

// EntityRouteBundles returns the per-entity handler bundles for arabica's
// simple entities (bean, roaster, grinder, brewer). Recipe and brew have
// additional endpoints and stay registered explicitly in routing.go.
func (h *Handlers) EntityRouteBundles() []handlers.EntityRouteBundle {
	return []handlers.EntityRouteBundle{
		{
			RecordType: lexicons.RecordTypeBean,
			Create:     h.HandleBeanCreate,
			Update:     h.HandleBeanUpdate,
			Delete:     h.HandleBeanDelete,
			View:       h.HandleBeanView,
			Backlinks:  h.HandleBeanBacklinks,
			OGImage:    h.HandleBeanOGImage,
			ModalNew:   h.HandleBeanModalNew,
			ModalEdit:  h.HandleBeanModalEdit,
		},
		{
			RecordType: lexicons.RecordTypeRoaster,
			Create:     h.HandleRoasterCreate,
			Update:     h.HandleRoasterUpdate,
			Delete:     h.HandleRoasterDelete,
			View:       h.HandleRoasterView,
			Backlinks:  h.HandleRoasterBacklinks,
			OGImage:    h.HandleRoasterOGImage,
			ModalNew:   h.HandleRoasterModalNew,
			ModalEdit:  h.HandleRoasterModalEdit,
		},
		{
			RecordType: lexicons.RecordTypeGrinder,
			Create:     h.HandleGrinderCreate,
			Update:     h.HandleGrinderUpdate,
			Delete:     h.HandleGrinderDelete,
			View:       h.HandleGrinderView,
			Backlinks:  h.HandleGrinderBacklinks,
			OGImage:    h.HandleGrinderOGImage,
			ModalNew:   h.HandleGrinderModalNew,
			ModalEdit:  h.HandleGrinderModalEdit,
		},
		{
			RecordType: lexicons.RecordTypeBrewer,
			Create:     h.HandleBrewerCreate,
			Update:     h.HandleBrewerUpdate,
			Delete:     h.HandleBrewerDelete,
			View:       h.HandleBrewerView,
			Backlinks:  h.HandleBrewerBacklinks,
			OGImage:    h.HandleBrewerOGImage,
			ModalNew:   h.HandleBrewerModalNew,
			ModalEdit:  h.HandleBrewerModalEdit,
		},
	}
}
