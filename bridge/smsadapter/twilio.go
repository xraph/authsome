// Package smsadapter provides SMS sender implementations.
package smsadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/xraph/authsome/bridge"
)

// TwilioSender implements bridge.SMSSender via the Twilio REST API.
type TwilioSender struct {
	accountSID string
	authToken  string
	fromNumber string
	client     *http.Client
}

// Compile-time check.
var _ bridge.SMSSender = (*TwilioSender)(nil)

// NewTwilioSender creates a Twilio SMS sender.
func NewTwilioSender(accountSID, authToken, fromNumber string) *TwilioSender {
	return &TwilioSender{
		accountSID: accountSID,
		authToken:  authToken,
		fromNumber: fromNumber,
		client:     &http.Client{},
	}
}

// SendSMS sends an SMS via the Twilio Messages API.
func (t *TwilioSender) SendSMS(ctx context.Context, msg *bridge.SMSMessage) error {
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", t.accountSID)

	data := url.Values{}
	data.Set("To", msg.To)
	data.Set("From", t.fromNumber)
	data.Set("Body", msg.Body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("sms/twilio: create request: %w", err)
	}

	req.SetBasicAuth(t.accountSID, t.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("sms/twilio: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var buf bytes.Buffer
		_ = json.NewEncoder(&buf).Encode(resp.Body)
		return fmt.Errorf("sms/twilio: API error (status %d)", resp.StatusCode)
	}

	return nil
}
