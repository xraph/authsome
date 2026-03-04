package notification

import (
	"context"

	"github.com/xraph/authsome/id"
)

// Store defines the persistence interface for notification operations.
type Store interface {
	CreateNotification(ctx context.Context, n *Notification) error
	GetNotification(ctx context.Context, notifID id.NotificationID) (*Notification, error)
	MarkSent(ctx context.Context, notifID id.NotificationID) error
	ListUserNotifications(ctx context.Context, userID id.UserID) ([]*Notification, error)
}
