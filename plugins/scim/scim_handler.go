package scim

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
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

func (p *Plugin) handleServiceProviderConfig(_ forge.Context, _ *struct{}) (*map[string]any, error) { //nolint:gocritic // ptrToRefParam: framework requires pointer return
	config := map[string]any{
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

func (p *Plugin) handleSchemas(_ forge.Context, _ *struct{}) (*map[string]any, error) { //nolint:gocritic // ptrToRefParam: framework requires pointer return
	schemas := map[string]any{
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

func (p *Plugin) handleResourceTypes(_ forge.Context, _ *struct{}) (*map[string]any, error) { //nolint:gocritic // ptrToRefParam: framework requires pointer return
	types := map[string]any{
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

func (p *Plugin) handleListUsers(ctx forge.Context, _ *struct{}) (*ListResponse, error) {
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
			return nil, forge.InternalError(err)
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
			return nil, forge.InternalError(err)
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

func (p *Plugin) handleGetUser(ctx forge.Context, req *scimUserPathParam) (*UserResource, error) {
	_, err := p.authenticateSCIM(ctx)
	if err != nil {
		return nil, err
	}

	userID, err := id.ParseUserID(req.UserID)
	if err != nil {
		return nil, forge.NotFound("user not found")
	}

	u, err := p.authStore.GetUser(ctx.Context(), userID)
	if err != nil {
		return nil, forge.NotFound("user not found")
	}

	return UserToSCIM(u, p.config.BasePath), nil
}

func (p *Plugin) handleCreateUser(ctx forge.Context, _ *struct{}) (*UserResource, error) {
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
		return nil, forge.InternalError(err)
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

	u, action, err := p.service.ProvisionUser(ctx.Context(), cfg, &scimUser)
	if err != nil {
		p.service.RecordLog(ctx.Context(), cfg.ID, ActionUpdateUser, "User", scimUser.ExternalID, req.UserID, LogStatusError, err.Error())
		return nil, forge.InternalError(err)
	}

	p.service.RecordLog(ctx.Context(), cfg.ID, action, "User", scimUser.ExternalID, u.ID.String(), LogStatusSuccess, "")
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

	userID, err := id.ParseUserID(req.UserID)
	if err != nil {
		return nil, forge.NotFound("user not found")
	}

	u, err := p.authStore.GetUser(ctx.Context(), userID)
	if err != nil {
		return nil, forge.NotFound("user not found")
	}

	// Apply SCIM PATCH operations.
	for _, op := range patch.Operations {
		if strings.EqualFold(op.Op, "replace") {
			p.applyUserPatchReplace(u, op)
		}
	}

	u.UpdatedAt = time.Now()
	if err := p.authStore.UpdateUser(ctx.Context(), u); err != nil {
		p.service.RecordLog(ctx.Context(), cfg.ID, ActionUpdateUser, "User", "", req.UserID, LogStatusError, err.Error())
		return nil, forge.InternalError(err)
	}

	// Check if user was deactivated.
	if u.Banned {
		p.service.RecordLog(ctx.Context(), cfg.ID, ActionSuspendUser, "User", "", u.ID.String(), LogStatusSuccess, "")
	} else {
		p.service.RecordLog(ctx.Context(), cfg.ID, ActionUpdateUser, "User", "", u.ID.String(), LogStatusSuccess, "")
	}

	return UserToSCIM(u, p.config.BasePath), nil
}

func (p *Plugin) handleDeleteUser(ctx forge.Context, req *scimUserPathParam) (*struct{}, error) {
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
		return nil, forge.InternalError(err)
	}

	p.service.RecordLog(ctx.Context(), cfg.ID, ActionSuspendUser, "User", "", req.UserID, LogStatusSuccess, "")
	p.audit(ctx.Context(), "scim.suspend_user", "user", req.UserID, "", cfg.ID.String(), bridge.OutcomeSuccess)

	return nil, ctx.NoContent(http.StatusNoContent)
}

// ──────────────────────────────────────────────────
// Group SCIM endpoints
// ──────────────────────────────────────────────────

func (p *Plugin) handleListGroups(ctx forge.Context, _ *struct{}) (*ListResponse, error) {
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
		return nil, forge.InternalError(err)
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
	_, err := p.authenticateSCIM(ctx)
	if err != nil {
		return nil, err
	}

	teamID, err := id.ParseTeamID(req.GroupID)
	if err != nil {
		return nil, forge.NotFound("group not found")
	}

	team, err := p.authStore.GetTeam(ctx.Context(), teamID)
	if err != nil {
		return nil, forge.NotFound("group not found")
	}

	return TeamToSCIMGroup(team, nil, p.config.BasePath), nil
}

func (p *Plugin) handleCreateGroup(ctx forge.Context, _ *struct{}) (*GroupResource, error) {
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
		return nil, forge.InternalError(err)
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

	team, action, err := p.service.ProvisionGroup(ctx.Context(), cfg, &scimGroup)
	if err != nil {
		p.service.RecordLog(ctx.Context(), cfg.ID, ActionUpdateGroup, "Group", scimGroup.ExternalID, req.GroupID, LogStatusError, err.Error())
		return nil, forge.InternalError(err)
	}

	p.service.RecordLog(ctx.Context(), cfg.ID, action, "Group", scimGroup.ExternalID, team.ID.String(), LogStatusSuccess, "")
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

	teamID, err := id.ParseTeamID(req.GroupID)
	if err != nil {
		return nil, forge.NotFound("group not found")
	}

	team, err := p.authStore.GetTeam(ctx.Context(), teamID)
	if err != nil {
		return nil, forge.NotFound("group not found")
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
		return nil, forge.InternalError(err)
	}

	p.service.RecordLog(ctx.Context(), cfg.ID, ActionUpdateGroup, "Group", "", team.ID.String(), LogStatusSuccess, "")
	return TeamToSCIMGroup(team, nil, p.config.BasePath), nil
}

func (p *Plugin) handleDeleteGroup(ctx forge.Context, req *scimGroupPathParam) (*struct{}, error) {
	cfg, err := p.authenticateSCIM(ctx)
	if err != nil {
		return nil, err
	}

	teamID, err := id.ParseTeamID(req.GroupID)
	if err != nil {
		return nil, forge.NotFound("group not found")
	}

	if err := p.authStore.DeleteTeam(ctx.Context(), teamID); err != nil {
		p.service.RecordLog(ctx.Context(), cfg.ID, ActionDeleteGroup, "Group", "", req.GroupID, LogStatusError, err.Error())
		return nil, forge.InternalError(err)
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
func (p *Plugin) applyUserPatchReplace(u *user.User, op Operation) {
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
			u.Email = v
		}
	case "":
		// Bulk replace: value is a map of attributes.
		if m, ok := op.Value.(map[string]any); ok {
			if active, ok := m["active"].(bool); ok {
				u.Banned = !active
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
}
