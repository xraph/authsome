package maileradapter_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/bridge/maileradapter"
)

func TestResendMailer_SendEmail_Success(t *testing.T) {
	var receivedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/emails", r.URL.Path)
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"msg_123"}`))
	}))
	defer server.Close()

	mailer := maileradapter.NewResendMailer(
		"test-api-key",
		"noreply@example.com",
		maileradapter.WithResendBaseURL(server.URL),
	)

	err := mailer.SendEmail(context.Background(), &bridge.EmailMessage{
		To:      []string{"user@example.com"},
		Subject: "Welcome!",
		HTML:    "<h1>Hello</h1>",
	})
	require.NoError(t, err)

	assert.Equal(t, "noreply@example.com", receivedBody["from"])
	assert.Equal(t, "Welcome!", receivedBody["subject"])
	assert.Equal(t, "<h1>Hello</h1>", receivedBody["html"])
	assert.Equal(t, []any{"user@example.com"}, receivedBody["to"])
}

func TestResendMailer_SendEmail_OverrideFrom(t *testing.T) {
	var receivedFrom string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		_ = json.Unmarshal(body, &req)
		receivedFrom = req["from"].(string)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mailer := maileradapter.NewResendMailer(
		"key",
		"default@example.com",
		maileradapter.WithResendBaseURL(server.URL),
	)

	err := mailer.SendEmail(context.Background(), &bridge.EmailMessage{
		To:      []string{"user@example.com"},
		From:    "custom@example.com",
		Subject: "Test",
	})
	require.NoError(t, err)
	assert.Equal(t, "custom@example.com", receivedFrom)
}

func TestResendMailer_SendEmail_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":"invalid_api_key"}`))
	}))
	defer server.Close()

	mailer := maileradapter.NewResendMailer(
		"bad-key",
		"noreply@example.com",
		maileradapter.WithResendBaseURL(server.URL),
	)

	err := mailer.SendEmail(context.Background(), &bridge.EmailMessage{
		To:      []string{"user@example.com"},
		Subject: "Test",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API error 422")
}

func TestSMTPMailer_ImplementsMailer(t *testing.T) {
	// Just verify the type satisfies the interface at compile time.
	var _ bridge.Mailer = maileradapter.NewSMTPMailer("localhost", "587", "", "", "noreply@example.com")
}
