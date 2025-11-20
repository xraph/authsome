package oidcprovider

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/internal/errs"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// AdminHandler handles admin-only OAuth client management endpoints
type AdminHandler struct {
	clientRepo      *repo.OAuthClientRepository
	registrationSvc *RegistrationService
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(clientRepo *repo.OAuthClientRepository, registrationSvc *RegistrationService) *AdminHandler {
	return &AdminHandler{
		clientRepo:      clientRepo,
		registrationSvc: registrationSvc,
	}
}

// RegisterClient handles dynamic client registration (admin only)
func (h *AdminHandler) RegisterClient(c forge.Context) error {
	ctx := c.Request().Context()

	// Extract context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("app context required"))
	}

	envID, ok := contexts.GetEnvironmentID(ctx)
	if !ok || envID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("environment context required"))
	}

	// Org context is optional
	orgID, ok := contexts.GetOrganizationID(ctx)
	var orgIDPtr *xid.ID
	if ok && !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	// Parse request
	var req ClientRegistrationRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request body"))
	}

	// Register client
	response, err := h.registrationSvc.RegisterClient(ctx, &req, appID, envID, orgIDPtr)
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusCreated, response)
}

// ListClients lists all OAuth clients for the current app/env/org
func (h *AdminHandler) ListClients(c forge.Context) error {
	ctx := c.Request().Context()

	// Extract context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("app context required"))
	}

	envID, ok := contexts.GetEnvironmentID(ctx)
	if !ok || envID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("environment context required"))
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Request().URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.Request().URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// Check if org-specific or app-level listing
	orgID, ok := contexts.GetOrganizationID(ctx)
	var clients []*schema.OAuthClient
	var total int
	var err error

	if ok && !orgID.IsNil() {
		// List org-specific clients
		clients, total, err = h.clientRepo.ListByOrg(ctx, appID, envID, orgID, pageSize, offset)
	} else {
		// List app-level clients
		clients, total, err = h.clientRepo.ListByApp(ctx, appID, envID, pageSize, offset)
	}

	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.DatabaseError("list clients", err))
	}

	// Build response
	summaries := make([]ClientSummary, len(clients))
	for i, client := range clients {
		summaries[i] = ClientSummary{
			ClientID:        client.ClientID,
			Name:            client.Name,
			ApplicationType: client.ApplicationType,
			CreatedAt:       client.CreatedAt.Format(time.RFC3339),
			IsOrgLevel:      client.OrganizationID != nil,
		}
	}

	totalPages := (total + pageSize - 1) / pageSize

	response := ClientsListResponse{
		Clients:    summaries,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	return c.JSON(http.StatusOK, response)
}

// GetClient retrieves detailed information about an OAuth client
func (h *AdminHandler) GetClient(c forge.Context) error {
	ctx := c.Request().Context()

	// Get client ID from path
	clientID := c.Param("clientId")
	if clientID == "" {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("client_id required"))
	}

	// Find client
	client, err := h.clientRepo.FindByClientID(ctx, clientID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.DatabaseError("find client", err))
	}
	if client == nil {
		return c.JSON(http.StatusNotFound, errs.NotFound("client not found"))
	}

	// Build detailed response
	response := ClientDetailsResponse{
		ClientID:                client.ClientID,
		Name:                    client.Name,
		ApplicationType:         client.ApplicationType,
		RedirectURIs:            client.RedirectURIs,
		PostLogoutRedirectURIs:  client.PostLogoutRedirectURIs,
		GrantTypes:              client.GrantTypes,
		ResponseTypes:           client.ResponseTypes,
		AllowedScopes:           client.AllowedScopes,
		TokenEndpointAuthMethod: client.TokenEndpointAuthMethod,
		RequirePKCE:             client.RequirePKCE,
		RequireConsent:          client.RequireConsent,
		TrustedClient:           client.TrustedClient,
		LogoURI:                 client.LogoURI,
		PolicyURI:               client.PolicyURI,
		TosURI:                  client.TosURI,
		Contacts:                client.Contacts,
		CreatedAt:               client.CreatedAt.Format(time.RFC3339),
		UpdatedAt:               client.UpdatedAt.Format(time.RFC3339),
		IsOrgLevel:              client.OrganizationID != nil,
	}

	if client.OrganizationID != nil {
		response.OrganizationID = client.OrganizationID.String()
	}

	return c.JSON(http.StatusOK, response)
}

