// Package spa serves the SvelteKit SPA build output from an embedded
// filesystem, with a server-rendered <head> for OpenGraph metadata and
// social sharing.
//
// The SvelteKit app (under web/) builds to web/build/, producing:
//   - index.html — the SPA fallback page
//   - _app/immutable/** — versioned JS chunks
//
// At build time, the SvelteKit output is copied into internal/web/spa/build/
// (overwriting the placeholder) so the go:embed directive always resolves.
//
// In production, index.html is NOT served as-is. Instead, ShellHandler
// reads the built template, injects server-side <head> content (OG tags,
// title, theme script, CSP nonce, CSS link, traceparent, body data
// attributes) at the <!--ARABICA_SPA_HEAD--> marker in app.html, and
// serves the result. This ensures crawlers and social media bots see
// correct metadata without executing JavaScript.
//
// During migration, shared routing registers the shell only for explicit
// app-owned page patterns. Ported pages use SvelteKit routes, unported pages
// keep their existing templ handlers, and unknown direct loads remain 404s.
package spa

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/middleware"
	"tangled.org/arabica.social/arabica/internal/web/assets"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"
)

// ShellData holds the server-side data injected into the SPA shell's <head>.
// All values are pre-computed by the handler before template execution —
// the template only reads fields, never calls methods.
type ShellData struct {
	Title        string
	BrandName    string
	BrandTagline string
	AppName      string
	// SiteDescription is the app-level meta description from domain.Brand.
	SiteDescription string
	OGTitle         string
	OGDescription   string
	OGImage         string
	OGImageAlt      string
	OGType          string
	OGUrl           string
	CSPNonce        string
	Traceparent     string
	UserDID         string
	IsAuthenticated bool
	// StylesheetHref is the cache-busted CSS URL (the existing global CSS
	// bundle). The SPA continues to use the same stylesheet during migration.
	StylesheetHref string

	// Session data injected as data-* attributes on <body> so the SvelteKit
	// header can render the authenticated state without an extra API call.
	// All optional — empty values are omitted from the HTML.
	UserHandle              string
	UserDisplayName         string
	UserAvatar              string
	IsModerator             bool
	UnreadNotificationCount int
	TemperatureUnit         string

	// Pre-computed values for the template (derived from the fields above).
	PageTitle       string
	LightThemeColor string
	DarkThemeColor  string
	TwitterCardType string
}

// ShellHandler serves the SvelteKit SPA shell with server-side <head>
// injection. It reads the embedded index.html (produced by the SvelteKit
// build), injects OG/meta tags and scripts at the <!--ARABICA_SPA_HEAD-->
// marker, and serves the result.
//
// Static SvelteKit assets (_app/immutable/**) are served by AssetHandler.
type ShellHandler struct {
	indexHTML    []byte
	headTemplate *template.Template
	appName      string
	brand        domain.BrandConfig
	manifest     assets.Manifest
	// sessionResolver, when set, provides per-request session data
	// (profile, unread count, moderator flag) for the authenticated user.
	// When nil, only the DID from context is injected.
	sessionResolver SessionResolver
	// ogResolver, when set, resolves entity-specific OG metadata for the
	// request URL. This allows social sharing and crawlers to see
	// record-specific title, description, and image on SPA-owned entity
	// view pages without executing JavaScript. When nil, default brand-level
	// OG tags are used.
	ogResolver OGResolver
	// devDir, when non-empty, is a path to the SvelteKit build output on
	// disk (e.g. web/build). ServeHTTP re-reads index.html from it on every
	// request so `vite build --watch` output appears on the next browser
	// refresh without restarting the Go server. Empty in production, where
	// the embedded copy read at construction time is always used.
	devDir string
}

// SessionData carries the authenticated user's display state for the SPA
// shell. The resolver returns zero values for unauthenticated requests.
type SessionData struct {
	Handle                  string
	DisplayName             string
	Avatar                  string
	IsModerator             bool
	UnreadNotificationCount int
	// TemperatureUnit is the user's preferred brew temperature display
	// unit ("recorded", "celsius", "fahrenheit"). Injected as a body data
	// attribute so the SPA can format temperatures without an extra API call.
	TemperatureUnit string
}

// SessionResolver looks up session data for a DID. The shell handler calls
// it once per request to populate the <body> data attributes. Implementations
// should be cheap (cache-backed) since this runs on every SPA page load.
type SessionResolver func(ctx context.Context, did string) SessionData

