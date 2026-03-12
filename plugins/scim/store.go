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
	CreateToken(ctx context.Context, t *SCIMToken) error
	GetToken(ctx context.Context, tokenID id.SCIMTokenID) (*SCIMToken, error)
	ListTokens(ctx context.Context, configID id.SCIMConfigID) ([]*SCIMToken, error)
	DeleteToken(ctx context.Context, tokenID id.SCIMTokenID) error
	FindTokenByHash(ctx context.Context, tokenHash string) (*SCIMToken, *SCIMConfig, error)

	// Provision logs
	CreateLog(ctx context.Context, l *SCIMProvisionLog) error
	ListLogs(ctx context.Context, configID id.SCIMConfigID, limit int) ([]*SCIMProvisionLog, error)
	ListAllLogs(ctx context.Context, appID string, limit int) ([]*SCIMProvisionLog, error)
	CountLogsByStatus(ctx context.Context, configID id.SCIMConfigID) (success, errors, skipped int, err error)
	CountAllLogsByStatus(ctx context.Context, appID string) (success, errors, skipped int, err error)
}
