package apikey

import "strings"

// ScopeToRBACMapping maps legacy scope strings to RBAC (action, resource) pairs
// This enables backward compatibility with existing scopes while migrating to RBAC
var ScopeToRBACMapping = map[string]RBACPermission{
	// User operations
	"users:read":   {Action: "view", Resource: "users"},
	"users:write":  {Action: "edit", Resource: "users"},
	"users:create": {Action: "create", Resource: "users"},
	"users:delete": {Action: "delete", Resource: "users"},
	"users:*":      {Action: "*", Resource: "users"},

	// Session operations
	"sessions:read":   {Action: "view", Resource: "sessions"},
	"sessions:create": {Action: "create", Resource: "sessions"},
	"sessions:delete": {Action: "delete", Resource: "sessions"},
	"sessions:*":      {Action: "*", Resource: "sessions"},

	// API key operations
	"apikeys:read":   {Action: "view", Resource: "apikeys"},
	"apikeys:write":  {Action: "edit", Resource: "apikeys"},
	"apikeys:create": {Action: "create", Resource: "apikeys"},
	"apikeys:delete": {Action: "delete", Resource: "apikeys"},
	"apikeys:*":      {Action: "*", Resource: "apikeys"},

	// Organization operations
	"organizations:read":   {Action: "view", Resource: "organizations"},
	"organizations:write":  {Action: "edit", Resource: "organizations"},
	"organizations:create": {Action: "create", Resource: "organizations"},
	"organizations:delete": {Action: "delete", Resource: "organizations"},
	"organizations:*":      {Action: "*", Resource: "organizations"},

	// Role operations
	"roles:read":   {Action: "view", Resource: "roles"},
	"roles:write":  {Action: "edit", Resource: "roles"},
	"roles:create": {Action: "create", Resource: "roles"},
	"roles:delete": {Action: "delete", Resource: "roles"},
	"roles:*":      {Action: "*", Resource: "roles"},

	// Permission operations
	"permissions:read":   {Action: "view", Resource: "permissions"},
	"permissions:write":  {Action: "edit", Resource: "permissions"},
	"permissions:create": {Action: "create", Resource: "permissions"},
	"permissions:delete": {Action: "delete", Resource: "permissions"},
	"permissions:*":      {Action: "*", Resource: "permissions"},

	// Admin operations
	"admin:full":  {Action: "*", Resource: "*"},
	"admin:users": {Action: "*", Resource: "users"},
	"admin:orgs":  {Action: "*", Resource: "organizations"},

	// Public/frontend-safe operations
	"app:identify": {Action: "identify", Resource: "app"},
	"public:read":  {Action: "view", Resource: "public"},
	"users:verify": {Action: "verify", Resource: "users"},
}

// RBACPermission represents a parsed RBAC permission
type RBACPermission struct {
	Action   string // e.g., "view", "edit", "create", "delete", "*"
	Resource string // e.g., "users", "sessions", "apikeys", "*"
}

// MapScopeToRBAC converts a legacy scope string to RBAC action and resource
// Returns ("", "") if no mapping exists
func MapScopeToRBAC(scope string) (action, resource string) {
	// Direct lookup
	if perm, exists := ScopeToRBACMapping[scope]; exists {
		return perm.Action, perm.Resource
	}

	// Try parsing as "resource:action" format
	parts := strings.Split(scope, ":")
	if len(parts) == 2 {
		return parts[1], parts[0] // action, resource
	}

	return "", ""
}

// CheckScopeOrRBAC checks if a scope string OR RBAC permission grants access
// This is the flexible check for backward compatibility
func CheckScopeOrRBAC(scopes []string, rbacPermissions []string, requiredAction, requiredResource string) bool {
	// Method 1: Check legacy scopes
	requiredScope := requiredResource + ":" + requiredAction
	for _, scope := range scopes {
		// Exact match
		if scope == requiredScope {
			return true
		}

		// Admin full access
		if scope == "admin:full" {
			return true
		}

		// Wildcard resource: "users:*"
		if scope == requiredResource+":*" {
			return true
		}

		// Map scope to RBAC and check
		action, resource := MapScopeToRBAC(scope)
		if matchRBACPermission(action, resource, requiredAction, requiredResource) {
			return true
		}
	}

	// Method 2: Check RBAC permissions
	for _, perm := range rbacPermissions {
		parts := strings.Split(perm, ":")
		if len(parts) == 2 {
			action, resource := parts[0], parts[1]
			if matchRBACPermission(action, resource, requiredAction, requiredResource) {
				return true
			}
		}
	}

	return false
}

// matchRBACPermission checks if an RBAC permission matches the required action/resource
func matchRBACPermission(permAction, permResource, requiredAction, requiredResource string) bool {
	// Full wildcard
	if permAction == "*" && permResource == "*" {
		return true
	}

	// Action wildcard
	if permAction == "*" && permResource == requiredResource {
		return true
	}

	// Resource wildcard
	if permAction == requiredAction && permResource == "*" {
		return true
	}

	// Exact match
	if permAction == requiredAction && permResource == requiredResource {
		return true
	}

	return false
}

// ConvertScopesToRBAC converts an array of scope strings to RBAC permissions
// Useful for migrating existing API keys from scopes to RBAC roles
func ConvertScopesToRBAC(scopes []string) []RBACPermission {
	result := []RBACPermission{}
	seen := make(map[string]bool)

	for _, scope := range scopes {
		action, resource := MapScopeToRBAC(scope)
		if action != "" && resource != "" {
			key := action + ":" + resource
			if !seen[key] {
				seen[key] = true
				result = append(result, RBACPermission{
					Action:   action,
					Resource: resource,
				})
			}
		}
	}

	return result
}

// GenerateSuggestedRole analyzes scopes and suggests appropriate RBAC roles
// Returns suggested role names based on the scope patterns
func GenerateSuggestedRole(scopes []string) string {
	hasAdmin := false
	hasUsers := false
	hasSessions := false
	hasAPIKeys := false
	hasOrgs := false

	for _, scope := range scopes {
		if scope == "admin:full" || scope == "*:*" {
			hasAdmin = true
		}
		if strings.HasPrefix(scope, "users:") {
			hasUsers = true
		}
		if strings.HasPrefix(scope, "sessions:") {
			hasSessions = true
		}
		if strings.HasPrefix(scope, "apikeys:") {
			hasAPIKeys = true
		}
		if strings.HasPrefix(scope, "organizations:") {
			hasOrgs = true
		}
	}

	// Suggest role based on patterns
	if hasAdmin {
		return "admin"
	}
	if hasUsers && hasSessions && hasAPIKeys && hasOrgs {
		return "platform_manager"
	}
	if hasUsers && hasSessions {
		return "user_manager"
	}
	if hasUsers {
		return "user_viewer"
	}
	if hasAPIKeys {
		return "api_key_manager"
	}

	return "custom_role"
}
