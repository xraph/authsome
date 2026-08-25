package scim

import (
	"context"

	"github.com/xraph/authsome/id"
)

// Store defines the persistence interface for SCIM entities.
type Store interface {
	// Config CRUD
	CreateConfig(ctx context.Context, c *SCIMConfig) error
	GetConfig(ctx context.Context, configID id.SCIMConfigID) (*SCIMConfig, error)
	UpdateConfig(ctx context.Context, c *SCIMConfig) error
	DeleteConfig(ctx context.Context, configID id.SCIMConfigID) error
	ListConfigs(ctx context.Context, appID string) ([]*SCIMConfig, error)
	ListConfigsByOrg(ctx context.Context, orgID id.OrgID) ([]*SCIMConfig, error)

	// Token CRUD
	CreateToken(ctx context.Context, t *Token) error
	GetToken(ctx context.Context, tokenID id.SCIMTokenID) (*Token, error)
	UpdateToken(ctx context.Context, t *Token) error
	ListTokens(ctx context.Context, configID id.SCIMConfigID) ([]*Token, error)
	DeleteToken(ctx context.Context, tokenID id.SCIMTokenID) error
	// FindTokenByPlaintext resolves a presented bearer token to its stored
	// record and owning config. Token hashes are salted bcrypt digests, so the
	// plaintext cannot be hashed and looked up directly; implementations must
	// scan candidates and bcrypt-compare. Expiry is not checked here.
	FindTokenByPlaintext(ctx context.Context, plaintext string) (*Token, *SCIMConfig, error)

	// Provision logs
	CreateLog(ctx context.Context, l *ProvisionLog) error
	ListLogs(ctx context.Context, configID id.SCIMConfigID, limit int) ([]*ProvisionLog, error)
	ListAllLogs(ctx context.Context, appID string, limit int) ([]*ProvisionLog, error)
	CountLogsByStatus(ctx context.Context, configID id.SCIMConfigID) (success, errors, skipped int, err error)
	CountAllLogsByStatus(ctx context.Context, appID string) (success, errors, skipped int, err error)
}
