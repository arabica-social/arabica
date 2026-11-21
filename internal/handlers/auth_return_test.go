package handlers

import "testing"

func TestAuthReturnPath(t *testing.T) {
	for _, path := range []string{"/", "/settings", "/?resume_reauth=1", "/brews/new?recipe=demo#results"} {
		if got := authReturnPath(path, "/fallback"); got != path {
			t.Errorf("authReturnPath(%q) = %q, want the input path", path, got)
		}
	}
	for _, path := range []string{"", "settings", "https://example.com/", "//example.com/", `/settings\example.com`, "/bad%escape"} {
		if got := authReturnPath(path, "/fallback"); got != "/fallback" {
			t.Errorf("authReturnPath(%q) = %q, want /fallback", path, got)
		}
	}
}
