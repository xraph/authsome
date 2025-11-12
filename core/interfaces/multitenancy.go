package interfaces

import (
	"context"
	"errors"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/forms"
	"github.com/xraph/authsome/core/jwt"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
)

// Context keys for multi-tenancy
type contextKey string

const (
	OrganizationContextKey contextKey = "organization_id"
	AppContextKey          contextKey = "app_id"
)

func GetAppID(ctx context.Context) (xid.ID, error) {
	appID, ok := ctx.Value(AppContextKey).(xid.ID)
	if !ok {
		return xid.ID{}, errors.New("app context not found")
	}
	return appID, nil
}

func SetAppID(ctx context.Context, appID xid.ID) context.Context {
	return context.WithValue(ctx, AppContextKey, appID)
}

func GetOrganizationID(ctx context.Context) (xid.ID, error) {
	orgID, ok := ctx.Value(OrganizationContextKey).(xid.ID)
	if !ok {
		return xid.ID{}, errors.New("organization context not found")
	}
	return orgID, nil
}

func SetOrganizationID(ctx context.Context, orgID xid.ID) context.Context {
	return context.WithValue(ctx, OrganizationContextKey, orgID)
}

// MultiTenantUserService extends user service with organization-aware operations
type MultiTenantUserService interface {
	// Core user service methods (delegated)
	user.Service

	// Multi-tenant specific methods
	CreateWithOrganization(ctx context.Context, req *user.CreateUserRequest, orgID string) (*user.User, error)
	FindByEmailInOrganization(ctx context.Context, email, orgID string) (*user.User, error)
	ListByOrganization(ctx context.Context, orgID string, limit, offset int) ([]*user.User, error)
	CountByOrganization(ctx context.Context, orgID string) (int, error)
}

// MultiTenantSessionService extends session service with organization-aware operations
type MultiTenantSessionService interface {
	// Core session service methods (delegated)
	session.Service

	// Multi-tenant specific methods
	CreateWithOrganization(ctx context.Context, req *session.CreateSessionRequest, orgID string) (*session.Session, error)
	FindByTokenInOrganization(ctx context.Context, token, orgID string) (*session.Session, error)
	ListByOrganization(ctx context.Context, orgID string, limit, offset int) ([]*session.Session, error)
	RevokeAllByOrganization(ctx context.Context, orgID string) error
}

// MultiTenantAuthService extends auth service with organization-aware operations
type MultiTenantAuthService interface {
	// Core auth service methods (delegated)
	auth.Service

	// Multi-tenant specific methods
	SignUpWithOrganization(ctx context.Context, req *auth.SignUpRequest, orgID string) (*auth.AuthResponse, error)
	SignInWithOrganization(ctx context.Context, req *auth.SignInRequest, orgID string) (*auth.AuthResponse, error)
	GetSessionWithOrganization(ctx context.Context, token, orgID string) (*auth.AuthResponse, error)
}

// MultiTenantJWTService extends JWT service with organization-aware operations
type MultiTenantJWTService interface {
	// Core JWT service methods (delegated)
	jwt.Service

	// Multi-tenant specific methods
	CreateKeyForOrganization(ctx context.Context, req *jwt.CreateJWTKeyRequest, orgID string) (*jwt.JWTKey, error)
	GetJWKSForOrganization(ctx context.Context, orgID string) (*jwt.JWKSResponse, error)
	ListKeysForOrganization(ctx context.Context, req *jwt.ListJWTKeysRequest) (*jwt.ListJWTKeysResponse, error)
}

// MultiTenantAPIKeyService extends API key service with organization-aware operations
type MultiTenantAPIKeyService interface {
	// Core API key service methods (delegated)
	apikey.Service

	// Multi-tenant specific methods
	CreateForOrganization(ctx context.Context, req *apikey.CreateAPIKeyRequest, orgID string) (*apikey.APIKey, error)
	ListForOrganization(ctx context.Context, req *apikey.ListAPIKeysRequest) (*apikey.ListAPIKeysResponse, error)
	ValidateForOrganization(ctx context.Context, key, orgID string) (*apikey.APIKey, error)
}

// MultiTenantFormsService extends forms service with organization-aware operations
type MultiTenantFormsService interface {
	// Core forms service methods (delegated)
	forms.Service

	// Multi-tenant specific methods
	CreateForOrganization(ctx context.Context, req *forms.CreateFormRequest, orgID string) (*forms.Form, error)
	GetByOrganizationAndType(ctx context.Context, orgID string, formType string) (*forms.Form, error)
	ListByOrganization(ctx context.Context, orgID string, limit, offset int) ([]*forms.Form, error)
}

// OrganizationContextProvider provides organization context for requests
type OrganizationContextProvider interface {
	GetOrganizationID(ctx context.Context) (string, error)
	SetOrganizationID(ctx context.Context, orgID string) context.Context
	ValidateOrganizationAccess(ctx context.Context, userID xid.ID, orgID string) error
}

// ConfigProvider provides organization-scoped configuration
type ConfigProvider interface {
	GetConfig(ctx context.Context, key string, orgID string) (interface{}, error)
	SetConfig(ctx context.Context, key string, orgID string, value interface{}) error
	DeleteConfig(ctx context.Context, key string, orgID string) error
	ListConfigs(ctx context.Context, orgID string) (map[string]interface{}, error)
}
