package scim

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/apitypes"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/user"
)

// ──────────────────────────────────────────────────
// Request types
// ──────────────────────────────────────────────────

type scimUserPathParam struct {
	UserID string `path:"userId"`
}

type scimGroupPathParam struct {
	GroupID string `path:"groupId"`
}

// ──────────────────────────────────────────────────
// SCIM Discovery endpoints
// ──────────────────────────────────────────────────

func (p *Plugin) handleServiceProviderConfig(_ forge.Context, _ *apitypes.Empty) (*ServiceProviderConfig, error) {
	config := ServiceProviderConfig{
		"schemas": []string{SchemaServiceConfig},
		"patch": map[string]any{
			"supported": true,
		},
		"bulk": map[string]any{
			"supported":      false,
			"maxPayloadSize": 0,
		},
		"filter": map[string]any{
			"supported":  true,
			"maxResults": 100,
		},
		"changePassword": map[string]any{
			"supported": false,
		},
		"sort": map[string]any{
			"supported": false,
		},
		"etag": map[string]any{
			"supported": false,
		},
		"authenticationSchemes": []map[string]any{
			{
				"type":        "oauthbearertoken",
				"name":        "OAuth Bearer Token",
				"description": "Authentication scheme using the OAuth Bearer Token Standard",
			},
		},
	}
	return &config, nil
}

func (p *Plugin) handleSchemas(_ forge.Context, _ *apitypes.Empty) (*SchemaList, error) {
	schemas := SchemaList{
		"schemas":      []string{SchemaListResponse},
		"totalResults": 2,
		"Resources": []map[string]any{
			{
				"id":   SchemaUser,
				"name": "User",
			},
			{
				"id":   SchemaGroup,
				"name": "Group",
			},
		},
	}
	return &schemas, nil
}

func (p *Plugin) handleResourceTypes(_ forge.Context, _ *apitypes.Empty) (*ResourceTypeList, error) {
	types := ResourceTypeList{
		"schemas":      []string{SchemaListResponse},
		"totalResults": 2,
		"Resources": []map[string]any{
			{
				"schemas":  []string{SchemaResourceType},
				"id":       "User",
				"name":     "User",
				"endpoint": "/Users",
				"schema":   SchemaUser,
			},
			{
				"schemas":  []string{SchemaResourceType},
				"id":       "Group",
				"name":     "Group",
				"endpoint": "/Groups",
				"schema":   SchemaGroup,
			},
		},
	}
	return &types, nil
}

// ──────────────────────────────────────────────────
// User SCIM endpoints
// ──────────────────────────────────────────────────

func (p *Plugin) handleListUsers(ctx forge.Context, _ *apitypes.Empty) (*ListResponse, error) {
	cfg, err := p.authenticateSCIM(ctx)
	if err != nil {
		return nil, err
	}

	baseURL := p.config.BasePath

	// List users. For org-scoped configs, list org members; otherwise all users.
	var users []*UserResource
	if !cfg.OrgID.IsNil() && p.authStore != nil {
		members, err := p.authStore.ListMembers(ctx.Context(), cfg.OrgID)
		if err != nil {
			return nil, middleware.InternalError(ctx, err)
		}
		for _, m := range members {
			u, err := p.authStore.GetUser(ctx.Context(), m.UserID)
			if err != nil {
				continue
			}
			users = append(users, UserToSCIM(u, baseURL))
		}
	} else if p.authStore != nil {
		result, err := p.authStore.ListUsers(ctx.Context(), &user.Query{
			AppID: cfg.AppID,
		})
		if err != nil {
			return nil, middleware.InternalError(ctx, err)
		}
		for _, u := range result.Users {
			users = append(users, UserToSCIM(u, baseURL))
		}
	}

	resources := make([]any, 0, len(users))
	for _, u := range users {
		resources = append(resources, u)
	}

	return &ListResponse{
		Schemas:      []string{SchemaListResponse},
		TotalResults: len(resources),
		StartIndex:   1,
		ItemsPerPage: len(resources),
		Resources:    resources,
	}, nil
}

