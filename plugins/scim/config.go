package scim

// Config holds plugin-level configuration for the SCIM plugin.
type Config struct {
	// BasePath is the HTTP prefix for SCIM endpoints (default: "/scim/v2").
	BasePath string

	// TokenLength is the number of random bytes for generated bearer tokens (default: 32).
	TokenLength int

	// MaxLogEntries is the maximum number of provision logs to keep per config (default: 1000).
	MaxLogEntries int
}

func (c *Config) defaults() {
	if c.BasePath == "" {
		c.BasePath = "/scim/v2"
	}
	if c.TokenLength <= 0 {
		c.TokenLength = 32
	}
	if c.MaxLogEntries <= 0 {
		c.MaxLogEntries = 1000
	}
}
