//go:build integration

package mongo_test

import (
	"testing"

	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/store/storetest"
)

// TestConformance runs the shared cross-backend store contract suite against
// MongoDB. One store is shared across the sub-tests (they isolate with random
// ids). Skips unless AUTHSOME_MONGO_URI is set; run with `-tags integration`.
func TestConformance(t *testing.T) {
	s := setupTestStore(t)
	storetest.RunConformance(t, func(_ *testing.T) store.Store { return s })
}
