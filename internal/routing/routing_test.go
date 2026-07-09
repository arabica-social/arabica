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
	RegisterEntityRoutes(arabicaMux, http.NewCrossOriginProtection(), arabicaapp.New(), bundles, false)
	assertRouteStatus(t, arabicaMux, "GET", "/beans/alice.test/r1", http.StatusOK)
	assertRouteStatus(t, arabicaMux, "GET", "/teas/alice.test/r1", http.StatusNotFound)

	oolongMux := http.NewServeMux()
	RegisterEntityRoutes(oolongMux, http.NewCrossOriginProtection(), oolongapp.New(), bundles, false)
	assertRouteStatus(t, oolongMux, "GET", "/beans/alice.test/r1", http.StatusNotFound)
	assertRouteStatus(t, oolongMux, "GET", "/teas/alice.test/r1", http.StatusOK)
}

func TestRegisterEntityRoutesSkipsPageViewsInSPAMode(t *testing.T) {
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
	RegisterEntityRoutes(mux, http.NewCrossOriginProtection(), arabicaapp.New(), bundles, true)

	// Page views and backlinks pages are skipped in SPA mode.
	assertRouteStatus(t, mux, "GET", "/beans/alice.test/r1", http.StatusNotFound)
	assertRouteStatus(t, mux, "GET", "/beans/alice.test/r1/backlinks", http.StatusNotFound)

	// JSON view/backlinks, OG image, mutations and modal partials remain.
	assertRouteStatus(t, mux, "GET", "/api/beans/alice.test/r1", http.StatusOK)
	assertRouteStatus(t, mux, "GET", "/api/beans/alice.test/r1/backlinks", http.StatusOK)
	assertRouteStatus(t, mux, "GET", "/beans/alice.test/r1/og-image", http.StatusOK)
	assertRouteStatus(t, mux, "GET", "/api/modals/bean/new", http.StatusOK)
	assertRouteStatus(t, mux, "GET", "/api/modals/bean/r1", http.StatusOK)
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
