package routing_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	arabicaapp "tangled.org/arabica.social/arabica/internal/arabica/app"

	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/routing"
)

func TestArabicaRouteContracts(t *testing.T) {
	for _, shell := range []bool{false, true} {
		t.Run(map[bool]string{false: "without shell", true: "with shell"}[shell], func(t *testing.T) {
			app := arabicaapp.New()
			h := handlers.NewHandler(nil, nil, nil, nil, nil, handlers.Config{})
			h.SetApp(app)
			var spa http.Handler
			if shell {
				spa = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Test-SPA", "yes")
					w.WriteHeader(http.StatusOK)
				})
			}
			router := routing.SetupRouter(routing.Config{App: app, Handlers: h, SPAHandler: spa, Logger: zerolog.Nop(), DisableRateLimit: true})
			for _, pattern := range routing.SPAOwnedRoutes() {
				path := strings.TrimPrefix(pattern, "GET ")
				path = strings.NewReplacer("{$}", "", "{actor}", "alice.test", "{id}", "r1").Replace(path)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
				if shell && path != "/chat" {
					require.Equal(t, 200, w.Code, path)
					require.Equal(t, "yes", w.Header().Get("X-Test-SPA"), path)
				} else {
					require.Equal(t, 404, w.Code, path)
				}
			}
			for _, path := range []string{"/unknown-page", "/api/unknown", "/api/modals/recipe/new", "/api/ai/status", "/api/session", "/healthz", "/og-image"} {
				w := httptest.NewRecorder()
				router.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
				require.Empty(t, w.Header().Get("X-Test-SPA"), path)
				if strings.Contains(path, "unknown") || path == "/api/modals/recipe/new" {
					require.Equal(t, 404, w.Code, path)
				}
			}
			for _, path := range []string{"/api/ai/chat", "/api/beans", "/api/recipes", "/api/brews", "/auth/login", "/logout", "/api/likes/toggle"} {
				req := httptest.NewRequest("POST", path, strings.NewReader(`{}`))
				req.Header.Set("Sec-Fetch-Site", "cross-site")
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				expected := http.StatusForbidden
				if path == "/api/ai/chat" {
					expected = http.StatusNotFound
				}
				require.Equal(t, expected, w.Code, path)
			}
		})
	}
}
