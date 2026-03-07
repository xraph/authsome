package webhook

import (
	"context"

	"github.com/xraph/authsome/id"
)

// Store defines the persistence interface for webhook operations.
type Store interface {
	CreateWebhook(ctx context.Context, w *Webhook) error
	GetWebhook(ctx context.Context, webhookID id.WebhookID) (*Webhook, error)
	UpdateWebhook(ctx context.Context, w *Webhook) error
	DeleteWebhook(ctx context.Context, webhookID id.WebhookID) error
	ListWebhooks(ctx context.Context, appID id.AppID) ([]*Webhook, error)
}
