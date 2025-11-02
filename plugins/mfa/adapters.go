package mfa

import (
	"context"
	"fmt"

	"github.com/rs/xid"
)

// FactorAdapter defines the interface for integrating authentication factors
// Each adapter wraps an existing plugin (twofa, emailotp, phone, passkey)
type FactorAdapter interface {
	// Type returns the factor type this adapter handles
	Type() FactorType

	// Enroll initiates factor enrollment for a user
	// Returns provisioning data needed to complete enrollment
	Enroll(ctx context.Context, userID xid.ID, metadata map[string]any) (*FactorEnrollmentResponse, error)

	// VerifyEnrollment verifies the enrollment (e.g., user scanned QR code and provides first TOTP)
	VerifyEnrollment(ctx context.Context, enrollmentID xid.ID, proof string) error

	// Challenge initiates a verification challenge (sends code, displays options, etc.)
	Challenge(ctx context.Context, factor *Factor, metadata map[string]any) (*Challenge, error)

	// Verify verifies the challenge response
	Verify(ctx context.Context, challenge *Challenge, response string, data map[string]any) (bool, error)

	// IsAvailable checks if this factor type is available/configured
	IsAvailable() bool
}

// FactorAdapterRegistry manages available factor adapters
type FactorAdapterRegistry struct {
	adapters map[FactorType]FactorAdapter
}

// NewFactorAdapterRegistry creates a new adapter registry
func NewFactorAdapterRegistry() *FactorAdapterRegistry {
	return &FactorAdapterRegistry{
		adapters: make(map[FactorType]FactorAdapter),
	}
}

// Register registers a factor adapter
func (r *FactorAdapterRegistry) Register(adapter FactorAdapter) {
	r.adapters[adapter.Type()] = adapter
}

// Get retrieves a factor adapter by type
func (r *FactorAdapterRegistry) Get(factorType FactorType) (FactorAdapter, error) {
	adapter, ok := r.adapters[factorType]
	if !ok {
		return nil, fmt.Errorf("no adapter registered for factor type: %s", factorType)
	}
	return adapter, nil
}

// List returns all available factor types
func (r *FactorAdapterRegistry) List() []FactorType {
	types := make([]FactorType, 0, len(r.adapters))
	for t := range r.adapters {
		types = append(types, t)
	}
	return types
}

// GetAvailable returns only available factor types
func (r *FactorAdapterRegistry) GetAvailable() []FactorType {
	types := make([]FactorType, 0, len(r.adapters))
	for t, adapter := range r.adapters {
		if adapter.IsAvailable() {
			types = append(types, t)
		}
	}
	return types
}

// BaseFactorAdapter provides common functionality for adapters
type BaseFactorAdapter struct {
	factorType FactorType
	available  bool
}

// Type returns the factor type
func (b *BaseFactorAdapter) Type() FactorType {
	return b.factorType
}

// IsAvailable checks if the factor is available
func (b *BaseFactorAdapter) IsAvailable() bool {
	return b.available
}
