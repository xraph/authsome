package bridge

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
	"github.com/xraph/forgeui/bridge"
)

// =============================================================================
// Input/Output Types
// =============================================================================

// GetClientsInput is the input for listing OAuth clients.
type GetClientsInput struct {
	AppID    string `json:"appId"`
	Page     int    `json:"page,omitempty"`
	PageSize int    `json:"pageSize,omitempty"`
	Search   string `json:"search,omitempty"`
}

// GetClientsOutput is the output for listing OAuth clients.
type GetClientsOutput struct {
	Data       []ClientDTO    `json:"data"`
	Pagination *PaginationDTO `json:"pagination"`
}

// PaginationDTO represents pagination info.
type PaginationDTO struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"totalPages"`
}

// ClientDTO represents an OAuth client in API responses.
type ClientDTO struct {
	ID                string    `json:"id"`
	ClientID          string    `json:"clientId"`
	ClientName        string    `json:"clientName"`
	ApplicationType   string    `json:"applicationType"`
	LogoURI           string    `json:"logoUri,omitempty"`
	GrantTypes        []string  `json:"grantTypes"`
	ResponseTypes     []string  `json:"responseTypes"`
	RedirectURIs      []string  `json:"redirectUris"`
	AllowedScopes     []string  `json:"allowedScopes"`
	RequirePKCE       bool      `json:"requirePkce"`
	RequireConsent    bool      `json:"requireConsent"`
	TrustedClient     bool      `json:"trustedClient"`
	OrganizationID    string    `json:"organizationId,omitempty"`
	IsOrgLevel        bool      `json:"isOrgLevel"`
	TokenEndpointAuth string    `json:"tokenEndpointAuth"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

// GetClientInput is the input for getting a single client.
type GetClientInput struct {
	ClientID string `json:"clientId"`
}

// GetClientOutput is the output for getting a single client.
type GetClientOutput struct {
	Data ClientDTO `json:"data"`
}

// CreateClientInput is the input for creating an OAuth client.
type CreateClientInput struct {
	AppID                   string   `json:"appId"`
	ClientName              string   `json:"clientName"`
	ApplicationType         string   `json:"applicationType,omitempty"` // web, native, spa
	LogoURI                 string   `json:"logoUri,omitempty"`
	RedirectURIs            []string `json:"redirectUris,omitempty"`
	PostLogoutRedirectURIs  []string `json:"postLogoutRedirectUris,omitempty"`
	GrantTypes              []string `json:"grantTypes,omitempty"`
	ResponseTypes           []string `json:"responseTypes,omitempty"`
	AllowedScopes           []string `json:"allowedScopes,omitempty"`
	TokenEndpointAuthMethod string   `json:"tokenEndpointAuthMethod,omitempty"` // client_secret_basic, client_secret_post, none
	RequirePKCE             bool     `json:"requirePkce,omitempty"`
	RequireConsent          bool     `json:"requireConsent,omitempty"`
	TrustedClient           bool     `json:"trustedClient,omitempty"`
	OrganizationID          string   `json:"organizationId,omitempty"` // If set, client is org-specific
	PolicyURI               string   `json:"policyUri,omitempty"`
	TosURI                  string   `json:"tosUri,omitempty"`
	Contacts                []string `json:"contacts,omitempty"`
}

// CreateClientOutput is the output for creating an OAuth client.
type CreateClientOutput struct {
	Data ClientWithSecretDTO `json:"data"`
}

// ClientWithSecretDTO includes the client secret (only returned on creation).
type ClientWithSecretDTO struct {
	ClientDTO

	ClientSecret string `json:"clientSecret,omitempty"`
}

// UpdateClientInput is the input for updating an OAuth client.
type UpdateClientInput struct {
	ClientID                string   `json:"clientId"`
	ClientName              string   `json:"clientName,omitempty"`
	ApplicationType         string   `json:"applicationType,omitempty"`
	LogoURI                 string   `json:"logoUri,omitempty"`
	RedirectURIs            []string `json:"redirectUris,omitempty"`
	PostLogoutRedirectURIs  []string `json:"postLogoutRedirectUris,omitempty"`
	GrantTypes              []string `json:"grantTypes,omitempty"`
	ResponseTypes           []string `json:"responseTypes,omitempty"`
	AllowedScopes           []string `json:"allowedScopes,omitempty"`
	TokenEndpointAuthMethod string   `json:"tokenEndpointAuthMethod,omitempty"`
	RequirePKCE             bool     `json:"requirePkce,omitempty"`
	RequireConsent          bool     `json:"requireConsent,omitempty"`
	TrustedClient           bool     `json:"trustedClient,omitempty"`
	PolicyURI               string   `json:"policyUri,omitempty"`
	TosURI                  string   `json:"tosUri,omitempty"`
	Contacts                []string `json:"contacts,omitempty"`
}

