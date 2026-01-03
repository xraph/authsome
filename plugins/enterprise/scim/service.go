package scim

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/core/webhook"
	orgplugin "github.com/xraph/authsome/plugins/organization"
	"golang.org/x/crypto/bcrypt"
)

// SCIMOrgService defines a unified interface for organization/app operations
// Supports both app mode (multitenancy) and organization mode (organization plugin)
type SCIMOrgService interface {
	// Member operations
	AddMember(ctx context.Context, orgID, userID xid.ID, role string) (interface{}, error)
	IsUserMember(ctx context.Context, orgID, userID xid.ID) (bool, error)
	ListMembers(ctx context.Context, orgID xid.ID, limit, offset int) ([]interface{}, error)

	// Team operations
	CreateTeam(ctx context.Context, orgID xid.ID, req interface{}) (interface{}, error)
	GetTeam(ctx context.Context, id xid.ID) (interface{}, error)
	ListTeams(ctx context.Context, orgID xid.ID, limit, offset int) ([]interface{}, error)
	UpdateTeam(ctx context.Context, id xid.ID, req interface{}) (interface{}, error)
	DeleteTeam(ctx context.Context, id xid.ID) error
	AddTeamMember(ctx context.Context, teamID, memberID xid.ID, role string) error
	RemoveTeamMember(ctx context.Context, teamID, memberID xid.ID) error
	ListTeamMembers(ctx context.Context, teamID xid.ID) ([]interface{}, error)

	// Get member ID from user ID (for team operations)
	GetMemberIDByUserID(ctx context.Context, orgID, userID xid.ID) (xid.ID, error)
}

// ServiceConfig holds service dependencies
type ServiceConfig struct {
	Config         *Config
	Repository     *Repository
	UserService    user.ServiceInterface // Use interface to support decorated services
	OrgService     interface{}           // Can be *app.ServiceImpl or *orgplugin.ServiceImpl
	AuditService   *audit.Service
	WebhookService *webhook.Service
}

// Service provides SCIM provisioning business logic
type Service struct {
	config         *Config
	repo           *Repository
	userService    user.ServiceInterface // Use interface to support decorated services
	orgService     interface{}           // Can be *app.ServiceImpl or *orgplugin.ServiceImpl
	scimOrgService SCIMOrgService        // Unified interface adapter
	auditService   *audit.Service
	webhookService *webhook.Service
	metrics        *Metrics
	mode           string // "app" or "organization"
}

// NewService creates a new SCIM service
func NewService(cfg ServiceConfig) *Service {
	service := &Service{
		config:         cfg.Config,
		repo:           cfg.Repository,
		userService:    cfg.UserService,
		orgService:     cfg.OrgService,
		auditService:   cfg.AuditService,
		webhookService: cfg.WebhookService,
		metrics:        GetMetrics(),
	}

	// Detect mode and create appropriate adapter
	if appSvc, ok := cfg.OrgService.(*app.ServiceImpl); ok {
		service.scimOrgService = &appServiceAdapter{service: appSvc}
		service.mode = "app"
	} else if orgSvc, ok := cfg.OrgService.(*orgplugin.Service); ok {
		service.scimOrgService = &orgServiceAdapter{service: orgSvc}
		service.mode = "organization"
	} else {
		// Fallback: try to use as app service
		if cfg.OrgService != nil {
			service.scimOrgService = &appServiceAdapter{service: cfg.OrgService.(*app.ServiceImpl)}
			service.mode = "app"
		}
	}

	return service
}

// getOrgService returns the unified SCIM organization service adapter
func (s *Service) getOrgService() SCIMOrgService {
	if s.scimOrgService == nil {
		panic("SCIM plugin requires multitenancy or organization plugin - organization service not available")
	}
	return s.scimOrgService
}

// User Provisioning Operations

// CreateUser provisions a new user via SCIM
func (s *Service) CreateUser(ctx context.Context, scimUser *SCIMUser, orgID xid.ID) (*SCIMUser, error) {
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
		}
	}

	// Add user to organization/app
	_, err = s.getOrgService().AddMember(ctx, orgID, createdUser.ID, s.config.UserProvisioning.DefaultRole)
	if err != nil {
		// Rollback user creation?
		return nil, fmt.Errorf("failed to add user to organization: %w", err)
	}

	// Sync groups if provided
	if len(scimUser.Groups) > 0 && s.config.GroupSync.Enabled {
		if err := s.syncUserGroups(ctx, createdUser.ID, scimUser.Groups, orgID); err != nil {
		}
	}

	// Convert back to SCIM format
	resultUser := s.mapAuthSomeToSCIMUser(createdUser, scimUser.ExternalID)

	// Record metrics
	s.metrics.RecordUserOperation("create")
	s.metrics.RecordOperation("CREATE_USER", "success", orgID.String())

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
func (s *Service) GetUser(ctx context.Context, id, orgID xid.ID) (*SCIMUser, error) {
	authUser, err := s.userService.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Verify user belongs to organization/app
	isMember, err := s.getOrgService().IsUserMember(ctx, orgID, authUser.ID)
	if err != nil || !isMember {
		return nil, fmt.Errorf("user not found in organization")
	}

	scimUser := s.mapAuthSomeToSCIMUser(authUser, "")

	// Record metrics
	s.metrics.RecordUserOperation("read")
	s.metrics.RecordOperation("GET_USER", "success", orgID.String())

	return scimUser, nil
}

