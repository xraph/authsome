package idverification

import (
	"testing"
	"time"

	"github.com/xraph/authsome/internal/errs"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if !config.Enabled {
		t.Error("Expected Enabled to be true")
	}

	if config.DefaultProvider != "onfido" {
		t.Errorf("Expected DefaultProvider 'onfido', got %s", config.DefaultProvider)
	}

	if config.SessionExpiryDuration != 24*time.Hour {
		t.Errorf("Expected SessionExpiryDuration 24h, got %v", config.SessionExpiryDuration)
	}

	if !config.RequireDocumentVerification {
		t.Error("Expected RequireDocumentVerification to be true")
	}

	if config.MaxAllowedRiskScore != 70 {
		t.Errorf("Expected MaxAllowedRiskScore 70, got %d", config.MaxAllowedRiskScore)
	}

	if config.MinConfidenceScore != 80 {
		t.Errorf("Expected MinConfidenceScore 80, got %d", config.MinConfidenceScore)
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Run("valid default config", func(t *testing.T) {
		config := DefaultConfig()
		config.Onfido.Enabled = true
		config.Onfido.APIToken = "test_token"

		err := config.Validate()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("disabled config", func(t *testing.T) {
		config := DefaultConfig()
		config.Enabled = false

		err := config.Validate()
		if err != nil {
			t.Errorf("Expected no error for disabled config, got %v", err)
		}
	})

	t.Run("no provider enabled", func(t *testing.T) {
		config := DefaultConfig()
		config.Onfido.Enabled = false
		config.Jumio.Enabled = false
		config.StripeIdentity.Enabled = false

		err := config.Validate()
		if !errs.Is(err, ErrNoProviderEnabled) {
			t.Errorf("Expected ErrNoProviderEnabled, got %v", err)
		}
	})

	t.Run("empty default provider", func(t *testing.T) {
		config := DefaultConfig()
		config.DefaultProvider = ""
		config.Onfido.Enabled = true
		config.Onfido.APIToken = "test"

		err := config.Validate()
		if !errs.Is(err, ErrInvalidDefaultProvider) {
			t.Errorf("Expected ErrInvalidDefaultProvider, got %v", err)
		}
	})

	t.Run("default provider not enabled", func(t *testing.T) {
		config := DefaultConfig()
		config.DefaultProvider = "onfido"
		config.Onfido.Enabled = false
		config.Jumio.Enabled = true
		config.Jumio.APIToken = "test"
		config.Jumio.APISecret = "secret"

		err := config.Validate()
		if !errs.Is(err, ErrProviderNotEnabled) {
			t.Errorf("Expected ErrProviderNotEnabled, got %v", err)
		}
	})

	t.Run("onfido missing API token", func(t *testing.T) {
		config := DefaultConfig()
		config.DefaultProvider = "onfido"
		config.Onfido.Enabled = true
		config.Onfido.APIToken = ""

		err := config.Validate()
		if !errs.Is(err, ErrMissingAPIToken) {
			t.Errorf("Expected ErrMissingAPIToken, got %v", err)
		}
	})

	t.Run("jumio missing credentials", func(t *testing.T) {
		config := DefaultConfig()
		config.DefaultProvider = "jumio"
		config.Jumio.Enabled = true
		config.Jumio.APIToken = "test"
		config.Jumio.APISecret = ""

		err := config.Validate()
		if !errs.Is(err, ErrMissingAPICredentials) {
			t.Errorf("Expected ErrMissingAPICredentials, got %v", err)
		}
	})

	t.Run("stripe missing API key", func(t *testing.T) {
		config := DefaultConfig()
		config.DefaultProvider = "stripe_identity"
		config.StripeIdentity.Enabled = true
		config.StripeIdentity.APIKey = ""

		err := config.Validate()
		if !errs.Is(err, ErrMissingAPIKey) {
			t.Errorf("Expected ErrMissingAPIKey, got %v", err)
		}
	})

	t.Run("unsupported provider", func(t *testing.T) {
		config := DefaultConfig()
		config.DefaultProvider = "invalid_provider"
		config.Onfido.Enabled = true
		config.Onfido.APIToken = "test"

		err := config.Validate()
		if !errs.Is(err, ErrUnsupportedProvider) {
			t.Errorf("Expected ErrUnsupportedProvider, got %v", err)
		}
	})

	t.Run("invalid risk score - negative", func(t *testing.T) {
		config := DefaultConfig()
		config.MaxAllowedRiskScore = -1
		config.Onfido.Enabled = true
		config.Onfido.APIToken = "test"

		err := config.Validate()
		if !errs.Is(err, ErrInvalidRiskScore) {
			t.Errorf("Expected ErrInvalidRiskScore, got %v", err)
		}
	})

	t.Run("invalid risk score - too high", func(t *testing.T) {
		config := DefaultConfig()
		config.MaxAllowedRiskScore = 101
		config.Onfido.Enabled = true
		config.Onfido.APIToken = "test"

		err := config.Validate()
		if !errs.Is(err, ErrInvalidRiskScore) {
			t.Errorf("Expected ErrInvalidRiskScore, got %v", err)
		}
	})

	t.Run("invalid confidence score", func(t *testing.T) {
		config := DefaultConfig()
		config.MinConfidenceScore = 150
		config.Onfido.Enabled = true
		config.Onfido.APIToken = "test"

		err := config.Validate()
		if !errs.Is(err, ErrInvalidConfidenceScore) {
			t.Errorf("Expected ErrInvalidConfidenceScore, got %v", err)
		}
	})

	t.Run("invalid minimum age", func(t *testing.T) {
		config := DefaultConfig()
		config.RequireAgeVerification = true
		config.MinimumAge = -5
		config.Onfido.Enabled = true
		config.Onfido.APIToken = "test"

		err := config.Validate()
		if !errs.Is(err, ErrInvalidMinimumAge) {
			t.Errorf("Expected ErrInvalidMinimumAge, got %v", err)
		}
	})

	t.Run("invalid rate limit", func(t *testing.T) {
		config := DefaultConfig()
		config.RateLimitEnabled = true
		config.MaxVerificationsPerHour = -10
		config.Onfido.Enabled = true
		config.Onfido.APIToken = "test"

		err := config.Validate()
		if !errs.Is(err, ErrInvalidRateLimit) {
			t.Errorf("Expected ErrInvalidRateLimit, got %v", err)
		}
	})

	t.Run("invalid max attempts", func(t *testing.T) {
		config := DefaultConfig()
		config.MaxVerificationAttempts = 0
		config.Onfido.Enabled = true
		config.Onfido.APIToken = "test"

		err := config.Validate()
		if !errs.Is(err, ErrInvalidMaxAttempts) {
			t.Errorf("Expected ErrInvalidMaxAttempts, got %v", err)
		}
	})

	t.Run("valid jumio config", func(t *testing.T) {
		config := DefaultConfig()
		config.DefaultProvider = "jumio"
		config.Jumio.Enabled = true
		config.Jumio.APIToken = "test_token"
		config.Jumio.APISecret = "test_secret"

		err := config.Validate()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("valid stripe config", func(t *testing.T) {
		config := DefaultConfig()
		config.DefaultProvider = "stripe_identity"
		config.StripeIdentity.Enabled = true
		config.StripeIdentity.APIKey = "sk_test_123"

		err := config.Validate()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}

func TestOnfidoConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Onfido.Region != "eu" {
		t.Errorf("Expected Onfido region 'eu', got %s", config.Onfido.Region)
	}

	if !config.Onfido.DocumentCheck.Enabled {
		t.Error("Expected DocumentCheck to be enabled")
	}

	if !config.Onfido.FacialCheck.Enabled {
		t.Error("Expected FacialCheck to be enabled")
	}

	if config.Onfido.FacialCheck.Variant != "video" {
		t.Errorf("Expected FacialCheck variant 'video', got %s", config.Onfido.FacialCheck.Variant)
	}
}

func TestJumioConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Jumio.DataCenter != "us" {
		t.Errorf("Expected Jumio data center 'us', got %s", config.Jumio.DataCenter)
	}

	if config.Jumio.VerificationType != "identity" {
		t.Errorf("Expected verification type 'identity', got %s", config.Jumio.VerificationType)
	}

	if !config.Jumio.EnableLiveness {
		t.Error("Expected EnableLiveness to be true")
	}
}

func TestStripeIdentityConfig(t *testing.T) {
	config := DefaultConfig()

	if !config.StripeIdentity.RequireLiveCapture {
		t.Error("Expected RequireLiveCapture to be true")
	}

	if len(config.StripeIdentity.AllowedTypes) != 1 {
		t.Errorf("Expected 1 allowed type, got %d", len(config.StripeIdentity.AllowedTypes))
	}

	if config.StripeIdentity.AllowedTypes[0] != "document" {
		t.Errorf("Expected allowed type 'document', got %s", config.StripeIdentity.AllowedTypes[0])
	}
}

func TestConfigDefaults(t *testing.T) {
	config := DefaultConfig()

	// Test accepted documents
	expectedDocs := []string{"passport", "drivers_license", "national_id"}
	if len(config.AcceptedDocuments) != len(expectedDocs) {
		t.Errorf("Expected %d accepted documents, got %d", len(expectedDocs), len(config.AcceptedDocuments))
	}

	// Test webhook events
	if len(config.WebhookEvents) != 3 {
		t.Errorf("Expected 3 webhook events, got %d", len(config.WebhookEvents))
	}

	// Test compliance settings
	if !config.GDPRCompliant {
		t.Error("Expected GDPR compliant by default")
	}

	if config.ComplianceMode != "standard" {
		t.Errorf("Expected compliance mode 'standard', got %s", config.ComplianceMode)
	}

	// Test retention
	if !config.RetainDocuments {
		t.Error("Expected document retention to be enabled")
	}

	if config.DocumentRetentionPeriod != 90*24*time.Hour {
		t.Errorf("Expected retention period 90 days, got %v", config.DocumentRetentionPeriod)
	}
}
