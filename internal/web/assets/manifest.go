package assets

// Manifest gives templates explicit access to cache-busted asset URLs.
//
// The zero value is usable: it falls back to the package registries so tests
// and older call sites that build LayoutData directly continue to render.
type Manifest struct {
	css *Bundle
	js  *JSAssets
}

func NewManifest(css *Bundle, js *JSAssets) Manifest {
	return Manifest{css: css, js: js}
}

func (m Manifest) StylesheetHref(appName string) string {
	if m.css != nil {
		return m.css.Href()
	}
	return HrefFor(appName)
}

func (m Manifest) ScriptHref(name string) string {
	if m.js != nil {
		if href := m.js.Href(name); href != "" {
			return href
		}
	}
	return JSHrefFor(name)
}