// UpdateClientOutput is the output for updating an OAuth client.
type UpdateClientOutput struct {
	Data ClientDTO `json:"data"`
}

// DeleteClientInput is the input for deleting an OAuth client.
type DeleteClientInput struct {
	ClientID string `json:"clientId"`
}

// DeleteClientOutput is the output for deleting an OAuth client.
type DeleteClientOutput struct {
	Success bool `json:"success"`
}

// RegenerateSecretInput is the input for regenerating a client secret.
type RegenerateSecretInput struct {
	ClientID string `json:"clientId"`
}

// RegenerateSecretOutput is the output for regenerating a client secret.
type RegenerateSecretOutput struct {
	Data struct {
		ClientSecret string `json:"clientSecret"`
	} `json:"data"`
}

// GetClientStatsInput is the input for getting client statistics.
type GetClientStatsInput struct {
	ClientID string `json:"clientId"`
}

// GetClientStatsOutput is the output for getting client statistics.
type GetClientStatsOutput struct {
	Data ClientStatsDTO `json:"data"`
}

// ClientStatsDTO represents client usage statistics.
type ClientStatsDTO struct {
	TotalTokens     int64 `json:"totalTokens"`
	ActiveTokens    int64 `json:"activeTokens"`
	TotalUsers      int64 `json:"totalUsers"`
	TokensToday     int64 `json:"tokensToday"`
	TokensThisWeek  int64 `json:"tokensThisWeek"`
	TokensThisMonth int64 `json:"tokensThisMonth"`
}

// =============================================================================
// Bridge Functions
// =============================================================================

