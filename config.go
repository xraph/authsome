package authsome

// Mode represents the operation mode
type Mode int

const (
    // ModeStandalone represents single-tenant mode
    ModeStandalone Mode = iota
    // ModeSaaS represents multi-tenant mode
    ModeSaaS
)

// Config represents the root configuration
type Config struct {
    // Mode determines if running in standalone or SaaS mode
    Mode Mode

    // BasePath is the base path for auth routes
    BasePath string

    // TrustedOrigins for CORS
    TrustedOrigins []string

    // Secret for signing tokens
    Secret string

    // RBACEnforce toggles handler-level RBAC enforcement (off by default)
    RBACEnforce bool
}