//go:build integration

package postgres_test

import (
	"testing"

	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/store/storetest"
)

// TestConformance runs the shared cross-backend store contract suite against
// PostgreSQL (via testcontainers). One store is shared across the sub-tests —
// they isolate themselves with random ids — to avoid spinning a container per
// case. Requires Docker; run with `-tags integration`.
func TestConformance(t *testing.T) {
	s := setupTestStore(t)
	storetest.RunConformance(t, func(_ *testing.T) store.Store { return s })
}
