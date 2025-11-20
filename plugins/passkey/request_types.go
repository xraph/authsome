package passkey

// Request types for passkey plugin with proper validation tags

// BeginRegisterRequest initiates passkey registration
type BeginRegisterRequest struct {
	UserID              string `json:"userId" validate:"required,xid"`
	Name                string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	AuthenticatorType   string `json:"authenticatorType,omitempty" validate:"omitempty,oneof=platform cross-platform"`
	RequireResidentKey  bool   `json:"requireResidentKey"`
	UserVerification    string `json:"userVerification,omitempty" validate:"omitempty,oneof=required preferred discouraged"`
}

// FinishRegisterRequest completes passkey registration with credential attestation
type FinishRegisterRequest struct {
	UserID   string                 `json:"userId" validate:"required,xid"`
	Name     string                 `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Response map[string]interface{} `json:"response" validate:"required"` // WebAuthn PublicKeyCredential
}

// BeginLoginRequest initiates passkey authentication
type BeginLoginRequest struct {
	UserID           string `json:"userId,omitempty" validate:"omitempty,xid"` // Optional for discoverable credentials
	UserVerification string `json:"userVerification,omitempty" validate:"omitempty,oneof=required preferred discouraged"`
}

// FinishLoginRequest completes passkey authentication
type FinishLoginRequest struct {
	Response map[string]interface{} `json:"response" validate:"required"` // WebAuthn PublicKeyCredential assertion
	Remember bool                   `json:"remember"`
}

// ListPasskeysRequest retrieves user's passkeys
type ListPasskeysRequest struct {
	UserID string `query:"userId" validate:"required,xid"`
}

// UpdatePasskeyRequest updates passkey metadata (name)
type UpdatePasskeyRequest struct {
	ID   string `path:"id" validate:"required,xid"`
	Name string `json:"name" validate:"required,min=1,max=100"`
}

// DeletePasskeyRequest deletes a passkey
type DeletePasskeyRequest struct {
	ID string `path:"id" validate:"required,xid"`
}

// GetPasskeyRequest retrieves a single passkey
type GetPasskeyRequest struct {
	ID string `path:"id" validate:"required,xid"`
}

