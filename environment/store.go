package environment

import (
	"context"

	"github.com/xraph/authsome/id"
)

// Store defines the persistence interface for environment operations.
type Store interface {
	// CreateEnvironment persists a new environment.
	CreateEnvironment(ctx context.Context, e *Environment) error

	// GetEnvironment retrieves an environment by ID.
	GetEnvironment(ctx context.Context, envID id.EnvironmentID) (*Environment, error)

	// GetEnvironmentBySlug retrieves an environment by app ID and slug.
	GetEnvironmentBySlug(ctx context.Context, appID id.AppID, slug string) (*Environment, error)

	// GetDefaultEnvironment retrieves the default environment for an app.
	GetDefaultEnvironment(ctx context.Context, appID id.AppID) (*Environment, error)

	// UpdateEnvironment updates an existing environment.
	UpdateEnvironment(ctx context.Context, e *Environment) error

	// DeleteEnvironment removes an environment by ID.
	// Returns an error if the environment is the default for its app.
	DeleteEnvironment(ctx context.Context, envID id.EnvironmentID) error

	// ListEnvironments returns all environments for an app.
	ListEnvironments(ctx context.Context, appID id.AppID) ([]*Environment, error)

	// SetDefaultEnvironment sets the given environment as the default for its app,
	// clearing the default flag on any previously default environment.
	SetDefaultEnvironment(ctx context.Context, appID id.AppID, envID id.EnvironmentID) error
}
