package authsome

import (
	"testing"
)

func TestNew(t *testing.T) {
	auth := New()
	if auth == nil {
		t.Fatal("expected auth instance, got nil")
	}

	// Mode removed - multitenancy and organization plugins control behavior
	// No mode check needed; auth initialized successfully
}

func TestWithBasePath(t *testing.T) {
	basePath := "/auth/v2"

	auth := New(WithBasePath(basePath))
	if auth.config.BasePath != basePath {
		t.Errorf("expected base path %s, got %s", basePath, auth.config.BasePath)
	}
}
