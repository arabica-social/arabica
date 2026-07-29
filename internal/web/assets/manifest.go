package assets

// Manifest provides cache-busted asset URLs to the web shell.
//
// The zero value is usable: it falls back to the package registry so tests
// and callers without an injected bundle continue to render.
type Manifest struct {
	css *Bundle
}

func NewManifest(css *Bundle) Manifest {
	return Manifest{css: css}
}

func (m Manifest) StylesheetHref(appName string) string {
	if m.css != nil {
		return m.css.Href()
	}
	return HrefFor(appName)
}
