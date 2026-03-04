package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/notification"
	"github.com/xraph/authsome/store"
)

// CreateNotification persists a new notification.
func (s *Store) CreateNotification(ctx context.Context, n *notification.Notification) error {
	m := toNotificationModel(n)

	_, err := s.mdb.NewInsert(m).Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: create notification: %w", err)
	}

	return nil
}

// GetNotification returns a notification by ID.
func (s *Store) GetNotification(ctx context.Context, notifID id.NotificationID) (*notification.Notification, error) {
	var m notificationModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"_id": notifID.String()}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get notification: %w", err)
	}

	return fromNotificationModel(&m)
}

// MarkSent marks a notification as sent.
func (s *Store) MarkSent(ctx context.Context, notifID id.NotificationID) error {
	t := now()

	res, err := s.mdb.NewUpdate((*notificationModel)(nil)).
		Filter(bson.M{"_id": notifID.String()}).
		Set("sent", true).
		Set("sent_at", t).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: mark sent: %w", err)
	}

	if res.MatchedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// ListUserNotifications returns all notifications for a user, ordered by creation date descending.
func (s *Store) ListUserNotifications(ctx context.Context, userID id.UserID) ([]*notification.Notification, error) {
	var models []notificationModel

	err := s.mdb.NewFind(&models).
		Filter(bson.M{"user_id": userID.String()}).
		Sort(bson.D{{Key: "created_at", Value: -1}}).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("authsome/mongo: list user notifications: %w", err)
	}

	result := make([]*notification.Notification, 0, len(models))

	for i := range models {
		n, err := fromNotificationModel(&models[i])
		if err != nil {
			return nil, err
		}

		result = append(result, n)
	}

	return result, nil
}
