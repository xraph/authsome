package auth

// Config represents authentication configuration
type Config struct {
	RequireEmailVerification bool `json:"requireEmailVerification"`
}
