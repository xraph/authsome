// Package app defines the multi-app (tenant) domain entity and its store interface.
package app

import (
	"time"

	"github.com/xraph/authsome/id"
)

// App represents a platform-level tenant. Each app has its own user pool,
// configuration, and authentication settings.
type App struct {
	ID         id.AppID  `json:"id"`
	Name       string    `json:"name"`
	Slug       string    `json:"slug"`
	Logo       string    `json:"logo,omitempty"`
	IsPlatform bool      `json:"is_platform"`
	Metadata   Metadata  `json:"metadata,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Metadata holds arbitrary app metadata as typed key-value pairs.
type Metadata map[string]string