// OGData carries entity-specific OpenGraph metadata for the SPA shell.
// When an OGResolver returns non-zero values, they override the default
// brand-level OG tags in the shell's <head>.
type OGData struct {
	Title       string
	Description string
	Image       string
	ImageAlt    string
	URL         string
	Type        string
}

// OGResolver resolves entity-specific OG metadata for a request URL.
// The shell handler calls it for every SPA page load. Implementations
// should inspect the URL path, and if it matches an entity-view pattern
// (e.g. /beans/{actor}/{id}), load the record and return its OG metadata.
// If the URL is not an entity view or the record can't be resolved,
// return zero-value OGData and the shell falls back to brand defaults.
type OGResolver func(r *http.Request) OGData

// NewShellHandler creates a handler that serves the SPA shell. It reads
// the embedded index.html at construction time. The assets manifest
// provides cache-busted CSS hrefs for the <head>.
func NewShellHandler(manifest assets.Manifest, appName string, brand domain.BrandConfig) (*ShellHandler, error) {
	fsys := EmbeddedFS()
	indexBytes, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		return nil, fmt.Errorf("read embedded index.html: %w", err)
	}

	// Verify the head marker is present (app.html must contain it).
	if !bytes.Contains(indexBytes, []byte(headMarker)) {
		return nil, fmt.Errorf("spa: embedded index.html missing arabica-spa-head marker — check web/src/app.html")
	}

	tmpl := template.Must(template.New("head").Parse(headFragmentTemplate))

	return &ShellHandler{
		indexHTML:    indexBytes,
		headTemplate: tmpl,
		appName:      appName,
		brand:        brand,
		manifest:     manifest,
	}, nil
}

// SetSessionResolver installs a resolver that populates the <body> data
// attributes with the authenticated user's profile, unread notification
// count, and moderator flag. Must be called before the handler starts
// serving requests. Passing nil disables session injection (DID only).
func (h *ShellHandler) SetSessionResolver(r SessionResolver) {
	h.sessionResolver = r
}

// SetOGResolver installs a resolver that populates entity-specific OG
// metadata (title, description, image, URL) for entity-view URL patterns.
// When nil or when the resolver returns empty data, the shell falls back
// to default brand-level OG tags.
func (h *ShellHandler) SetOGResolver(r OGResolver) {
	h.ogResolver = r
}

// SetDevDir enables dev mode for the shell handler. When set to a non-empty
// path (the SvelteKit build output directory, e.g. web/build), ServeHTTP
// re-reads index.html from disk on every request so edits produced by
// `vite build --watch` appear on the next refresh without a Go restart.
// When empty (the default), the embedded copy read at construction time is
// always used. The dev path falls back to the embedded copy if the on-disk
// file is missing, so dev works before the first vite build completes.
func (h *ShellHandler) SetDevDir(dir string) {
	h.devDir = dir
}

