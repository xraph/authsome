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
	UpdateApp(ctx context.Context, a *App) error
	DeleteApp(ctx context.Context, appID id.AppID) error
	ListApps(ctx context.Context) ([]*App, error)
}