// UpdateClient updates an existing OAuth client
func (h *AdminHandler) UpdateClient(c forge.Context) error {
	ctx := c.Request().Context()

	// Get client ID from path
	clientID := c.Param("clientId")
	if clientID == "" {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("client_id required"))
	}

	// Parse request
	var req ClientUpdateRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request body"))
	}

	// Find existing client
	client, err := h.clientRepo.FindByClientID(ctx, clientID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.DatabaseError("find client", err))
	}
	if client == nil {
		return c.JSON(http.StatusNotFound, errs.NotFound("client not found"))
	}

	// Update fields if provided
	if req.Name != "" {
		client.Name = req.Name
	}
	if len(req.RedirectURIs) > 0 {
		client.RedirectURIs = req.RedirectURIs
		client.RedirectURI = req.RedirectURIs[0] // Update legacy field
	}
	if len(req.PostLogoutRedirectURIs) > 0 {
		client.PostLogoutRedirectURIs = req.PostLogoutRedirectURIs
	}
	if len(req.GrantTypes) > 0 {
		client.GrantTypes = req.GrantTypes
	}
	if len(req.ResponseTypes) > 0 {
		client.ResponseTypes = req.ResponseTypes
	}
	if len(req.AllowedScopes) > 0 {
		client.AllowedScopes = req.AllowedScopes
	}
	if req.TokenEndpointAuthMethod != "" {
		client.TokenEndpointAuthMethod = req.TokenEndpointAuthMethod
	}
	if req.RequirePKCE != nil {
		client.RequirePKCE = *req.RequirePKCE
	}
	if req.RequireConsent != nil {
		client.RequireConsent = *req.RequireConsent
	}
	if req.TrustedClient != nil {
		client.TrustedClient = *req.TrustedClient
	}
	if req.LogoURI != "" {
		client.LogoURI = req.LogoURI
	}
	if req.PolicyURI != "" {
		client.PolicyURI = req.PolicyURI
	}
	if req.TosURI != "" {
		client.TosURI = req.TosURI
	}
	if len(req.Contacts) > 0 {
		client.Contacts = req.Contacts
	}

	// Update in database
	if err := h.clientRepo.Update(ctx, client); err != nil {
		return c.JSON(http.StatusInternalServerError, errs.DatabaseError("update client", err))
	}

	// Build response
	response := ClientDetailsResponse{
		ClientID:                client.ClientID,
		Name:                    client.Name,
		ApplicationType:         client.ApplicationType,
		RedirectURIs:            client.RedirectURIs,
		PostLogoutRedirectURIs:  client.PostLogoutRedirectURIs,
		GrantTypes:              client.GrantTypes,
		ResponseTypes:           client.ResponseTypes,
		AllowedScopes:           client.AllowedScopes,
		TokenEndpointAuthMethod: client.TokenEndpointAuthMethod,
		RequirePKCE:             client.RequirePKCE,
		RequireConsent:          client.RequireConsent,
		TrustedClient:           client.TrustedClient,
		LogoURI:                 client.LogoURI,
		PolicyURI:               client.PolicyURI,
		TosURI:                  client.TosURI,
		Contacts:                client.Contacts,
		CreatedAt:               client.CreatedAt.Format(time.RFC3339),
		UpdatedAt:               client.UpdatedAt.Format(time.RFC3339),
		IsOrgLevel:              client.OrganizationID != nil,
	}

	if client.OrganizationID != nil {
		response.OrganizationID = client.OrganizationID.String()
	}

	return c.JSON(http.StatusOK, response)
}

// DeleteClient deletes an OAuth client
func (h *AdminHandler) DeleteClient(c forge.Context) error {
	ctx := c.Request().Context()

	// Get client ID from path
	clientID := c.Param("clientId")
	if clientID == "" {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("client_id required"))
	}

	// Find client
	client, err := h.clientRepo.FindByClientID(ctx, clientID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.DatabaseError("find client", err))
	}
	if client == nil {
		return c.JSON(http.StatusNotFound, errs.NotFound("client not found"))
	}

	// Delete client
	if err := h.clientRepo.Delete(ctx, client.ID); err != nil {
		return c.JSON(http.StatusInternalServerError, errs.DatabaseError("delete client", err))
	}

	return c.JSON(http.StatusNoContent, nil)
}
