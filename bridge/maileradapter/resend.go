// Package maileradapter provides bridge.Mailer implementations for
// transactional email delivery (Resend HTTP API and standard SMTP).
package maileradapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/xraph/authsome/bridge"
)

// ResendMailer delivers email via the Resend HTTP API.
type ResendMailer struct {
	apiKey     string
	fromAddr   string
	httpClient *http.Client
	baseURL    string
}

// ResendOption configures the Resend mailer.
type ResendOption func(*ResendMailer)

// WithResendHTTPClient sets a custom HTTP client (useful for testing).
func WithResendHTTPClient(c *http.Client) ResendOption {
	return func(m *ResendMailer) { m.httpClient = c }
}

// WithResendBaseURL overrides the Resend API base URL (useful for testing).
func WithResendBaseURL(url string) ResendOption {
	return func(m *ResendMailer) { m.baseURL = url }
}

// NewResendMailer creates a Mailer backed by the Resend HTTP API.
func NewResendMailer(apiKey, fromAddr string, opts ...ResendOption) *ResendMailer {
	m := &ResendMailer{
		apiKey:   apiKey,
		fromAddr: fromAddr,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://api.resend.com",
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

var _ bridge.Mailer = (*ResendMailer)(nil)

// resendRequest is the JSON body for the Resend /emails endpoint.
type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html,omitempty"`
	Text    string   `json:"text,omitempty"`
}

// SendEmail delivers a message via the Resend API.
func (m *ResendMailer) SendEmail(ctx context.Context, msg *bridge.EmailMessage) error {
	from := msg.From
	if from == "" {
		from = m.fromAddr
	}

	body := resendRequest{
		From:    from,
		To:      msg.To,
		Subject: msg.Subject,
		HTML:    msg.HTML,
		Text:    msg.Text,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("resend: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+"/emails", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("resend: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("resend: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024)) //nolint:errcheck // best-effort read
		return fmt.Errorf("resend: API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
