// Package assets bundles CSS source files at startup and serves the result
// over a single content-hashed URL.
//
// Source files are embedded via go:embed; the production build path produces
// one immutable byte slice per app (arabica, oolong, …) with a sha256-derived
// cache buster. The dev path opts in via DevDir and re-reads the directory on
// every request, so editing a CSS file under that path and refreshing the
// browser shows the change without a server restart.
package assets

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"sort"
	"sync"
)

//go:embed css/tokens.css css/reset.css css/utilities.css css/components/*.css css/themes/*.css
var embedded embed.FS

// Config configures a Bundle.
type Config struct {
	// AppName selects the theme file appended after the base layers. The
	// arabica app uses no extra theme; any other name expects
	// css/themes/<AppName>.css to exist.
	AppName string

	// DevDir, if non-empty, is a filesystem path containing the CSS source
	// tree (mirroring the embed layout: tokens.css, reset.css, utilities.css,
	// components/*.css, themes/*.css). When set, the handler re-reads and
	// re-hashes on every request so edits are picked up without a restart.
	DevDir string
}

// Bundle holds the bundled bytes and serves them.
type Bundle struct {
	appName string
	devDir  string

	once  sync.Once
	mu    sync.RWMutex
	bytes []byte
	hash  string
	err   error
}

// New creates a Bundle. Build errors surface lazily on first request so that
// startup never fails on a malformed CSS file — but the error is returned by
// MustBuild for callers that want to fail-fast.
func New(cfg Config) *Bundle {
	return &Bundle{appName: cfg.AppName, devDir: cfg.DevDir}
}

// MustBuild builds the bundle eagerly (production mode only) and panics on
// error. Useful as a startup smoke test.
func (b *Bundle) MustBuild() {
	if b.devDir != "" {
		return // dev builds happen per-request
	}
	if _, _, err := b.current(); err != nil {
		panic(fmt.Errorf("cssbundle %q: %w", b.appName, err))
	}
}

// Href returns the URL the layout template should link, including a content
// hash query parameter for cache busting.
func (b *Bundle) Href() string {
	hash, _ := b.hashOnly()
	return b.URLPath() + "?h=" + hash
}

// URLPath is the path part of the served URL, e.g. /static/css/output.css.
// Routing wires this path to b.Handler.
func (b *Bundle) URLPath() string {
	if b.appName == "" || b.appName == "arabica" {
		return "/static/css/output.css"
	}
	return "/static/css/output-" + b.appName + ".css"
}

// Handler returns an http.Handler that serves the bundled bytes with ETag
// support and an immutable Cache-Control header in production. In dev mode
// it sets Cache-Control: no-cache so edits show up on refresh.
func (b *Bundle) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bytes, hash, err := b.current()
		if err != nil {
			http.Error(w, "css bundle build failed", http.StatusInternalServerError)
			return
		}
		etag := `"` + hash + `"`
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.Header().Set("ETag", etag)
		if b.devDir != "" {
			w.Header().Set("Cache-Control", "no-cache")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		_, _ = w.Write(bytes)
	})
}

// current returns the active bundle bytes + hash, reading from disk every call
// in dev mode or memoizing once in production.
func (b *Bundle) current() ([]byte, string, error) {
	if b.devDir != "" {
		return b.build(os.DirFS(b.devDir), ".")
	}
	b.once.Do(func() {
		sub, err := fs.Sub(embedded, "css")
		if err != nil {
			b.err = err
			return
		}
		b.bytes, b.hash, b.err = b.build(sub, ".")
	})
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.bytes, b.hash, b.err
}

// hashOnly returns the hash without forcing a build of the full bytes slice
// when one is already cached. It's used by Href, which renders into many
// templ pages — keeps that path off the lock when possible.
func (b *Bundle) hashOnly() (string, error) {
	_, hash, err := b.current()
	return hash, err
}

// build concatenates the bundle from fsys rooted at root. Order: tokens,
// reset, utilities, components/*.css (sorted), and finally the theme overlay
// for non-arabica apps.
func (b *Bundle) build(fsys fs.FS, root string) ([]byte, string, error) {
	parts := []string{
		path.Join(root, "tokens.css"),
		path.Join(root, "reset.css"),
		path.Join(root, "utilities.css"),
	}
	comps, err := fs.Glob(fsys, path.Join(root, "components", "*.css"))
	if err != nil {
		return nil, "", fmt.Errorf("glob components: %w", err)
	}
	sort.Strings(comps)
	parts = append(parts, comps...)
	if b.appName != "" && b.appName != "arabica" {
		parts = append(parts, path.Join(root, "themes", b.appName+".css"))
	}

	var buf []byte
	for _, p := range parts {
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return nil, "", fmt.Errorf("read %s: %w", p, err)
		}
		buf = append(buf, data...)
		if len(data) > 0 && data[len(data)-1] != '\n' {
			buf = append(buf, '\n')
		}
	}
	if len(buf) == 0 {
		return nil, "", errors.New("empty bundle (no source files found)")
	}
	sum := sha256.Sum256(buf)
	return buf, hex.EncodeToString(sum[:])[:16], nil
}

// --- package-level registry, for the templ helper to look up bundles by app name ---

var (
	registryMu sync.RWMutex
	registry   = map[string]*Bundle{}
)

// Register makes a bundle discoverable by its app name. Call once per bundle
// from main.go after construction.
func Register(b *Bundle) {
	name := b.appName
	if name == "" {
		name = "arabica"
	}
	registryMu.Lock()
	registry[name] = b
	registryMu.Unlock()
}

// HrefFor returns the cache-busted CSS URL for the given app name. Falls
// back to a static path if no bundle is registered (e.g. early-render paths
// that run before main wires up registration, or in tests).
func HrefFor(appName string) string {
	if appName == "" {
		appName = "arabica"
	}
	registryMu.RLock()
	b, ok := registry[appName]
	registryMu.RUnlock()
	if !ok {
		if appName == "arabica" {
			return "/static/css/output.css"
		}
		return "/static/css/output-" + appName + ".css"
	}
	return b.Href()
}

// Registered returns all registered bundles. Routing uses this to wire one
// handler per bundle without needing to know app names ahead of time.
func Registered() []*Bundle {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*Bundle, 0, len(registry))
	for _, b := range registry {
		out = append(out, b)
	}
	return out
}
