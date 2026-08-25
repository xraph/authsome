// Package dpop implements RFC 9449 Demonstrating Proof of Possession.
//
// A DPoP proof is a short-lived JWT signed by a key the client holds. Binding
// an issued token to that key's thumbprint means a token on its own is not
// enough to use it, which is the property bearer tokens lack.
package dpop

import "errors"

// Sentinel errors. Callers map these onto the RFC 9449 wire codes: everything
// here becomes invalid_dpop_proof at the token endpoint and invalid_token at a
// protected resource, except ErrNonceRequired which becomes use_dpop_nonce.
var (
	ErrMalformedProof = errors.New("dpop: malformed proof")
	ErrUnsupportedAlg = errors.New("dpop: unsupported proof algorithm")
	ErrBadSignature   = errors.New("dpop: proof signature invalid")
	ErrMethodMismatch = errors.New("dpop: htm does not match request method")
	ErrURIMismatch    = errors.New("dpop: htu does not match request URI")
	ErrStaleProof     = errors.New("dpop: proof iat outside acceptable window")
	ErrNonceRequired  = errors.New("dpop: nonce required")
	ErrNonceMismatch  = errors.New("dpop: nonce mismatch")
	ErrATHMismatch    = errors.New("dpop: ath does not match presented token")
	ErrKeyMismatch    = errors.New("dpop: proof key does not match bound token")
	ErrReplayed       = errors.New("dpop: proof jti replayed")
)