// ServeHTTP serves the SPA index.html with injected <head> content.
func (h *ShellHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := ShellData{
		AppName:         h.appName,
		BrandName:       h.brand.DisplayName,
		BrandTagline:    h.brand.Tagline,
		SiteDescription: h.brand.SiteDescription,
		LightThemeColor: h.brand.LightThemeColor,
		DarkThemeColor:  h.brand.DarkThemeColor,
		StylesheetHref:  h.manifest.StylesheetHref(h.appName),
	}

	// Populate from request context if available (auth state, traceparent,
	// CSP nonce).
	if did, ok := didFromContext(r.Context()); ok {
		data.UserDID = did
		data.IsAuthenticated = true

		// Resolve session data (profile, unread count, moderator flag) for
		// the authenticated user. The resolver is cache-backed and cheap.
		if h.sessionResolver != nil {
			session := h.sessionResolver(r.Context(), did)
			data.UserHandle = session.Handle
			data.UserDisplayName = session.DisplayName
			data.UserAvatar = session.Avatar
			data.IsModerator = session.IsModerator
			data.UnreadNotificationCount = session.UnreadNotificationCount
			data.TemperatureUnit = session.TemperatureUnit
		}
	}
	data.CSPNonce = middleware.CSPNonceFromContext(r.Context())
	if tp := traceparentFromContext(r.Context()); tp != "" {
		data.Traceparent = tp
	}

	// Resolve entity-specific OG metadata for entity-view URL patterns.
	// Falls back to brand defaults when the URL is not an entity view or
	// the resolver returns empty data.
	if h.ogResolver != nil {
		og := h.ogResolver(r)
		if og.Title != "" {
			data.OGTitle = og.Title
			data.Title = og.Title
		}
		if og.Description != "" {
			data.OGDescription = og.Description
		}
		if og.Image != "" {
			data.OGImage = og.Image
		}
		if og.ImageAlt != "" {
			data.OGImageAlt = og.ImageAlt
		}
		if og.URL != "" {
			data.OGUrl = og.URL
		}
		if og.Type != "" {
			data.OGType = og.Type
		}
	}

	// Default OG metadata when no entity-specific data was resolved.
	if data.OGTitle == "" {
		data.OGTitle = data.BrandName
	}
	if data.OGDescription == "" {
		data.OGDescription = data.siteDescription()
	}
	if data.OGType == "" {
		data.OGType = "website"
	}

	// Pre-compute derived values for the template.
	data.SiteDescription = data.siteDescription()
	data.PageTitle = data.pageTitle()
	data.LightThemeColor = data.lightThemeColor()
	data.DarkThemeColor = data.darkThemeColor()
	data.TwitterCardType = data.twitterCardType()

	// Render the head fragment.
	var headBuf bytes.Buffer
	if err := h.headTemplate.Execute(&headBuf, data); err != nil {
		http.Error(w, "SPA shell render failed", http.StatusInternalServerError)
		return
	}

	// Inject the head fragment at the marker, replacing it. In dev mode,
	// re-read index.html from disk so changes from `vite build --watch` are
	// served on the next refresh without a Go restart; fall back to the
	// embedded copy read at construction if the dev file is missing.
	indexBytes := h.indexHTML
	if h.devDir != "" {
		if b, err := fs.ReadFile(os.DirFS(h.devDir), "index.html"); err == nil {
			indexBytes = b
		}
	}
	result := bytes.Replace(indexBytes, []byte(headMarker), headBuf.Bytes(), 1)

	// Inject body data attributes (user-did, app) into the <body> tag.
	if data.UserDID != "" || data.AppName != "" {
		result = injectBodyAttrs(result, data)
	}

	// The SvelteKit adapter-static emits a small inline bootstrap script
	// that loads the app and start chunks. It has no nonce, so the CSP
	// blocks it. Add the request nonce to that inline script.
	result = injectSvelteKitScriptNonce(result, data.CSPNonce)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(result)
}

// injectBodyAttrs adds data-* attributes to the <body> tag in the HTML.
// These are read by the SvelteKit app to determine auth state, which app
// (arabica/oolong) is running, and the authenticated user's display
// profile / unread count / moderator flag. data-frontend is an explicit
// browser-test marker proving that a direct load reached the SvelteKit shell.
func injectBodyAttrs(html []byte, data ShellData) []byte {
	bodyTag := []byte("<body")
	if !bytes.Contains(html, bodyTag) {
		return html
	}

	attrs := []byte(` data-frontend="sveltekit"`)
	if data.UserDID != "" {
		attrs = append(attrs, []byte(` data-user-did="`+data.UserDID+`"`)...)
	}
	if data.AppName != "" {
		attrs = append(attrs, []byte(` data-app="`+data.AppName+`"`)...)
	}
	if data.UserHandle != "" {
		attrs = append(attrs, []byte(` data-user-handle="`+htmlEscapeAttr(data.UserHandle)+`"`)...)
	}
	if data.UserDisplayName != "" {
		attrs = append(attrs, []byte(` data-user-display="`+htmlEscapeAttr(data.UserDisplayName)+`"`)...)
	}
	if data.UserAvatar != "" {
		attrs = append(attrs, []byte(` data-user-avatar="`+htmlEscapeAttr(data.UserAvatar)+`"`)...)
	}
	if data.IsModerator {
		attrs = append(attrs, []byte(` data-is-moderator="true"`)...)
	}
	if data.UnreadNotificationCount > 0 {
		attrs = append(attrs, []byte(fmt.Sprintf(` data-unread-notifications="%d"`, data.UnreadNotificationCount))...)
	}
	if data.TemperatureUnit != "" {
		attrs = append(attrs, []byte(` data-temperature-unit="`+htmlEscapeAttr(data.TemperatureUnit)+`"`)...)
	}

	// Insert attributes right after "<body".
	idx := bytes.Index(html, bodyTag)
	if idx < 0 {
		return html
	}
	insertAt := idx + len(bodyTag)
	result := make([]byte, 0, len(html)+len(attrs))
	result = append(result, html[:insertAt]...)
	result = append(result, attrs...)
	result = append(result, html[insertAt:]...)
	return result
}

