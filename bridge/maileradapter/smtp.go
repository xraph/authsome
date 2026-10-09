package maileradapter

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

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

	// Every SMTP conversation runs under a deadline: the request's own when
	// it has one, else defaultSMTPTimeout. A mail server that accepts the
	// connection and then says nothing must not hold a sign-up open.
	deadline := time.Now().Add(defaultSMTPTimeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	dialCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	var conn net.Conn
	var err error
	if m.useTLS {
		dialer := &tls.Dialer{NetDialer: &net.Dialer{}, Config: m.tlsConfig()}
		conn, err = dialer.DialContext(dialCtx, "tcp", addr)
	} else {
		conn, err = (&net.Dialer{}).DialContext(dialCtx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("smtp: dial: %w", err)
	}
	defer conn.Close()
	if deadlineErr := conn.SetDeadline(deadline); deadlineErr != nil {
		return fmt.Errorf("smtp: set deadline: %w", deadlineErr)
	}

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return fmt.Errorf("smtp: new client: %w", err)
	}
	defer client.Close()

	// A plain connection upgrades with STARTTLS when the server offers it,
	// as smtp.SendMail did before the deadline made a hand-rolled
	// conversation necessary.
	if !m.useTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if tlsErr := client.StartTLS(m.tlsConfig()); tlsErr != nil {
				return fmt.Errorf("smtp: starttls: %w", tlsErr)
			}
		}
	}
	return deliver(client, from, msg.To, body.String(), auth)
}

// defaultSMTPTimeout bounds an SMTP conversation whose context has no
// deadline of its own.
const defaultSMTPTimeout = 30 * time.Second

func (m *SMTPMailer) tlsConfig() *tls.Config {
	return &tls.Config{ServerName: m.host, MinVersion: tls.VersionTLS12}
}

// deliver runs the SMTP conversation on an open, deadline-bound client.
func deliver(client *smtp.Client, from string, to []string, body string, auth smtp.Auth) error {
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
