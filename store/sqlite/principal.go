package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
)

// GetPrincipal is a stub — not yet implemented for the SQLite backend.
func (s *Store) GetPrincipal(_ context.Context, _ principal.Ref) (*principal.Principal, error) {
	return nil, fmt.Errorf("sqlite: GetPrincipal: not implemented")
}

// ListPrincipals is a stub — not yet implemented for the SQLite backend.
func (s *Store) ListPrincipals(_ context.Context, _ *principal.Query) ([]*principal.Principal, error) {
	return nil, fmt.Errorf("sqlite: ListPrincipals: not implemented")
}

// CreateDelegation is a stub — not yet implemented for the SQLite backend.
func (s *Store) CreateDelegation(_ context.Context, _ *principal.Delegation) error {
	return fmt.Errorf("sqlite: CreateDelegation: not implemented")
}

// GetDelegation is a stub — not yet implemented for the SQLite backend.
func (s *Store) GetDelegation(_ context.Context, _ id.DelegationID) (*principal.Delegation, error) {
	return nil, fmt.Errorf("sqlite: GetDelegation: not implemented")
}

// FindActiveDelegation is a stub — not yet implemented for the SQLite backend.
func (s *Store) FindActiveDelegation(
	_ context.Context, _ id.AppID, _, _ principal.Ref, _ principal.GrantKind,
) (*principal.Delegation, error) {
	return nil, fmt.Errorf("sqlite: FindActiveDelegation: not implemented")
}

// ListDelegations is a stub — not yet implemented for the SQLite backend.
func (s *Store) ListDelegations(_ context.Context, _ *principal.DelegationQuery) ([]*principal.Delegation, error) {
	return nil, fmt.Errorf("sqlite: ListDelegations: not implemented")
}

// RevokeDelegation is a stub — not yet implemented for the SQLite backend.
func (s *Store) RevokeDelegation(_ context.Context, _ id.DelegationID, _ time.Time) error {
	return fmt.Errorf("sqlite: RevokeDelegation: not implemented")
}
