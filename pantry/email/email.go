// pantry/email/email.go
// Package email provides a simple interface for sending emails via SMTP.
// It wraps github.com/wneessen/go-mail with sensible defaults for typical
// web application use cases like password resets and email verification.
package email

import (
	"context"
	"fmt"
	"time"

	"github.com/wneessen/go-mail"
)

// Config holds SMTP server configuration.
type Config struct {
	// Host is the SMTP server hostname (e.g., "email-smtp.us-east-1.amazonaws.com")
	Host string

	// Port is the SMTP server port (typically 587 for STARTTLS, 465 for SSL)
	Port int

	// Username for SMTP authentication
	Username string

	// Password for SMTP authentication
	Password string

	// FromAddress is the default sender email address
	FromAddress string

	// FromName is the default sender display name (optional)
	FromName string

	// UseTLS enables STARTTLS (default: true, recommended for port 587)
	UseTLS bool

	// UseSSL enables implicit SSL/TLS (for port 465)
	UseSSL bool

	// Timeout for SMTP operations (default: 30 seconds)
	Timeout time.Duration
}

// Sender sends emails using the configured SMTP server.
type Sender struct {
	cfg Config
}

// NewSender creates a new email sender with the given configuration.
func NewSender(cfg Config) *Sender {
	if cfg.Port == 0 {
		cfg.Port = 587
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	// Default to TLS unless SSL is explicitly enabled
	if !cfg.UseSSL && cfg.Port != 465 {
		cfg.UseTLS = true
	}
	return &Sender{cfg: cfg}
}

// Message represents an email message to be sent.
type Message struct {
	To          []string // Recipient email addresses
	Subject     string   // Email subject line
	TextBody    string   // Plain text body (optional if HTMLBody is set)
	HTMLBody    string   // HTML body (optional if TextBody is set)
	ReplyTo     string   // Reply-To address (optional)
	Attachments []Attachment
}

// Attachment represents a file attachment.
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// Send sends an email message.
func (s *Sender) Send(ctx context.Context, msg Message) error {
	if len(msg.To) == 0 {
		return fmt.Errorf("email: no recipients specified")
	}
	if msg.TextBody == "" && msg.HTMLBody == "" {
		return fmt.Errorf("email: message body is empty")
	}

	m := mail.NewMsg()

	// Set sender
	if s.cfg.FromName != "" {
		if err := m.FromFormat(s.cfg.FromName, s.cfg.FromAddress); err != nil {
			return fmt.Errorf("email: invalid from address: %w", err)
		}
	} else {
		if err := m.From(s.cfg.FromAddress); err != nil {
			return fmt.Errorf("email: invalid from address: %w", err)
		}
	}

	// Set recipients
	if err := m.To(msg.To...); err != nil {
		return fmt.Errorf("email: invalid to address: %w", err)
	}

	// Set reply-to if specified
	if msg.ReplyTo != "" {
		if err := m.ReplyTo(msg.ReplyTo); err != nil {
			return fmt.Errorf("email: invalid reply-to address: %w", err)
		}
	}

	m.Subject(msg.Subject)

	// Set body content
	if msg.TextBody != "" && msg.HTMLBody != "" {
		m.SetBodyString(mail.TypeTextPlain, msg.TextBody)
		m.AddAlternativeString(mail.TypeTextHTML, msg.HTMLBody)
	} else if msg.HTMLBody != "" {
		m.SetBodyString(mail.TypeTextHTML, msg.HTMLBody)
	} else {
		m.SetBodyString(mail.TypeTextPlain, msg.TextBody)
	}

	// Add attachments
	for _, att := range msg.Attachments {
		m.AttachReader(att.Filename, bytesReader{data: att.Data})
	}

	// Build client options
	opts := []mail.Option{
		mail.WithPort(s.cfg.Port),
		mail.WithTimeout(s.cfg.Timeout),
	}

	if s.cfg.Username != "" {
		opts = append(opts, mail.WithSMTPAuth(mail.SMTPAuthPlain))
		opts = append(opts, mail.WithUsername(s.cfg.Username))
		opts = append(opts, mail.WithPassword(s.cfg.Password))
	}

	if s.cfg.UseSSL {
		opts = append(opts, mail.WithSSL())
	} else if s.cfg.UseTLS {
		opts = append(opts, mail.WithTLSPortPolicy(mail.TLSMandatory))
	}

	// Create client and send
	c, err := mail.NewClient(s.cfg.Host, opts...)
	if err != nil {
		return fmt.Errorf("email: failed to create client: %w", err)
	}

	if err := c.DialAndSendWithContext(ctx, m); err != nil {
		return fmt.Errorf("email: failed to send: %w", err)
	}

	return nil
}

// SendSimple sends a simple text email with minimal configuration.
func (s *Sender) SendSimple(ctx context.Context, to, subject, body string) error {
	return s.Send(ctx, Message{
		To:       []string{to},
		Subject:  subject,
		TextBody: body,
	})
}

// SendHTML sends an HTML email with a plain text fallback.
func (s *Sender) SendHTML(ctx context.Context, to, subject, textBody, htmlBody string) error {
	return s.Send(ctx, Message{
		To:       []string{to},
		Subject:  subject,
		TextBody: textBody,
		HTMLBody: htmlBody,
	})
}

// bytesReader wraps a byte slice to implement io.Reader
type bytesReader struct {
	data []byte
	pos  int
}

func (r bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, fmt.Errorf("EOF")
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
