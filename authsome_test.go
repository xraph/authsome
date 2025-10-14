package authsome

import (
    "testing"
)

func TestNew(t *testing.T) {
    auth := New()
    if auth == nil {
        t.Fatal("expected auth instance, got nil")
    }

    if auth.config.Mode != ModeStandalone {
        t.Errorf("expected standalone mode, got %v", auth.config.Mode)
    }
}

func TestWithMode(t *testing.T) {
    auth := New(WithMode(ModeSaaS))
    if auth.config.Mode != ModeSaaS {
        t.Errorf("expected SaaS mode, got %v", auth.config.Mode)
    }
}