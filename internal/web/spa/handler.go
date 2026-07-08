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
// During migration, the shell route is a catch-all that only activates for
// paths not handled by existing templ routes. This allows page-by-page
// migration: ported pages use SvelteKit routes, unported pages fall through
// to their existing templ handlers.
package spa

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"text/template"

	"tangled.org/arabica.social/arabica/internal/web/assets"
)

// ShellData holds the server-side data injected into the SPA shell's <head>.
// All values are pre-computed by the handler before template execution —
// the template only reads fields, never calls methods.
type ShellData struct {
	Title           string
	BrandName       string
	BrandTagline    string
	AppName         string
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

	// Pre-computed values for the template (derived from the fields above).
	SiteDescription  string
	PageTitle        string
	LightThemeColor  string
	DarkThemeColor   string
	TwitterCardType  string
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
	brandName    string
	manifest     assets.Manifest
}

// NewShellHandler creates a handler that serves the SPA shell. It reads
// the embedded index.html at construction time. The assets manifest
// provides cache-busted CSS hrefs for the <head>.
func NewShellHandler(manifest assets.Manifest, appName string) (*ShellHandler, error) {
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
		brandName:     brandNameForApp(appName),
		manifest:     manifest,
	}, nil
}

// ServeHTTP serves the SPA index.html with injected <head> content.
func (h *ShellHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := ShellData{
		AppName:        h.appName,
		BrandName:      h.brandName,
		BrandTagline:   brandTaglineForApp(h.appName),
		StylesheetHref: h.manifest.StylesheetHref(h.appName),
	}

	// Populate from request context if available (auth state, traceparent).
	if did, ok := didFromContext(r.Context()); ok {
		data.UserDID = did
		data.IsAuthenticated = true
	}
	if tp := traceparentFromContext(r.Context()); tp != "" {
		data.Traceparent = tp
	}

	// Default OG metadata. Entity-specific OG (title, image, description)
	// will be set by middleware or a pre-shell resolver in later phases.
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

	// Inject the head fragment at the marker, replacing it.
	result := bytes.Replace(h.indexHTML, []byte(headMarker), headBuf.Bytes(), 1)

	// Inject body data attributes (user-did, app) into the <body> tag.
	if data.UserDID != "" || data.AppName != "" {
		result = injectBodyAttrs(result, data)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(result)
}

// injectBodyAttrs adds data-user-did and data-app attributes to the <body>
// tag in the HTML. These are read by the SvelteKit app to determine auth
// state and which app (arabica/oolong) is running.
func injectBodyAttrs(html []byte, data ShellData) []byte {
	bodyTag := []byte("<body")
	if !bytes.Contains(html, bodyTag) {
		return html
	}

	var attrs []byte
	if data.UserDID != "" {
		attrs = append(attrs, []byte(` data-user-did="`+data.UserDID+`"`)...)
	}
	if data.AppName != "" {
		attrs = append(attrs, []byte(` data-app="`+data.AppName+`"`)...)
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

func (d ShellData) siteDescription() string {
	if d.AppName == "oolong" {
		return d.BrandName + " is a tea tracking app built on AT Protocol. Your steep logs, teas, and teaware are stored in your own Personal Data Server, giving you full ownership and portability."
	}
	return d.BrandName + " is a coffee brew tracking app built on AT Protocol. Your brewing data is stored in your own Personal Data Server, giving you full ownership and portability."
}

func brandNameForApp(appName string) string {
	if appName == "oolong" {
		return "Oolong"
	}
	return "Arabica"
}

func brandTaglineForApp(appName string) string {
	if appName == "oolong" {
		return "Your tea, your data"
	}
	return "Your brew, your data"
}

func (d ShellData) pageTitle() string {
	if d.Title == "" || d.Title == "Home" {
		return d.BrandName
	}
	return d.Title + " - " + d.BrandName
}

func (d ShellData) lightThemeColor() string {
	if d.AppName == "oolong" {
		return "#b8d5aa"
	}
	return "#4a2c2a"
}

func (d ShellData) darkThemeColor() string {
	if d.AppName == "oolong" {
		return "#162018"
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
	fsys := EmbeddedFS()
	fileServer := http.FileServer(http.FS(fsys))
	return http.StripPrefix("/_app/", fileServer)
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
	return did, ok
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
				var t = localStorage.getItem('arabica-theme');
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
		<link rel="stylesheet" href="{{.StylesheetHref}}"/>
		<link rel="manifest" href="/static/manifest.json"/>
		{{if .Traceparent}}<meta name="traceparent" content="{{.Traceparent}}"/>{{end}}
`
