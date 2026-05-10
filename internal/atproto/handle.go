package atproto

import (
	"strings"

	"golang.org/x/net/idna"
)

// NormalizeHandle converts a handle to its ASCII (punycode) form for
// resolution and storage. Idempotent on plain ASCII input. Returns the
// lowercased input unchanged if IDN conversion fails.
func NormalizeHandle(handle string) string {
	h := strings.TrimPrefix(handle, "@")
	h = strings.ToLower(strings.TrimSpace(h))
	if h == "" {
		return ""
	}
	ascii, err := idna.Lookup.ToASCII(h)
	if err != nil {
		return h
	}
	return ascii
}

// DisplayHandle converts a handle to its Unicode form for display. If the
// input is already Unicode (or plain ASCII with no xn-- labels), it is
// returned unchanged. Falls back to the input on conversion error.
func DisplayHandle(handle string) string {
	if handle == "" {
		return ""
	}
	unicode, err := idna.Display.ToUnicode(handle)
	if err != nil {
		return handle
	}
	return unicode
}
