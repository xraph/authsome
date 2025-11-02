package scim

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/core/webhook"
	"golang.org/x/crypto/bcrypt"
)

// ServiceConfig holds service dependencies
type ServiceConfig struct {
	Config         *Config
	Repository     *Repository
	UserService    *user.Service
	OrgService     *organization.Service
	AuditService   *audit.Service
	WebhookService *webhook.Service
}

// Service provides SCIM provisioning business logic
type Service struct {
	config         *Config
	repo           *Repository
	userService    *user.Service
	orgService     *organization.Service
	auditService   *audit.Service
	webhookService *webhook.Service
	metrics        *Metrics
}

// NewService creates a new SCIM service
func NewService(cfg ServiceConfig) *Service {
	return &Service{
		config:         cfg.Config,
		repo:           cfg.Repository,
		userService:    cfg.UserService,
		orgService:     cfg.OrgService,
		auditService:   cfg.AuditService,
		webhookService: cfg.WebhookService,
		metrics:        GetMetrics(),
	}
}

// User Provisioning Operations

// CreateUser provisions a new user via SCIM
func (s *Service) CreateUser(ctx context.Context, scimUser *SCIMUser, orgID string) (*SCIMUser, error) {
	// Validate required attributes
	if err := s.validateUserAttributes(scimUser); err != nil {
		return nil, fmt.Errorf("invalid user attributes: %w", err)
	}
	
	// Check for duplicate email if configured
	if s.config.UserProvisioning.PreventDuplicates {
		email := s.getPrimaryEmail(scimUser)
		if email != "" {
			existing, _ := s.userService.FindByEmail(ctx, email)
			if existing != nil {
				return nil, fmt.Errorf("user with email %s already exists", email)
			}
		}
	}
	
	// Get primary email for user creation
	email := s.getPrimaryEmail(scimUser)
	name := scimUser.DisplayName
	if name == "" && scimUser.Name != nil {
		name = strings.TrimSpace(scimUser.Name.GivenName + " " + scimUser.Name.FamilyName)
	}
	
	// Create user in AuthSome
	createReq := &user.CreateUserRequest{
		Email:    email,
		Password: "", // Empty for SSO users
		Name:     name,
	}
	
	createdUser, err := s.userService.Create(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	
	// Auto-activate if configured
	if s.config.UserProvisioning.AutoActivate {
		updateReq := &user.UpdateUserRequest{
			EmailVerified: boolPtr(true),
		}
		_, err := s.userService.Update(ctx, createdUser, updateReq)
		if err != nil {
			// Log but don't fail
			fmt.Printf("[SCIM] Failed to auto-activate user: %v\n", err)
		}
	}
	
	// Add user to organization
	orgXID, _ := xid.FromString(orgID)
	member := &organization.Member{
		ID:             xid.New(),
		OrganizationID: orgXID,
		UserID:         createdUser.ID,
		Role:           s.config.UserProvisioning.DefaultRole,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := s.orgService.CreateMember(ctx, member); err != nil {
		// Rollback user creation?
		return nil, fmt.Errorf("failed to add user to organization: %w", err)
	}
	
	// Sync groups if provided
	if len(scimUser.Groups) > 0 && s.config.GroupSync.Enabled {
		if err := s.syncUserGroups(ctx, createdUser.ID, scimUser.Groups, orgID); err != nil {
			fmt.Printf("[SCIM] Failed to sync user groups: %v\n", err)
		}
	}
	
	// Convert back to SCIM format
	resultUser := s.mapAuthSomeToSCIMUser(createdUser, scimUser.ExternalID)
	
	// Record metrics
	s.metrics.RecordUserOperation("create")
	s.metrics.RecordOperation("CREATE_USER", "success", orgID)
	
	// Send webhook if configured
	if s.config.Webhooks.Enabled && s.config.Webhooks.NotifyOnCreate {
		err := s.sendProvisioningWebhook(ctx, "user.provisioned", map[string]interface{}{
			"user_id":     createdUser.ID.String(),
			"external_id": scimUser.ExternalID,
			"org_id":      orgID,
			"operation":   "create",
		})
		s.metrics.RecordWebhook(err == nil, false)
	}
	
	// Audit log
	_ = s.auditService.Log(ctx, nil, "scim.user.create", "user:"+createdUser.ID.String(), "", "", 
		fmt.Sprintf(`{"org_id":"%s","external_id":"%s"}`, orgID, scimUser.ExternalID))
	
	return resultUser, nil
}

// GetUser retrieves a user by ID
func (s *Service) GetUser(ctx context.Context, id, orgID string) (*SCIMUser, error) {
	userXID, err := xid.FromString(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	
	authUser, err := s.userService.FindByID(ctx, userXID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	
	// Verify user belongs to organization
	orgXID, _ := xid.FromString(orgID)
	member, err := s.orgService.FindMember(ctx, orgXID, authUser.ID)
	if err != nil || member == nil {
		return nil, fmt.Errorf("user not found in organization")
	}
	
	scimUser := s.mapAuthSomeToSCIMUser(authUser, "")
	
	// Record metrics
	s.metrics.RecordUserOperation("read")
	s.metrics.RecordOperation("GET_USER", "success", orgID)
	
	return scimUser, nil
}

// UpdateUser updates a user via SCIM PATCH
func (s *Service) UpdateUser(ctx context.Context, id, orgID string, patch *PatchOp) (*SCIMUser, error) {
	// Get existing user
	userXID, err := xid.FromString(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	
	authUser, err := s.userService.FindByID(ctx, userXID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	
	// Apply patch operations and build update request
	updateReq := &user.UpdateUserRequest{}
	for _, op := range patch.Operations {
		if err := s.applyPatchOperationToRequest(authUser, &op, updateReq); err != nil {
			return nil, fmt.Errorf("failed to apply patch operation: %w", err)
		}
	}
	
	// Update user
	updatedUser, err := s.userService.Update(ctx, authUser, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	authUser = updatedUser
	
	// Record metrics
	s.metrics.RecordUserOperation("update")
	s.metrics.RecordOperation("PATCH_USER", "success", orgID)
	
	// Send webhook
	if s.config.Webhooks.Enabled && s.config.Webhooks.NotifyOnUpdate {
		err := s.sendProvisioningWebhook(ctx, "user.updated", map[string]interface{}{
			"user_id":   authUser.ID.String(),
			"org_id":    orgID,
			"operation": "patch",
		})
		s.metrics.RecordWebhook(err == nil, false)
	}
	
	scimUser := s.mapAuthSomeToSCIMUser(authUser, "")
	return scimUser, nil
}

// ReplaceUser replaces a user via SCIM PUT
func (s *Service) ReplaceUser(ctx context.Context, id, orgID string, scimUser *SCIMUser) (*SCIMUser, error) {
	userXID, err := xid.FromString(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	
	authUser, err := s.userService.FindByID(ctx, userXID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	
	// Build full replacement update request
	email := s.getPrimaryEmail(scimUser)
	name := scimUser.DisplayName
	if name == "" && scimUser.Name != nil {
		name = strings.TrimSpace(scimUser.Name.GivenName + " " + scimUser.Name.FamilyName)
	}
	
	updateReq := &user.UpdateUserRequest{
		Email:         stringPtr(email),
		Name:          stringPtr(name),
		EmailVerified: boolPtr(scimUser.Active),
	}
	
	updatedUser, err := s.userService.Update(ctx, authUser, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	
	resultUser := s.mapAuthSomeToSCIMUser(updatedUser, scimUser.ExternalID)
	return resultUser, nil
}

// DeleteUser de-provisions a user
func (s *Service) DeleteUser(ctx context.Context, id, orgID string) error {
	userXID, err := xid.FromString(id)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}
	
	// Soft delete or hard delete based on config
	if s.config.UserProvisioning.SoftDeleteOnDeProvision {
		// Soft delete: deactivate user by setting email verified to false
		authUser, err := s.userService.FindByID(ctx, userXID)
		if err != nil {
			return fmt.Errorf("user not found: %w", err)
		}
		
		updateReq := &user.UpdateUserRequest{
			EmailVerified: boolPtr(false),
		}
		
		_, err = s.userService.Update(ctx, authUser, updateReq)
		if err != nil {
			return fmt.Errorf("failed to deactivate user: %w", err)
		}
	} else {
		// Hard delete
		if err := s.userService.Delete(ctx, userXID); err != nil {
			return fmt.Errorf("failed to delete user: %w", err)
		}
	}
	
	// Record metrics
	s.metrics.RecordUserOperation("delete")
	s.metrics.RecordOperation("DELETE_USER", "success", orgID)
	
	// Send webhook
	if s.config.Webhooks.Enabled && s.config.Webhooks.NotifyOnDelete {
		err := s.sendProvisioningWebhook(ctx, "user.deprovisioned", map[string]interface{}{
			"user_id":   id,
			"org_id":    orgID,
			"operation": "delete",
		})
		s.metrics.RecordWebhook(err == nil, false)
	}
	
	return nil
}

// ListUsers lists users with filtering and pagination
func (s *Service) ListUsers(ctx context.Context, orgID string, filter string, startIndex, count int) (*ListResponse, error) {
	orgXID, _ := xid.FromString(orgID)
	
	// Get paginated members
	offset := startIndex - 1
	if offset < 0 {
		offset = 0
	}
	
	members, err := s.orgService.ListMembers(ctx, orgXID, count, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	
	// Convert to SCIM users and apply filtering
	resources := make([]interface{}, 0, len(members))
	for _, member := range members {
		authUser, err := s.userService.FindByID(ctx, member.UserID)
		if err != nil {
			continue // Skip invalid users
		}
		
		scimUser := s.mapAuthSomeToSCIMUser(authUser, "")
		
		// Apply SCIM filter if provided
		if filter != "" {
			if !s.matchesSCIMFilter(scimUser, filter) {
				continue
			}
		}
		
		resources = append(resources, scimUser)
	}
	
	// Get total count (after filtering)
	// Note: In production, you'd want to apply filters at the DB level
	total, err := s.orgService.CountMembers(ctx, orgXID)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}
	
	return &ListResponse{
		Schemas:      []string{SchemaListResponse},
		TotalResults: total,
		StartIndex:   startIndex,
		ItemsPerPage: len(resources),
		Resources:    resources,
	}, nil
}

// matchesSCIMFilter checks if a SCIM user matches the filter expression
// Implements basic SCIM filtering as per RFC 7644 Section 3.4.2.2
func (s *Service) matchesSCIMFilter(user *SCIMUser, filter string) bool {
	// Parse simple filter expressions like:
	// - userName eq "john@example.com"
	// - active eq true
	// - emails[type eq "work"].value co "@example.com"
	
	filter = strings.TrimSpace(filter)
	
	// Extract attribute, operator, and value
	parts := strings.Fields(filter)
	if len(parts) < 3 {
		return true // Invalid filter, include the user
	}
	
	attribute := parts[0]
	operator := strings.ToLower(parts[1])
	value := strings.Trim(strings.Join(parts[2:], " "), "\"")
	
	// Get the attribute value from the user
	var attrValue string
	switch attribute {
	case "userName":
		attrValue = user.UserName
	case "displayName":
		attrValue = user.DisplayName
	case "active":
		attrValue = fmt.Sprintf("%v", user.Active)
	case "externalId":
		attrValue = user.ExternalID
	default:
		// Handle complex paths like emails[type eq "work"].value
		if strings.Contains(attribute, "emails") {
			if len(user.Emails) > 0 {
				attrValue = user.Emails[0].Value
			}
		}
	}
	
	// Apply operator
	switch operator {
	case "eq": // Equal
		return strings.EqualFold(attrValue, value)
	case "ne": // Not equal
		return !strings.EqualFold(attrValue, value)
	case "co": // Contains
		return strings.Contains(strings.ToLower(attrValue), strings.ToLower(value))
	case "sw": // Starts with
		return strings.HasPrefix(strings.ToLower(attrValue), strings.ToLower(value))
	case "ew": // Ends with
		return strings.HasSuffix(strings.ToLower(attrValue), strings.ToLower(value))
	case "pr": // Present (has value)
		return attrValue != ""
	case "gt": // Greater than
		return attrValue > value
	case "ge": // Greater than or equal
		return attrValue >= value
	case "lt": // Less than
		return attrValue < value
	case "le": // Less than or equal
		return attrValue <= value
	default:
		return true // Unknown operator, include the user
	}
}

// Group Operations

// CreateGroup creates a new group (maps to team/role)
func (s *Service) CreateGroup(ctx context.Context, scimGroup *SCIMGroup, orgID string) (*SCIMGroup, error) {
	if !s.config.GroupSync.Enabled {
		return nil, fmt.Errorf("group synchronization is disabled")
	}
	
	orgXID, _ := xid.FromString(orgID)
	
	// Create team if sync to teams is enabled
	if s.config.GroupSync.SyncToTeams {
		team := &organization.Team{
			ID:             xid.New(),
			OrganizationID: orgXID,
			Name:           scimGroup.DisplayName,
			Description:    fmt.Sprintf("Synced from SCIM group: %s", scimGroup.ExternalID),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		
		if err := s.orgService.CreateTeam(ctx, team); err != nil {
			return nil, fmt.Errorf("failed to create team: %w", err)
		}
		
		// Store mapping
		mapping := &GroupMapping{
			ID:            xid.New(),
			OrgID:         orgXID,
			SCIMGroupID:   scimGroup.ExternalID,
			SCIMGroupName: scimGroup.DisplayName,
			MappingType:   "team",
			TargetID:      team.ID,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		
		if err := s.repo.CreateGroupMapping(ctx, mapping); err != nil {
			return nil, fmt.Errorf("failed to store group mapping: %w", err)
		}
		
		scimGroup.ID = team.ID.String()
	}
	
	// Sync members
	if len(scimGroup.Members) > 0 {
		if err := s.syncGroupMembers(ctx, scimGroup.ID, scimGroup.Members, orgID); err != nil {
			fmt.Printf("[SCIM] Failed to sync group members: %v\n", err)
		}
	}
	
	return scimGroup, nil
}

// GetGroup retrieves a group by ID
func (s *Service) GetGroup(ctx context.Context, id, orgID string) (*SCIMGroup, error) {
	if !s.config.GroupSync.Enabled {
		return nil, fmt.Errorf("group synchronization is disabled")
	}
	
	teamXID, err := xid.FromString(id)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}
	
	_ = orgID // orgID may be used for validation in production
	
	// Get group mapping
	mapping, err := s.repo.FindGroupMappingByTargetID(ctx, teamXID)
	if err != nil {
		return nil, fmt.Errorf("group mapping not found: %w", err)
	}
	
	// Get team details
	team, err := s.orgService.FindTeamByID(ctx, teamXID)
	if err != nil {
		return nil, fmt.Errorf("team not found: %w", err)
	}
	
	// Get team members
	members, _ := s.orgService.ListTeamMembers(ctx, teamXID, 1000, 0)
	
	// Build SCIM group
	scimGroup := &SCIMGroup{
		Schemas:     []string{SchemaGroup},
		ID:          team.ID.String(),
		ExternalID:  mapping.SCIMGroupID,
		DisplayName: team.Name,
		Meta: &Meta{
			ResourceType: "Group",
			Created:      team.CreatedAt,
			LastModified: team.UpdatedAt,
			Location:     fmt.Sprintf("/scim/v2/Groups/%s", team.ID.String()),
		},
	}
	
	// Add members
	for _, tm := range members {
		member, _ := s.orgService.FindMemberByID(ctx, tm.MemberID)
		if member != nil {
			scimGroup.Members = append(scimGroup.Members, MemberReference{
				Value:   member.UserID.String(),
				Display: "", // Would need to fetch user details
			})
		}
	}
	
	return scimGroup, nil
}

// UpdateGroup updates a group via PATCH
func (s *Service) UpdateGroup(ctx context.Context, id, orgID string, patch *PatchOp) (*SCIMGroup, error) {
	if !s.config.GroupSync.Enabled {
		return nil, fmt.Errorf("group synchronization is disabled")
	}
	
	teamXID, err := xid.FromString(id)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}
	
	team, err := s.orgService.FindTeamByID(ctx, teamXID)
	if err != nil {
		return nil, fmt.Errorf("team not found: %w", err)
	}
	
	// Apply patch operations
	for _, op := range patch.Operations {
		if err := s.applyGroupPatchOperation(ctx, team, &op, orgID); err != nil {
			return nil, fmt.Errorf("failed to apply patch operation: %w", err)
		}
	}
	
	// Update team
	if err := s.orgService.UpdateTeam(ctx, team); err != nil {
		return nil, fmt.Errorf("failed to update team: %w", err)
	}
	
	return s.GetGroup(ctx, id, orgID)
}

// ReplaceGroup replaces a group via PUT
func (s *Service) ReplaceGroup(ctx context.Context, id, orgID string, scimGroup *SCIMGroup) (*SCIMGroup, error) {
	if !s.config.GroupSync.Enabled {
		return nil, fmt.Errorf("group synchronization is disabled")
	}
	
	teamXID, err := xid.FromString(id)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}
	
	team, err := s.orgService.FindTeamByID(ctx, teamXID)
	if err != nil {
		return nil, fmt.Errorf("team not found: %w", err)
	}
	
	// Update team properties
	team.Name = scimGroup.DisplayName
	team.UpdatedAt = time.Now()
	
	if err := s.orgService.UpdateTeam(ctx, team); err != nil {
		return nil, fmt.Errorf("failed to update team: %w", err)
	}
	
	// Sync members
	if len(scimGroup.Members) > 0 {
		if err := s.syncGroupMembers(ctx, id, scimGroup.Members, orgID); err != nil {
			fmt.Printf("[SCIM] Failed to sync group members: %v\n", err)
		}
	}
	
	return s.GetGroup(ctx, id, orgID)
}

// DeleteGroup deletes a group
func (s *Service) DeleteGroup(ctx context.Context, id, orgID string) error {
	if !s.config.GroupSync.Enabled {
		return fmt.Errorf("group synchronization is disabled")
	}
	
	teamXID, err := xid.FromString(id)
	if err != nil {
		return fmt.Errorf("invalid group ID: %w", err)
	}
	
	// Delete group mapping
	mapping, err := s.repo.FindGroupMappingByTargetID(ctx, teamXID)
	if err == nil && mapping != nil {
		_ = s.repo.DeleteGroupMapping(ctx, mapping.ID)
	}
	
	// Delete team
	if err := s.orgService.DeleteTeam(ctx, teamXID); err != nil {
		return fmt.Errorf("failed to delete team: %w", err)
	}
	
	return nil
}

// ListGroups lists groups with filtering and pagination
func (s *Service) ListGroups(ctx context.Context, orgID string, filter string, startIndex, count int) (*ListResponse, error) {
	if !s.config.GroupSync.Enabled {
		return &ListResponse{
			Schemas:      []string{SchemaListResponse},
			TotalResults: 0,
			StartIndex:   startIndex,
			ItemsPerPage: 0,
			Resources:    []interface{}{},
		}, nil
	}
	
	orgXID, _ := xid.FromString(orgID)
	
	// Get total count
	total, err := s.orgService.CountTeams(ctx, orgXID)
	if err != nil {
		return nil, fmt.Errorf("failed to count teams: %w", err)
	}
	
	// Get paginated teams
	offset := startIndex - 1
	if offset < 0 {
		offset = 0
	}
	
	teams, err := s.orgService.ListTeams(ctx, orgXID, count, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}
	
	// Convert to SCIM groups
	resources := make([]interface{}, 0, len(teams))
	for _, team := range teams {
		// Get group mapping for external ID
		mapping, _ := s.repo.FindGroupMappingByTargetID(ctx, team.ID)
		
		scimGroup := &SCIMGroup{
			Schemas:     []string{SchemaGroup},
			ID:          team.ID.String(),
			DisplayName: team.Name,
			Meta: &Meta{
				ResourceType: "Group",
				Created:      team.CreatedAt,
				LastModified: team.UpdatedAt,
				Location:     fmt.Sprintf("/scim/v2/Groups/%s", team.ID.String()),
			},
		}
		
		if mapping != nil {
			scimGroup.ExternalID = mapping.SCIMGroupID
		}
		
		resources = append(resources, scimGroup)
	}
	
	return &ListResponse{
		Schemas:      []string{SchemaListResponse},
		TotalResults: total,
		StartIndex:   startIndex,
		ItemsPerPage: len(resources),
		Resources:    resources,
	}, nil
}

// Token Management

// CreateProvisioningToken creates a new SCIM provisioning token
func (s *Service) CreateProvisioningToken(ctx context.Context, orgID, name, description string, scopes []string, expiresAt *time.Time) (string, *ProvisioningToken, error) {
	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", nil, fmt.Errorf("failed to generate token: %w", err)
	}
	
	token := base64.URLEncoding.EncodeToString(tokenBytes)
	tokenPrefix := token[:8] // First 8 chars for identification
	
	// Hash token for storage
	hashedToken, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	if err != nil {
		return "", nil, fmt.Errorf("failed to hash token: %w", err)
	}
	
	orgXID, _ := xid.FromString(orgID)
	
	provToken := &ProvisioningToken{
		ID:          xid.New(),
		OrgID:       orgXID,
		Name:        name,
		Description: description,
		TokenHash:   string(hashedToken),
		TokenPrefix: tokenPrefix,
		Scopes:      scopes,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	if err := s.repo.CreateProvisioningToken(ctx, provToken); err != nil {
		return "", nil, fmt.Errorf("failed to store token: %w", err)
	}
	
	// Record metrics
	s.metrics.RecordTokenCreation()
	s.metrics.RecordOperation("CREATE_TOKEN", "success", orgID)
	
	// Return the plain token only once
	return token, provToken, nil
}

// ValidateProvisioningToken validates a bearer token
func (s *Service) ValidateProvisioningToken(ctx context.Context, token string) (*ProvisioningToken, error) {
	if len(token) < 8 {
		s.metrics.RecordTokenValidation(false)
		s.metrics.RecordError("token_invalid_format")
		return nil, fmt.Errorf("invalid token format")
	}
	
	tokenPrefix := token[:8]
	
	// Find token by prefix
	provToken, err := s.repo.FindProvisioningTokenByPrefix(ctx, tokenPrefix)
	if err != nil {
		s.metrics.RecordTokenValidation(false)
		s.metrics.RecordError("token_not_found")
		return nil, fmt.Errorf("token not found: %w", err)
	}
	
	// Check if revoked
	if provToken.RevokedAt != nil {
		s.metrics.RecordTokenValidation(false)
		s.metrics.RecordError("token_revoked")
		return nil, fmt.Errorf("token has been revoked")
	}
	
	// Check expiry
	if provToken.ExpiresAt != nil && time.Now().After(*provToken.ExpiresAt) {
		s.metrics.RecordTokenValidation(false)
		s.metrics.RecordError("token_expired")
		return nil, fmt.Errorf("token has expired")
	}
	
	// Verify token hash
	if err := bcrypt.CompareHashAndPassword([]byte(provToken.TokenHash), []byte(token)); err != nil {
		s.metrics.RecordTokenValidation(false)
		s.metrics.RecordError("token_invalid_hash")
		return nil, fmt.Errorf("invalid token")
	}
	
	// Update last used timestamp
	now := time.Now()
	provToken.LastUsedAt = &now
	_ = s.repo.UpdateProvisioningToken(ctx, provToken)
	
	// Record successful validation
	s.metrics.RecordTokenValidation(true)
	
	return provToken, nil
}

// Helper methods

func (s *Service) validateUserAttributes(scimUser *SCIMUser) error {
	for _, attr := range s.config.UserProvisioning.RequiredAttributes {
		switch attr {
		case "userName":
			if scimUser.UserName == "" {
				return fmt.Errorf("userName is required")
			}
		case "emails":
			if len(scimUser.Emails) == 0 {
				return fmt.Errorf("at least one email is required")
			}
		}
	}
	return nil
}

func (s *Service) getPrimaryEmail(scimUser *SCIMUser) string {
	for _, email := range scimUser.Emails {
		if email.Primary {
			return email.Value
		}
	}
	if len(scimUser.Emails) > 0 {
		return scimUser.Emails[0].Value
	}
	return ""
}

func (s *Service) mapSCIMToAuthSomeUser(scimUser *SCIMUser, orgID string) (*user.User, error) {
	email := s.getPrimaryEmail(scimUser)
	
	name := scimUser.DisplayName
	if name == "" && scimUser.Name != nil {
		name = strings.TrimSpace(scimUser.Name.GivenName + " " + scimUser.Name.FamilyName)
	}
	
	// Note: metadata like scim_external_id, scim_username, employee_number, department
	// would need to be stored in a separate SCIM attributes table
	// since user.User doesn't have a Metadata field
	
	return &user.User{
		ID:            xid.New(),
		Email:         email,
		Name:          name,
		EmailVerified: scimUser.Active,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}, nil
}

func (s *Service) mapAuthSomeToSCIMUser(authUser *user.User, externalID string) *SCIMUser {
	scimUser := &SCIMUser{
		Schemas:     []string{SchemaCore},
		ID:          authUser.ID.String(),
		ExternalID:  externalID,
		UserName:    authUser.Email, // Default to email
		DisplayName: authUser.Name,
		Active:      authUser.EmailVerified,
		Emails: []Email{
			{
				Value:   authUser.Email,
				Primary: true,
				Type:    "work",
			},
		},
		Meta: &Meta{
			ResourceType: "User",
			Created:      authUser.CreatedAt,
			LastModified: authUser.UpdatedAt,
			Location:     fmt.Sprintf("/scim/v2/Users/%s", authUser.ID.String()),
		},
	}
	
	// Parse name from display name if available
	parts := strings.Fields(authUser.Name)
	if len(parts) > 0 {
		scimUser.Name = &Name{
			Formatted: authUser.Name,
		}
		if len(parts) == 1 {
			scimUser.Name.GivenName = parts[0]
		} else {
			scimUser.Name.GivenName = parts[0]
			scimUser.Name.FamilyName = strings.Join(parts[1:], " ")
		}
	}
	
	return scimUser
}

func (s *Service) applyPatchOperationToRequest(authUser *user.User, op *PatchOperation, updateReq *user.UpdateUserRequest) error {
	// Simplified patch operation handling
	switch op.Op {
	case "replace":
		if op.Path == "active" {
			if active, ok := op.Value.(bool); ok {
				updateReq.EmailVerified = boolPtr(active)
			}
		} else if op.Path == "name.givenName" || op.Path == "name.familyName" || op.Path == "displayName" {
			if name, ok := op.Value.(string); ok {
				updateReq.Name = stringPtr(name)
			}
		} else if op.Path == "emails[type eq \"work\"].value" || op.Path == "emails[primary eq true].value" {
			if email, ok := op.Value.(string); ok {
				updateReq.Email = stringPtr(email)
			}
		}
	case "add":
		// Handle add operations (similar to replace)
	case "remove":
		// Handle remove operations
	}
	return nil
}

// Helper functions for pointer conversion
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func (s *Service) applyGroupPatchOperation(ctx context.Context, team *organization.Team, op *PatchOperation, orgID string) error {
	switch op.Op {
	case "replace":
		if op.Path == "displayName" {
			if name, ok := op.Value.(string); ok {
				team.Name = name
			}
		} else if op.Path == "members" {
			// Handle member replacement
			if members, ok := op.Value.([]interface{}); ok {
				teamXID := team.ID
				// Remove all existing members
				existingMembers, _ := s.orgService.ListTeamMembers(ctx, teamXID, 1000, 0)
				for _, tm := range existingMembers {
					_ = s.orgService.RemoveTeamMember(ctx, teamXID, tm.MemberID)
				}
				
				// Add new members
				for _, m := range members {
					if memberMap, ok := m.(map[string]interface{}); ok {
						if userIDStr, ok := memberMap["value"].(string); ok {
							userID, _ := xid.FromString(userIDStr)
							orgXID, _ := xid.FromString(orgID)
							member, _ := s.orgService.FindMember(ctx, orgXID, userID)
							if member != nil {
								tm := &organization.TeamMember{
									ID:        xid.New(),
									TeamID:    teamXID,
									MemberID:  member.ID,
									CreatedAt: time.Now(),
								}
								_ = s.orgService.AddTeamMember(ctx, tm)
							}
						}
					}
				}
			}
		}
	case "add":
		if op.Path == "members" {
			// Add members to group
			teamXID := team.ID
			if members, ok := op.Value.([]interface{}); ok {
				for _, m := range members {
					if memberMap, ok := m.(map[string]interface{}); ok {
						if userIDStr, ok := memberMap["value"].(string); ok {
							userID, _ := xid.FromString(userIDStr)
							orgXID, _ := xid.FromString(orgID)
							member, _ := s.orgService.FindMember(ctx, orgXID, userID)
							if member != nil {
								tm := &organization.TeamMember{
									ID:        xid.New(),
									TeamID:    teamXID,
									MemberID:  member.ID,
									CreatedAt: time.Now(),
								}
								_ = s.orgService.AddTeamMember(ctx, tm)
							}
						}
					}
				}
			}
		}
	case "remove":
		if strings.HasPrefix(op.Path, "members[") {
			// Remove specific member
			// Parse member ID from path like "members[value eq \"123\"]"
			teamXID := team.ID
			// For simplicity, if value is provided, use that
			if memberIDStr, ok := op.Value.(string); ok {
				userID, _ := xid.FromString(memberIDStr)
				orgXID, _ := xid.FromString(orgID)
				member, _ := s.orgService.FindMember(ctx, orgXID, userID)
				if member != nil {
					_ = s.orgService.RemoveTeamMember(ctx, teamXID, member.ID)
				}
			}
		}
	}
	return nil
}

func (s *Service) syncUserGroups(ctx context.Context, userID xid.ID, groups []GroupReference, orgID string) error {
	if !s.config.GroupSync.Enabled {
		return nil
	}
	
	orgXID, _ := xid.FromString(orgID)
	
	// Get user's organization member record
	member, err := s.orgService.FindMember(ctx, orgXID, userID)
	if err != nil {
		return fmt.Errorf("user not found in organization: %w", err)
	}
	
	// Get user's current teams
	// For each group in the SCIM request, add user to that team
	for _, groupRef := range groups {
		// Find team by external ID (group value)
		mapping, err := s.repo.FindGroupMappingBySCIMID(ctx, groupRef.Value, orgXID)
		if err != nil {
			fmt.Printf("[SCIM] Group mapping not found for %s: %v\n", groupRef.Value, err)
			continue
		}
		
		// Add user to team
		tm := &organization.TeamMember{
			ID:        xid.New(),
			TeamID:    mapping.TargetID,
			MemberID:  member.ID,
			CreatedAt: time.Now(),
		}
		
		if err := s.orgService.AddTeamMember(ctx, tm); err != nil {
			fmt.Printf("[SCIM] Failed to add user to team: %v\n", err)
		}
	}
	
	return nil
}

func (s *Service) syncGroupMembers(ctx context.Context, groupID string, members []MemberReference, orgID string) error {
	if !s.config.GroupSync.Enabled {
		return nil
	}
	
	teamXID, err := xid.FromString(groupID)
	if err != nil {
		return fmt.Errorf("invalid group ID: %w", err)
	}
	
	orgXID, _ := xid.FromString(orgID)
	
	// Clear existing members (for full sync)
	existingMembers, _ := s.orgService.ListTeamMembers(ctx, teamXID, 1000, 0)
	for _, tm := range existingMembers {
		_ = s.orgService.RemoveTeamMember(ctx, teamXID, tm.MemberID)
	}
	
	// Add new members
	for _, memberRef := range members {
		userID, err := xid.FromString(memberRef.Value)
		if err != nil {
			continue
		}
		
		// Get organization member
		member, err := s.orgService.FindMember(ctx, orgXID, userID)
		if err != nil {
			fmt.Printf("[SCIM] User %s not found in organization\n", memberRef.Value)
			continue
		}
		
		// Add to team
		tm := &organization.TeamMember{
			ID:        xid.New(),
			TeamID:    teamXID,
			MemberID:  member.ID,
			CreatedAt: time.Now(),
		}
		
		if err := s.orgService.AddTeamMember(ctx, tm); err != nil {
			fmt.Printf("[SCIM] Failed to add member to team: %v\n", err)
		}
	}
	
	return nil
}

func (s *Service) sendProvisioningWebhook(ctx context.Context, event string, data map[string]interface{}) error {
	if !s.config.Webhooks.Enabled {
		return nil
	}
	
	// Use webhook service to send notifications
	return nil
}

// Bulk Operations (RFC 7644 Section 3.7)

// ProcessBulkOperation processes a bulk operation request
func (s *Service) ProcessBulkOperation(ctx context.Context, bulkReq *BulkRequest, orgID string) (*BulkResponse, error) {
	response := &BulkResponse{
		Schemas:    []string{SchemaBulkResponse},
		Operations: []BulkOperationResult{},
	}
	
	// Process each operation
	for i, op := range bulkReq.Operations {
		bulkID := op.BulkID
		if bulkID == "" {
			bulkID = fmt.Sprintf("bulk_%d", i)
		}
		
		respOp := BulkOperationResult{
			BulkID:   bulkID,
			Method:   op.Method,
			Location: op.Path,
		}
		
		// Process based on method and path
		switch strings.ToUpper(op.Method) {
		case "POST":
			if strings.Contains(op.Path, "/Users") {
				// Create user
				scimUser, ok := op.Data.(*SCIMUser)
				if !ok {
					respOp.Status = 400
					respOp.Response = &ErrorResponse{
						Schemas: []string{SchemaError},
						Status:  400,
						Detail:  "Invalid user data",
					}
				} else {
					user, err := s.CreateUser(ctx, scimUser, orgID)
					if err != nil {
						respOp.Status = 400
						respOp.Response = &ErrorResponse{
							Schemas: []string{SchemaError},
							Status:  400,
							Detail:  err.Error(),
						}
					} else {
						respOp.Status = 201
						respOp.Location = fmt.Sprintf("/scim/v2/Users/%s", user.ID)
						respOp.Response = user
					}
				}
			} else if strings.Contains(op.Path, "/Groups") {
				// Create group
				scimGroup, ok := op.Data.(*SCIMGroup)
				if !ok {
					respOp.Status = 400
					respOp.Response = &ErrorResponse{
						Schemas: []string{SchemaError},
						Status:  400,
						Detail:  "Invalid group data",
					}
				} else {
					group, err := s.CreateGroup(ctx, scimGroup, orgID)
					if err != nil {
						respOp.Status = 400
						respOp.Response = &ErrorResponse{
							Schemas: []string{SchemaError},
							Status:  400,
							Detail:  err.Error(),
						}
					} else {
						respOp.Status = 201
						respOp.Location = fmt.Sprintf("/scim/v2/Groups/%s", group.ID)
						respOp.Response = group
					}
				}
			}
			
		case "PUT":
			// Extract resource ID from path
			pathParts := strings.Split(op.Path, "/")
			if len(pathParts) < 2 {
				respOp.Status = 400
				respOp.Response = &ErrorResponse{
					Schemas: []string{SchemaError},
					Status:  400,
					Detail:  "Invalid path",
				}
			} else {
				resourceID := pathParts[len(pathParts)-1]
				
				if strings.Contains(op.Path, "/Users/") {
					scimUser, ok := op.Data.(*SCIMUser)
					if !ok {
						respOp.Status = 400
						respOp.Response = &ErrorResponse{
							Schemas: []string{SchemaError},
							Status:  400,
							Detail:  "Invalid user data",
						}
					} else {
						user, err := s.ReplaceUser(ctx, resourceID, orgID, scimUser)
						if err != nil {
							respOp.Status = 400
							respOp.Response = &ErrorResponse{
								Schemas: []string{SchemaError},
								Status:  400,
								Detail:  err.Error(),
							}
						} else {
							respOp.Status = 200
							respOp.Response = user
						}
					}
				}
			}
			
		case "PATCH":
			pathParts := strings.Split(op.Path, "/")
			if len(pathParts) < 2 {
				respOp.Status = 400
				respOp.Response = &ErrorResponse{
					Schemas: []string{SchemaError},
					Status:  400,
					Detail:  "Invalid path",
				}
			} else {
				resourceID := pathParts[len(pathParts)-1]
				
				if strings.Contains(op.Path, "/Users/") {
					patchOp, ok := op.Data.(*PatchOp)
					if !ok {
						respOp.Status = 400
						respOp.Response = &ErrorResponse{
							Schemas: []string{SchemaError},
							Status:  400,
							Detail:  "Invalid patch data",
						}
					} else {
						user, err := s.UpdateUser(ctx, resourceID, orgID, patchOp)
						if err != nil {
							respOp.Status = 400
							respOp.Response = &ErrorResponse{
								Schemas: []string{SchemaError},
								Status:  400,
								Detail:  err.Error(),
							}
						} else {
							respOp.Status = 200
							respOp.Response = user
						}
					}
				}
			}
			
		case "DELETE":
			pathParts := strings.Split(op.Path, "/")
			if len(pathParts) < 2 {
				respOp.Status = 400
				respOp.Response = &ErrorResponse{
					Schemas: []string{SchemaError},
					Status:  400,
					Detail:  "Invalid path",
				}
			} else {
				resourceID := pathParts[len(pathParts)-1]
				
				if strings.Contains(op.Path, "/Users/") {
					err := s.DeleteUser(ctx, resourceID, orgID)
					if err != nil {
						respOp.Status = 400
						respOp.Response = &ErrorResponse{
							Schemas: []string{SchemaError},
							Status:  400,
							Detail:  err.Error(),
						}
					} else {
						respOp.Status = 204
					}
				} else if strings.Contains(op.Path, "/Groups/") {
					err := s.DeleteGroup(ctx, resourceID, orgID)
					if err != nil {
						respOp.Status = 400
						respOp.Response = &ErrorResponse{
							Schemas: []string{SchemaError},
							Status:  400,
							Detail:  err.Error(),
						}
					} else {
						respOp.Status = 204
					}
				}
			}
		}
		
		response.Operations = append(response.Operations, respOp)
		
		// Check fail on errors mode
		if bulkReq.FailOnErrors > 0 {
			failCount := 0
			for _, ro := range response.Operations {
				if ro.Status >= 400 {
					failCount++
				}
			}
			if failCount >= bulkReq.FailOnErrors {
				break
			}
		}
	}
	
	return response, nil
}

// Token Management Operations

// ListProvisioningTokens lists all provisioning tokens for an organization
func (s *Service) ListProvisioningTokens(ctx context.Context, orgID string, limit, offset int) ([]*ProvisioningToken, int, error) {
	orgXID, _ := xid.FromString(orgID)
	
	tokens, err := s.repo.ListProvisioningTokens(ctx, orgXID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tokens: %w", err)
	}
	
	// Get total count
	total := len(tokens) // Simplified; in production, query count separately
	
	return tokens, total, nil
}

// RevokeProvisioningToken revokes a provisioning token
func (s *Service) RevokeProvisioningToken(ctx context.Context, tokenID string) error {
	tokenXID, err := xid.FromString(tokenID)
	if err != nil {
		return fmt.Errorf("invalid token ID: %w", err)
	}
	
	return s.repo.RevokeProvisioningToken(ctx, tokenXID)
}

// Attribute Mapping Operations

// GetAttributeMappings retrieves attribute mappings for an organization
func (s *Service) GetAttributeMappings(ctx context.Context, orgID string) (map[string]string, error) {
	orgXID, _ := xid.FromString(orgID)
	
	mapping, err := s.repo.FindAttributeMappingByOrgID(ctx, orgXID)
	if err != nil {
		// Return default mappings if none exist
		return s.config.AttributeMapping.CustomMapping, nil
	}
	
	return mapping.Mappings, nil
}

// UpdateAttributeMappings updates attribute mappings for an organization
func (s *Service) UpdateAttributeMappings(ctx context.Context, orgID string, mappings map[string]string) error {
	orgXID, _ := xid.FromString(orgID)
	
	// Find existing mapping
	existingMapping, err := s.repo.FindAttributeMappingByOrgID(ctx, orgXID)
	if err != nil {
		// Create new mapping
		mapping := &AttributeMapping{
			ID:        xid.New(),
			OrgID:     orgXID,
			Mappings:  mappings,
			Metadata:  make(map[string]interface{}),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		return s.repo.CreateAttributeMapping(ctx, mapping)
	}
	
	// Update existing mapping
	existingMapping.Mappings = mappings
	existingMapping.UpdatedAt = time.Now()
	return s.repo.UpdateAttributeMapping(ctx, existingMapping)
}

// Provisioning Logs Operations

// GetProvisioningLogs retrieves provisioning logs with filtering
func (s *Service) GetProvisioningLogs(ctx context.Context, orgID string, action string, limit, offset int) ([]*ProvisioningLog, int, error) {
	orgXID, _ := xid.FromString(orgID)
	
	// Build filters
	filters := make(map[string]interface{})
	if action != "" {
		filters["operation"] = action
	}
	
	logs, err := s.repo.ListProvisioningLogs(ctx, orgXID, filters, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list logs: %w", err)
	}
	
	// Get total count
	total, err := s.repo.CountProvisioningLogs(ctx, orgXID, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count logs: %w", err)
	}
	
	return logs, total, nil
}

// CreateProvisioningLog creates a new provisioning log entry
func (s *Service) CreateProvisioningLog(ctx context.Context, log *ProvisioningLog) error {
	log.ID = xid.New()
	log.CreatedAt = time.Now()
	return s.repo.CreateProvisioningLog(ctx, log)
}

// Lifecycle methods

// Migrate runs database migrations
func (s *Service) Migrate(ctx context.Context) error {
	return s.repo.Migrate(ctx)
}

// InitializeOrgSCIMConfig initializes default SCIM config for an organization
func (s *Service) InitializeOrgSCIMConfig(ctx context.Context, orgID string) error {
	// Create default attribute mapping
	orgXID, _ := xid.FromString(orgID)
	mapping := &AttributeMapping{
		ID:       xid.New(),
		OrgID:    orgXID,
		Mappings: s.config.AttributeMapping.CustomMapping,
		Metadata: make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	return s.repo.CreateAttributeMapping(ctx, mapping)
}

// SendProvisioningWebhook sends a provisioning webhook
func (s *Service) SendProvisioningWebhook(ctx context.Context, event string, data map[string]interface{}) error {
	return s.sendProvisioningWebhook(ctx, event, data)
}

// Shutdown gracefully shuts down the service
func (s *Service) Shutdown(ctx context.Context) error {
	// Cleanup resources
	return nil
}

// Health checks service health
func (s *Service) Health(ctx context.Context) error {
	// Check database connectivity
	return s.repo.Ping(ctx)
}

