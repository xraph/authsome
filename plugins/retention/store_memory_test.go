package retention

import "testing"

func TestMemoryStoreConformance(t *testing.T) {
	runStoreConformance(t, func(_ *testing.T) Store { return NewMemoryStore() })
}
