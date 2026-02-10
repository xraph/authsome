package phone

import (
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidatePhone tests phone number validation.
func TestValidatePhone(t *testing.T) {
	tests := []struct {
		name    string
		phone   string
		wantErr bool
		errType error
	}{
		{
			name:    "valid US number",
			phone:   "+12345678901",
			wantErr: false,
		},
		{
			name:    "valid UK number",
			phone:   "+442071838750",
			wantErr: false,
		},
		{
			name:    "valid short number",
			phone:   "+123456789",
			wantErr: false,
		},
		{
			name:    "empty phone",
			phone:   "",
			wantErr: true,
			errType: ErrMissingPhone,
		},
		{
			name:    "missing plus sign",
			phone:   "12345678901",
			wantErr: true,
			errType: ErrInvalidPhoneFormat,
		},
		{
			name:    "starts with zero",
			phone:   "+0123456789",
			wantErr: true,
			errType: ErrInvalidPhoneFormat,
		},
		{
			name:    "contains spaces",
			phone:   "+1 234 567 8901",
			wantErr: true,
			errType: ErrInvalidPhoneFormat,
		},
		{
			name:    "contains dashes",
			phone:   "+1-234-567-8901",
			wantErr: true,
			errType: ErrInvalidPhoneFormat,
		},
		{
			name:    "contains letters",
			phone:   "+1234567890a",
			wantErr: true,
			errType: ErrInvalidPhoneFormat,
		},
		{
			name:    "too short",
			phone:   "+12",
			wantErr: true,
			errType: ErrInvalidPhoneFormat,
		},
		{
			name:    "too long",
			phone:   "+123456789012345678",
			wantErr: true,
			errType: ErrInvalidPhoneFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePhone(tt.phone)
			if tt.wantErr {
				require.Error(t, err)

				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestGenerateSecureCode tests secure code generation.
func TestGenerateSecureCode(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantErr bool
	}{
		{
			name:    "4 digit code",
			length:  4,
			wantErr: false,
		},
		{
			name:    "6 digit code",
			length:  6,
			wantErr: false,
		},
		{
			name:    "8 digit code",
			length:  8,
			wantErr: false,
		},
		{
			name:    "zero length",
			length:  0,
			wantErr: true,
		},
		{
			name:    "negative length",
			length:  -1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := generateSecureCode(tt.length)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Len(t, code, tt.length)

			// Verify it's numeric
			matched, err := regexp.MatchString(`^\d+$`, code)
			require.NoError(t, err)
			assert.True(t, matched, "code should be numeric")

			// Verify it has leading zeros if needed
			if tt.length > 0 {
				assert.Len(t, code, tt.length)
			}
		})
	}
}

// TestGenerateSecureCodeRandomness tests that codes are sufficiently random.
func TestGenerateSecureCodeRandomness(t *testing.T) {
	codes := make(map[string]bool)
	iterations := 100
	length := 6

	for range iterations {
		code, err := generateSecureCode(length)
		require.NoError(t, err)

		codes[code] = true
	}

	// We should have close to 100 unique codes (very unlikely to have many duplicates)
	uniqueCount := len(codes)
	assert.Greater(t, uniqueCount, 90, "should generate mostly unique codes")
}

// TestServiceErrors tests error conditions.
func TestServiceErrors(t *testing.T) {
	t.Run("invalid phone format", func(t *testing.T) {
		err := validatePhone("invalid-phone")
		assert.ErrorIs(t, err, ErrInvalidPhoneFormat)
	})

	t.Run("missing phone", func(t *testing.T) {
		err := validatePhone("")
		assert.ErrorIs(t, err, ErrMissingPhone)
	})
}

// TestRateLimitConfig tests rate limit configuration.
func TestRateLimitConfig(t *testing.T) {
	config := DefaultConfig()

	assert.True(t, config.RateLimit.Enabled)
	assert.False(t, config.RateLimit.UseRedis)
	assert.Equal(t, 1*time.Minute, config.RateLimit.SendCodePerPhone.Window)
	assert.Equal(t, 3, config.RateLimit.SendCodePerPhone.Max)
	assert.Equal(t, 1*time.Hour, config.RateLimit.SendCodePerIP.Window)
	assert.Equal(t, 20, config.RateLimit.SendCodePerIP.Max)
}

// TestConfigOptions tests functional configuration options.
func TestConfigOptions(t *testing.T) {
	t.Run("WithCodeLength", func(t *testing.T) {
		p := NewPlugin(WithCodeLength(8))
		assert.Equal(t, 8, p.defaultConfig.CodeLength)
	})

	t.Run("WithExpiryMinutes", func(t *testing.T) {
		p := NewPlugin(WithExpiryMinutes(15))
		assert.Equal(t, 15, p.defaultConfig.ExpiryMinutes)
	})

	t.Run("WithMaxAttempts", func(t *testing.T) {
		p := NewPlugin(WithMaxAttempts(3))
		assert.Equal(t, 3, p.defaultConfig.MaxAttempts)
	})

	t.Run("WithSMSProvider", func(t *testing.T) {
		p := NewPlugin(WithSMSProvider("aws_sns"))
		assert.Equal(t, "aws_sns", p.defaultConfig.SMSProvider)
	})

	t.Run("WithAllowImplicitSignup", func(t *testing.T) {
		p := NewPlugin(WithAllowImplicitSignup(false))
		assert.False(t, p.defaultConfig.AllowImplicitSignup)
	})

	t.Run("WithDevExposeCode", func(t *testing.T) {
		p := NewPlugin(WithDevExposeCode(true))
		assert.True(t, p.defaultConfig.DevExposeCode)
	})

	t.Run("multiple options", func(t *testing.T) {
		p := NewPlugin(
			WithCodeLength(8),
			WithExpiryMinutes(20),
			WithMaxAttempts(10),
			WithSMSProvider("twilio"),
			WithDevExposeCode(true),
		)

		assert.Equal(t, 8, p.defaultConfig.CodeLength)
		assert.Equal(t, 20, p.defaultConfig.ExpiryMinutes)
		assert.Equal(t, 10, p.defaultConfig.MaxAttempts)
		assert.Equal(t, "twilio", p.defaultConfig.SMSProvider)
		assert.True(t, p.defaultConfig.DevExposeCode)
	})
}

// TestPluginID tests plugin identification.
func TestPluginID(t *testing.T) {
	p := NewPlugin()
	assert.Equal(t, "phone", p.ID())
}

// Integration test notes:
// Full integration tests would require:
// 1. Real database connection (using testcontainers)
// 2. Mock notification adapter
// 3. Testing full send code -> verify flow
// 4. Testing rate limiting with actual storage
// 5. Testing implicit signup flow
// 6. Testing error scenarios with database failures
// 7. Testing concurrent access patterns
//
// These should be implemented in a separate integration test file
// with proper test fixtures and database setup.
