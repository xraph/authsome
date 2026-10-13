package hook

import "strings"

// Audit categories. An audit sink such as Chronicle refuses a record with no
// category, so every event the bus dispatches carries one: the call site's
// own choice when it set one, otherwise CategoryFor's.
const (
	CategoryAuth    = "auth"
	CategoryAccess  = "access"
	CategoryAccount = "account"
	CategoryOrg     = "org"
	CategoryAdmin   = "admin"
	CategorySystem  = "system"
)

// CategoryFor returns the category an event gets when its call site left
// Category empty. It goes by the action's leading segment ("org" for
// "org.member.add"), then by the resource, and falls back to CategorySystem,
// so the answer is never empty.
func CategoryFor(action, resource string) string {
	prefix := action
	if i := strings.IndexByte(prefix, '.'); i >= 0 {
		prefix = prefix[:i]
	}
	switch prefix {
	case "auth", "session", "device", "passkey", "social", "sso", "mfa":
		return CategoryAuth
	case "rbac", "principal", "permission":
		return CategoryAccess
	case "user":
		return CategoryAccount
	case "org":
		return CategoryOrg
	case "admin", "app", "environment", "webhook", "audit":
		return CategoryAdmin
	case "settings", "apikey", "waitlist", "consent", "scim", "subscription", "oauth2":
		return prefix
	}
	if strings.HasPrefix(action, "dpop_") {
		return CategoryAuth
	}

	switch resource {
	case ResourceUser:
		return CategoryAccount
	case ResourceSession, ResourceDevice, ResourcePasskey:
		return CategoryAuth
	case ResourceOrganization, ResourceMember, ResourceTeam, ResourceInvitation:
		return CategoryOrg
	case ResourceRole, ResourcePermission:
		return CategoryAccess
	case ResourceApp, ResourceEnvironment, ResourceWebhook:
		return CategoryAdmin
	case ResourceAPIKey, ResourceWaitlist:
		return resource
	}
	return CategorySystem
}