// scimError maps a service failure to the SCIM response. A row outside the
// configuration's scope is answered exactly like a missing one.
func scimError(ctx forge.Context, err error, notFound string) error {
	if errors.Is(err, ErrOutOfScope) {
		return forge.NotFound(notFound)
	}
	return middleware.InternalError(ctx, err)
}

func (p *Plugin) handleGetUser(ctx forge.Context, req *scimUserPathParam) (*UserResource, error) {
	cfg, err := p.authenticateSCIM(ctx)
	if err != nil {
		return nil, err
	}

	u, err := p.service.LoadUser(ctx.Context(), cfg, req.UserID)
	if err != nil {
		return nil, scimError(ctx, err, "user not found")
	}

	return UserToSCIM(u, p.config.BasePath), nil
}

func (p *Plugin) handleCreateUser(ctx forge.Context, _ *apitypes.Empty) (*UserResource, error) {
	cfg, err := p.authenticateSCIM(ctx)
	if err != nil {
		return nil, err
	}

	var scimUser UserResource
	if decodeErr := json.NewDecoder(ctx.Request().Body).Decode(&scimUser); decodeErr != nil {
		return nil, forge.BadRequest("invalid SCIM user payload")
	}

	scimUser.Active = true

	u, action, err := p.service.ProvisionUser(ctx.Context(), cfg, &scimUser)
	if err != nil {
		p.service.RecordLog(ctx.Context(), cfg.ID, action, "User", scimUser.ExternalID, "", LogStatusError, err.Error())
		return nil, scimError(ctx, err, "user not found")
	}

	p.service.RecordLog(ctx.Context(), cfg.ID, action, "User", scimUser.ExternalID, u.ID.String(), LogStatusSuccess, "")
	p.audit(ctx.Context(), "scim."+action, "user", u.ID.String(), "", cfg.ID.String(), bridge.OutcomeSuccess)

	result := UserToSCIM(u, p.config.BasePath)
	return nil, ctx.JSON(http.StatusCreated, result)
}

func (p *Plugin) handleReplaceUser(ctx forge.Context, req *scimUserPathParam) (*UserResource, error) {
	cfg, err := p.authenticateSCIM(ctx)
	if err != nil {
		return nil, err
	}

	var scimUser UserResource
	if decodeErr := json.NewDecoder(ctx.Request().Body).Decode(&scimUser); decodeErr != nil {
		return nil, forge.BadRequest("invalid SCIM user payload")
	}

	scimUser.ID = req.UserID

	// PUT targets the path id, never whichever account the body's email
	// happens to resolve to.
	u, err := p.service.LoadUser(ctx.Context(), cfg, req.UserID)
	if err != nil {
		return nil, scimError(ctx, err, "user not found")
	}
	if err := p.service.ReplaceUser(ctx.Context(), cfg, u, &scimUser); err != nil {
		p.service.RecordLog(ctx.Context(), cfg.ID, ActionUpdateUser, "User", scimUser.ExternalID, req.UserID, LogStatusError, err.Error())
		return nil, scimError(ctx, err, "user not found")
	}

	p.service.RecordLog(ctx.Context(), cfg.ID, ActionUpdateUser, "User", scimUser.ExternalID, u.ID.String(), LogStatusSuccess, "")
	return UserToSCIM(u, p.config.BasePath), nil
}

