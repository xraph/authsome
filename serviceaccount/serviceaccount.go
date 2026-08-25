// Package serviceaccount defines the service account domain entity and its store interface.
// Service accounts are non-human principals used for machine-to-machine authentication,
// providing a first-class alternative to impersonating fake user rows.
package serviceaccount

import (
	"context"
	"errors"
	"time"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
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

	// Kind is which sort of non-human principal this is. Rows written before
	// the column existed carry the empty string, which reads as
	// principal.KindService.
	Kind principal.Kind `json:"kind,omitempty"`
	// OwnerUserID is the human answerable for this principal. Zero for a
	// workload that nobody owns personally, such as a CI runner.
	OwnerUserID id.UserID `json:"owner_user_id,omitempty"`
	// ParentID is the registered principal that minted this one. Set only on
	// ephemeral children.
	ParentID id.ServiceAccountID `json:"parent_id,omitempty"`
	// ExpiresAt is a hard cutoff. Nil means durable.
	ExpiresAt *time.Time       `json:"expires_at,omitempty"`
	OrgID     id.OrgID         `json:"org_id,omitempty"`
	EnvID     id.EnvironmentID `json:"env_id,omitempty"`
}

// ToPrincipal renders svc as a resolved principal.
//
// An empty Kind reads as service_account, matching rows written before the
// column existed. That fallback lives here so the four store backends cannot
// each pick a different one.
func (svc *ServiceAccount) ToPrincipal() *principal.Principal {
	kind := svc.Kind
	if kind == "" {
		kind = principal.KindService
	}
	p := &principal.Principal{
		Ref:       principal.Ref{Kind: kind, ID: svc.ID.String()},
		AppID:     svc.AppID,
		OrgID:     svc.OrgID,
		EnvID:     svc.EnvID,
		Name:      svc.Name,
		Scopes:    svc.Scopes,
		ExpiresAt: svc.ExpiresAt,
		Disabled:  !svc.Active,
	}
	if !svc.OwnerUserID.IsNil() {
		owner := principal.UserRef(svc.OwnerUserID)
		p.Owner = &owner
	}
	if !svc.ParentID.IsNil() {
		parent := principal.Ref{Kind: kind, ID: svc.ParentID.String()}
		p.Parent = &parent
	}
	return p
}

// Query contains filters for listing service accounts.
type Query struct {
	AppID      id.AppID
	Active     *bool
	Kind       principal.Kind
	ActiveOnly bool
	Limit      int
	Cursor     string
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
