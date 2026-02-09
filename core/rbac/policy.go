package rbac

// Policy represents a single RBAC rule parsed from an expression
// Example: "user:read,write on project:* where owner = true".
type Policy struct {
	Subject   string   `json:"subject"`   // e.g. "user", "role:admin"
	Actions   []string `json:"actions"`   // e.g. ["read", "write"]
	Resource  string   `json:"resource"`  // e.g. "project:*"
	Condition string   `json:"condition"` // optional condition, e.g. "owner = true"
}

// Context captures what we're trying to authorize.
type Context struct {
	Subject  string            `json:"subject"`  // e.g. "user", "role:admin"
	Action   string            `json:"action"`   // e.g. "read"
	Resource string            `json:"resource"` // e.g. "project:123"
	Vars     map[string]string `json:"vars"`     // context variables used by conditions
}