// UpdateUser updates a user via SCIM PATCH
func (s *Service) UpdateUser(ctx context.Context, id, orgID xid.ID, patch *PatchOp) (*SCIMUser, error) {
	authUser, err := s.userService.FindByID(ctx, id)
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
	s.metrics.RecordOperation("PATCH_USER", "success", orgID.String())

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
func (s *Service) ReplaceUser(ctx context.Context, id, orgID xid.ID, scimUser *SCIMUser) (*SCIMUser, error) {
	authUser, err := s.userService.FindByID(ctx, id)
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
func (s *Service) DeleteUser(ctx context.Context, id, orgID xid.ID) error {
	// Soft delete or hard delete based on config
	if s.config.UserProvisioning.SoftDeleteOnDeProvision {
		// Soft delete: deactivate user by setting email verified to false
		authUser, err := s.userService.FindByID(ctx, id)
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
		if err := s.userService.Delete(ctx, id); err != nil {
			return fmt.Errorf("failed to delete user: %w", err)
		}
	}

	// Record metrics
	s.metrics.RecordUserOperation("delete")
	s.metrics.RecordOperation("DELETE_USER", "success", orgID.String())

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
func (s *Service) ListUsers(ctx context.Context, orgID xid.ID, filter string, startIndex, count int) (*ListResponse, error) {
	// Get paginated members
	offset := startIndex - 1
	if offset < 0 {
		offset = 0
	}

	memberList, err := s.getOrgService().ListMembers(ctx, orgID, count, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Convert to SCIM users and apply filtering
	resources := make([]interface{}, 0, len(memberList))
	for _, memberInterface := range memberList {
		// Extract UserID from member (works for both app.Member and schema.OrganizationMember)
		var userID xid.ID
		switch m := memberInterface.(type) {
		case *app.Member:
			userID = m.UserID
		default:
			// For organization plugin, use type assertion with reflection
			// Both types have UserID field, so we can use a helper
			userID = extractUserIDFromMember(memberInterface)
			if userID.IsNil() {
				continue // Skip if we can't extract user ID
			}
		}

		authUser, err := s.userService.FindByID(ctx, userID)
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
	// Get total by counting all members (use large limit)
	totalMemberList, err := s.getOrgService().ListMembers(ctx, orgID, 10000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}
	total := len(totalMemberList)

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
func (s *Service) CreateGroup(ctx context.Context, scimGroup *SCIMGroup, orgID xid.ID) (*SCIMGroup, error) {
	if !s.config.GroupSync.Enabled {
		return nil, fmt.Errorf("group synchronization is disabled")
	}

	// Create team if sync to teams is enabled
	if s.config.GroupSync.SyncToTeams {
		description := fmt.Sprintf("Synced from SCIM group: %s", scimGroup.ExternalID)

		// Create team request based on mode
		var teamReq interface{}
		if s.mode == "organization" {
			desc := description
			teamReq = &orgplugin.CreateTeamRequest{
				Name:        scimGroup.DisplayName,
				Description: &desc,
			}
		} else {
			// App mode
			desc := description
			teamReq = &app.CreateTeamRequest{
				Name:        scimGroup.DisplayName,
				Description: &desc,
			}
		}

		teamInterface, err := s.getOrgService().CreateTeam(ctx, orgID, teamReq)
		if err != nil {
			return nil, fmt.Errorf("failed to create team: %w", err)
		}

		// Extract team ID from interface
		var teamID xid.ID
		switch t := teamInterface.(type) {
		case *app.Team:
			teamID = t.ID
		default:
			// Use reflection to get ID
			val := reflect.ValueOf(teamInterface)
			if val.Kind() == reflect.Ptr {
				val = val.Elem()
			}
			if val.Kind() == reflect.Struct {
				idField := val.FieldByName("ID")
				if idField.IsValid() && idField.CanInterface() {
					if id, ok := idField.Interface().(xid.ID); ok {
						teamID = id
					}
				}
			}
		}

		// Update team with SCIM provisioning information
		if err := s.updateTeamProvisioningInfo(ctx, teamID, scimGroup.ExternalID); err != nil {
			// Log warning but don't fail the operation
		}

		// Store mapping
		orgXID := orgID
		mapping := &GroupMapping{
			ID:             xid.New(),
			AppID:          xid.ID{},
			EnvironmentID:  xid.ID{},
			OrganizationID: orgXID,
			SCIMGroupID:    scimGroup.ExternalID,
			SCIMGroupName:  scimGroup.DisplayName,
			MappingType:    "team",
			TargetID:       teamID,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		if err := s.repo.CreateGroupMapping(ctx, mapping); err != nil {
			return nil, fmt.Errorf("failed to store group mapping: %w", err)
		}

		scimGroup.ID = teamID.String()
	}

	// Sync members
	if len(scimGroup.Members) > 0 {
		scimGroupID, err := xid.FromString(scimGroup.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid group ID: %w", err)
		}
		if err := s.syncGroupMembers(ctx, scimGroupID, scimGroup.Members, orgID); err != nil {
		}
	}

	return scimGroup, nil
}

// GetGroup retrieves a group by ID
func (s *Service) GetGroup(ctx context.Context, id, orgID xid.ID) (*SCIMGroup, error) {
	if !s.config.GroupSync.Enabled {
		return nil, fmt.Errorf("group synchronization is disabled")
	}

	// Get group mapping
	mapping, err := s.repo.FindGroupMappingByTargetID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("group mapping not found: %w", err)
	}

	// Get team details
	teamInterface, err := s.getOrgService().GetTeam(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("team not found: %w", err)
	}

	// Extract team info using reflection
	var teamID xid.ID
	var teamName string
	var createdAt, updatedAt time.Time

	switch t := teamInterface.(type) {
	case *app.Team:
		teamID = t.ID
		teamName = t.Name
		createdAt = t.CreatedAt
		updatedAt = t.UpdatedAt
	default:
		// Use reflection
		val := reflect.ValueOf(teamInterface)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		if val.Kind() == reflect.Struct {
			if idField := val.FieldByName("ID"); idField.IsValid() {
				if id, ok := idField.Interface().(xid.ID); ok {
					teamID = id
				}
			}
			if nameField := val.FieldByName("Name"); nameField.IsValid() {
				if name, ok := nameField.Interface().(string); ok {
					teamName = name
				}
			}
			if createdAtField := val.FieldByName("CreatedAt"); createdAtField.IsValid() {
				if ca, ok := createdAtField.Interface().(time.Time); ok {
					createdAt = ca
				}
			}
			if updatedAtField := val.FieldByName("UpdatedAt"); updatedAtField.IsValid() {
				if ua, ok := updatedAtField.Interface().(time.Time); ok {
					updatedAt = ua
				}
			}
		}
	}

	// Get team members
	memberList, _ := s.getOrgService().ListTeamMembers(ctx, id)

	// Build SCIM group
	scimGroup := &SCIMGroup{
		Schemas:     []string{SchemaGroup},
		ID:          teamID.String(),
		ExternalID:  mapping.SCIMGroupID,
		DisplayName: teamName,
		Meta: &SCIMMeta{
			ResourceType: "Group",
			Created:      createdAt,
			LastModified: updatedAt,
			Location:     fmt.Sprintf("/scim/v2/Groups/%s", teamID),
		},
	}

	// Add members
	for _, memberInterface := range memberList {
		userID := extractUserIDFromMember(memberInterface)
		if !userID.IsNil() {
			scimGroup.Members = append(scimGroup.Members, MemberReference{
				Value:   userID.String(),
				Display: "", // Would need to fetch user details
			})
		}
	}

	return scimGroup, nil
}

// UpdateGroup updates a group via PATCH
func (s *Service) UpdateGroup(ctx context.Context, id, orgID xid.ID, patch *PatchOp) (*SCIMGroup, error) {
	if !s.config.GroupSync.Enabled {
		return nil, fmt.Errorf("group synchronization is disabled")
	}

	teamInterface, err := s.getOrgService().GetTeam(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("team not found: %w", err)
	}

	// Apply patch operations
	for _, op := range patch.Operations {
		if err := s.applyGroupPatchOperation(ctx, teamInterface, &op, orgID); err != nil {
			return nil, fmt.Errorf("failed to apply patch operation: %w", err)
		}
	}

	// Extract team info for update request
	var teamName string
	var teamDesc *string
	var teamMetadata map[string]interface{}

	switch t := teamInterface.(type) {
	case *app.Team:
		teamName = t.Name
		teamDesc = &t.Description
		teamMetadata = t.Metadata
	default:
		// Use reflection
		val := reflect.ValueOf(teamInterface)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		if val.Kind() == reflect.Struct {
			if nameField := val.FieldByName("Name"); nameField.IsValid() {
				if name, ok := nameField.Interface().(string); ok {
					teamName = name
				}
			}
			if descField := val.FieldByName("Description"); descField.IsValid() {
				if desc, ok := descField.Interface().(*string); ok {
					teamDesc = desc
				} else if descStr, ok := descField.Interface().(string); ok {
					teamDesc = &descStr
				}
			}
			if metaField := val.FieldByName("Metadata"); metaField.IsValid() {
				if meta, ok := metaField.Interface().(map[string]interface{}); ok {
					teamMetadata = meta
				}
			}
		}
	}

	// Update team
	var updateReq interface{}
	if s.mode == "organization" {
		updateReq = &orgplugin.UpdateTeamRequest{
			Name:        &teamName,
			Description: teamDesc,
			Metadata:    teamMetadata,
		}
	} else {
		updateReq = &app.UpdateTeamRequest{
			Name:        &teamName,
			Description: teamDesc,
			Metadata:    teamMetadata,
		}
	}

	if _, err := s.getOrgService().UpdateTeam(ctx, id, updateReq); err != nil {
		return nil, fmt.Errorf("failed to update team: %w", err)
	}

	return s.GetGroup(ctx, id, orgID)
}

// ReplaceGroup replaces a group via PUT
func (s *Service) ReplaceGroup(ctx context.Context, id, orgID xid.ID, scimGroup *SCIMGroup) (*SCIMGroup, error) {
	if !s.config.GroupSync.Enabled {
		return nil, fmt.Errorf("group synchronization is disabled")
	}

	// Update team properties
	var updateReq interface{}
	if s.mode == "organization" {
		name := scimGroup.DisplayName
		updateReq = &orgplugin.UpdateTeamRequest{
			Name: &name,
		}
	} else {
		name := scimGroup.DisplayName
		updateReq = &app.UpdateTeamRequest{
			Name: &name,
		}
	}

	if _, err := s.getOrgService().UpdateTeam(ctx, id, updateReq); err != nil {
		return nil, fmt.Errorf("failed to update team: %w", err)
	}

	// Sync members
	if len(scimGroup.Members) > 0 {
		if err := s.syncGroupMembers(ctx, id, scimGroup.Members, orgID); err != nil {
		}
	}

	return s.GetGroup(ctx, id, orgID)
}

// DeleteGroup deletes a group
func (s *Service) DeleteGroup(ctx context.Context, id, orgID xid.ID) error {
	if !s.config.GroupSync.Enabled {
		return fmt.Errorf("group synchronization is disabled")
	}

	// Delete group mapping
	mapping, err := s.repo.FindGroupMappingByTargetID(ctx, id)
	if err == nil && mapping != nil {
		_ = s.repo.DeleteGroupMapping(ctx, mapping.ID)
	}

	// Delete team
	if err := s.getOrgService().DeleteTeam(ctx, id); err != nil {
		return fmt.Errorf("failed to delete team: %w", err)
	}

	return nil
}

// ListGroups lists groups with filtering and pagination
func (s *Service) ListGroups(ctx context.Context, orgID xid.ID, filter string, startIndex, count int) (*ListResponse, error) {
	if !s.config.GroupSync.Enabled {
		return &ListResponse{
			Schemas:      []string{SchemaListResponse},
			TotalResults: 0,
			StartIndex:   startIndex,
			ItemsPerPage: 0,
			Resources:    []interface{}{},
		}, nil
	}

	// Get paginated teams
	offset := startIndex - 1
	if offset < 0 {
		offset = 0
	}

	teamList, err := s.getOrgService().ListTeams(ctx, orgID, count, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}

	// Convert to SCIM groups
	resources := make([]interface{}, 0, len(teamList))
	for _, teamInterface := range teamList {
		// Extract team info using reflection
		var teamID xid.ID
		var teamName string
		var createdAt, updatedAt time.Time

		switch t := teamInterface.(type) {
		case *app.Team:
			teamID = t.ID
			teamName = t.Name
			createdAt = t.CreatedAt
			updatedAt = t.UpdatedAt
		default:
			// Use reflection
			val := reflect.ValueOf(teamInterface)
			if val.Kind() == reflect.Ptr {
				val = val.Elem()
			}
			if val.Kind() == reflect.Struct {
				if idField := val.FieldByName("ID"); idField.IsValid() {
					if id, ok := idField.Interface().(xid.ID); ok {
						teamID = id
					}
				}
				if nameField := val.FieldByName("Name"); nameField.IsValid() {
					if name, ok := nameField.Interface().(string); ok {
						teamName = name
					}
				}
				if createdAtField := val.FieldByName("CreatedAt"); createdAtField.IsValid() {
					if ca, ok := createdAtField.Interface().(time.Time); ok {
						createdAt = ca
					}
				}
				if updatedAtField := val.FieldByName("UpdatedAt"); updatedAtField.IsValid() {
					if ua, ok := updatedAtField.Interface().(time.Time); ok {
						updatedAt = ua
					}
				}
			}
		}

		// Get group mapping for external ID
		mapping, _ := s.repo.FindGroupMappingByTargetID(ctx, teamID)

		scimGroup := &SCIMGroup{
			Schemas:     []string{SchemaGroup},
			ID:          teamID.String(),
			DisplayName: teamName,
			Meta: &SCIMMeta{
				ResourceType: "Group",
				Created:      createdAt,
				LastModified: updatedAt,
				Location:     fmt.Sprintf("/scim/v2/Groups/%s", teamID),
			},
		}

		if mapping != nil {
			scimGroup.ExternalID = mapping.SCIMGroupID
		}

		resources = append(resources, scimGroup)
	}

	// Get total count (approximation using len(teams))
	total := len(resources)

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
// Updated for 3-tier architecture: App → Environment → Organization
func (s *Service) CreateProvisioningToken(ctx context.Context, appID, envID, orgID xid.ID, name, description string, scopes []string, expiresAt *time.Time) (string, *ProvisioningToken, error) {
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

	provToken := &ProvisioningToken{
		ID:             xid.New(),
		AppID:          appID,
		EnvironmentID:  envID,
		OrganizationID: orgID,
		Name:           name,
		Description:    description,
		TokenHash:      string(hashedToken),
		TokenPrefix:    tokenPrefix,
		Scopes:         scopes,
		ExpiresAt:      expiresAt,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.repo.CreateProvisioningToken(ctx, provToken); err != nil {
		return "", nil, fmt.Errorf("failed to store token: %w", err)
	}

	// Record metrics
	s.metrics.RecordTokenCreation()
	s.metrics.RecordOperation("CREATE_TOKEN", "success", orgID.String())

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

func (s *Service) mapSCIMToAuthSomeUser(scimUser *SCIMUser, orgID xid.ID) (*user.User, error) {
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
		Meta: &SCIMMeta{
			ResourceType: "User",
			Created:      authUser.CreatedAt,
			LastModified: authUser.UpdatedAt,
			Location:     fmt.Sprintf("/scim/v2/Users/%s", authUser.ID.String()),
		},
	}

	// Parse name from display name if available
	parts := strings.Fields(authUser.Name)
	if len(parts) > 0 {
		scimUser.Name = &SCIMName{
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

func (s *Service) applyGroupPatchOperation(ctx context.Context, teamInterface interface{}, op *PatchOperation, orgID xid.ID) error {
	// Extract team ID from interface
	var teamID xid.ID
	switch t := teamInterface.(type) {
	case *app.Team:
		teamID = t.ID
	default:
		// Use reflection
		val := reflect.ValueOf(teamInterface)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		if val.Kind() == reflect.Struct {
			if idField := val.FieldByName("ID"); idField.IsValid() {
				if id, ok := idField.Interface().(xid.ID); ok {
					teamID = id
				}
			}
		}
	}

	switch op.Op {
	case "replace":
		if op.Path == "displayName" {
			// Name update will be handled by UpdateTeam call
		} else if op.Path == "members" {
			// Handle member replacement
			if members, ok := op.Value.([]interface{}); ok {
				// Remove all existing members
				existingMemberList, _ := s.getOrgService().ListTeamMembers(ctx, teamID)
				for _, memberInterface := range existingMemberList {
					// Extract member ID
					memberID := extractMemberIDFromInterface(memberInterface)
					if !memberID.IsNil() {
						_ = s.getOrgService().RemoveTeamMember(ctx, teamID, memberID)
					}
				}

				// Add new members
				for _, m := range members {
					if memberMap, ok := m.(map[string]interface{}); ok {
						if userIDStr, ok := memberMap["value"].(string); ok {
							userXID, err := xid.FromString(userIDStr)
							if err != nil {
								continue
							}

							// Find member by userID - check if user is member of organization
							isMember, _ := s.getOrgService().IsUserMember(ctx, orgID, userXID)
							if isMember {
								// Get member ID
								memberID, err := s.getOrgService().GetMemberIDByUserID(ctx, orgID, userXID)
								if err == nil {
									_ = s.getOrgService().AddTeamMember(ctx, teamID, memberID, "member")
								}
							}
						}
					}
				}
			}
		}
	case "add":
		if op.Path == "members" {
			// Add members to group
			if members, ok := op.Value.([]interface{}); ok {
				for _, m := range members {
					if memberMap, ok := m.(map[string]interface{}); ok {
						if userIDStr, ok := memberMap["value"].(string); ok {
							userID, err := xid.FromString(userIDStr)
							if err != nil {
								continue
							}
							// Find member by userID
							isMember, _ := s.getOrgService().IsUserMember(ctx, orgID, userID)
							if isMember {
								// Get member ID
								memberID, err := s.getOrgService().GetMemberIDByUserID(ctx, orgID, userID)
								if err == nil {
									_ = s.getOrgService().AddTeamMember(ctx, teamID, memberID, "member")
								}
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
			// For simplicity, if value is provided, use that
			if userIDStr, ok := op.Value.(string); ok {
				userID, err := xid.FromString(userIDStr)
				if err == nil {
					// Get member ID
					memberID, err := s.getOrgService().GetMemberIDByUserID(ctx, orgID, userID)
					if err == nil {
						_ = s.getOrgService().RemoveTeamMember(ctx, teamID, memberID)
					}
				}
			}
		}
	}
	return nil
}

// Helper to extract member ID from member interface
func extractMemberIDFromInterface(member interface{}) xid.ID {
	switch m := member.(type) {
	case *app.Member:
		return m.ID
	default:
		// Use reflection
		val := reflect.ValueOf(member)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		if val.Kind() == reflect.Struct {
			if idField := val.FieldByName("ID"); idField.IsValid() {
				if id, ok := idField.Interface().(xid.ID); ok {
					return id
				}
			}
		}
	}
	return xid.ID{}
}

func (s *Service) syncUserGroups(ctx context.Context, userID xid.ID, groups []GroupReference, orgID xid.ID) error {
	if !s.config.GroupSync.Enabled {
		return nil
	}

	// Get member ID by user ID
	memberID, err := s.getOrgService().GetMemberIDByUserID(ctx, orgID, userID)
	if err != nil {
		return fmt.Errorf("user not found in organization: %w", err)
	}

	// Get user's current teams
	// For each group in the SCIM request, add user to that team
	for _, groupRef := range groups {
		// Find team by external ID (group value)
		mapping, err := s.repo.FindGroupMappingBySCIMID(ctx, xid.ID{}, xid.ID{}, orgID, groupRef.Value)
		if err != nil {
			continue
		}

		// Add user to team
		teamID := mapping.TargetID
		if err := s.getOrgService().AddTeamMember(ctx, teamID, memberID, "member"); err != nil {
		}
	}

	return nil
}

func (s *Service) syncGroupMembers(ctx context.Context, groupID xid.ID, members []MemberReference, orgID xid.ID) error {
	if !s.config.GroupSync.Enabled {
		return nil
	}

	// Clear existing members (for full sync)
	existingMemberList, _ := s.getOrgService().ListTeamMembers(ctx, groupID)
	for _, memberInterface := range existingMemberList {
		memberID := extractMemberIDFromInterface(memberInterface)
		if !memberID.IsNil() {
			_ = s.getOrgService().RemoveTeamMember(ctx, groupID, memberID)
		}
	}

	// Add new members
	for _, memberRef := range members {
		// memberRef.Value is the userID
		userIDStr := memberRef.Value
		userID, err := xid.FromString(userIDStr)
		if err != nil {
			continue
		}

		// Get member ID by user ID
		memberID, err := s.getOrgService().GetMemberIDByUserID(ctx, orgID, userID)
		if err != nil {
			continue
		}

		// Add to team
		if err := s.getOrgService().AddTeamMember(ctx, groupID, memberID, "member"); err != nil {
			continue
		}

		// Mark this team membership as SCIM-provisioned
		if err := s.repo.UpdateTeamMemberProvisioningInfo(ctx, groupID, memberID, strPtr("scim")); err != nil {
			// Log warning but don't fail the operation
		}
	}

	return nil
}

// updateTeamProvisioningInfo updates a team with SCIM provisioning information
// This uses direct database access to update provisioning fields since the service layer
// doesn't expose these fields in update requests
func (s *Service) updateTeamProvisioningInfo(ctx context.Context, teamID xid.ID, externalID string) error {
	provisionedBy := "scim"

	// Use direct repository access to update provisioning fields
	// We can't use the service layer since CreateTeamRequest doesn't include these fields
	return s.repo.UpdateTeamProvisioningInfo(ctx, teamID, &provisionedBy, &externalID)
}

// strPtr is a helper function to create a string pointer
func strPtr(s string) *string {
	return &s
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
func (s *Service) ProcessBulkOperation(ctx context.Context, bulkReq *BulkRequest, orgID xid.ID) (*BulkResponse, error) {
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
				resID := pathParts[len(pathParts)-1]
				resourceID, err := xid.FromString(resID)
				if err != nil {
					respOp.Status = 400
				}

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
				resID := pathParts[len(pathParts)-1]
				resourceID, err := xid.FromString(resID)
				if err != nil {
					respOp.Status = 400
				}

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
				resID := pathParts[len(pathParts)-1]
				resourceID, err := xid.FromString(resID)
				if err != nil {
					respOp.Status = 400
				}

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
// Updated for 3-tier architecture
func (s *Service) ListProvisioningTokens(ctx context.Context, appID, envID, orgID xid.ID, limit, offset int) ([]*ProvisioningToken, int, error) {
	tokens, err := s.repo.ListProvisioningTokens(ctx, appID, envID, orgID, limit, offset)
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
// Updated for 3-tier architecture: App → Environment → Organization
func (s *Service) GetAttributeMappings(ctx context.Context, appID, envID, orgID xid.ID) (map[string]string, error) {
	mapping, err := s.repo.FindAttributeMappingByOrganization(ctx, appID, envID, orgID)
	if err != nil {
		// Return default mappings if none exist
		return s.config.AttributeMapping.CustomMapping, nil
	}

	return mapping.Mappings, nil
}

// UpdateAttributeMappings updates attribute mappings for an organization
// Updated for 3-tier architecture: App → Environment → Organization
func (s *Service) UpdateAttributeMappings(ctx context.Context, appID, envID, orgID xid.ID, mappings map[string]string) error {
	// Find existing mapping
	existingMapping, err := s.repo.FindAttributeMappingByOrganization(ctx, appID, envID, orgID)
	if err != nil {
		// Create new mapping
		mapping := &AttributeMapping{
			ID:             xid.New(),
			AppID:          appID,
			EnvironmentID:  envID,
			OrganizationID: orgID,
			Mappings:       mappings,
			Metadata:       make(map[string]interface{}),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
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
// Updated for 3-tier architecture
func (s *Service) GetProvisioningLogs(ctx context.Context, appID, envID, orgID xid.ID, action string, limit, offset int) ([]*ProvisioningLog, int, error) {
	// Build filters
	filters := make(map[string]interface{})
	if action != "" {
		filters["operation"] = action
	}

	logs, err := s.repo.ListProvisioningLogs(ctx, appID, envID, orgID, filters, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list logs: %w", err)
	}

	// Get total count
	total, err := s.repo.CountProvisioningLogs(ctx, appID, envID, orgID, filters)
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
// Updated for 3-tier architecture: App → Environment → Organization
func (s *Service) InitializeOrgSCIMConfig(ctx context.Context, appID, envID, orgID xid.ID) error {
	// Create default attribute mapping
	mapping := &AttributeMapping{
		ID:             xid.New(),
		AppID:          appID,
		EnvironmentID:  envID,
		OrganizationID: orgID,
		Mappings:       s.config.AttributeMapping.CustomMapping,
		Metadata:       make(map[string]interface{}),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
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

// Helper function to extract UserID from member interface
func extractUserIDFromMember(member interface{}) xid.ID {
	// Try app.Member
	if m, ok := member.(*app.Member); ok {
		return m.UserID
	}
	// Try schema.OrganizationMember (from organization plugin)
	if m, ok := member.(interface{ GetUserID() xid.ID }); ok {
		return m.GetUserID()
	}
	// Last resort: use reflection (works for both app.Member and schema.OrganizationMember)
	val := reflect.ValueOf(member)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() == reflect.Struct {
		userIDField := val.FieldByName("UserID")
		if userIDField.IsValid() && userIDField.CanInterface() {
			if userID, ok := userIDField.Interface().(xid.ID); ok {
				return userID
			}
		}
	}
	return xid.ID{}
}

// Adapter implementations for unified interface

// appServiceAdapter adapts multitenancy app service to SCIMOrgService interface
type appServiceAdapter struct {
	service *app.ServiceImpl
}

func (a *appServiceAdapter) AddMember(ctx context.Context, orgID, userID xid.ID, role string) (interface{}, error) {
	member := &app.Member{
		AppID:  orgID,
		UserID: userID,
		Role:   app.MemberRole(role),
		Status: app.MemberStatusActive,
	}
	return a.service.CreateMember(ctx, member)
}

func (a *appServiceAdapter) IsUserMember(ctx context.Context, orgID, userID xid.ID) (bool, error) {
	return a.service.IsUserMember(ctx, orgID, userID)
}

func (a *appServiceAdapter) ListMembers(ctx context.Context, orgID xid.ID, limit, offset int) ([]interface{}, error) {
	filter := &app.ListMembersFilter{
		AppID: orgID,
	}
	filter.Limit = limit
	filter.Offset = offset
	response, err := a.service.ListMembers(ctx, filter)
	if err != nil {
		return nil, err
	}
	result := make([]interface{}, len(response.Data))
	for i, m := range response.Data {
		result[i] = m
	}
	return result, nil
}

func (a *appServiceAdapter) CreateTeam(ctx context.Context, orgID xid.ID, req interface{}) (interface{}, error) {
	// For now, return a placeholder since team creation interface needs proper implementation
	// TODO: Implement team creation with new app service interface
	return nil, fmt.Errorf("team creation via SCIM not yet implemented for app mode")
}

func (a *appServiceAdapter) GetTeam(ctx context.Context, id xid.ID) (interface{}, error) {
	return a.service.FindTeamByID(ctx, id)
}

func (a *appServiceAdapter) ListTeams(ctx context.Context, orgID xid.ID, limit, offset int) ([]interface{}, error) {
	filter := &app.ListTeamsFilter{
		AppID: orgID,
	}
	filter.Limit = limit
	filter.Offset = offset
	response, err := a.service.ListTeams(ctx, filter)
	if err != nil {
		return nil, err
	}
	result := make([]interface{}, len(response.Data))
	for i, t := range response.Data {
		result[i] = t
	}
	return result, nil
}

func (a *appServiceAdapter) UpdateTeam(ctx context.Context, id xid.ID, req interface{}) (interface{}, error) {
	// For now, return a placeholder since team update interface needs proper implementation
	// TODO: Implement team update with new app service interface
	return nil, fmt.Errorf("team update via SCIM not yet implemented for app mode")
}

func (a *appServiceAdapter) DeleteTeam(ctx context.Context, id xid.ID) error {
	return a.service.DeleteTeam(ctx, id)
}

func (a *appServiceAdapter) AddTeamMember(ctx context.Context, teamID, memberID xid.ID, role string) error {
	tm := &app.TeamMember{
		TeamID:   teamID,
		MemberID: memberID,
		// Note: Role management should be handled through member roles, not team member roles
	}
	_, err := a.service.AddTeamMember(ctx, tm)
	return err
}

func (a *appServiceAdapter) RemoveTeamMember(ctx context.Context, teamID, memberID xid.ID) error {
	return a.service.RemoveTeamMember(ctx, teamID, memberID)
}

func (a *appServiceAdapter) ListTeamMembers(ctx context.Context, teamID xid.ID) ([]interface{}, error) {
	filter := &app.ListTeamMembersFilter{
		TeamID: teamID,
	}
	filter.Limit = 10000
	filter.Offset = 0
	response, err := a.service.ListTeamMembers(ctx, filter)
	if err != nil {
		return nil, err
	}
	result := make([]interface{}, len(response.Data))
	for i, m := range response.Data {
		result[i] = m
	}
	return result, nil
}

func (a *appServiceAdapter) GetMemberIDByUserID(ctx context.Context, orgID, userID xid.ID) (xid.ID, error) {
	filter := &app.ListMembersFilter{
		AppID: orgID,
	}
	filter.Limit = 10000
	filter.Offset = 0
	response, err := a.service.ListMembers(ctx, filter)
	if err != nil {
		return xid.ID{}, err
	}
	for _, m := range response.Data {
		if m.UserID == userID {
			return m.ID, nil
		}
	}
	return xid.ID{}, fmt.Errorf("member not found for user %s in org %s", userID, orgID)
}

// orgServiceAdapter adapts organization plugin service to SCIMOrgService interface
type orgServiceAdapter struct {
	service *orgplugin.Service
}

func (a *orgServiceAdapter) AddMember(ctx context.Context, orgID, userID xid.ID, role string) (interface{}, error) {
	return a.service.AddMember(ctx, orgID, userID, role)
}

func (a *orgServiceAdapter) IsUserMember(ctx context.Context, orgID, userID xid.ID) (bool, error) {
	return a.service.IsMember(ctx, orgID, userID)
}

func (a *orgServiceAdapter) ListMembers(ctx context.Context, orgID xid.ID, limit, offset int) ([]interface{}, error) {
	// TODO: Organization service needs to be updated to use filter-based pagination
	// For now, return empty to prevent errors
	return []interface{}{}, fmt.Errorf("organization service list members needs pagination filter update")
}

func (a *orgServiceAdapter) CreateTeam(ctx context.Context, orgID xid.ID, req interface{}) (interface{}, error) {
	teamReq, ok := req.(*orgplugin.CreateTeamRequest)
	if !ok {
		// Try to convert from app service request
		if appReq, ok := req.(*app.CreateTeamRequest); ok {
			desc := appReq.Description
			teamReq = &orgplugin.CreateTeamRequest{
				Name:        appReq.Name,
				Description: desc,
				Metadata:    appReq.Metadata,
			}
		} else {
			return nil, fmt.Errorf("invalid team request type")
		}
	}
	// For SCIM operations, we use a system user ID (zero xid) or get from context
	// In practice, SCIM tokens should have a created_by field we can use
	systemUserID := xid.ID{} // Zero xid for system operations
	return a.service.CreateTeam(ctx, orgID, teamReq, systemUserID)
}

func (a *orgServiceAdapter) GetTeam(ctx context.Context, id xid.ID) (interface{}, error) {
	// TODO: Organization service needs GetTeam method
	return nil, fmt.Errorf("organization service get team not yet implemented")
}

func (a *orgServiceAdapter) ListTeams(ctx context.Context, orgID xid.ID, limit, offset int) ([]interface{}, error) {
	// TODO: Organization service needs to be updated to use filter-based pagination
	// For now, return empty to prevent errors
	return []interface{}{}, fmt.Errorf("organization service list teams needs pagination filter update")
}

func (a *orgServiceAdapter) UpdateTeam(ctx context.Context, id xid.ID, req interface{}) (interface{}, error) {
	teamReq, ok := req.(*orgplugin.UpdateTeamRequest)
	if !ok {
		// Try to convert from app service request
		if appReq, ok := req.(*app.UpdateTeamRequest); ok {
			teamReq = &orgplugin.UpdateTeamRequest{
				Name:        appReq.Name,
				Description: appReq.Description,
				Metadata:    appReq.Metadata,
			}
		} else {
			return nil, fmt.Errorf("invalid team update request type")
		}
	}
	// For SCIM operations, we use a system user ID
	systemUserID := xid.ID{} // Zero xid for system operations
	return a.service.UpdateTeam(ctx, id, teamReq, systemUserID)
}

func (a *orgServiceAdapter) DeleteTeam(ctx context.Context, id xid.ID) error {
	systemUserID := xid.ID{} // Zero xid for system operations
	return a.service.DeleteTeam(ctx, id, systemUserID)
}

func (a *orgServiceAdapter) AddTeamMember(ctx context.Context, teamID, memberID xid.ID, role string) error {
	systemUserID := xid.ID{} // Zero xid for system operations
	return a.service.AddTeamMember(ctx, teamID, memberID, systemUserID)
}

func (a *orgServiceAdapter) RemoveTeamMember(ctx context.Context, teamID, memberID xid.ID) error {
	systemUserID := xid.ID{} // Zero xid for system operations
	return a.service.RemoveTeamMember(ctx, teamID, memberID, systemUserID)
}

func (a *orgServiceAdapter) ListTeamMembers(ctx context.Context, teamID xid.ID) ([]interface{}, error) {
	// TODO: Organization service needs to be updated to use filter-based pagination
	// For now, return empty to prevent errors
	return []interface{}{}, fmt.Errorf("organization service list team members needs pagination filter update")
}

func (a *orgServiceAdapter) GetMemberIDByUserID(ctx context.Context, orgID, userID xid.ID) (xid.ID, error) {
	// TODO: Organization service needs to be updated to use filter-based pagination
	// For now, return error
	return xid.ID{}, fmt.Errorf("organization service get member needs pagination filter update")
}
