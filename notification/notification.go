// Package notification defines the notification domain entity and its store interface.
package notification

import (
	"time"

	"github.com/xraph/authsome/id"
)

// Notification represents a notification sent to a user.
type Notification struct {
	ID        id.NotificationID `json:"id"`
	AppID     id.AppID          `json:"app_id"`
	EnvID     id.EnvironmentID  `json:"env_id"`
	UserID    id.UserID         `json:"user_id"`
	Type      string            `json:"type"`
	Channel   Channel           `json:"channel"`
	Subject   string            `json:"subject,omitempty"`
	Body      string            `json:"body"`
	Sent      bool              `json:"sent"`
	SentAt    *time.Time        `json:"sent_at,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// Channel identifies the delivery mechanism.
type Channel string

const (
	ChannelEmail Channel = "email"
	ChannelSMS   Channel = "sms"
	ChannelPush  Channel = "push"
)
