package session

import (
	"net/http"
	"testing"
)

func TestDefaultCookieConfig(t *testing.T) {
	config := DefaultCookieConfig()

	if config.Enabled {
		t.Error("Expected Enabled to be false by default")
	}
	if config.Name != "authsome_session" {
		t.Errorf("Expected Name to be 'authsome_session', got '%s'", config.Name)
	}
	if config.Path != "/" {
		t.Errorf("Expected Path to be '/', got '%s'", config.Path)
	}
	if !config.HttpOnly {
		t.Error("Expected HttpOnly to be true by default")
	}
	if config.SameSite != "Lax" {
		t.Errorf("Expected SameSite to be 'Lax', got '%s'", config.SameSite)
	}
}

func TestCookieConfigMerge(t *testing.T) {
	base := CookieConfig{
		Enabled:  true,
		Name:     "base_session",
		Path:     "/",
		HttpOnly: true,
		SameSite: "Lax",
	}

	override := CookieConfig{
		Name:     "override_session",
		SameSite: "Strict",
		// Note: Enabled and HttpOnly are false (zero values) here
		// With the new merge logic, zero values don't override base values
		// This prevents app metadata from accidentally disabling globally-enabled features
	}

	merged := base.Merge(&override)

	// Check that override values are applied
	if merged.Name != "override_session" {
		t.Errorf("Expected Name to be 'override_session', got '%s'", merged.Name)
	}
	if merged.SameSite != "Strict" {
		t.Errorf("Expected SameSite to be 'Strict', got '%s'", merged.SameSite)
	}

	// Check that base values are preserved for non-overridden fields
	if merged.Path != "/" {
		t.Errorf("Expected Path to be '/' from base, got '%s'", merged.Path)
	}

	// NEW BEHAVIOR: Zero values in override don't override base values
	// This prevents app metadata without explicit "enabled: true" from
	// accidentally disabling globally-enabled cookies
	if !merged.Enabled {
		t.Error("Expected Enabled to remain true from base (zero value in override shouldn't disable)")
	}
	if !merged.HttpOnly {
		t.Error("Expected HttpOnly to remain true from base (zero value in override shouldn't disable)")
	}
}

func TestCookieConfigMergeWithNil(t *testing.T) {
	base := CookieConfig{
		Enabled:  true,
		Name:     "test_session",
		Path:     "/api",
		HttpOnly: true,
	}

	merged := base.Merge(nil)

	// Should return a copy of base config
	if merged.Enabled != base.Enabled {
		t.Error("Expected Enabled to match base")
	}
	if merged.Name != base.Name {
		t.Error("Expected Name to match base")
	}
	if merged.Path != base.Path {
		t.Error("Expected Path to match base")
	}
	if merged.HttpOnly != base.HttpOnly {
		t.Error("Expected HttpOnly to match base")
	}
}

func TestCookieConfigMergeWithPointerFields(t *testing.T) {
	secureTrue := true
	secureFalse := false
	maxAge3600 := 3600
	maxAge7200 := 7200

	base := CookieConfig{
		Enabled: true,
		Name:    "base_session",
		Secure:  &secureTrue,
		MaxAge:  &maxAge3600,
	}

	override := CookieConfig{
		Secure: &secureFalse,
		MaxAge: &maxAge7200,
	}

	merged := base.Merge(&override)

	// Check that pointer overrides are applied
	if merged.Secure == nil || *merged.Secure != false {
		t.Error("Expected Secure to be false from override")
	}
	if merged.MaxAge == nil || *merged.MaxAge != 7200 {
		t.Errorf("Expected MaxAge to be 7200 from override, got %v", *merged.MaxAge)
	}

	// Check that non-overridden values are preserved
	if merged.Name != "base_session" {
		t.Errorf("Expected Name to be 'base_session' from base, got '%s'", merged.Name)
	}
}

func TestParseSameSite(t *testing.T) {
	tests := []struct {
		input    string
		expected http.SameSite
	}{
		{"Strict", http.SameSiteStrictMode},
		{"strict", http.SameSiteStrictMode},
		{"Lax", http.SameSiteLaxMode},
		{"lax", http.SameSiteLaxMode},
		{"", http.SameSiteLaxMode}, // Default
		{"None", http.SameSiteNoneMode},
		{"none", http.SameSiteNoneMode},
		{"invalid", http.SameSiteLaxMode}, // Invalid defaults to Lax
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseSameSite(tt.input)
			if result != tt.expected {
				t.Errorf("ParseSameSite(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCookieConfigMergeDomain(t *testing.T) {
	base := CookieConfig{
		Enabled: true,
		Name:    "base_session",
		Domain:  ".example.com",
	}

	override := CookieConfig{
		Domain: ".app.example.com",
	}

	merged := base.Merge(&override)

	if merged.Domain != ".app.example.com" {
		t.Errorf("Expected Domain to be '.app.example.com', got '%s'", merged.Domain)
	}
}

func TestCookieConfigMergeEnabled(t *testing.T) {
	// Test that Enabled=false in override doesn't disable base Enabled=true
	// This is intentional - we can't distinguish between explicit false and unset (zero value)
	// To prevent app metadata from accidentally disabling globally-enabled cookies,
	// we only override Enabled if the override sets it to true
	base := CookieConfig{
		Enabled: true,
		Name:    "base_session",
	}

	override := CookieConfig{
		Enabled: false, // This is the same as not setting it (zero value)
	}

	merged := base.Merge(&override)

	// NEW BEHAVIOR: base.Enabled=true is preserved even when override.Enabled=false
	// This prevents app metadata from accidentally disabling globally-enabled cookies
	if !merged.Enabled {
		t.Error("Expected Enabled to remain true from base (override.Enabled=false shouldn't disable)")
	}
}

func TestCookieConfigMergeEnabledExplicitTrue(t *testing.T) {
	// Test that Enabled=true in override can enable cookies even when base is disabled
	base := CookieConfig{
		Enabled: false,
		Name:    "base_session",
	}

	override := CookieConfig{
		Enabled: true, // Explicit enable
	}

	merged := base.Merge(&override)

	if !merged.Enabled {
		t.Error("Expected Enabled to be true from override (explicit enable)")
	}
}
