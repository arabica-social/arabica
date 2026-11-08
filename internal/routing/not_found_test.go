package routing_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	arabicaapp "tangled.org/arabica.social/arabica/internal/arabica/app"
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/routing"
)

func TestUnknownPageRouteContracts(t *testing.T) {
	for _, withShell := range []bool{false, true} {
		app := arabicaapp.New()
		h := handlers.NewHandler(nil, nil, nil, nil, nil, handlers.Config{})
		h.SetApp(app)
		var shell http.Handler
		if withShell {
			shell = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("X-Test-SPA", "yes")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("<html>app shell</html>"))
			})
		}
		router := routing.SetupRouter(routing.Config{App: app, Handlers: h, SPAHandler: shell, Logger: zerolog.Nop(), DisableRateLimit: true})
		for _, path := range []string{"/chatt", "/unknown-page", "/unknown/deep/path", "/apiary", "/static-page", "/chat"} {
			for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost} {
				t.Run(method+path+map[bool]string{false: "/without-shell", true: "/with-shell"}[withShell], func(t *testing.T) {
					rec := httptest.NewRecorder()
					router.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
					require.Equal(t, http.StatusNotFound, rec.Code)
					if withShell && method != http.MethodPost {
						require.Equal(t, "yes", rec.Header().Get("X-Test-SPA"))
						require.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))
						require.Equal(t, "no-store", rec.Result().Header.Get("Cache-Control"))
						require.Contains(t, rec.Body.String(), "app shell")
					} else {
						require.Empty(t, rec.Header().Get("X-Test-SPA"))
					}
				})
			}
		}
		for _, path := range []string{"/api", "/api/unknown", "/api/modals/recipe/new", "/static", "/static/missing.css", "/_app", "/_app/missing.js", "/.well-known", "/.well-known/unknown", "/auth", "/auth/unknown", "/oauth", "/oauth/unknown"} {
			t.Run(path+map[bool]string{false: "/without-shell", true: "/with-shell"}[withShell], func(t *testing.T) {
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
				expected := http.StatusNotFound
				if path == "/static" || path == "/_app" {
					expected = http.StatusTemporaryRedirect
				}
				require.Equal(t, expected, rec.Code)
				require.Empty(t, rec.Header().Get("X-Test-SPA"))
			})
		}
	}
}