func (p *Plugin) handlePatchUser(ctx forge.Context, req *scimUserPathParam) (*UserResource, error) {
	cfg, err := p.authenticateSCIM(ctx)
	if err != nil {
		return nil, err
	}

	var patch PatchOp
	if decodeErr := json.NewDecoder(ctx.Request().Body).Decode(&patch); decodeErr != nil {
		return nil, forge.BadRequest("invalid SCIM patch payload")
	}

	u, err := p.service.LoadUser(ctx.Context(), cfg, req.UserID)
	if err != nil {
		return nil, scimError(ctx, err, "user not found")
	}

	// Apply SCIM PATCH operations.
	wasBanned := u.Banned
	newUserName := ""
	for _, op := range patch.Operations {
		if strings.EqualFold(op.Op, "replace") {
			if name := p.applyUserPatchReplace(u, op); name != "" {
				newUserName = name
			}
		}
	}
	// userName is the primary email and goes through the email-record API,
	// not through the user's email field.
	if newUserName != "" {
		if err := p.service.ChangeUserName(ctx.Context(), cfg, u, newUserName); err != nil {
			p.service.RecordLog(ctx.Context(), cfg.ID, ActionUpdateUser, "User", "", req.UserID, LogStatusError, err.Error())
			return nil, scimError(ctx, err, "user not found")
		}
	}

	u.UpdatedAt = time.Now()
	if err := p.authStore.UpdateUser(ctx.Context(), u); err != nil {
		p.service.RecordLog(ctx.Context(), cfg.ID, ActionUpdateUser, "User", "", req.UserID, LogStatusError, err.Error())
		return nil, middleware.InternalError(ctx, err)
	}
	if u.Banned && !wasBanned {
		if err := p.service.enforceDeactivation(ctx.Context(), cfg, u); err != nil {
			p.service.RecordLog(ctx.Context(), cfg.ID, ActionSuspendUser, "User", "", req.UserID, LogStatusError, err.Error())
			return nil, middleware.InternalError(ctx, err)
		}
	}

	// PATCH {"path":"active","value":false} (and the pathless bulk-replace
	// shape) is the deactivation Okta and Entra ID send by default, and it
	// bans the user (see applyUserPatchReplace) without going through
	// AdminBanUser or DeactivateUser. Plugins watching AfterUserUpdate need
	// to see every path that can flip Banned, not just the two named ones.
	if p.plugins != nil {
		p.plugins.EmitAfterUserUpdate(ctx.Context(), u)
	}

	// Check if user was deactivated.
	if u.Banned {
		p.service.RecordLog(ctx.Context(), cfg.ID, ActionSuspendUser, "User", "", u.ID.String(), LogStatusSuccess, "")
	} else {
		p.service.RecordLog(ctx.Context(), cfg.ID, ActionUpdateUser, "User", "", u.ID.String(), LogStatusSuccess, "")
	}

	return UserToSCIM(u, p.config.BasePath), nil
}

func (p *Plugin) handleDeleteUser(ctx forge.Context, req *scimUserPathParam) (*apitypes.Empty, error) {
	cfg, err := p.authenticateSCIM(ctx)
	if err != nil {
		return nil, err
	}

	userID, err := id.ParseUserID(req.UserID)
	if err != nil {
		return nil, forge.NotFound("user not found")
	}

	if err := p.service.DeactivateUser(ctx.Context(), cfg, userID); err != nil {
		p.service.RecordLog(ctx.Context(), cfg.ID, ActionSuspendUser, "User", "", req.UserID, LogStatusError, err.Error())
		return nil, scimError(ctx, err, "user not found")
	}

	p.service.RecordLog(ctx.Context(), cfg.ID, ActionSuspendUser, "User", "", req.UserID, LogStatusSuccess, "")
	p.audit(ctx.Context(), "scim.suspend_user", "user", req.UserID, "", cfg.ID.String(), bridge.OutcomeSuccess)

	return nil, ctx.NoContent(http.StatusNoContent)
}

// ──────────────────────────────────────────────────
// Group SCIM endpoints
// ──────────────────────────────────────────────────

func (p *Plugin) handleListGroups(ctx forge.Context, _ *apitypes.Empty) (*ListResponse, error) {
	cfg, err := p.authenticateSCIM(ctx)
	if err != nil {
		return nil, err
	}

	if !cfg.GroupSync || cfg.OrgID.IsNil() {
		return &ListResponse{
			Schemas:      []string{SchemaListResponse},
			TotalResults: 0,
			StartIndex:   1,
			ItemsPerPage: 0,
			Resources:    []any{},
		}, nil
	}

	baseURL := p.config.BasePath
	teams, err := p.authStore.ListTeams(ctx.Context(), cfg.OrgID)
	if err != nil {
		return nil, middleware.InternalError(ctx, err)
	}

	resources := make([]any, 0, len(teams))
	for _, t := range teams {
		resources = append(resources, TeamToSCIMGroup(t, nil, baseURL))
	}

	return &ListResponse{
		Schemas:      []string{SchemaListResponse},
		TotalResults: len(resources),
		StartIndex:   1,
		ItemsPerPage: len(resources),
		Resources:    resources,
	}, nil
}

