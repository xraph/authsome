// Package serviceaccount defines the service account domain entity and its store interface.
// Service accounts are non-human principals used for machine-to-machine authentication,
// providing a first-class alternative to impersonating fake user rows.
package serviceaccount

import (
	"context"
	"errors"
	"time"

	"github.com/xraph/authsome/id"
)

// ErrNotFound is returned when a service account cannot be found.
var ErrNotFound = errors.New("serviceaccount: not found")

// ServiceAccount is a non-human principal for machine-to-machine authentication.
type ServiceAccount struct {
	ID          id.ServiceAccountID `json:"id"`
	AppID       id.AppID            `json:"app_id"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Scopes      []string            `json:"scopes,omitempty"`
	Active      bool                `json:"active"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// Query contains filters for listing service accounts.
type Query struct {
	AppID  id.AppID
	Active *bool
	Limit  int
	Cursor string
}

// List is the result of a service account listing query.
type List struct {
	ServiceAccounts []*ServiceAccount `json:"service_accounts"`
	NextCursor      string            `json:"next_cursor,omitempty"`
	Total           int               `json:"total"`
}

// Store is the persistence interface for service accounts.
type Store interface {
	// CreateServiceAccount stores a new service account.
	CreateServiceAccount(ctx context.Context, svc *ServiceAccount) error

	// GetServiceAccount returns a service account by ID.
	GetServiceAccount(ctx context.Context, svcID id.ServiceAccountID) (*ServiceAccount, error)

	// ListServiceAccounts returns service accounts matching the query.
	ListServiceAccounts(ctx context.Context, q *Query) (*List, error)

	// UpdateServiceAccount updates an existing service account.
	UpdateServiceAccount(ctx context.Context, svc *ServiceAccount) error

	// DeleteServiceAccount permanently deletes a service account.
	DeleteServiceAccount(ctx context.Context, svcID id.ServiceAccountID) error
}
