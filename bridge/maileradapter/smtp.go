package maileradapter

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"github.com/xraph/authsome/bridge"
)

// SMTPMailer delivers email via standard SMTP.
type SMTPMailer struct {
	host     string
	port     string
	username string
	password string
	fromAddr string
	useTLS   bool
}

// SMTPOption configures the SMTP mailer.
type SMTPOption func(*SMTPMailer)

// WithSMTPTLS enables TLS for the SMTP connection.
func WithSMTPTLS(useTLS bool) SMTPOption {
	return func(m *SMTPMailer) { m.useTLS = useTLS }
}

// NewSMTPMailer creates a Mailer backed by standard SMTP.
func NewSMTPMailer(host, port, username, password, fromAddr string, opts ...SMTPOption) *SMTPMailer {
	m := &SMTPMailer{
		host:     host,
		port:     port,
		username: username,
		password: password,
		fromAddr: fromAddr,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

var _ bridge.Mailer = (*SMTPMailer)(nil)

// SendEmail delivers a message via SMTP.
func (m *SMTPMailer) SendEmail(ctx context.Context, msg *bridge.EmailMessage) error {
	from := msg.From
	if from == "" {
		from = m.fromAddr
	}

	addr := net.JoinHostPort(m.host, m.port)

	// Build RFC 2822 message
	var body strings.Builder
	body.WriteString("From: " + from + "\r\n")
	body.WriteString("To: " + strings.Join(msg.To, ", ") + "\r\n")
	body.WriteString("Subject: " + msg.Subject + "\r\n")
	body.WriteString("MIME-Version: 1.0\r\n")

	content := msg.Text
	contentType := "text/plain"
	if msg.HTML != "" {
		content = msg.HTML
		contentType = "text/html"
	}
	body.WriteString("Content-Type: " + contentType + "; charset=UTF-8\r\n")
	body.WriteString("\r\n")
	body.WriteString(content)

	var auth smtp.Auth
	if m.username != "" {
		auth = smtp.PlainAuth("", m.username, m.password, m.host)
	}

	if m.useTLS {
		return m.sendWithTLS(ctx, addr, from, msg.To, body.String(), auth)
	}

	if err := smtp.SendMail(addr, auth, from, msg.To, []byte(body.String())); err != nil {
		return fmt.Errorf("smtp: send mail: %w", err)
	}
	return nil
}

// sendWithTLS establishes a TLS connection and sends the email.
func (m *SMTPMailer) sendWithTLS(ctx context.Context, addr, from string, to []string, body string, auth smtp.Auth) error {
	tlsConfig := &tls.Config{
		ServerName: m.host,
		MinVersion: tls.VersionTLS12,
	}

	dialer := &tls.Dialer{Config: tlsConfig}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp: tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return fmt.Errorf("smtp: new client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if authErr := client.Auth(auth); authErr != nil {
			return fmt.Errorf("smtp: auth: %w", authErr)
		}
	}

	if mailErr := client.Mail(from); mailErr != nil {
		return fmt.Errorf("smtp: mail from: %w", mailErr)
	}
	for _, recipient := range to {
		if rcptErr := client.Rcpt(recipient); rcptErr != nil {
			return fmt.Errorf("smtp: rcpt to %s: %w", recipient, rcptErr)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp: data: %w", err)
	}
	if _, err := w.Write([]byte(body)); err != nil {
		return fmt.Errorf("smtp: write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp: close data: %w", err)
	}

	return client.Quit()
}