func (p *Plugin) handleGetGroup(ctx forge.Context, req *scimGroupPathParam) (*GroupResource, error) {
	cfg, err := p.authenticateSCIM(ctx)
	if err != nil {
		return nil, err
	}

	team, err := p.service.LoadTeam(ctx.Context(), cfg, req.GroupID)
	if err != nil {
		return nil, scimError(ctx, err, "group not found")
	}

	return TeamToSCIMGroup(team, nil, p.config.BasePath), nil
}

func (p *Plugin) handleCreateGroup(ctx forge.Context, _ *apitypes.Empty) (*GroupResource, error) {
	cfg, err := p.authenticateSCIM(ctx)
	if err != nil {
		return nil, err
	}

	var scimGroup GroupResource
	if decodeErr := json.NewDecoder(ctx.Request().Body).Decode(&scimGroup); decodeErr != nil {
		return nil, forge.BadRequest("invalid SCIM group payload")
	}

	team, action, err := p.service.ProvisionGroup(ctx.Context(), cfg, &scimGroup)
	if err != nil {
		p.service.RecordLog(ctx.Context(), cfg.ID, action, "Group", scimGroup.ExternalID, "", LogStatusError, err.Error())
		return nil, middleware.InternalError(ctx, err)
	}

	p.service.RecordLog(ctx.Context(), cfg.ID, action, "Group", scimGroup.ExternalID, team.ID.String(), LogStatusSuccess, "")
	p.audit(ctx.Context(), "scim."+action, "group", team.ID.String(), "", cfg.ID.String(), bridge.OutcomeSuccess)

	result := TeamToSCIMGroup(team, nil, p.config.BasePath)
	return nil, ctx.JSON(http.StatusCreated, result)
}

func (p *Plugin) handleReplaceGroup(ctx forge.Context, req *scimGroupPathParam) (*GroupResource, error) {
	cfg, err := p.authenticateSCIM(ctx)
	if err != nil {
		return nil, err
	}

	var scimGroup GroupResource
	if decodeErr := json.NewDecoder(ctx.Request().Body).Decode(&scimGroup); decodeErr != nil {
		return nil, forge.BadRequest("invalid SCIM group payload")
	}
	scimGroup.ID = req.GroupID

	// PUT targets the path id inside the configuration's organization.
	team, err := p.service.LoadTeam(ctx.Context(), cfg, req.GroupID)
	if err != nil {
		return nil, scimError(ctx, err, "group not found")
	}
	if name := strings.TrimSpace(scimGroup.DisplayName); name != "" {
		team.Name = name
	}
	team.UpdatedAt = time.Now()
	if err := p.authStore.UpdateTeam(ctx.Context(), team); err != nil {
		p.service.RecordLog(ctx.Context(), cfg.ID, ActionUpdateGroup, "Group", scimGroup.ExternalID, req.GroupID, LogStatusError, err.Error())
		return nil, middleware.InternalError(ctx, err)
	}

	p.service.RecordLog(ctx.Context(), cfg.ID, ActionUpdateGroup, "Group", scimGroup.ExternalID, team.ID.String(), LogStatusSuccess, "")
	return TeamToSCIMGroup(team, nil, p.config.BasePath), nil
}

