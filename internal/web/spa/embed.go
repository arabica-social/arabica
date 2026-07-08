// Package spa serves the SvelteKit SPA build output from an embedded
// filesystem, with a server-rendered <head> for OpenGraph metadata and
// social sharing.
//
// The SvelteKit app (under web/) builds to web/build/, producing:
//   - index.html — the SPA fallback page (served for all non-API, non-static
//     routes that don't match a registered Go handler)
//   - _app/immutable/** — versioned JS chunks
//
// At build time, the SvelteKit output is copied into internal/web/spa/build/
// (overwriting the placeholder) so the go:embed directive always resolves.
// In production, index.html is NOT served as-is. Instead, SPAShellHandler
// reads the template, injects server-side <head> content (OG tags, title,
// theme script, CSP nonce, CSS/JS hrefs, traceparent), and serves the
// result. This ensures crawlers and social media bots see correct metadata
// without executing JavaScript.
//
// During migration, the shell route is a catch-all that only activates for
// paths not handled by existing templ routes. This allows page-by-page
// migration: ported pages use SvelteKit routes, unported pages fall through
// to their existing templ handlers.
package spa

import (
	"embed"
	"io/fs"
)

//go:embed all:build
var buildFS embed.FS

// EmbeddedFS returns the embedded SvelteKit build filesystem, rooted at the
// build/ directory. Callers can walk it to serve static assets.
func EmbeddedFS() fs.FS {
	sub, err := fs.Sub(buildFS, "build")
	if err != nil {
		panic("spa: embedded build/ directory not found: " + err.Error())
	}
	return sub
}
