package ai

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"syscall"
	"testing"
)

func TestSafeNetworkDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name     string
		err      error
		category string
	}{
		{"plaintext HTTP", tls.RecordHeaderError{RecordHeader: [5]byte{'H', 'T', 'T', 'P', '/'}}, "tls_protocol_mismatch"},
		{"TLS", tls.RecordHeaderError{}, "tls_handshake"},
		{"DNS", &net.DNSError{Err: "secret-value", Name: "secret-host"}, "dns"},
		{"refused", syscall.ECONNREFUSED, "connection_refused"},
		{"unreachable", syscall.ENETUNREACH, "network_unreachable"},
		{"denied", syscall.EACCES, "network_denied"},
		{"unknown", errors.New("secret-value"), "connection"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := networkFailure(&url.Error{Op: "Post", URL: "https://secret-host/secret-value", Err: tc.err})
			if DescribeFailure(err).Category != tc.category {
				t.Fatalf("wrong category: %s", DescribeFailure(err).Category)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("diagnostic leaks upstream details")
			}
		})
	}
	if DescribeFailure(context.DeadlineExceeded).Category != "timeout" || DescribeFailure(context.Canceled).Category != "canceled" {
		t.Fatal("context diagnostic missing")
	}
}
func TestTLSProtocolMismatchDiagnosis(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	client := testClient(t, testConfig(strings.Replace(server.URL, "http://", "https://", 1)))
	_, err := client.Complete(context.Background(), testMessages, nil, nil)
	if DescribeFailure(err).Category != "tls_protocol_mismatch" {
		t.Fatalf("wrong diagnostic: %v", err)
	}
}