func (p *Plugin) handlePatchGroup(ctx forge.Context, req *scimGroupPathParam) (*GroupResource, error) {
	cfg, err := p.authenticateSCIM(ctx)
	if err != nil {
		return nil, err
	}

	var patch PatchOp
	if decodeErr := json.NewDecoder(ctx.Request().Body).Decode(&patch); decodeErr != nil {
		return nil, forge.BadRequest("invalid SCIM patch payload")
	}

	team, err := p.service.LoadTeam(ctx.Context(), cfg, req.GroupID)
	if err != nil {
		return nil, scimError(ctx, err, "group not found")
	}

	// Apply PATCH operations (simplified: handle displayName changes).
	for _, op := range patch.Operations {
		if strings.EqualFold(op.Op, "replace") {
			if op.Path == "displayName" {
				if name, ok := op.Value.(string); ok {
					team.Name = name
				}
			}
		}
	}

	team.UpdatedAt = time.Now()
	if err := p.authStore.UpdateTeam(ctx.Context(), team); err != nil {
		p.service.RecordLog(ctx.Context(), cfg.ID, ActionUpdateGroup, "Group", "", req.GroupID, LogStatusError, err.Error())
		return nil, middleware.InternalError(ctx, err)
	}

	p.service.RecordLog(ctx.Context(), cfg.ID, ActionUpdateGroup, "Group", "", team.ID.String(), LogStatusSuccess, "")
	return TeamToSCIMGroup(team, nil, p.config.BasePath), nil
}

func (p *Plugin) handleDeleteGroup(ctx forge.Context, req *scimGroupPathParam) (*apitypes.Empty, error) {
	cfg, err := p.authenticateSCIM(ctx)
	if err != nil {
		return nil, err
	}

	team, err := p.service.LoadTeam(ctx.Context(), cfg, req.GroupID)
	if err != nil {
		return nil, scimError(ctx, err, "group not found")
	}

	if err := p.authStore.DeleteTeam(ctx.Context(), team.ID); err != nil {
		p.service.RecordLog(ctx.Context(), cfg.ID, ActionDeleteGroup, "Group", "", req.GroupID, LogStatusError, err.Error())
		return nil, middleware.InternalError(ctx, err)
	}

	p.service.RecordLog(ctx.Context(), cfg.ID, ActionDeleteGroup, "Group", "", req.GroupID, LogStatusSuccess, "")
	p.audit(ctx.Context(), "scim.delete_group", "group", req.GroupID, "", cfg.ID.String(), bridge.OutcomeSuccess)

	return nil, ctx.NoContent(http.StatusNoContent)
}

// ──────────────────────────────────────────────────
// Auth + Helpers
// ──────────────────────────────────────────────────

// authenticateSCIM validates the Bearer token from the request.
func (p *Plugin) authenticateSCIM(ctx forge.Context) (*SCIMConfig, error) {
	auth := ctx.Request().Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
		return nil, forge.Unauthorized("SCIM bearer token required")
	}

	token := strings.TrimPrefix(auth, "Bearer ")
	_, cfg, err := p.service.ValidateToken(ctx.Context(), token)
	if err != nil {
		return nil, forge.Unauthorized("invalid or expired SCIM token")
	}

	if !cfg.Enabled {
		return nil, forge.Forbidden("SCIM configuration is disabled")
	}

	return cfg, nil
}

// applyUserPatchReplace handles SCIM PATCH replace operations for a user.
// applyUserPatchReplace applies one replace operation to u and returns the
// userName it named, if any, for the caller to change through the
// email-record API rather than by writing the user's email field.
func (p *Plugin) applyUserPatchReplace(u *user.User, op Operation) (userName string) {
	switch op.Path {
	case "active":
		if active, ok := op.Value.(bool); ok {
			u.Banned = !active
		}
	case "name.givenName":
		if v, ok := op.Value.(string); ok {
			u.FirstName = v
		}
	case "name.familyName":
		if v, ok := op.Value.(string); ok {
			u.LastName = v
		}
	case "userName":
		if v, ok := op.Value.(string); ok {
			userName = v
		}
	case "":
		// Bulk replace: value is a map of attributes.
		if m, ok := op.Value.(map[string]any); ok {
			if active, ok := m["active"].(bool); ok {
				u.Banned = !active
			}
			if v, ok := m["userName"].(string); ok {
				userName = v
			}
			if name, ok := m["name"].(map[string]any); ok {
				if gn, ok := name["givenName"].(string); ok {
					u.FirstName = gn
				}
				if fn, ok := name["familyName"].(string); ok {
					u.LastName = fn
				}
			}
		}
	}
	return userName
}
