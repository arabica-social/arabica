package teahandlers

import (
	"context"
	"net/http"

	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/lexicons"
	"tangled.org/arabica.social/arabica/internal/middleware"
	teapages "tangled.org/arabica.social/arabica/internal/oolong/web/pages"
	"tangled.org/arabica.social/arabica/internal/routing"
	"tangled.org/arabica.social/arabica/internal/web/components"
)

// Routes owns Oolong-specific HTTP route registration. Keeping this in the app
// package prevents the shared router from importing tea handlers.
type Routes struct{}

func StaticPages() handlers.StaticPageRenderers {
	return handlers.StaticPageRenderers{
		About: func(ctx context.Context, w http.ResponseWriter, data *components.LayoutData) error {
			return teapages.About(data).Render(ctx, w)
		},
		Terms: func(ctx context.Context, w http.ResponseWriter, data *components.LayoutData) error {
			return teapages.Terms(data).Render(ctx, w)
		},
	}
}

func (Routes) RegisterAppRoutes(mux *http.ServeMux, ctx routing.AppRouteContext) {
	h := New(ctx.Handlers)
	cop := ctx.CSRF

	mux.HandleFunc("GET /api/data", h.HandleOolongAPIListAll)
	mux.Handle("GET /api/get-started-card", middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandleOolongGetStartedCard)))
	mux.Handle("GET /api/onboarding/station-form/{kind}", middleware.RequireHTMXMiddleware(http.HandlerFunc(h.HandleOolongOnboardingStationForm)))

	mux.HandleFunc("GET /onboarding", h.HandleOolongOnboarding)
	mux.HandleFunc("GET /my-tea", h.HandleMyTea)
	mux.Handle("POST /api/tea/refresh", cop.Handler(http.HandlerFunc(h.HandleTeaRefresh)))
	mux.HandleFunc("GET /brews/new", h.HandleOolongSteepNew)
	mux.HandleFunc("GET /brews/{id}/edit", h.HandleOolongSteepEdit)
	mux.HandleFunc("GET /teas/new", h.HandleOolongTeaNew)
	mux.HandleFunc("GET /teas/{id}/edit", h.HandleOolongTeaEdit)

	routing.RegisterEntityRoutes(mux, cop, ctx.App, h.EntityRouteBundles())
	mux.HandleFunc("GET /profile/{actor}", h.HandleOolongProfile)
}

// EntityRouteBundles returns the per-entity handler bundles for oolong's
// public surface. Mirrors the shared
// (*handlers.Handler).EntityRouteBundles contract; see entity_routes.go
// for the per-field semantics.
//
// Tea uses a full-page form (/teas/new + /teas/{id}/edit) rather than a
// modal partial, so its ModalNew/ModalEdit slots stay nil — the route
// registrar skips nil slots.
func (h *Handlers) EntityRouteBundles() []handlers.EntityRouteBundle {
	return []handlers.EntityRouteBundle{
		{
			RecordType: lexicons.RecordTypeOolongTea,
			Create:     h.HandleTeaCreate,
			Update:     h.HandleTeaUpdate,
			Delete:     h.HandleTeaDelete,
			View:       h.HandleTeaView,
		},
		{
			RecordType: lexicons.RecordTypeOolongVendor,
			Create:     h.HandleOolongVendorCreate,
			Update:     h.HandleOolongVendorUpdate,
			Delete:     h.HandleOolongVendorDelete,
			View:       h.HandleOolongVendorView,
			ModalNew:   h.HandleOolongVendorModalNew,
			ModalEdit:  h.HandleOolongVendorModalEdit,
		},
		{
			RecordType: lexicons.RecordTypeOolongVessel,
			Create:     h.HandleOolongVesselCreate,
			Update:     h.HandleOolongVesselUpdate,
			Delete:     h.HandleOolongVesselDelete,
			View:       h.HandleOolongVesselView,
			ModalNew:   h.HandleOolongVesselModalNew,
			ModalEdit:  h.HandleOolongVesselModalEdit,
		},
		{
			RecordType: lexicons.RecordTypeOolongInfuser,
			Create:     h.HandleOolongInfuserCreate,
			Update:     h.HandleOolongInfuserUpdate,
			Delete:     h.HandleOolongInfuserDelete,
			View:       h.HandleOolongInfuserView,
			ModalNew:   h.HandleOolongInfuserModalNew,
			ModalEdit:  h.HandleOolongInfuserModalEdit,
		},
		{
			RecordType: lexicons.RecordTypeOolongBrew,
			Create:     h.HandleOolongBrewCreate,
			Update:     h.HandleOolongBrewUpdate,
			Delete:     h.HandleOolongBrewDelete,
			View:       h.HandleOolongBrewView,
			ModalNew:   h.HandleOolongBrewModalNew,
			ModalEdit:  h.HandleOolongBrewModalEdit,
		},
	}
}
