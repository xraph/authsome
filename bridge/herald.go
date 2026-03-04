package bridge

import (
	"context"
	"errors"
)

// ErrHeraldNotAvailable is returned when Herald is not configured.
var ErrHeraldNotAvailable = errors.New("authsome: herald not available")

// Herald is the bridge interface for the Herald notification system.
// It provides a unified API for sending notifications across multiple channels.
type Herald interface {
	// Send sends a notification via a specific channel.
	Send(ctx context.Context, req *HeraldSendRequest) error
	// Notify sends a notification across multiple channels using a template.
	Notify(ctx context.Context, req *HeraldNotifyRequest) error
}

// HeraldSendRequest describes a notification to send on a single channel.
type HeraldSendRequest struct {
	AppID    string            `json:"app_id"`
	EnvID    string            `json:"env_id,omitempty"`
	OrgID    string            `json:"org_id,omitempty"`
	UserID   string            `json:"user_id,omitempty"`
	Channel  string            `json:"channel"`
	Template string            `json:"template"`
	Locale   string            `json:"locale,omitempty"`
	To       []string          `json:"to"`
	Data     map[string]any    `json:"data,omitempty"`
	Async    bool              `json:"async,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// HeraldNotifyRequest describes a multi-channel notification using a template.
type HeraldNotifyRequest struct {
	AppID    string            `json:"app_id"`
	EnvID    string            `json:"env_id,omitempty"`
	OrgID    string            `json:"org_id,omitempty"`
	UserID   string            `json:"user_id,omitempty"`
	Template string            `json:"template"`
	Locale   string            `json:"locale,omitempty"`
	To       []string          `json:"to"`
	Data     map[string]any    `json:"data,omitempty"`
	Channels []string          `json:"channels"`
	Async    bool              `json:"async,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}
