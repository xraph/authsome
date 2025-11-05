package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/xraph/authsome/core/notification"
)

// SMTPConfig holds SMTP configuration
type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	FromName string `json:"from_name"`
	UseTLS   bool   `json:"use_tls"`
}

// SMTPProvider implements notification.Provider for SMTP email
type SMTPProvider struct {
	config SMTPConfig
}

// NewSMTPProvider creates a new SMTP email provider
func NewSMTPProvider(config SMTPConfig) *SMTPProvider {
	return &SMTPProvider{
		config: config,
	}
}

// ID returns the provider ID
func (p *SMTPProvider) ID() string {
	return "smtp"
}

// Type returns the notification type this provider handles
func (p *SMTPProvider) Type() notification.NotificationType {
	return notification.NotificationTypeEmail
}

// Send sends an email notification
func (p *SMTPProvider) Send(ctx context.Context, notif *notification.Notification) error {
	// Validate recipient is an email address
	if !strings.Contains(notif.Recipient, "@") {
		return fmt.Errorf("invalid email address: %s", notif.Recipient)
	}

	// Create SMTP auth
	auth := smtp.PlainAuth("", p.config.Username, p.config.Password, p.config.Host)

	// Build message
	from := p.config.From
	if p.config.FromName != "" {
		from = fmt.Sprintf("%s <%s>", p.config.FromName, p.config.From)
	}

	msg := fmt.Sprintf("From: %s\r\n", from)
	msg += fmt.Sprintf("To: %s\r\n", notif.Recipient)
	msg += fmt.Sprintf("Subject: %s\r\n", notif.Subject)
	msg += "MIME-Version: 1.0\r\n"
	msg += "Content-Type: text/html; charset=UTF-8\r\n"
	msg += "\r\n"
	msg += notif.Body

	// Send email
	addr := fmt.Sprintf("%s:%d", p.config.Host, p.config.Port)

	if p.config.UseTLS {
		return p.sendWithTLS(addr, auth, p.config.From, []string{notif.Recipient}, []byte(msg))
	}

	return smtp.SendMail(addr, auth, p.config.From, []string{notif.Recipient}, []byte(msg))
}

// GetStatus gets the delivery status of a notification
func (p *SMTPProvider) GetStatus(ctx context.Context, providerID string) (notification.NotificationStatus, error) {
	// SMTP doesn't provide delivery status tracking
	return notification.NotificationStatusSent, nil
}

// ValidateConfig validates the provider configuration
func (p *SMTPProvider) ValidateConfig() error {
	return p.Validate()
}

// sendWithTLS sends email with TLS encryption
func (p *SMTPProvider) sendWithTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	// Create TLS config
	tlsConfig := &tls.Config{
		ServerName: p.config.Host,
	}

	// Connect to server
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect with TLS: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, p.config.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Quit()

	// Authenticate
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send message
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}
	defer writer.Close()

	_, err = writer.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// Validate validates the provider configuration
func (p *SMTPProvider) Validate() error {
	if p.config.Host == "" {
		return fmt.Errorf("SMTP host is required")
	}
	if p.config.Port == 0 {
		return fmt.Errorf("SMTP port is required")
	}
	if p.config.From == "" {
		return fmt.Errorf("from email address is required")
	}
	if !strings.Contains(p.config.From, "@") {
		return fmt.Errorf("invalid from email address")
	}
	return nil
}
