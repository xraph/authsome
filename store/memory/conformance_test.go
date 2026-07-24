package memory_test

import (
	"testing"

	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/store/storetest"
)

// TestConformance runs the shared cross-backend store contract suite against
// the in-memory backend. It runs in normal CI (no Docker required).
func TestConformance(t *testing.T) {
	storetest.RunConformance(t, func(t *testing.T) store.Store {
		return memory.New()
	})
}
