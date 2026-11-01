package atproto

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

var (
	carrierGradeNAT = netip.MustParsePrefix("100.64.0.0/10")
	// IPv6 transition ranges embed an arbitrary IPv4 destination, so a public-looking
	// address can still reach loopback or RFC1918 space. indigo's ssrf package excludes
	// 6to4 for the same reason; the others are the same shape.
	sixToFour = netip.MustParsePrefix("2002::/16")    // 6to4: IPv4 in bits 16-48
	teredo    = netip.MustParsePrefix("2001::/32")    // Teredo: IPv4 server in the address
	nat64     = netip.MustParsePrefix("64:ff9b::/96") // NAT64 well-known prefix: IPv4 in the low 32 bits
)

// privateDestination includes non-unicast addresses as well as private networks.
// Unmap prevents IPv4-mapped IPv6 addresses from bypassing IPv4 restrictions.
func privateDestination(ip netip.Addr) bool {
	ip = ip.Unmap()
	return !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() || carrierGradeNAT.Contains(ip) ||
		sixToFour.Contains(ip) || teredo.Contains(ip) || nat64.Contains(ip)
}

// guardPublicDial runs after DNS resolution, immediately before connecting to
// this exact IP. A hostname check followed by a separate dial would allow DNS
// rebinding to change the destination between validation and connection.
func guardPublicDial(network, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("invalid public destination: %w", err)
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return fmt.Errorf("invalid public destination IP: %w", err)
	}
	if privateDestination(ip) {
		return fmt.Errorf("private destination blocked: %s", ip)
	}
	return nil
}

func publicTransport(allowPrivate bool) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// A proxy could resolve the origin itself and bypass our destination guard.
	// Public fetches must connect directly, even when HTTP(S)_PROXY is configured.
	transport.Proxy = nil
	dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	if !allowPrivate {
		dialer.Control = guardPublicDial
	}
	transport.DialContext = dialer.DialContext
	return transport
}

var proxyWarning sync.Once

// warnProxyBypassed runs once per process. Public fetches must dial directly for
// the destination guard to see the address that is actually connected to, which
// means an operator-configured egress proxy is silently not used. Say it once,
// loudly, instead of letting every public fetch fail without explanation.
func warnProxyBypassed(allowPrivate bool) {
	if allowPrivate {
		return
	}
	proxyWarning.Do(func() {
		for _, name := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy", "ALL_PROXY", "all_proxy"} {
			if os.Getenv(name) != "" {
				log.Warn().Str("variable", name).
					Msg("public atproto fetches bypass the configured egress proxy so the destination guard cannot be evaded; direct egress is required")
			}
		}
	})
}
