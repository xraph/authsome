package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Service handles OIDC operations
type Service struct {
	httpClient *http.Client
	jwksCache  map[string]*CachedJWKS
	cacheMutex sync.RWMutex
}

// NewService creates a new OIDC service
func NewService() *Service {
	return &Service{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		jwksCache: make(map[string]*CachedJWKS),
	}
}

// JWKS represents a JSON Web Key Set
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kty string `json:"kty"` // Key Type
	Use string `json:"use,omitempty"` // Public Key Use
	Kid string `json:"kid,omitempty"` // Key ID
	Alg string `json:"alg,omitempty"` // Algorithm
	N   string `json:"n,omitempty"`   // RSA modulus
	E   string `json:"e,omitempty"`   // RSA exponent
	X   string `json:"x,omitempty"`   // EC/OKP x coordinate
	Y   string `json:"y,omitempty"`   // EC y coordinate
	Crv string `json:"crv,omitempty"` // EC curve / OKP subtype
}

// CachedJWKS represents cached JWKS with expiration
type CachedJWKS struct {
	JWKS      *JWKS
	ExpiresAt time.Time
}

// OIDCTokenResponse represents the response from token endpoint
type OIDCTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// OIDCUserInfo represents user information from userinfo endpoint
type OIDCUserInfo struct {
	Sub               string `json:"sub"`
	Name              string `json:"name,omitempty"`
	GivenName         string `json:"given_name,omitempty"`
	FamilyName        string `json:"family_name,omitempty"`
	Email             string `json:"email,omitempty"`
	EmailVerified     bool   `json:"email_verified,omitempty"`
	Picture           string `json:"picture,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty"`
}

// PKCEChallenge represents PKCE challenge data
type PKCEChallenge struct {
	CodeVerifier  string
	CodeChallenge string
	Method        string
}

// GeneratePKCEChallenge generates a PKCE challenge for OAuth2 flow
func (s *Service) GeneratePKCEChallenge() (*PKCEChallenge, error) {
	// Generate code verifier (43-128 characters)
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)
	
	// Generate code challenge using SHA256
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])
	
	return &PKCEChallenge{
		CodeVerifier:  codeVerifier,
		CodeChallenge: codeChallenge,
		Method:        "S256",
	}, nil
}

