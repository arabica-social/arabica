package assets

// Manifest gives templates explicit access to cache-busted asset URLs.
//
// The zero value is usable: it falls back to the package registry so tests
// and older call sites that build LayoutData directly continue to render.
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
