package ratelimit

import "time"

// Rule defines a rate limit window and maximum allowed requests.
type Rule struct {
	Window time.Duration `json:"window"`
	Max    int           `json:"max"`
}

// Config for the rate limiting service.
type Config struct {
	Enabled     bool
	DefaultRule Rule
	// Rules allows per-path custom rate limit rules keyed by request path
	// e.g., "/api/auth/signin": {Window: time.Minute, Max: 20}
	Rules map[string]Rule
}

func defaultConfig() Config {
	return Config{
		Enabled:     true,
		DefaultRule: Rule{Window: time.Minute, Max: 60},
		Rules:       map[string]Rule{},
	}
}