// ExchangeCodeForTokens exchanges authorization code for tokens
func (s *Service) ExchangeCodeForTokens(ctx context.Context, tokenEndpoint, clientID, clientSecret, code, redirectURI, codeVerifier string) (*OIDCTokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", clientID)
	if clientSecret != "" {
		data.Set("client_secret", clientSecret)
	}
	if codeVerifier != "" {
		data.Set("code_verifier", codeVerifier)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp OIDCTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenResp, nil
}

// FetchJWKS fetches JWKS from the given URL with caching
func (s *Service) FetchJWKS(ctx context.Context, jwksURL string) (*JWKS, error) {
	s.cacheMutex.RLock()
	if cached, exists := s.jwksCache[jwksURL]; exists && time.Now().Before(cached.ExpiresAt) {
		s.cacheMutex.RUnlock()
		return cached.JWKS, nil
	}
	s.cacheMutex.RUnlock()

	req, err := http.NewRequestWithContext(ctx, "GET", jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWKS request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS fetch failed with status %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	// Cache for 1 hour
	s.cacheMutex.Lock()
	s.jwksCache[jwksURL] = &CachedJWKS{
		JWKS:      &jwks,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	s.cacheMutex.Unlock()

	return &jwks, nil
}

// GetPublicKeyFromJWK converts a JWK to a public key for JWT verification
func (s *Service) GetPublicKeyFromJWK(jwk *JWK) (interface{}, error) {
	switch jwk.Kty {
	case "RSA":
		// Decode RSA modulus and exponent
		nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
		if err != nil {
			return nil, fmt.Errorf("failed to decode RSA modulus: %w", err)
		}
		
		eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
		if err != nil {
			return nil, fmt.Errorf("failed to decode RSA exponent: %w", err)
		}

		// Convert to big integers
		n := new(big.Int).SetBytes(nBytes)
		e := new(big.Int).SetBytes(eBytes)

		// Create RSA public key
		return &rsa.PublicKey{
			N: n,
			E: int(e.Int64()),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported key type: %s", jwk.Kty)
	}
}

// ValidateIDTokenWithJWKS validates an ID token using remote JWKS
func (s *Service) ValidateIDTokenWithJWKS(ctx context.Context, idToken, jwksURL, expectedIssuer, expectedAudience, expectedNonce string) (*jwt.MapClaims, error) {
	// Parse token to get header
	token, err := jwt.Parse(idToken, func(token *jwt.Token) (interface{}, error) {
		// Get key ID from token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing kid in token header")
		}

		// Fetch JWKS
		jwks, err := s.FetchJWKS(ctx, jwksURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
		}

		// Find matching key
		for _, jwk := range jwks.Keys {
			if jwk.Kid == kid {
				return s.GetPublicKeyFromJWK(&jwk)
			}
		}

		return nil, fmt.Errorf("key with kid %s not found in JWKS", kid)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to validate token signature: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Validate issuer
	if iss, ok := claims["iss"].(string); !ok || iss != expectedIssuer {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %v", expectedIssuer, claims["iss"])
	}

	// Validate audience
	if aud, ok := claims["aud"].(string); !ok || aud != expectedAudience {
		// Check if audience is an array
		if audArray, ok := claims["aud"].([]interface{}); ok {
			found := false
			for _, a := range audArray {
				if audStr, ok := a.(string); ok && audStr == expectedAudience {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("invalid audience: expected %s not found in %v", expectedAudience, claims["aud"])
			}
		} else {
			return nil, fmt.Errorf("invalid audience: expected %s, got %v", expectedAudience, claims["aud"])
		}
	}

	// Validate nonce if provided
	if expectedNonce != "" {
		if nonce, ok := claims["nonce"].(string); !ok || nonce != expectedNonce {
			return nil, fmt.Errorf("invalid nonce: expected %s, got %v", expectedNonce, claims["nonce"])
		}
	}

	// Validate expiration
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return nil, fmt.Errorf("token expired")
		}
	} else {
		return nil, fmt.Errorf("missing or invalid exp claim")
	}

	// Validate issued at time
	if iat, ok := claims["iat"].(float64); ok {
		if time.Now().Unix() < int64(iat) {
			return nil, fmt.Errorf("token used before issued")
		}
	} else {
		return nil, fmt.Errorf("missing or invalid iat claim")
	}

	return &claims, nil
}

// GetUserInfo fetches user information from the userinfo endpoint
func (s *Service) GetUserInfo(ctx context.Context, userinfoEndpoint, accessToken string) (*OIDCUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", userinfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create userinfo request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("userinfo request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var userInfo OIDCUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode userinfo response: %w", err)
	}
	
	return &userInfo, nil
}

// ValidateIDToken validates an OIDC ID token using JWKS for signature verification
func (s *Service) ValidateIDToken(ctx context.Context, tokenString, jwksURL, issuer, clientID, nonce string) (*jwt.MapClaims, error) {
	return s.ValidateIDTokenWithJWKS(ctx, tokenString, jwksURL, issuer, clientID, nonce)
}

// RefreshTokens refreshes access tokens using a refresh token
func (s *Service) RefreshTokens(ctx context.Context, tokenEndpoint, clientID, clientSecret, refreshToken string) (*OIDCTokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", clientID)
	
	req, err := http.NewRequestWithContext(ctx, "POST", tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	
	if clientSecret != "" {
		req.SetBasicAuth(clientID, clientSecret)
	}
	
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("refresh failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var tokenResp OIDCTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode refresh response: %w", err)
	}
	
	return &tokenResp, nil
}