package api

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"github.com/xraph/authsome/tokenformat"
)

// TestJwtToJWK_RSAKey verifies RSA key conversion to JWKS format.
func TestJwtToJWK_RSAKey(t *testing.T) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	jwtFmt, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodRS256,
		SigningKey:    privKey,
		VerifyKey:     &privKey.PublicKey,
		KeyID:         "test-rsa-kid",
		Issuer:        "https://example.com",
		Audience:      "api",
	})
	if err != nil {
		t.Fatalf("create JWT format: %v", err)
	}

	jwk := jwtToJWK(jwtFmt)

	if jwk == nil {
		t.Fatal("jwtToJWK returned nil for RSA key")
	}
	if jwk.KTY != "RSA" {
		t.Errorf("KTY = %q, want RSA", jwk.KTY)
	}
	if jwk.Use != "sig" {
		t.Errorf("Use = %q, want sig", jwk.Use)
	}
	if jwk.KID != "test-rsa-kid" {
		t.Errorf("KID = %q, want test-rsa-kid", jwk.KID)
	}
	if jwk.ALG != "RS256" {
		t.Errorf("ALG = %q, want RS256", jwk.ALG)
	}
	if jwk.N == "" {
		t.Error("N (modulus) is empty")
	}
	if jwk.E == "" {
		t.Error("E (exponent) is empty")
	}

	// Verify N and E are valid base64url.
	if _, err := base64.RawURLEncoding.DecodeString(jwk.N); err != nil {
		t.Errorf("N is not valid base64url: %v", err)
	}
	if _, err := base64.RawURLEncoding.DecodeString(jwk.E); err != nil {
		t.Errorf("E is not valid base64url: %v", err)
	}
}

// TestJwtToJWK_ECDSAKey verifies ECDSA P-256 key conversion to JWKS format.
func TestJwtToJWK_ECDSAKey(t *testing.T) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ECDSA key: %v", err)
	}

	jwtFmt, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodES256,
		SigningKey:    privKey,
		VerifyKey:     &privKey.PublicKey,
		KeyID:         "test-ec-kid",
		Issuer:        "https://example.com",
		Audience:      "api",
	})
	if err != nil {
		t.Fatalf("create JWT format: %v", err)
	}

	jwk := jwtToJWK(jwtFmt)

	if jwk == nil {
		t.Fatal("jwtToJWK returned nil for ECDSA key")
	}
	if jwk.KTY != "EC" {
		t.Errorf("KTY = %q, want EC", jwk.KTY)
	}
	if jwk.CRV != "P-256" {
		t.Errorf("CRV = %q, want P-256", jwk.CRV)
	}
	if jwk.Use != "sig" {
		t.Errorf("Use = %q, want sig", jwk.Use)
	}
	if jwk.KID != "test-ec-kid" {
		t.Errorf("KID = %q, want test-ec-kid", jwk.KID)
	}
	if jwk.ALG != "ES256" {
		t.Errorf("ALG = %q, want ES256", jwk.ALG)
	}

	// Verify X and Y are valid base64url and have the correct length.
	xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		t.Errorf("X is not valid base64url: %v", err)
	}
	if len(xBytes) != 32 {
		t.Errorf("X decoded to %d bytes, want 32 (P-256 field size)", len(xBytes))
	}

	yBytes, err := base64.RawURLEncoding.DecodeString(jwk.Y)
	if err != nil {
		t.Errorf("Y is not valid base64url: %v", err)
	}
	if len(yBytes) != 32 {
		t.Errorf("Y decoded to %d bytes, want 32 (P-256 field size)", len(yBytes))
	}
}

