package routing

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"tangled.org/arabica.social/arabica/internal/atplatform/apps"
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func TestRegisterEntityRoutesFiltersUnionByAppDescriptors(t *testing.T) {
	bundles := []handlers.EntityRouteBundle{
		{RecordType: lexicons.RecordTypeBean, View: okHandler("bean")},
		{RecordType: lexicons.RecordTypeOolongTea, View: okHandler("tea")},
	}

	arabicaMux := http.NewServeMux()
	registerEntityRoutes(arabicaMux, http.NewCrossOriginProtection(), apps.NewArabica(), bundles)
	assertRouteStatus(t, arabicaMux, "GET", "/beans/alice.test/r1", http.StatusOK)
	assertRouteStatus(t, arabicaMux, "GET", "/teas/alice.test/r1", http.StatusNotFound)

	oolongMux := http.NewServeMux()
	registerEntityRoutes(oolongMux, http.NewCrossOriginProtection(), apps.NewOolong(), bundles)
	assertRouteStatus(t, oolongMux, "GET", "/beans/alice.test/r1", http.StatusNotFound)
	assertRouteStatus(t, oolongMux, "GET", "/teas/alice.test/r1", http.StatusOK)
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
