package email

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// Config holds SMTP configuration for sending email.
type Config struct {
	Host       string
	Port       int
	User       string
	Pass       string
	From       string
	AdminEmail string
}

// Sender sends email via SMTP.
type Sender struct {
	cfg Config
}

// NewSender creates a new email Sender.
func NewSender(cfg Config) *Sender {
	return &Sender{cfg: cfg}
}

// Enabled returns true if SMTP is configured.
func (s *Sender) Enabled() bool {
	return s.cfg.Host != ""
}

// AdminEmail returns the configured admin email address.
func (s *Sender) AdminEmail() string {
	return s.cfg.AdminEmail
}

// Send sends an email to the given recipient.
func (s *Sender) Send(to, subject, body string) error {
	if !s.Enabled() {
		return nil
	}

	addr := net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.Port))

	// Extract domain from From address for Message-ID
	domain := s.cfg.Host
	if parts := strings.SplitN(s.cfg.From, "@", 2); len(parts) == 2 {
		domain = parts[1]
	}

	// Generate a random Message-ID
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	messageID := fmt.Sprintf("<%x.%d@%s>", randBytes, time.Now().UnixNano(), domain)

	msg := fmt.Sprintf("From: Arabica <%s>\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMessage-ID: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.cfg.From, to, subject, time.Now().UTC().Format(time.RFC1123Z), messageID, body)

	var auth smtp.Auth
	if s.cfg.User != "" {
		auth = smtp.PlainAuth("", s.cfg.User, s.cfg.Pass, s.cfg.Host)
	}

	if s.cfg.Port == 465 {
		return s.sendImplicitTLS(addr, auth, msg, to)
	}
	return s.sendSTARTTLS(addr, auth, msg, to)
}

// sendImplicitTLS connects over TLS directly (port 465).
func (s *Sender) sendImplicitTLS(addr string, auth smtp.Auth, msg, to string) error {
	tlsConfig := &tls.Config{ServerName: s.cfg.Host}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer client.Close()

	return s.sendWithClient(client, auth, msg, to)
}

// sendSTARTTLS connects in plaintext then upgrades via STARTTLS.
func (s *Sender) sendSTARTTLS(addr string, auth smtp.Auth, msg, to string) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{ServerName: s.cfg.Host}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}

	return s.sendWithClient(client, auth, msg, to)
}

// sendWithClient performs the SMTP conversation on an established client.
func (s *Sender) sendWithClient(client *smtp.Client, auth smtp.Auth, msg, to string) error {
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(s.cfg.From); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	return client.Quit()
}
