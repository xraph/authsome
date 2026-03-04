package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/webhook"
)

// CreateWebhook persists a new webhook.
func (s *Store) CreateWebhook(ctx context.Context, w *webhook.Webhook) error {
	m := toWebhookModel(w)

	_, err := s.mdb.NewInsert(m).Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: create webhook: %w", err)
	}

	return nil
}

// GetWebhook returns a webhook by ID.
func (s *Store) GetWebhook(ctx context.Context, webhookID id.WebhookID) (*webhook.Webhook, error) {
	var m webhookModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"_id": webhookID.String()}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get webhook: %w", err)
	}

	return fromWebhookModel(&m)
}

// UpdateWebhook modifies an existing webhook.
func (s *Store) UpdateWebhook(ctx context.Context, w *webhook.Webhook) error {
	m := toWebhookModel(w)
	m.UpdatedAt = now()

	res, err := s.mdb.NewUpdate(m).
		Filter(bson.M{"_id": m.ID}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: update webhook: %w", err)
	}

	if res.MatchedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// DeleteWebhook removes a webhook.
func (s *Store) DeleteWebhook(ctx context.Context, webhookID id.WebhookID) error {
	res, err := s.mdb.NewDelete((*webhookModel)(nil)).
		Filter(bson.M{"_id": webhookID.String()}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: delete webhook: %w", err)
	}

	if res.DeletedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// ListWebhooks returns all webhooks for an app, ordered by creation date descending.
func (s *Store) ListWebhooks(ctx context.Context, appID id.AppID) ([]*webhook.Webhook, error) {
	var models []webhookModel

	err := s.mdb.NewFind(&models).
		Filter(bson.M{"app_id": appID.String()}).
		Sort(bson.D{{Key: "created_at", Value: -1}}).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("authsome/mongo: list webhooks: %w", err)
	}

	result := make([]*webhook.Webhook, 0, len(models))

	for i := range models {
		w, err := fromWebhookModel(&models[i])
		if err != nil {
			return nil, err
		}

		result = append(result, w)
	}

	return result, nil
}
