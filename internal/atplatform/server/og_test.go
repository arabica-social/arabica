package server

import (
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	arabicaapp "tangled.org/arabica.social/arabica/internal/arabica/app"
	arabicahandlers "tangled.org/arabica.social/arabica/internal/arabica/handlers"
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/ogcard"
	"tangled.org/arabica.social/arabica/internal/routing"
	"tangled.org/arabica.social/arabica/internal/web/assets"
	"tangled.org/arabica.social/arabica/internal/web/spa"
)

// Exercise the production resolver, embedded shell, app route registration,
// and card renderer together. A crawler does not run the Svelte application.
func TestHomepageOpenGraph(t *testing.T) {
	app := arabicaapp.New()
	h := handlers.NewHandler(nil, nil, nil, nil, nil, handlers.Config{
		PublicURL: "https://arabica.social",
	})
	h.SetApp(app)
	shell, err := spa.NewShellHandler(assets.NewManifest(nil), app.Name, app.Brand)
	require.NoError(t, err)
	shell.SetOGResolver(func(r *http.Request) spa.OGData {
		return resolvePageOG(r, h, app)
	})
	router := routing.SetupRouter(routing.Config{
		App: app, Handlers: h, AppRoutes: arabicahandlers.Routes{},
		SPAHandler: shell, Logger: zerolog.Nop(), DisableRateLimit: true,
	})

	checkImage := func(t *testing.T, imageURL string) {
		t.Helper()
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, imageURL, nil))
		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
		assert.Equal(t, "public, max-age=86400", w.Header().Get("Cache-Control"))
		img, err := png.Decode(w.Body)
		require.NoError(t, err, "the advertised image must be a complete PNG")
		assert.Equal(t, 1200, img.Bounds().Dx())
		assert.Equal(t, 630, img.Bounds().Dy())
		require.NotNil(t, ogcard.GetLogoFor(app.Name), "embedded site logo must decode")
	}

	t.Run("image endpoint", func(t *testing.T) {
		checkImage(t, "/og-image")
	})
	for _, path := range []string{"/", "/?sort=popular&utm_source=share"} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "http://internal:8080"+path, nil)
			req.Header.Set("User-Agent", "Twitterbot/1.0")
			router.ServeHTTP(w, req)
			require.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
			body := w.Body.String()
			assert.Contains(t, body, `data-frontend="sveltekit"`)
			head, _, found := strings.Cut(body, "</head>")
			require.True(t, found)
			for _, tag := range []string{
				`property="og:title" content="Arabica"`,
				`property="og:description" content="` + app.Brand.SiteDescription + `"`,
				`property="og:type" content="website"`,
				`property="og:url" content="https://arabica.social/"`,
				`property="og:image:width" content="1200"`,
				`property="og:image:height" content="630"`,
				`property="og:image:alt" content="Arabica"`,
				`name="twitter:card" content="summary_large_image"`,
				`name="twitter:image" content="https://arabica.social/og-image"`,
			} {
				assert.Contains(t, head, tag)
			}
			images := regexp.MustCompile(`<meta property="og:image" content="([^"]+)"`).FindAllStringSubmatch(head, -1)
			require.Len(t, images, 1, "homepage must advertise exactly one image without JavaScript")
			imageURL := images[0][1]
			parsed, err := url.Parse(imageURL)
			require.NoError(t, err)
			require.True(t, parsed.IsAbs())
			assert.Equal(t, "https://arabica.social/og-image", imageURL)
			checkImage(t, imageURL)
		})
	}
}

func TestResolvePageOGHomepageURL(t *testing.T) {
	app := arabicaapp.New()
	for _, tc := range []struct {
		name, publicURL, requestURL, forwardedProto, wantBase string
	}{
		{"configured", "https://arabica.social", "http://internal:8080/", "", "https://arabica.social"},
		{"trailing slash", "https://arabica.social/", "http://internal:8080/", "", "https://arabica.social"},
		{"local HTTP", "", "http://localhost:18910/", "", "http://localhost:18910"},
		{"TLS", "", "https://arabica.social/", "", "https://arabica.social"},
		{"HTTPS proxy", "", "http://arabica.social/", "https", "https://arabica.social"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := handlers.NewHandler(nil, nil, nil, nil, nil, handlers.Config{PublicURL: tc.publicURL})
			r := httptest.NewRequest(http.MethodGet, tc.requestURL, nil)
			r.Header.Set("X-Forwarded-Proto", tc.forwardedProto)
			og := resolvePageOG(r, h, app)
			assert.Equal(t, tc.wantBase+"/", og.URL)
			assert.Equal(t, tc.wantBase+"/og-image", og.Image)
			assert.Equal(t, app.Brand.DisplayName, og.ImageAlt)
			assert.Empty(t, og.Title, "retain the shell's brand title")
			assert.Empty(t, og.Description, "retain the shell's brand description")
		})
	}
}

func TestResolvePageOGOtherPages(t *testing.T) {
	app := arabicaapp.New()
	h := handlers.NewHandler(nil, nil, nil, nil, nil, handlers.Config{PublicURL: "https://arabica.social"})
	for _, path := range []string{"/about", "/settings", "/api/beans/alice.test/r1"} {
		assert.Empty(t, resolvePageOG(httptest.NewRequest(http.MethodGet, path, nil), h, app), path)
	}
	og := resolvePageOG(httptest.NewRequest(http.MethodGet, "/beans/alice.test/r1", nil), h, app)
	assert.Equal(t, "article", og.Type)
	assert.Equal(t, "https://arabica.social/beans/alice.test/r1", og.URL)
	assert.Equal(t, "https://arabica.social/beans/alice.test/r1/og-image", og.Image)
}
