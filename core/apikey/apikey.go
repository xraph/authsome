package apikey

import (
	"github.com/xraph/authsome/core/base"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// APIKey represents an API key with its metadata (DTO)
// Updated for V2 architecture: App → Environment → Organization
type APIKey = base.APIKey

// FromSchemaAPIKey converts a schema.APIKey to APIKey DTO
func FromSchemaAPIKey(s *schema.APIKey) *APIKey {
	return base.FromSchemaAPIKey(s)
}

// FromSchemaAPIKeys converts multiple schema.APIKey to APIKey DTOs
func FromSchemaAPIKeys(keys []*schema.APIKey) []*APIKey {
	return base.FromSchemaAPIKeys(keys)
}

// CreateAPIKeyRequest represents a request to create an API key
// Updated for V2 architecture
type CreateAPIKeyRequest = base.CreateAPIKeyRequest

// UpdateAPIKeyRequest represents a request to update an API key
type UpdateAPIKeyRequest = base.UpdateAPIKeyRequest

// ListAPIKeysResponse is a type alias for the paginated response
type ListAPIKeysResponse = pagination.PageResponse[*APIKey]

// RotateAPIKeyRequest represents a request to rotate an API key
// Updated for V2 architecture
type RotateAPIKeyRequest = base.RotateAPIKeyRequest

// VerifyAPIKeyRequest represents a request to verify an API key
type VerifyAPIKeyRequest = base.VerifyAPIKeyRequest

// VerifyAPIKeyResponse represents a response from API key verification
type VerifyAPIKeyResponse = base.VerifyAPIKeyResponse
