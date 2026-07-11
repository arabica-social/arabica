package routing

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	arabicaapp "tangled.org/arabica.social/arabica/internal/arabica/app"
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/lexicons"
	oolongapp "tangled.org/arabica.social/arabica/internal/oolong/app"
)

func TestRegisterEntityRoutesFiltersUnionByAppDescriptors(t *testing.T) {
	bundles := []handlers.EntityRouteBundle{
		{RecordType: lexicons.RecordTypeBean, View: okHandler("bean")},
		{RecordType: lexicons.RecordTypeOolongTea, View: okHandler("tea")},
	}

	arabicaMux := http.NewServeMux()
	RegisterEntityRoutes(arabicaMux, http.NewCrossOriginProtection(), arabicaapp.New(), bundles, NewPageRoutes(nil, nil))
	assertRouteStatus(t, arabicaMux, "GET", "/beans/alice.test/r1", http.StatusOK)
	assertRouteStatus(t, arabicaMux, "GET", "/teas/alice.test/r1", http.StatusNotFound)

	oolongMux := http.NewServeMux()
	RegisterEntityRoutes(oolongMux, http.NewCrossOriginProtection(), oolongapp.New(), bundles, NewPageRoutes(nil, nil))
	assertRouteStatus(t, oolongMux, "GET", "/beans/alice.test/r1", http.StatusNotFound)
	assertRouteStatus(t, oolongMux, "GET", "/teas/alice.test/r1", http.StatusOK)
}

func TestRegisterEntityRoutesUsesExplicitPageOwnership(t *testing.T) {
	bundles := []handlers.EntityRouteBundle{
		{
			RecordType:    lexicons.RecordTypeBean,
			View:          okHandler("bean-view"),
			JSONView:      okHandler("bean-json"),
			Backlinks:     okHandler("bean-backlinks"),
			JSONBacklinks: okHandler("bean-json-backlinks"),
			OGImage:       okHandler("bean-og"),
			ModalNew:      okHandler("bean-modal-new"),
			ModalEdit:     okHandler("bean-modal-edit"),
		},
	}

	mux := http.NewServeMux()
	pages := NewPageRoutes(okHandler("spa"), []string{"GET /beans/{actor}/{id}"})
	RegisterEntityRoutes(mux, http.NewCrossOriginProtection(), arabicaapp.New(), bundles, pages)

	// Only the explicitly owned detail route uses the SPA. Backlinks retain
	// their working legacy handler when not listed as SPA-owned.
	assertRouteBody(t, mux, "GET", "/beans/alice.test/r1", http.StatusOK, "spa")
	assertRouteBody(t, mux, "GET", "/beans/alice.test/r1/backlinks", http.StatusOK, "bean-backlinks")

	// When the backlinks pattern IS SPA-owned, it routes to the SPA shell.
	pagesWithBacklinks := NewPageRoutes(okHandler("spa"), []string{
		"GET /beans/{actor}/{id}",
		"GET /beans/{actor}/{id}/backlinks",
	})
	mux2 := http.NewServeMux()
	RegisterEntityRoutes(mux2, http.NewCrossOriginProtection(), arabicaapp.New(), bundles, pagesWithBacklinks)
	assertRouteBody(t, mux2, "GET", "/beans/alice.test/r1/backlinks", http.StatusOK, "spa")

	// JSON view/backlinks, OG image, mutations and modal partials remain.
	assertRouteStatus(t, mux, "GET", "/api/beans/alice.test/r1", http.StatusOK)
	assertRouteStatus(t, mux, "GET", "/api/beans/alice.test/r1/backlinks", http.StatusOK)
	assertRouteStatus(t, mux, "GET", "/beans/alice.test/r1/og-image", http.StatusOK)
	assertRouteStatus(t, mux, "GET", "/api/modals/bean/new", http.StatusOK)
	assertRouteStatus(t, mux, "GET", "/api/modals/bean/r1", http.StatusOK)
}

func TestPageRoutesRegisterExplicitOwners(t *testing.T) {
	mux := http.NewServeMux()
	pages := NewPageRoutes(okHandler("spa"), []string{
		"GET /about",
		"GET /beans/{actor}/{id}",
		"GET /roasters/new",
	})

	pages.Register(mux, "GET /about", okHandler("legacy-about"))
	pages.Register(mux, "GET /manage", okHandler("legacy-manage"))
	pages.Register(mux, "GET /beans/{actor}/{id}", okHandler("legacy-bean"))
	pages.Register(mux, "GET /roasters/new", http.HandlerFunc(http.NotFound))
	mux.HandleFunc("GET /api/data", okHandler("api"))
	mux.HandleFunc("GET /static/app.css", okHandler("static"))
	mux.HandleFunc("/", http.NotFound)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{name: "SPA-owned static page", path: "/about", wantStatus: http.StatusOK, wantBody: "spa"},
		{name: "legacy-owned page", path: "/manage", wantStatus: http.StatusOK, wantBody: "legacy-manage"},
		{name: "SPA-owned dynamic page", path: "/beans/alice.test/r1", wantStatus: http.StatusOK, wantBody: "spa"},
		{name: "SPA-only create page", path: "/roasters/new", wantStatus: http.StatusOK, wantBody: "spa"},
		{name: "API remains independent", path: "/api/data", wantStatus: http.StatusOK, wantBody: "api"},
		{name: "static asset remains independent", path: "/static/app.css", wantStatus: http.StatusOK, wantBody: "static"},
		{name: "unknown route", path: "/not-a-real-page", wantStatus: http.StatusNotFound, wantBody: "404 page not found\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Equal(t, tt.wantBody, w.Body.String())
		})
	}
}

func okHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}
}

func assertRouteStatus(t *testing.T, h http.Handler, method, path string, want int) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, want, w.Code)
}

func assertRouteBody(t *testing.T, h http.Handler, method, path string, wantStatus int, wantBody string) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, wantStatus, w.Code)
	assert.Equal(t, wantBody, w.Body.String())
}
