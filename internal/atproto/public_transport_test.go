package atproto

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"tangled.org/pdewey.com/atp"
)

func TestPublicClientPrivateDestination(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if !strings.HasPrefix(r.Header.Get("User-Agent"), userAgent) {
			t.Error("missing Arabica User-Agent")
		}
		io.WriteString(w, `{"did":"did:plc:test","handle":"test.example"}`)
	}))
	defer server.Close()
	old := atp.PublicAPIBase
	atp.PublicAPIBase = server.URL
	t.Cleanup(func() { atp.PublicAPIBase = old })

	t.Run("secure by default", func(t *testing.T) {
		t.Setenv("ARABICA_ALLOW_PRIVATE_NETWORKS", "")
		_, err := NewPublicClient().GetProfile(context.Background(), "test.example")
		if err == nil || !strings.Contains(err.Error(), "private destination blocked") {
			t.Fatalf("expected private destination blocked, got %v (server hits: %d)", err, hits.Load())
		}
		if hits.Load() != 0 {
			t.Fatal("private server was reached")
		}
	})
	t.Run("explicit local development opt out", func(t *testing.T) {
		t.Setenv("ARABICA_ALLOW_PRIVATE_NETWORKS", "true")
		profile, err := NewPublicClient().GetProfile(context.Background(), "test.example")
		if err != nil {
			t.Fatal(err)
		}
		if profile.DID != "did:plc:test" {
			t.Fatalf("unexpected profile: %+v", profile)
		}
	})
}

func TestPrivateDestination(t *testing.T) {
	tests := []struct {
		ip      string
		blocked bool
	}{
		{"127.0.0.1", true}, {"127.42.1.2", true},
		{"10.0.0.1", true}, {"172.16.0.1", true}, {"172.31.255.255", true}, {"192.168.1.1", true},
		{"169.254.1.1", true}, {"169.254.169.254", true},
		{"100.64.0.0", true}, {"100.127.255.255", true},
		{"0.0.0.0", true}, {"::", true}, {"::1", true},
		{"fc00::1", true}, {"fdff::1", true}, {"fe80::1", true}, {"fe80::1%eth0", true},
		{"::ffff:127.0.0.1", true}, {"::ffff:10.0.0.1", true}, {"::ffff:100.64.0.1", true},
		{"224.0.0.1", true}, {"ff02::1", true},
		{"2002:7f00:1::", true}, {"2002:0a00:0001::", true}, {"2002:808:808::", true},
		{"2001:0:4136:e378::", true}, {"64:ff9b::7f00:1", true}, {"64:ff9b::808:808", true},
		{"8.8.8.8", false}, {"1.1.1.1", false}, {"2606:4700:4700::1111", false},
		{"::ffff:8.8.8.8", false},
		{"100.63.255.255", false}, {"100.128.0.0", false}, {"172.15.255.255", false}, {"172.32.0.0", false},
	}
	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := netip.MustParseAddr(tt.ip)
			if got := privateDestination(ip); got != tt.blocked {
				t.Fatalf("privateDestination(%s) = %v, want %v", ip, got, tt.blocked)
			}
			err := guardPublicDial("tcp", net.JoinHostPort(tt.ip, "443"), nil)
			if (err != nil) != tt.blocked {
				t.Fatalf("guardPublicDial(%s): %v", ip, err)
			}
		})
	}
}

func TestPublicClientResolvedLoopback(t *testing.T) {
	t.Setenv("ARABICA_ALLOW_PRIVATE_NETWORKS", "")
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer server.Close()
	old := atp.PublicAPIBase
	atp.PublicAPIBase = strings.Replace(server.URL, "127.0.0.1", "localhost", 1)
	t.Cleanup(func() { atp.PublicAPIBase = old })
	_, err := NewPublicClient().GetProfile(context.Background(), "test.example")
	if err == nil || !strings.Contains(err.Error(), "private destination blocked") {
		t.Fatalf("expected resolved loopback blocked, got %v", err)
	}
	if hits.Load() != 0 {
		t.Fatal("resolved private server was reached")
	}
}

func TestPublicClientRefusesRedirects(t *testing.T) {
	// Allow the initial local test server, but redirects must remain forbidden
	// even in development mode. This uses the same redirect policy as production.
	t.Setenv("ARABICA_ALLOW_PRIVATE_NETWORKS", "true")
	var hits atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redirect.Close()
	old := atp.PublicAPIBase
	atp.PublicAPIBase = redirect.URL
	t.Cleanup(func() { atp.PublicAPIBase = old })
	_, err := NewPublicClient().GetProfile(context.Background(), "test.example")
	if err == nil || !strings.Contains(err.Error(), "public client redirects blocked") {
		t.Fatalf("expected redirect blocked, got %v", err)
	}
	if hits.Load() != 0 {
		t.Fatal("redirect target was reached")
	}
}

func TestPublicTransportDirectDialOnly(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://proxy.example:3128")
	t.Setenv("HTTPS_PROXY", "http://proxy.example:3128")
	transport := publicTransport(false)
	defer transport.CloseIdleConnections()
	if transport.Proxy != nil {
		t.Fatal("a proxy can bypass destination validation")
	}
	if transport.DialContext == nil {
		t.Fatal("missing guarded dialer")
	}
	for _, address := range []string{"localhost:443", "not-an-address", "[invalid]:443"} {
		if err := guardPublicDial("tcp", address, nil); err == nil {
			t.Fatalf("accepted unresolved or malformed destination %q", address)
		}
	}
}

// TestPublicClientLivePublicEndpoint is opt-in so ordinary tests remain offline.
func TestPublicClientLivePublicEndpoint(t *testing.T) {
	if os.Getenv("ATPROTO_TEST_LIVE_PUBLIC") != "true" {
		t.Skip("set ATPROTO_TEST_LIVE_PUBLIC=true to exercise the public Bluesky API")
	}
	t.Setenv("ARABICA_ALLOW_PRIVATE_NETWORKS", "")
	profile, err := NewPublicClient().GetProfile(context.Background(), "bsky.app")
	if err != nil {
		t.Fatal(err)
	}
	if profile.DID == "" || profile.Handle != "bsky.app" {
		t.Fatalf("unexpected public profile: %+v", profile)
	}
	t.Logf("public profile fetched with guard enabled: %s (%s)", profile.Handle, profile.DID)
}
