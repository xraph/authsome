package app

import (
	"context"

	"github.com/xraph/authsome/id"
)

// Store defines the persistence interface for app (tenant) operations.
type Store interface {
	CreateApp(ctx context.Context, a *App) error
	GetApp(ctx context.Context, appID id.AppID) (*App, error)
	GetAppBySlug(ctx context.Context, slug string) (*App, error)
	GetAppByPublishableKey(ctx context.Context, key string) (*App, error)
	UpdateApp(ctx context.Context, a *App) error
	DeleteApp(ctx context.Context, appID id.AppID) error
	ListApps(ctx context.Context) ([]*App, error)
	// GetPlatformApp returns the single platform app (is_platform=true).
	// Returns store.ErrNotFound if no platform app exists.
	GetPlatformApp(ctx context.Context) (*App, error)
}