// injectSvelteKitScriptNonce adds the CSP nonce to the SvelteKit bootstrap
// inline script. The script is the only inline <script> in the generated
// index.html; it loads the start/app entry chunks and calls kit.start().
func injectSvelteKitScriptNonce(html []byte, nonce string) []byte {
	if nonce == "" {
		return html
	}
	// The inline script is the only <script> in the generated index.html
	// (the theme script is injected via the head template above). Target the
	// exact pattern SvelteKit emits so we don't accidentally rewrite other
	// scripts.
	target := []byte("<script>\n\t\t\t\t{\n\t\t\t\t\t__sveltekit_")
	replacement := []byte(`<script nonce="` + htmlEscapeAttr(nonce) + `">` + "\n\t\t\t\t{\n\t\t\t\t\t__sveltekit_")
	return bytes.Replace(html, target, replacement, 1)
}

// htmlEscapeAttr escapes a string for safe inclusion in a double-quoted HTML
// attribute value. It escapes the minimal set required by the HTML spec:
// &, ", <, >. This avoids pulling in html/template for a single use.
func htmlEscapeAttr(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '"':
			b.WriteString("&quot;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (d ShellData) siteDescription() string {
	if d.SiteDescription != "" {
		return d.SiteDescription
	}
	return d.BrandName + " is a brew tracking app built on AT Protocol. Your brewing data is stored in your own Personal Data Server, giving you full ownership and portability."
}

func (d ShellData) pageTitle() string {
	if d.Title == "" || d.Title == "Home" {
		return d.BrandName
	}
	return d.Title + " - " + d.BrandName
}

func (d ShellData) lightThemeColor() string {
	if d.LightThemeColor != "" {
		return d.LightThemeColor
	}
	return "#4a2c2a"
}

func (d ShellData) darkThemeColor() string {
	if d.DarkThemeColor != "" {
		return d.DarkThemeColor
	}
	return "#0F0A08"
}

func (d ShellData) twitterCardType() string {
	if d.OGImage != "" {
		return "summary_large_image"
	}
	return "summary"
}

// AssetHandler serves SvelteKit static assets (_app/immutable/**) from the
// embedded build filesystem. These are the versioned JS chunks produced by
// the SvelteKit build.
func AssetHandler() http.Handler {
	fsys, err := fs.Sub(EmbeddedFS(), "_app")
	if err != nil {
		panic("spa: embedded _app directory not found: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(fsys))
	stripped := http.StripPrefix("/_app/", fileServer)
	return contentTypeMiddleware(stripped)
}

// AssetHandlerWithDevDir serves SvelteKit static assets from a build
// directory on disk (e.g. web/build) instead of the embedded filesystem.
// Intended for dev mode: `vite build --watch` writes new chunks to the
// directory and they are served on the next refresh without a Go restart.
// Responses carry Cache-Control: no-cache so stale chunks are never cached
// by the browser. The _app subdirectory is resolved relative to devDir to
// mirror the embedded layout (build/_app/**).
func AssetHandlerWithDevDir(devDir string) http.Handler {
	fsys := os.DirFS(filepath.Join(devDir, "_app"))
	fileServer := http.FileServer(http.FS(fsys))
	stripped := http.StripPrefix("/_app/", fileServer)
	return contentTypeMiddleware(noCacheMiddleware(stripped))
}

// noCacheMiddleware sets Cache-Control: no-cache so dev-mode responses are
// always revalidated against the on-disk source rather than served from the
// browser cache. Only used by the disk-backed dev asset handler.
func noCacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		next.ServeHTTP(w, r)
	})
}

// contentTypeMiddleware ensures embedded assets are served with correct
// Content-Type headers. Go's http.FileServer relies on the host MIME
// database via mime.TypeByExtension, which can return text/plain on minimal
// systems (e.g. NixOS without mailcap), breaking module loading.
//
// The wrapper locks the Content-Type we compute so that the underlying
// FileServer cannot overwrite it with its own (possibly wrong) guess.
func contentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := contentTypeForPath(r.URL.Path)
		if ct != "" {
			w.Header().Set("Content-Type", ct)
			next.ServeHTTP(&contentTypeLockingResponseWriter{ResponseWriter: w, contentType: ct}, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// contentTypeLockingResponseWriter prevents http.FileServer from overwriting
// a Content-Type header that has already been set explicitly.
type contentTypeLockingResponseWriter struct {
	http.ResponseWriter
	contentType string
}

func (w *contentTypeLockingResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.Header().Set("Content-Type", w.contentType)
	w.ResponseWriter.WriteHeader(code)
}

func (w *contentTypeLockingResponseWriter) Write(p []byte) (int, error) {
	w.ResponseWriter.Header().Set("Content-Type", w.contentType)
	return w.ResponseWriter.Write(p)
}

func contentTypeForPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".js", ".mjs":
		return "text/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".json":
		return "application/json"
	case ".woff2":
		return "font/woff2"
	case ".woff":
		return "font/woff"
	case ".ttf":
		return "font/ttf"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".ico":
		return "image/x-icon"
	default:
		return ""
	}
}

