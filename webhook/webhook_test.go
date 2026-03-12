package webhook

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/id"
)

func TestWebhook_FieldsPopulated(t *testing.T) {
	w := &Webhook{
		ID:        id.NewWebhookID(),
		AppID:     id.NewAppID(),
		URL:       "https://example.com/webhook",
		Events:    []string{"user.created", "auth.signin"},
		Secret:    "whsec_test123",
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	assert.NotEmpty(t, w.ID.String())
	assert.NotEmpty(t, w.AppID.String())
	assert.Equal(t, "https://example.com/webhook", w.URL)
	assert.Equal(t, []string{"user.created", "auth.signin"}, w.Events)
	assert.Equal(t, "whsec_test123", w.Secret)
	assert.True(t, w.Active)
	assert.False(t, w.CreatedAt.IsZero())
	assert.False(t, w.UpdatedAt.IsZero())
}

func TestWebhook_EmptyEvents(t *testing.T) {
	w := &Webhook{
		ID:    id.NewWebhookID(),
		AppID: id.NewAppID(),
		URL:   "https://example.com/webhook",
	}
	assert.NotEmpty(t, w.ID.String())
	assert.NotEmpty(t, w.AppID.String())
	assert.Equal(t, "https://example.com/webhook", w.URL)
	assert.Nil(t, w.Events)
	assert.False(t, w.Active)
}

func TestWebhook_MultipleEvents(t *testing.T) {
	events := []string{"user.created", "user.deleted", "auth.signin", "session.created"}
	w := &Webhook{
		ID:     id.NewWebhookID(),
		Events: events,
	}
	assert.NotEmpty(t, w.ID.String())
	assert.Len(t, w.Events, 4)
	assert.Contains(t, w.Events, "user.created")
	assert.Contains(t, w.Events, "session.created")
}
