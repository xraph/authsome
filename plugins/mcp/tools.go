package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/user"
)

// Tool defines the interface for MCP tools
type Tool interface {
	Execute(ctx context.Context, arguments map[string]interface{}, plugin *Plugin) (string, error)
	Describe() ToolDescription
	RequiresAuth() bool
	RequiresAdmin() bool
}

// ToolDescription describes a tool for MCP clients
type ToolDescription struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolRegistry manages available tools
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register registers a tool handler
func (r *ToolRegistry) Register(name string, tool Tool) {
	r.tools[name] = tool
}

// List returns descriptions of all tools (filtered by mode)
func (r *ToolRegistry) List(mode Mode) []ToolDescription {
	var descriptions []ToolDescription
	for _, tool := range r.tools {
		// Filter based on mode
		if mode == ModeReadOnly && (tool.RequiresAuth() || tool.RequiresAdmin()) {
			continue
		}
		if mode == ModeAdmin && tool.RequiresAdmin() {
			continue
		}

		descriptions = append(descriptions, tool.Describe())
	}
	return descriptions
}

// Execute executes a tool by name
func (r *ToolRegistry) Execute(ctx context.Context, name string, arguments map[string]interface{}, plugin *Plugin) (string, error) {
	tool, exists := r.tools[name]
	if !exists {
		return "", fmt.Errorf("tool not found: %s", name)
	}

	// TODO: Check authorization based on tool requirements

	return tool.Execute(ctx, arguments, plugin)
}

// QueryUserTool finds users by email/ID/username
type QueryUserTool struct{}

func (t *QueryUserTool) Describe() ToolDescription {
	return ToolDescription{
		Name:        "query_user",
		Description: "Find user by email, ID, or username. Returns sanitized user data (no password hashes).",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"email": map[string]interface{}{
					"type":        "string",
					"description": "User email address",
				},
				"id": map[string]interface{}{
					"type":        "string",
					"description": "User ID (UUID)",
				},
				"username": map[string]interface{}{
					"type":        "string",
					"description": "Username",
				},
			},
			"oneOf": []map[string]interface{}{
				{"required": []string{"email"}},
				{"required": []string{"id"}},
				{"required": []string{"username"}},
			},
		},
	}
}

func (t *QueryUserTool) RequiresAuth() bool {
	return false // Read-only operation
}

func (t *QueryUserTool) RequiresAdmin() bool {
	return false
}

func (t *QueryUserTool) Execute(ctx context.Context, arguments map[string]interface{}, plugin *Plugin) (string, error) {
	if plugin.serviceRegistry == nil {
		return "", fmt.Errorf("service registry not available")
	}

	userService := plugin.serviceRegistry.UserService()
	if userService == nil {
		return "", fmt.Errorf("user service not available")
	}

	var foundUser *user.User
	var err error

	// Find by email
	if email, ok := arguments["email"].(string); ok {
		foundUser, err = userService.FindByEmail(ctx, email)
		if err != nil {
			return "", fmt.Errorf("user not found by email: %w", err)
		}
	} else if idStr, ok := arguments["id"].(string); ok {
		// Find by ID - parse xid from string
		parsedID, err := xid.FromString(idStr)
		if err != nil {
			return "", fmt.Errorf("invalid user ID format: %w", err)
		}
		foundUser, err = userService.FindByID(ctx, parsedID)
		if err != nil {
			return "", fmt.Errorf("user not found by ID: %w", err)
		}
	} else {
		return "", fmt.Errorf("must provide email or id")
	}

	// Sanitize user data (remove sensitive fields)
	sanitized := sanitizeUser(foundUser)

	data, err := json.MarshalIndent(sanitized, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal user: %w", err)
	}

	return string(data), nil
}

// CheckPermissionTool verifies RBAC permissions
type CheckPermissionTool struct{}

func (t *CheckPermissionTool) Describe() ToolDescription {
	return ToolDescription{
		Name:        "check_permission",
		Description: "Check if a user has a specific permission on a resource",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"user_id": map[string]interface{}{
					"type":        "string",
					"description": "User ID to check",
				},
				"action": map[string]interface{}{
					"type":        "string",
					"description": "Action (e.g., 'read', 'write', 'delete')",
				},
				"resource": map[string]interface{}{
					"type":        "string",
					"description": "Resource identifier (e.g., 'organization:123', 'user:*')",
				},
			},
			"required": []string{"user_id", "action", "resource"},
		},
	}
}

func (t *CheckPermissionTool) RequiresAuth() bool {
	return false
}

func (t *CheckPermissionTool) RequiresAdmin() bool {
	return false
}

func (t *CheckPermissionTool) Execute(ctx context.Context, arguments map[string]interface{}, plugin *Plugin) (string, error) {
	userID, _ := arguments["user_id"].(string)
	action, _ := arguments["action"].(string)
	resource, _ := arguments["resource"].(string)

	if userID == "" || action == "" || resource == "" {
		return "", fmt.Errorf("missing required arguments")
	}

	rbacService := plugin.serviceRegistry.RBACService()
	if rbacService == nil {
		return "", fmt.Errorf("RBAC service not available")
	}

	// TODO: Get user roles and check permission
	// For now, return a placeholder

	result := map[string]interface{}{
		"user_id":   userID,
		"action":    action,
		"resource":  resource,
		"permitted": false, // TODO: Actual check
		"reason":    "RBAC check not yet fully implemented in MCP plugin",
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(data), nil
}

// sanitizeUser removes sensitive fields from user data
func sanitizeUser(u *user.User) map[string]interface{} {
	if u == nil {
		return nil
	}

	return map[string]interface{}{
		"id":            u.ID,
		"email":         u.Email,
		"name":          u.Name,
		"emailVerified": u.EmailVerified,
		"image":         u.Image,
		"createdAt":     u.CreatedAt,
		"updatedAt":     u.UpdatedAt,
		// Explicitly exclude: Password, TwoFactorEnabled, etc.
	}
}
