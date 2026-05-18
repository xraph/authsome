package authsome

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/serviceaccount"
	"github.com/xraph/authsome/store"
)

// CreateServiceAccount creates a new service account for an app.
func (e *Engine) CreateServiceAccount(ctx context.Context, appID id.AppID, name, description string, scopes []string) (*serviceaccount.ServiceAccount, error) {
	if err := e.requireStarted(); err != nil {
		return nil, err
	}
	if appID.IsNil() {
		return nil, fmt.Errorf("authsome: app_id is required")
	}
	if name == "" {
		return nil, fmt.Errorf("authsome: name is required")
	}

	now := time.Now()
	svc := &serviceaccount.ServiceAccount{
		ID:          id.NewServiceAccountID(),
		AppID:       appID,
		Name:        name,
		Description: description,
		Scopes:      scopes,
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := e.store.CreateServiceAccount(ctx, svc); err != nil {
		return nil, fmt.Errorf("authsome: create service account: %w", err)
	}

	return svc, nil
}

// GetServiceAccount returns a service account by ID.
func (e *Engine) GetServiceAccount(ctx context.Context, svcID id.ServiceAccountID) (*serviceaccount.ServiceAccount, error) {
	if err := e.requireStarted(); err != nil {
		return nil, err
	}

	svc, err := e.store.GetServiceAccount(ctx, svcID)
	if err != nil {
		return nil, fmt.Errorf("authsome: get service account: %w", err)
	}

	return svc, nil
}

// ListServiceAccounts returns service accounts for an app.
func (e *Engine) ListServiceAccounts(ctx context.Context, appID id.AppID, limit int) (*serviceaccount.List, error) {
	if err := e.requireStarted(); err != nil {
		return nil, err
	}
	if appID.IsNil() {
		return nil, fmt.Errorf("authsome: app_id is required")
	}

	q := &serviceaccount.Query{
		AppID: appID,
		Limit: limit,
	}

	list, err := e.store.ListServiceAccounts(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("authsome: list service accounts: %w", err)
	}

	return list, nil
}

// DeleteServiceAccount deletes a service account by ID.
func (e *Engine) DeleteServiceAccount(ctx context.Context, svcID id.ServiceAccountID) error {
	if err := e.requireStarted(); err != nil {
		return err
	}

	if err := e.store.DeleteServiceAccount(ctx, svcID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("authsome: service account not found")
		}
		return fmt.Errorf("authsome: delete service account: %w", err)
	}

	return nil
}

// CreateServiceAccountAPIKey mints an API key bound to a service account (not a user).
// Returns the persisted APIKey and the plaintext secret (only returned once — not stored).
func (e *Engine) CreateServiceAccountAPIKey(ctx context.Context, svcAcctID id.ServiceAccountID, name string, scopes []string, expiresAt *time.Time) (*apikey.APIKey, string, error) {
	if err := e.requireStarted(); err != nil {
		return nil, "", err
	}

	// Verify the service account exists.
	svc, err := e.store.GetServiceAccount(ctx, svcAcctID)
	if err != nil {
		return nil, "", fmt.Errorf("authsome: get service account: %w", err)
	}

	// Generate a key pair.
	publicKey, secretKey, secretHash, publicPrefix, secretPrefix, err := apikey.GenerateKeyPair()
	if err != nil {
		return nil, "", fmt.Errorf("authsome: generate key pair: %w", err)
	}

	now := time.Now()
	k := &apikey.APIKey{
		ID:               id.NewAPIKeyID(),
		AppID:            svc.AppID,
		ServiceAccountID: svcAcctID,
		Name:             name,
		KeyHash:          secretHash,
		KeyPrefix:        secretPrefix,
		PublicKey:        publicKey,
		PublicKeyPrefix:  publicPrefix,
		Scopes:           scopes,
		ExpiresAt:        expiresAt,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := e.store.CreateAPIKey(ctx, k); err != nil {
		return nil, "", fmt.Errorf("authsome: create service account api key: %w", err)
	}

	return k, secretKey, nil
}
