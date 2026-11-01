package ai

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"syscall"
)

// ErrRequestTooLarge means the configured provider cap rejected the request locally.
var ErrRequestTooLarge = errors.New("ai: request exceeds configured provider limit")

// FailureDetails contains only application-defined categories and numeric status.
// It never contains URLs, credentials, headers, prompts, or provider error bodies.
type FailureDetails struct {
	Category   string
	HTTPStatus int
}
type failure struct {
	details FailureDetails
	message string
}

func (f *failure) Error() string { return f.message }
func failed(category, message string, status int) error {
	return &failure{details: FailureDetails{Category: category, HTTPStatus: status}, message: message}
}
func DescribeFailure(err error) FailureDetails {
	var f *failure
	switch {
	case errors.As(err, &f):
		return f.details
	case errors.Is(err, context.DeadlineExceeded):
		return FailureDetails{Category: "timeout"}
	case errors.Is(err, context.Canceled):
		return FailureDetails{Category: "canceled"}
	default:
		return FailureDetails{Category: "unknown"}
	}
}
func networkFailure(err error) error {
	var record tls.RecordHeaderError
	var certificate *tls.CertificateVerificationError
	var authority x509.UnknownAuthorityError
	var dns *net.DNSError
	var timeout net.Error
	switch {
	case errors.Is(err, http.ErrSchemeMismatch):
		return failed("tls_protocol_mismatch", "ai: provider TLS handshake received plaintext HTTP; check base_url and proxy scheme", 0)
	case errors.As(err, &record):
		if strings.HasPrefix(string(record.RecordHeader[:]), "HTTP/") {
			return failed("tls_protocol_mismatch", "ai: provider TLS handshake received plaintext HTTP; check base_url and proxy scheme", 0)
		}
		return failed("tls_handshake", "ai: provider TLS handshake failed", 0)
	case errors.As(err, &certificate), errors.As(err, &authority):
		return failed("tls_certificate", "ai: provider TLS certificate verification failed", 0)
	case errors.As(err, &dns):
		return failed("dns", "ai: provider DNS lookup failed", 0)
	case errors.As(err, &timeout) && timeout.Timeout():
		return failed("timeout", "ai: provider connection timed out", 0)
	case errors.Is(err, syscall.ECONNREFUSED):
		return failed("connection_refused", "ai: provider request failed (connection refused)", 0)
	case errors.Is(err, syscall.ENETUNREACH), errors.Is(err, syscall.EHOSTUNREACH):
		return failed("network_unreachable", "ai: provider network is unreachable", 0)
	case errors.Is(err, syscall.EACCES), errors.Is(err, syscall.EPERM):
		return failed("network_denied", "ai: provider network access denied", 0)
	default:
		return failed("connection", "ai: provider request failed", 0)
	}
}
func upstreamHTTPFailure(status int) error {
	return failed("upstream_http", fmt.Sprintf("ai: provider returned HTTP %d", status), status)
}
