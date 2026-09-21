package routing

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	arabicaapp "tangled.org/arabica.social/arabica/internal/arabica/app"
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/lexicons"
	"tangled.org/arabica.social/arabica/internal/metrics"
)

func TestRegisterEntityRoutesFiltersUnionByAppDescriptors(t *testing.T) {
	bundles := []handlers.EntityRouteBundle{
		{RecordType: lexicons.RecordTypeBean, JSONView: okHandler("bean-json")},
		{RecordType: lexicons.RecordTypeRoaster, JSONView: okHandler("roaster-json")},
	}

	arabicaMux := http.NewServeMux()
	RegisterEntityRoutes(arabicaMux, http.NewCrossOriginProtection(), arabicaapp.New(), bundles, NewPageRoutes(nil, nil))
	assertRouteStatus(t, arabicaMux, "GET", "/api/beans/alice.test/r1", http.StatusOK)
	assertRouteStatus(t, arabicaMux, "GET", "/api/roasters/alice.test/r1", http.StatusOK)
}

func TestRegisterEntityRoutesRegistersJSONAndOGAndMutations(t *testing.T) {
	bundles := []handlers.EntityRouteBundle{
		{
			RecordType:    lexicons.RecordTypeBean,
			JSONView:      okHandler("bean-json"),
			JSONBacklinks: okHandler("bean-json-backlinks"),
			OGImage:       okHandler("bean-og"),
			Create:        okHandler("bean-create"),
			Update:        okHandler("bean-update"),
			Delete:        okHandler("bean-delete"),
		},
	}

	mux := http.NewServeMux()
	RegisterEntityRoutes(mux, http.NewCrossOriginProtection(), arabicaapp.New(), bundles, NewPageRoutes(nil, nil))

	// JSON view/backlinks, OG image, and mutations are registered independently
	// of the SPA-owned page routes (which the shell serves directly).
	assertRouteStatus(t, mux, "GET", "/api/beans/alice.test/r1", http.StatusOK)
	assertRouteStatus(t, mux, "GET", "/api/beans/alice.test/r1/backlinks", http.StatusOK)
	assertRouteStatus(t, mux, "GET", "/beans/alice.test/r1/og-image", http.StatusOK)
	assertRouteStatus(t, mux, "POST", "/api/beans", http.StatusOK)
	assertRouteStatus(t, mux, "PUT", "/api/beans/r1", http.StatusOK)
	assertRouteStatus(t, mux, "DELETE", "/api/beans/r1", http.StatusOK)
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

type telemetryTestRoutes struct{}

func (telemetryTestRoutes) SPAOwnedRoutes() []string {
	return []string{
		"GET /{$}",
		"GET /explore",
		"GET /brews/{actor}/{id}",
	}
}

func (telemetryTestRoutes) RegisterAppRoutes(mux *http.ServeMux, ctx AppRouteContext) {
	page := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	ctx.Pages.Register(mux, "GET /explore", page)
	ctx.Pages.Register(mux, "GET /brews/{actor}/{id}", page)
}

func TestSetupRouterRecordsMatchedRoutesInMetricsAndTraces(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previousProvider := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		_ = provider.Shutdown(t.Context())
	})

	page := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	router := SetupRouter(Config{
		Handlers:         &handlers.Handler{},
		Logger:           zerolog.Nop(),
		AppRoutes:        telemetryTestRoutes{},
		SPAHandler:       page,
		DisableRateLimit: true,
	})

	tests := []struct {
		name       string
		path       string
		wantRoute  string
		wantStatus int
	}{
		{name: "health check", path: "/healthz", wantRoute: "/healthz", wantStatus: http.StatusOK},
		{name: "home", path: "/", wantRoute: "/", wantStatus: http.StatusNoContent},
		{name: "main page", path: "/explore", wantRoute: "/explore", wantStatus: http.StatusNoContent},
		{name: "dynamic page", path: "/brews/alice.test/3abc", wantRoute: "/brews/{actor}/{id}", wantStatus: http.StatusNoContent},
		{name: "unknown page", path: "/not-a-real-page", wantRoute: "/{unmatched}", wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := metrics.HTTPRequestsTotal.WithLabelValues(http.MethodGet, tt.wantRoute, strconv.Itoa(tt.wantStatus))
			before := counterValue(t, counter)

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Equal(t, before+1, counterValue(t, counter))
		})
	}

	ended := recorder.Ended()
	if assert.Len(t, ended, len(tests)) {
		for i, tt := range tests {
			assert.Equal(t, http.MethodGet+" "+tt.wantRoute, ended[i].Name())
		}
	}
}

func counterValue(t *testing.T, counter prometheus.Counter) float64 {
	t.Helper()
	var metric dto.Metric
	assert.NoError(t, counter.Write(&metric))
	return metric.GetCounter().GetValue()
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
