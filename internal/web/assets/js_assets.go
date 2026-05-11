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
	"sync"
)

//go:embed js/*.js
var jsEmbedded embed.FS

// JSConfig configures a JSAssets.
type JSConfig struct {
	// DevDir, if non-empty, points to a directory containing the JS source
	// files (mirroring the embed layout, e.g. internal/web/assets/js).
	// When set, every request re-reads the file from disk and re-hashes,
	// so editing a .js file and refreshing picks up the change.
	DevDir string
}

// JSAssets serves the embedded JavaScript files at /static/js/<name> with
// per-file content-hash cache busting.
type JSAssets struct {
	devDir string

	once     sync.Once
	mu       sync.RWMutex
	contents map[string][]byte // filename → bytes
	hashes   map[string]string // filename → hash
	err      error
}

// NewJSAssets builds a JSAssets ready to serve. In dev mode (DevDir set)
// nothing is loaded eagerly. In production the embed.FS is enumerated on
// first use; call MustBuild to fail fast on startup.
func NewJSAssets(cfg JSConfig) *JSAssets {
	return &JSAssets{devDir: cfg.DevDir}
}

// MustBuild builds the per-file caches eagerly (production only) and panics
// on error. Useful as a startup smoke test.
func (a *JSAssets) MustBuild() {
	if a.devDir != "" {
		return
	}
	if err := a.ensureLoaded(); err != nil {
		panic(fmt.Errorf("js assets: %w", err))
	}
}

// Href returns the cache-busted URL for a named JS file, or an empty string
// if the file isn't known. Templates should pass a constant string; missing
// files surface as an empty src so the page request fails loudly in dev.
func (a *JSAssets) Href(name string) string {
	hash, ok := a.hashFor(name)
	if !ok {
		return ""
	}
	return "/static/js/" + name + "?h=" + hash
}

// Handler returns the http.Handler that serves any registered .js file.
// Route it at GET /static/js/{name} — the handler reads the {name} path
// value and looks it up in the per-file cache (or reads from disk in
// dev mode).
func (a *JSAssets) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if name == "" {
			http.NotFound(w, r)
			return
		}
		bytes, hash, ok := a.lookup(name)
		if !ok {
			http.NotFound(w, r)
			return
		}
		etag := `"` + hash + `"`
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("ETag", etag)
		if a.devDir != "" {
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

// hashFor returns the hash for a file. In dev mode it re-reads the file so
// templ-rendered hrefs always reflect on-disk content.
func (a *JSAssets) hashFor(name string) (string, bool) {
	if a.devDir != "" {
		data, err := os.ReadFile(a.devDir + "/" + name)
		if err != nil {
			return "", false
		}
		sum := sha256.Sum256(data)
		return hex.EncodeToString(sum[:])[:16], true
	}
	if err := a.ensureLoaded(); err != nil {
		return "", false
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	h, ok := a.hashes[name]
	return h, ok
}

// lookup returns the file bytes + hash for a file.
func (a *JSAssets) lookup(name string) ([]byte, string, bool) {
	if a.devDir != "" {
		data, err := os.ReadFile(a.devDir + "/" + name)
		if err != nil {
			return nil, "", false
		}
		sum := sha256.Sum256(data)
		return data, hex.EncodeToString(sum[:])[:16], true
	}
	if err := a.ensureLoaded(); err != nil {
		return nil, "", false
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	data, ok := a.contents[name]
	if !ok {
		return nil, "", false
	}
	return data, a.hashes[name], true
}

// ensureLoaded reads the embedded js/*.js files once and caches them with
// their content hashes. Subsequent calls return immediately.
func (a *JSAssets) ensureLoaded() error {
	a.once.Do(func() {
		sub, err := fs.Sub(jsEmbedded, "js")
		if err != nil {
			a.err = err
			return
		}
		entries, err := fs.ReadDir(sub, ".")
		if err != nil {
			a.err = err
			return
		}
		contents := map[string][]byte{}
		hashes := map[string]string{}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			data, err := fs.ReadFile(sub, e.Name())
			if err != nil {
				a.err = fmt.Errorf("read %s: %w", e.Name(), err)
				return
			}
			sum := sha256.Sum256(data)
			contents[e.Name()] = data
			hashes[e.Name()] = hex.EncodeToString(sum[:])[:16]
		}
		if len(contents) == 0 {
			a.err = errors.New("no embedded JS files found")
			return
		}
		a.mu.Lock()
		a.contents = contents
		a.hashes = hashes
		a.mu.Unlock()
	})
	return a.err
}

// --- package-level registry, mirroring the CSS bundle ---

var (
	jsRegistryMu sync.RWMutex
	jsRegistered *JSAssets
)

// RegisterJS makes a JSAssets discoverable to the templ helper. The app is
// single-tenant from the JS perspective (the same script files serve every
// app), so there's only one registered instance.
func RegisterJS(a *JSAssets) {
	jsRegistryMu.Lock()
	jsRegistered = a
	jsRegistryMu.Unlock()
}

// JSHrefFor is the templ-facing helper. Returns /static/js/<name>?h=<hash>
// when the file is known and a JSAssets is registered; falls back to an
// un-hashed path so misconfiguration shows up as a missing-version src
// rather than a broken page.
func JSHrefFor(name string) string {
	jsRegistryMu.RLock()
	a := jsRegistered
	jsRegistryMu.RUnlock()
	if a == nil {
		return "/static/js/" + name
	}
	if h := a.Href(name); h != "" {
		return h
	}
	return "/static/js/" + name
}
