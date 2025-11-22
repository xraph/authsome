package base

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// SOCIAL ACCOUNT DTO (Data Transfer Object)
// =============================================================================

// SocialAccount represents a social account connection DTO
// This is separate from schema.SocialAccount to maintain proper separation of concerns
type SocialAccount struct {
	ID        xid.ID    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// User relationship
	UserID             xid.ID  `json:"userId"`
	AppID              xid.ID  `json:"appId"`
	UserOrganizationID *xid.ID `json:"userOrganizationId,omitempty"`

	// Provider information
	Provider   string `json:"provider"`
	ProviderID string `json:"providerId"`
	Email      string `json:"email,omitempty"`
	Name       string `json:"name,omitempty"`
	Avatar     string `json:"avatar,omitempty"`

	// OAuth tokens (access token excluded for security)
	TokenType        string     `json:"tokenType"`
	ExpiresAt        *time.Time `json:"expiresAt,omitempty"`
	RefreshExpiresAt *time.Time `json:"refreshExpiresAt,omitempty"`
	Scope            string     `json:"scope,omitempty"`

	// Account status
	Revoked   bool       `json:"revoked"`
	RevokedAt *time.Time `json:"revokedAt,omitempty"`
}

// FromSchemaSocialAccount converts a schema.SocialAccount to a SocialAccount DTO
func FromSchemaSocialAccount(sa *schema.SocialAccount) *SocialAccount {
	if sa == nil {
		return nil
	}

	return &SocialAccount{
		ID:                 sa.ID,
		CreatedAt:          sa.CreatedAt,
		UpdatedAt:          sa.UpdatedAt,
		UserID:             sa.UserID,
		AppID:              sa.AppID,
		UserOrganizationID: sa.UserOrganizationID,
		Provider:           sa.Provider,
		ProviderID:         sa.ProviderID,
		Email:              sa.Email,
		Name:               sa.Name,
		Avatar:             sa.Avatar,
		TokenType:          sa.TokenType,
		ExpiresAt:          sa.ExpiresAt,
		RefreshExpiresAt:   sa.RefreshExpiresAt,
		Scope:              sa.Scope,
		Revoked:            sa.Revoked,
		RevokedAt:          sa.RevokedAt,
	}
}

// FromSchemaSocialAccounts converts a slice of schema.SocialAccount to SocialAccount DTOs
func FromSchemaSocialAccounts(accounts []*schema.SocialAccount) []*SocialAccount {
	if accounts == nil {
		return nil
	}

	result := make([]*SocialAccount, len(accounts))
	for i, sa := range accounts {
		result[i] = FromSchemaSocialAccount(sa)
	}
	return result
}

// ToSchema converts the SocialAccount DTO to a schema.SocialAccount model
func (sa *SocialAccount) ToSchema() *schema.SocialAccount {
	return &schema.SocialAccount{
		ID:                 sa.ID,
		CreatedAt:          sa.CreatedAt,
		UpdatedAt:          sa.UpdatedAt,
		UserID:             sa.UserID,
		AppID:              sa.AppID,
		UserOrganizationID: sa.UserOrganizationID,
		Provider:           sa.Provider,
		ProviderID:         sa.ProviderID,
		Email:              sa.Email,
		Name:               sa.Name,
		Avatar:             sa.Avatar,
		TokenType:          sa.TokenType,
		ExpiresAt:          sa.ExpiresAt,
		RefreshExpiresAt:   sa.RefreshExpiresAt,
		Scope:              sa.Scope,
		Revoked:            sa.Revoked,
		RevokedAt:          sa.RevokedAt,
	}
}
