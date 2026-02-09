package username

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
)

// TestValidatePassword tests password validation rules.
func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		password    string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid password - meets min length",
			config: Config{
				MinPasswordLength: 8,
				MaxPasswordLength: 128,
			},
			password:    "password123",
			expectError: false,
		},
		{
			name: "invalid password - too short",
			config: Config{
				MinPasswordLength: 8,
				MaxPasswordLength: 128,
			},
			password:    "pass",
			expectError: true,
			errorMsg:    "password must be at least 8 characters",
		},
		{
			name: "invalid password - too long",
			config: Config{
				MinPasswordLength: 8,
				MaxPasswordLength: 20,
			},
			password:    "thispasswordiswaytoolongforthevalidator",
			expectError: true,
			errorMsg:    "password must be at most 20 characters",
		},
		{
			name: "invalid password - missing uppercase",
			config: Config{
				MinPasswordLength: 8,
				MaxPasswordLength: 128,
				RequireUppercase:  true,
			},
			password:    "password123",
			expectError: true,
			errorMsg:    "password must contain at least one uppercase letter",
		},
		{
			name: "valid password - has uppercase",
			config: Config{
				MinPasswordLength: 8,
				MaxPasswordLength: 128,
				RequireUppercase:  true,
			},
			password:    "Password123",
			expectError: false,
		},
		{
			name: "invalid password - missing lowercase",
			config: Config{
				MinPasswordLength: 8,
				MaxPasswordLength: 128,
				RequireLowercase:  true,
			},
			password:    "PASSWORD123",
			expectError: true,
			errorMsg:    "password must contain at least one lowercase letter",
		},
		{
			name: "valid password - has lowercase",
			config: Config{
				MinPasswordLength: 8,
				MaxPasswordLength: 128,
				RequireLowercase:  true,
			},
			password:    "Password123",
			expectError: false,
		},
		{
			name: "invalid password - missing number",
			config: Config{
				MinPasswordLength: 8,
				MaxPasswordLength: 128,
				RequireNumber:     true,
			},
			password:    "Password",
			expectError: true,
			errorMsg:    "password must contain at least one number",
		},
		{
			name: "valid password - has number",
			config: Config{
				MinPasswordLength: 8,
				MaxPasswordLength: 128,
				RequireNumber:     true,
			},
			password:    "Password123",
			expectError: false,
		},
		{
			name: "invalid password - missing special char",
			config: Config{
				MinPasswordLength:  8,
				MaxPasswordLength:  128,
				RequireSpecialChar: true,
			},
			password:    "Password123",
			expectError: true,
			errorMsg:    "password must contain at least one special character",
		},
		{
			name: "valid password - has special char",
			config: Config{
				MinPasswordLength:  8,
				MaxPasswordLength:  128,
				RequireSpecialChar: true,
			},
			password:    "Password123!",
			expectError: false,
		},
		{
			name: "valid password - all requirements met",
			config: Config{
				MinPasswordLength:  12,
				MaxPasswordLength:  128,
				RequireUppercase:   true,
				RequireLowercase:   true,
				RequireNumber:      true,
				RequireSpecialChar: true,
			},
			password:    "SecureP@ssw0rd",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &Service{config: tt.config}
			err := svc.ValidatePassword(tt.password)

			if tt.expectError {
				assert.Error(t, err)

				if tt.errorMsg != "" {
					// Check if it's an AuthsomeError and verify the context contains the message
					authErr := &errs.AuthsomeError{}
					if errs.As(err, &authErr) {
						// The specific reason is in the context
						assert.Contains(t, authErr.Message, "Password does not meet security requirements")
						// Check that the underlying error or context contains our specific message
						if authErr.Context != nil {
							reasonFound := false

							for _, v := range authErr.Context {
								if str, ok := v.(string); ok && strings.Contains(str, tt.errorMsg) {
									reasonFound = true

									break
								}
							}
							// If not in context, check if we can unwrap to get the original error
							if !reasonFound && authErr.Err != nil {
								assert.Contains(t, authErr.Err.Error(), tt.errorMsg)
							}
						}
					} else {
						// Fallback for non-AuthsomeError
						assert.Contains(t, err.Error(), tt.errorMsg)
					}
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestDefaultConfig tests the default configuration.
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, 8, cfg.MinPasswordLength)
	assert.Equal(t, 128, cfg.MaxPasswordLength)
	assert.False(t, cfg.RequireUppercase)
	assert.False(t, cfg.RequireLowercase)
	assert.False(t, cfg.RequireNumber)
	assert.False(t, cfg.RequireSpecialChar)
	assert.True(t, cfg.AllowUsernameLogin)

	// Account lockout defaults
	assert.True(t, cfg.LockoutEnabled)
	assert.Equal(t, 5, cfg.MaxFailedAttempts)
	assert.Equal(t, 15*time.Minute, cfg.LockoutDuration)
	assert.Equal(t, 10*time.Minute, cfg.FailedAttemptWindow)

	// Password history defaults
	assert.Equal(t, 5, cfg.PasswordHistorySize)
	assert.True(t, cfg.PreventPasswordReuse)

	// Password expiry defaults
	assert.False(t, cfg.PasswordExpiryEnabled)
	assert.Equal(t, 90, cfg.PasswordExpiryDays)
	assert.Equal(t, 7, cfg.PasswordExpiryWarning)

	// Rate limiting defaults
	assert.True(t, cfg.RateLimit.Enabled)
	assert.False(t, cfg.RateLimit.UseRedis)
	assert.Equal(t, "localhost:6379", cfg.RateLimit.RedisAddr)
	assert.Equal(t, 0, cfg.RateLimit.RedisDB)
}

// TestPluginOptions tests functional options.
func TestPluginOptions(t *testing.T) {
	tests := []struct {
		name   string
		option PluginOption
		check  func(*Plugin) bool
	}{
		{
			name:   "WithMinPasswordLength",
			option: WithMinPasswordLength(12),
			check: func(p *Plugin) bool {
				return p.defaultConfig.MinPasswordLength == 12
			},
		},
		{
			name:   "WithMaxPasswordLength",
			option: WithMaxPasswordLength(64),
			check: func(p *Plugin) bool {
				return p.defaultConfig.MaxPasswordLength == 64
			},
		},
		{
			name:   "WithRequireUppercase",
			option: WithRequireUppercase(true),
			check: func(p *Plugin) bool {
				return p.defaultConfig.RequireUppercase == true
			},
		},
		{
			name:   "WithRequireLowercase",
			option: WithRequireLowercase(true),
			check: func(p *Plugin) bool {
				return p.defaultConfig.RequireLowercase == true
			},
		},
		{
			name:   "WithRequireNumber",
			option: WithRequireNumber(true),
			check: func(p *Plugin) bool {
				return p.defaultConfig.RequireNumber == true
			},
		},
		{
			name:   "WithRequireSpecialChar",
			option: WithRequireSpecialChar(true),
			check: func(p *Plugin) bool {
				return p.defaultConfig.RequireSpecialChar == true
			},
		},
		{
			name:   "WithAllowUsernameLogin",
			option: WithAllowUsernameLogin(false),
			check: func(p *Plugin) bool {
				return p.defaultConfig.AllowUsernameLogin == false
			},
		},
		{
			name:   "WithLockoutEnabled",
			option: WithLockoutEnabled(false),
			check: func(p *Plugin) bool {
				return p.defaultConfig.LockoutEnabled == false
			},
		},
		{
			name:   "WithMaxFailedAttempts",
			option: WithMaxFailedAttempts(3),
			check: func(p *Plugin) bool {
				return p.defaultConfig.MaxFailedAttempts == 3
			},
		},
		{
			name:   "WithLockoutDuration",
			option: WithLockoutDuration(30 * time.Minute),
			check: func(p *Plugin) bool {
				return p.defaultConfig.LockoutDuration == 30*time.Minute
			},
		},
		{
			name:   "WithPasswordHistorySize",
			option: WithPasswordHistorySize(10),
			check: func(p *Plugin) bool {
				return p.defaultConfig.PasswordHistorySize == 10
			},
		},
		{
			name:   "WithPreventPasswordReuse",
			option: WithPreventPasswordReuse(false),
			check: func(p *Plugin) bool {
				return p.defaultConfig.PreventPasswordReuse == false
			},
		},
		{
			name:   "WithPasswordExpiryEnabled",
			option: WithPasswordExpiryEnabled(true),
			check: func(p *Plugin) bool {
				return p.defaultConfig.PasswordExpiryEnabled == true
			},
		},
		{
			name:   "WithPasswordExpiryDays",
			option: WithPasswordExpiryDays(60),
			check: func(p *Plugin) bool {
				return p.defaultConfig.PasswordExpiryDays == 60
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := NewPlugin(tt.option)
			require.NotNil(t, plugin)
			assert.True(t, tt.check(plugin), "option check failed")
		})
	}
}

// TestPluginID tests plugin ID.
func TestPluginID(t *testing.T) {
	plugin := NewPlugin()
	assert.Equal(t, "username", plugin.ID())
}

// TestPasswordExpiryCalculation tests password expiry logic
// Note: Uses account creation date as password change tracking is not in current schema.
func TestPasswordExpiryCalculation(t *testing.T) {
	svc := &Service{
		config: Config{
			PasswordExpiryEnabled: true,
			PasswordExpiryDays:    90,
		},
	}

	now := time.Now()

	tests := []struct {
		name             string
		createdAt        time.Time
		expectExpired    bool
		expectedDaysLeft int
	}{
		{
			name:             "account created 30 days ago - not expired",
			createdAt:        now.Add(-30 * 24 * time.Hour),
			expectExpired:    false,
			expectedDaysLeft: 60,
		},
		{
			name:             "account created 91 days ago - expired",
			createdAt:        now.Add(-91 * 24 * time.Hour),
			expectExpired:    true,
			expectedDaysLeft: 0,
		},
		{
			name:             "account created 89 days ago - not expired (edge case)",
			createdAt:        now.Add(-89 * 24 * time.Hour),
			expectExpired:    false,
			expectedDaysLeft: 1,
		},
		{
			name:             "account created 1 day ago - not expired",
			createdAt:        now.Add(-1 * 24 * time.Hour),
			expectExpired:    false,
			expectedDaysLeft: 89,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &user.User{
				CreatedAt: tt.createdAt,
			}

			expired := svc.isPasswordExpired(u)
			assert.Equal(t, tt.expectExpired, expired)

			daysLeft := svc.daysUntilPasswordExpiry(u)
			assert.Equal(t, tt.expectedDaysLeft, daysLeft)
		})
	}
}

// TestAccountLockoutError tests the AccountLockoutError type.
func TestAccountLockoutError(t *testing.T) {
	lockedUntil := time.Now().Add(15 * time.Minute)
	err := &AccountLockoutError{
		LockedUntil:   lockedUntil,
		LockedMinutes: 15,
	}

	assert.Contains(t, err.Error(), "account locked")
	assert.Contains(t, err.Error(), "15 minutes")
}