// GetClients lists OAuth clients with pagination and search.
func (bm *BridgeManager) GetClients(ctx bridge.Context, input GetClientsInput) (*GetClientsOutput, error) {
	goCtx, _, appID, err := bm.buildContextWithAppID(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Get environment ID
	envID, _ := contexts.GetEnvironmentID(goCtx)

	// Set pagination defaults
	page := max(input.Page, 1)

	pageSize := input.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Query clients
	clients, err := bm.clientRepo.ListByAppAndEnv(goCtx, appID, envID, page, pageSize)
	if err != nil {
		bm.logger.Error("failed to list OAuth clients",
			forge.F("error", err.Error()),
			forge.F("appId", appID.String()))

		return nil, errs.InternalServerError("failed to list clients", err)
	}

	// Convert to DTOs
	data := make([]ClientDTO, len(clients))
	for i, client := range clients {
		data[i] = clientToDTO(client)
	}

	// Get total count for pagination
	total, err := bm.clientRepo.CountByAppAndEnv(goCtx, appID, envID)
	if err != nil {
		total = int64(len(clients))
	}

	return &GetClientsOutput{
		Data: data,
		Pagination: &PaginationDTO{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: (total + int64(pageSize) - 1) / int64(pageSize),
		},
	}, nil
}

// GetClient retrieves a single OAuth client.
func (bm *BridgeManager) GetClient(ctx bridge.Context, input GetClientInput) (*GetClientOutput, error) {
	goCtx, _, _, err := bm.buildContext(ctx)
	if err != nil {
		return nil, err
	}

	client, err := bm.clientRepo.FindByClientID(goCtx, input.ClientID)
	if err != nil {
		return nil, errs.NotFound("client not found")
	}

	return &GetClientOutput{
		Data: clientToDTO(client),
	}, nil
}

// CreateClient creates a new OAuth client.
func (bm *BridgeManager) CreateClient(ctx bridge.Context, input CreateClientInput) (*CreateClientOutput, error) {
	goCtx, _, appID, err := bm.buildContextWithAppID(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Get environment ID
	envID, _ := contexts.GetEnvironmentID(goCtx)

	// Validate input
	if err := validateCreateClientInput(input); err != nil {
		return nil, err
	}

	// Generate client ID and secret
	clientID := "client_" + xid.New().String()

	clientSecret, hashedSecret, err := generateClientSecret()
	if err != nil {
		return nil, errs.InternalServerError("failed to generate client secret", err)
	}

	// Parse organization ID if provided
	var orgID *xid.ID

	if input.OrganizationID != "" {
		parsedOrgID, err := xid.FromString(input.OrganizationID)
		if err != nil {
			return nil, errs.BadRequest("invalid organizationId")
		}

		orgID = &parsedOrgID
	}

	// Create client model
	client := &schema.OAuthClient{
		ID:                      xid.New(),
		AppID:                   appID,
		EnvironmentID:           envID,
		OrganizationID:          orgID,
		ClientID:                clientID,
		ClientSecret:            hashedSecret,
		Name:                    input.ClientName,
		ApplicationType:         input.ApplicationType,
		LogoURI:                 input.LogoURI,
		RedirectURIs:            input.RedirectURIs,
		PostLogoutRedirectURIs:  input.PostLogoutRedirectURIs,
		GrantTypes:              input.GrantTypes,
		ResponseTypes:           input.ResponseTypes,
		AllowedScopes:           input.AllowedScopes,
		TokenEndpointAuthMethod: input.TokenEndpointAuthMethod,
		RequirePKCE:             input.RequirePKCE,
		RequireConsent:          input.RequireConsent,
		TrustedClient:           input.TrustedClient,
		PolicyURI:               input.PolicyURI,
		TosURI:                  input.TosURI,
		Contacts:                input.Contacts,
	}
	// Note: CreatedAt and UpdatedAt are set automatically by AuditableModel

	// Save to database
	if err := bm.clientRepo.Create(goCtx, client); err != nil {
		bm.logger.Error("failed to create OAuth client",
			forge.F("error", err.Error()),
			forge.F("clientName", input.ClientName))

		return nil, errs.InternalServerError("failed to create client", err)
	}

	bm.logger.Info("OAuth client created",
		forge.F("clientId", clientID),
		forge.F("clientName", input.ClientName),
		forge.F("appId", appID.String()))

	// Return client with secret (only time secret is returned in plaintext)
	dto := clientToDTO(client)

	return &CreateClientOutput{
		Data: ClientWithSecretDTO{
			ClientDTO:    dto,
			ClientSecret: clientSecret,
		},
	}, nil
}

// UpdateClient updates an existing OAuth client.
func (bm *BridgeManager) UpdateClient(ctx bridge.Context, input UpdateClientInput) (*UpdateClientOutput, error) {
	goCtx, _, _, err := bm.buildContext(ctx)
	if err != nil {
		return nil, err
	}

	// Find existing client
	client, err := bm.clientRepo.FindByClientID(goCtx, input.ClientID)
	if err != nil {
		return nil, errs.NotFound("client not found")
	}

	// Validate input
	if err := validateUpdateClientInput(input); err != nil {
		return nil, err
	}

	// Update fields
	client.Name = input.ClientName
	client.ApplicationType = input.ApplicationType
	client.LogoURI = input.LogoURI
	client.RedirectURIs = input.RedirectURIs
	client.PostLogoutRedirectURIs = input.PostLogoutRedirectURIs
	client.GrantTypes = input.GrantTypes
	client.ResponseTypes = input.ResponseTypes
	client.AllowedScopes = input.AllowedScopes
	client.TokenEndpointAuthMethod = input.TokenEndpointAuthMethod
	client.RequirePKCE = input.RequirePKCE
	client.RequireConsent = input.RequireConsent
	client.TrustedClient = input.TrustedClient
	client.PolicyURI = input.PolicyURI
	client.TosURI = input.TosURI
	client.Contacts = input.Contacts
	// Note: UpdatedAt is set automatically by AuditableModel

	// Save changes
	if err := bm.clientRepo.Update(goCtx, client); err != nil {
		bm.logger.Error("failed to update OAuth client",
			forge.F("error", err.Error()),
			forge.F("clientId", input.ClientID))

		return nil, errs.InternalServerError("failed to update client", err)
	}

	bm.logger.Info("OAuth client updated",
		forge.F("clientId", input.ClientID),
		forge.F("clientName", input.ClientName))

	return &UpdateClientOutput{
		Data: clientToDTO(client),
	}, nil
}

// DeleteClient deletes an OAuth client and revokes all associated tokens.
func (bm *BridgeManager) DeleteClient(ctx bridge.Context, input DeleteClientInput) (*DeleteClientOutput, error) {
	goCtx, _, _, err := bm.buildContext(ctx)
	if err != nil {
		return nil, err
	}

	// Find client to ensure it exists
	client, err := bm.clientRepo.FindByClientID(goCtx, input.ClientID)
	if err != nil {
		return nil, errs.NotFound("client not found")
	}

	// Revoke all tokens for this client
	if err := bm.tokenRepo.RevokeByClientID(goCtx, input.ClientID); err != nil {
		bm.logger.Error("failed to revoke client tokens",
			forge.F("error", err.Error()),
			forge.F("clientId", input.ClientID))
		// Continue with deletion even if token revocation fails
	}

	// Delete client
	if err := bm.clientRepo.Delete(goCtx, client.ID); err != nil {
		bm.logger.Error("failed to delete OAuth client",
			forge.F("error", err.Error()),
			forge.F("clientId", input.ClientID))

		return nil, errs.InternalServerError("failed to delete client", err)
	}

	bm.logger.Info("OAuth client deleted",
		forge.F("clientId", input.ClientID),
		forge.F("clientName", client.Name))

	return &DeleteClientOutput{
		Success: true,
	}, nil
}

// RegenerateSecret generates a new client secret.
func (bm *BridgeManager) RegenerateSecret(ctx bridge.Context, input RegenerateSecretInput) (*RegenerateSecretOutput, error) {
	goCtx, _, _, err := bm.buildContext(ctx)
	if err != nil {
		return nil, err
	}

	// Find client
	client, err := bm.clientRepo.FindByClientID(goCtx, input.ClientID)
	if err != nil {
		return nil, errs.NotFound("client not found")
	}

	// Check if client uses client secret (not for public clients)
	if client.TokenEndpointAuthMethod == "none" {
		return nil, errs.BadRequest("cannot regenerate secret for public client")
	}

	// Generate new secret
	newSecret, hashedSecret, err := generateClientSecret()
	if err != nil {
		return nil, errs.InternalServerError("failed to generate client secret", err)
	}

	// Update client
	client.ClientSecret = hashedSecret
	client.UpdatedAt = time.Now()

	if err := bm.clientRepo.Update(goCtx, client); err != nil {
		bm.logger.Error("failed to update client secret",
			forge.F("error", err.Error()),
			forge.F("clientId", input.ClientID))

		return nil, errs.InternalServerError("failed to regenerate secret", err)
	}

	bm.logger.Info("OAuth client secret regenerated",
		forge.F("clientId", input.ClientID))

	return &RegenerateSecretOutput{
		Data: struct {
			ClientSecret string `json:"clientSecret"`
		}{
			ClientSecret: newSecret,
		},
	}, nil
}

// GetClientStats retrieves usage statistics for a client.
func (bm *BridgeManager) GetClientStats(ctx bridge.Context, input GetClientStatsInput) (*GetClientStatsOutput, error) {
	goCtx, _, _, err := bm.buildContext(ctx)
	if err != nil {
		return nil, err
	}

	// Find client to ensure it exists
	_, err = bm.clientRepo.FindByClientID(goCtx, input.ClientID)
	if err != nil {
		return nil, errs.NotFound("client not found")
	}

	// Get token counts
	totalTokens, err := bm.tokenRepo.CountByClientID(goCtx, input.ClientID)
	if err != nil {
		totalTokens = 0
	}

	activeTokens, err := bm.tokenRepo.CountActiveByClientID(goCtx, input.ClientID)
	if err != nil {
		activeTokens = 0
	}

	// Get unique users count (tokens with distinct user IDs)
	totalUsers, err := bm.tokenRepo.CountUniqueUsersByClientID(goCtx, input.ClientID)
	if err != nil {
		totalUsers = 0
	}

	// Time-based counts
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := todayStart.AddDate(0, 0, -7)
	monthStart := todayStart.AddDate(0, -1, 0)

	tokensToday, _ := bm.tokenRepo.CountByClientIDSince(goCtx, input.ClientID, todayStart)
	tokensThisWeek, _ := bm.tokenRepo.CountByClientIDSince(goCtx, input.ClientID, weekStart)
	tokensThisMonth, _ := bm.tokenRepo.CountByClientIDSince(goCtx, input.ClientID, monthStart)

	return &GetClientStatsOutput{
		Data: ClientStatsDTO{
			TotalTokens:     totalTokens,
			ActiveTokens:    activeTokens,
			TotalUsers:      totalUsers,
			TokensToday:     tokensToday,
			TokensThisWeek:  tokensThisWeek,
			TokensThisMonth: tokensThisMonth,
		},
	}, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// clientToDTO converts a schema.OAuthClient to ClientDTO.
func clientToDTO(client *schema.OAuthClient) ClientDTO {
	dto := ClientDTO{
		ID:                client.ID.String(),
		ClientID:          client.ClientID,
		ClientName:        client.Name,
		ApplicationType:   client.ApplicationType,
		LogoURI:           client.LogoURI,
		GrantTypes:        client.GrantTypes,
		ResponseTypes:     client.ResponseTypes,
		RedirectURIs:      client.RedirectURIs,
		AllowedScopes:     client.AllowedScopes,
		RequirePKCE:       client.RequirePKCE,
		RequireConsent:    client.RequireConsent,
		TrustedClient:     client.TrustedClient,
		TokenEndpointAuth: client.TokenEndpointAuthMethod,
		CreatedAt:         client.CreatedAt,
		UpdatedAt:         client.UpdatedAt,
	}

	if client.OrganizationID != nil {
		dto.OrganizationID = client.OrganizationID.String()
		dto.IsOrgLevel = true
	}

	return dto
}

// generateClientSecret generates a secure random client secret.
func generateClientSecret() (plaintext, hashed string, err error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}

	// Encode as base64 for plaintext secret
	plaintext = "secret_" + base64.RawURLEncoding.EncodeToString(bytes)

	// Hash the secret for storage
	hash := sha256.Sum256([]byte(plaintext))
	hashed = base64.StdEncoding.EncodeToString(hash[:])

	return plaintext, hashed, nil
}

// validateCreateClientInput validates the create client input.
func validateCreateClientInput(input CreateClientInput) error {
	if input.ClientName == "" {
		return errs.BadRequest("clientName is required")
	}

	if len(input.RedirectURIs) == 0 {
		return errs.BadRequest("at least one redirectUri is required")
	}

	validTypes := map[string]bool{"web": true, "native": true, "spa": true}
	if !validTypes[input.ApplicationType] {
		return errs.BadRequest("applicationType must be 'web', 'native', or 'spa'")
	}

	validAuthMethods := map[string]bool{
		"client_secret_basic": true,
		"client_secret_post":  true,
		"none":                true,
	}
	if !validAuthMethods[input.TokenEndpointAuthMethod] {
		return errs.BadRequest("invalid tokenEndpointAuthMethod")
	}

	// Public clients must use PKCE
	if input.TokenEndpointAuthMethod == "none" && !input.RequirePKCE {
		return errs.BadRequest("public clients must require PKCE")
	}

	return nil
}

// validateUpdateClientInput validates the update client input.
func validateUpdateClientInput(input UpdateClientInput) error {
	if input.ClientName == "" {
		return errs.BadRequest("clientName is required")
	}

	if len(input.RedirectURIs) == 0 {
		return errs.BadRequest("at least one redirectUri is required")
	}

	validTypes := map[string]bool{"web": true, "native": true, "spa": true}
	if !validTypes[input.ApplicationType] {
		return errs.BadRequest("applicationType must be 'web', 'native', or 'spa'")
	}

	validAuthMethods := map[string]bool{
		"client_secret_basic": true,
		"client_secret_post":  true,
		"none":                true,
	}
	if !validAuthMethods[input.TokenEndpointAuthMethod] {
		return errs.BadRequest("invalid tokenEndpointAuthMethod")
	}

	// Public clients must use PKCE
	if input.TokenEndpointAuthMethod == "none" && !input.RequirePKCE {
		return errs.BadRequest("public clients must require PKCE")
	}

	return nil
}
