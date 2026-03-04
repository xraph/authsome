// Package store defines the aggregate persistence interface. Each subsystem
// (user, session, account, app, organization, device, webhook, notification,
// environment) defines its own store interface. The composite Store composes
// them all. Backends: PostgreSQL, SQLite, MongoDB, and Memory.
package store

import (
	"context"
	"errors"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/appsessionconfig"
	"github.com/xraph/authsome/device"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/notification"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
	"github.com/xraph/authsome/webhook"

	"github.com/xraph/grove/migrate"
)

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("authsome: not found")

// Store is the aggregate persistence interface.
// Each subsystem store is a composable interface.
// A single backend (postgres, sqlite, mongo, memory) implements all of them.
type Store interface {
	user.Store
	session.Store
	account.Store
	app.Store
	organization.Store
	device.Store
	webhook.Store
	notification.Store
	apikey.Store
	environment.Store
	formconfig.Store
	formconfig.BrandingStore
	appsessionconfig.Store

	// Migrate runs all schema migrations. Extra migration groups (e.g. from
	// plugins) are appended to the core group and orchestrated together.
	Migrate(ctx context.Context, extraGroups ...*migrate.Group) error

	// Ping checks database connectivity.
	Ping(ctx context.Context) error

	// Close closes the store connection.
	Close() error
}