// TestJwtToJWK_ECDSAKeyWithoutKID verifies that ECDSA keys without explicit KID
// get a correctly derived thumbprint-based KID computed from the encoded X and Y coordinates.
func TestJwtToJWK_ECDSAKeyWithoutKID(t *testing.T) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ECDSA key: %v", err)
	}

	// Create JWT format without a KeyID.
	jwtFmt, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodES256,
		SigningKey:    privKey,
		VerifyKey:     &privKey.PublicKey,
		Issuer:        "https://example.com",
		Audience:      "api",
	})
	if err != nil {
		t.Fatalf("create JWT format: %v", err)
	}

	jwk := jwtToJWK(jwtFmt)

	if jwk == nil {
		t.Fatal("jwtToJWK returned nil for ECDSA key without KID")
	}
	if jwk.KID == "" {
		t.Error("KID is empty, expected thumbprint-based KID for ECDSA without explicit KID")
	}

	// Verify the KID is a valid base64url string with the correct length.
	kidBytes, err := base64.RawURLEncoding.DecodeString(jwk.KID)
	if err != nil {
		t.Errorf("KID is not valid base64url: %v", err)
	}
	if len(kidBytes) != 8 {
		t.Errorf("KID decoded to %d bytes, want 8 (first 8 bytes of SHA256)", len(kidBytes))
	}

	// Verify the KID is derived correctly: SHA256(X + Y) truncated to 8 bytes, base64url encoded.
	// This recomputes the expectation from the JWK's own X and Y values (the encoded coordinates).
	expectedHash := sha256.Sum256([]byte(jwk.X + jwk.Y))
	expectedKID := base64.RawURLEncoding.EncodeToString(expectedHash[:8])
	if jwk.KID != expectedKID {
		t.Errorf("KID = %q, want %q (SHA256(X + Y)[:8])", jwk.KID, expectedKID)
	}
}

// TestJwtToJWK_ECDSAKeyKIDDeterminism verifies that the same ECDSA key encoded
// twice yields the same thumbprint-based KID.
func TestJwtToJWK_ECDSAKeyKIDDeterminism(t *testing.T) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ECDSA key: %v", err)
	}

	// Create JWT format and encode the key twice.
	jwtFmt, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodES256,
		SigningKey:    privKey,
		VerifyKey:     &privKey.PublicKey,
		Issuer:        "https://example.com",
		Audience:      "api",
	})
	if err != nil {
		t.Fatalf("create JWT format: %v", err)
	}

	jwk1 := jwtToJWK(jwtFmt)
	jwk2 := jwtToJWK(jwtFmt)

	if jwk1 == nil || jwk2 == nil {
		t.Fatal("jwtToJWK returned nil")
	}
	if jwk1.KID != jwk2.KID {
		t.Errorf("KID is not deterministic: first %q, second %q", jwk1.KID, jwk2.KID)
	}
	if jwk1.KID == "" {
		t.Error("KID is empty")
	}
}

// TestJwtToJWK_ECDSAKeyKIDUniqueness verifies that two different ECDSA keys
// receive different thumbprint-based KIDs.
func TestJwtToJWK_ECDSAKeyKIDUniqueness(t *testing.T) {
	privKey1, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate first ECDSA key: %v", err)
	}
	privKey2, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate second ECDSA key: %v", err)
	}

	jwtFmt1, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodES256,
		SigningKey:    privKey1,
		VerifyKey:     &privKey1.PublicKey,
		Issuer:        "https://example.com",
		Audience:      "api",
	})
	if err != nil {
		t.Fatalf("create first JWT format: %v", err)
	}

	jwtFmt2, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodES256,
		SigningKey:    privKey2,
		VerifyKey:     &privKey2.PublicKey,
		Issuer:        "https://example.com",
		Audience:      "api",
	})
	if err != nil {
		t.Fatalf("create second JWT format: %v", err)
	}

	jwk1 := jwtToJWK(jwtFmt1)
	jwk2 := jwtToJWK(jwtFmt2)

	if jwk1 == nil || jwk2 == nil {
		t.Fatal("jwtToJWK returned nil")
	}
	if jwk1.KID == "" || jwk2.KID == "" {
		t.Error("KID is empty")
	}
	if jwk1.KID == jwk2.KID {
		t.Errorf("KID is not unique: both keys have KID %q", jwk1.KID)
	}
}

// TestJwtToJWK_HMACKey verifies that HMAC keys are not published in JWKS.
func TestJwtToJWK_HMACKey(t *testing.T) {
	hmacKey := []byte("secret-key-for-testing-hmac")

	jwtFmt, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		SigningKey:    hmacKey,
		VerifyKey:     hmacKey,
		KeyID:         "hmac-kid",
		Issuer:        "https://example.com",
		Audience:      "api",
	})
	if err != nil {
		t.Fatalf("create JWT format: %v", err)
	}

	jwk := jwtToJWK(jwtFmt)

	if jwk != nil {
		t.Errorf("jwtToJWK returned %v for HMAC key, want nil (symmetric keys must never be published in JWKS)", jwk)
	}
}