// IsSPAAsset returns true if the request path is for a SvelteKit static
// asset (e.g. /_app/immutable/...).
func IsSPAAsset(path string) bool {
	return strings.HasPrefix(path, "/_app/")
}

// --- context keys for shell data injection ---

type contextKey int

const (
	didKey contextKey = iota
	traceparentKey
)

// WithDID stores the authenticated user's DID in the request context for
// the shell handler to read. Middleware or a pre-shell resolver calls this.
func WithDID(ctx context.Context, did string) context.Context {
	return context.WithValue(ctx, didKey, did)
}

func didFromContext(ctx context.Context) (string, bool) {
	did, ok := ctx.Value(didKey).(string)
	if ok && did != "" {
		return did, true
	}
	return atpmiddleware.GetDID(ctx)
}

// WithTraceparent stores the W3C traceparent in the request context.
func WithTraceparent(ctx context.Context, tp string) context.Context {
	return context.WithValue(ctx, traceparentKey, tp)
}

func traceparentFromContext(ctx context.Context) string {
	tp, _ := ctx.Value(traceparentKey).(string)
	return tp
}

// headMarker is a <meta> tag in web/src/app.html that SvelteKit preserves
// through the build. The Go shell replaces it with server-rendered <head>
// content (OG tags, title, theme script, etc.). HTML comments are stripped
// by the SvelteKit build, so a meta tag is used instead.
const headMarker = `<meta name="arabica-spa-head" content="" />`

// headFragmentTemplate is the server-rendered <head> content injected at
// the <!--ARABICA_SPA_HEAD--> marker. It mirrors the <head> structure from
// components/layout.templ: theme script, OG metadata, Twitter cards,
// theme-color, title, favicon, stylesheet, manifest, traceparent.
//
// SvelteKit's own %sveltekit.head% output (modulepreload links) remains
// after this fragment — both coexist in <head>.
const headFragmentTemplate = `
		<script nonce="{{.CSPNonce}}">
			(function() {
				var t = localStorage.getItem('{{.AppName}}-theme');
				if (t === 'dark' || t === 'light') document.documentElement.setAttribute('data-theme', t);
			})();
		</script>
		<meta name="description" content="{{.SiteDescription}}"/>
		<meta property="og:title" content="{{.OGTitle}}"/>
		<meta property="og:description" content="{{.OGDescription}}"/>
		<meta property="og:type" content="{{.OGType}}"/>
		<meta property="og:site_name" content="{{.BrandName}}"/>
		{{if .OGUrl}}<meta property="og:url" content="{{.OGUrl}}"/>{{end}}
		{{if .OGImage}}
		<meta property="og:image" content="{{.OGImage}}"/>
		<meta property="og:image:width" content="1200"/>
		<meta property="og:image:height" content="630"/>
		<meta property="og:image:alt" content="{{.OGImageAlt}}"/>
		{{end}}
		<meta name="twitter:card" content="{{.TwitterCardType}}"/>
		<meta name="twitter:title" content="{{.OGTitle}}"/>
		<meta name="twitter:description" content="{{.OGDescription}}"/>
		{{if .OGImage}}<meta name="twitter:image" content="{{.OGImage}}"/>{{end}}
		<meta name="theme-color" content="{{.LightThemeColor}}" media="(prefers-color-scheme: light)"/>
		<meta name="theme-color" content="{{.DarkThemeColor}}" media="(prefers-color-scheme: dark)"/>
		<title>{{.PageTitle}}</title>
		<link rel="icon" href="/static/favicon.svg" type="image/svg+xml"/>
		<link rel="icon" href="/static/favicon-32.svg" type="image/svg+xml" sizes="32x32"/>
		<link rel="apple-touch-icon" href="/static/icon-192.svg"/>
		<link rel="preload" href="/static/fonts/fraunces-latin.woff2" as="font" type="font/woff2" crossorigin/>
		<link rel="stylesheet" href="{{.StylesheetHref}}"/>
		<link rel="manifest" href="/static/manifest.json"/>
		{{if .Traceparent}}<meta name="traceparent" content="{{.Traceparent}}"/>{{end}}
`
