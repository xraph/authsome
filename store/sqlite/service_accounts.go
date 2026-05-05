package sqlite

import (
	"context"
	"fmt"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/serviceaccount"
)

// CreateServiceAccount is a stub — not yet implemented for the SQLite backend.
func (s *Store) CreateServiceAccount(_ context.Context, _ *serviceaccount.ServiceAccount) error {
	return fmt.Errorf("sqlite: CreateServiceAccount: not implemented")
}

// GetServiceAccount is a stub — not yet implemented for the SQLite backend.
func (s *Store) GetServiceAccount(_ context.Context, _ id.ServiceAccountID) (*serviceaccount.ServiceAccount, error) {
	return nil, fmt.Errorf("sqlite: GetServiceAccount: not implemented")
}

// ListServiceAccounts is a stub — not yet implemented for the SQLite backend.
func (s *Store) ListServiceAccounts(_ context.Context, _ *serviceaccount.Query) (*serviceaccount.List, error) {
	return nil, fmt.Errorf("sqlite: ListServiceAccounts: not implemented")
}

// UpdateServiceAccount is a stub — not yet implemented for the SQLite backend.
func (s *Store) UpdateServiceAccount(_ context.Context, _ *serviceaccount.ServiceAccount) error {
	return fmt.Errorf("sqlite: UpdateServiceAccount: not implemented")
}

// DeleteServiceAccount is a stub — not yet implemented for the SQLite backend.
func (s *Store) DeleteServiceAccount(_ context.Context, _ id.ServiceAccountID) error {
	return fmt.Errorf("sqlite: DeleteServiceAccount: not implemented")
}
