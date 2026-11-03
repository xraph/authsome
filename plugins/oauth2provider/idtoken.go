package oauth2provider

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/golang-jwt/jwt/v5"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/user"
)

// oidcRequest carries what the authorization request bound for the id_token.
// A grant that passes nil to issueTokens never gets an id_token, whatever its
// scopes: client credentials has no user and token exchange is not an OpenID
// Connect login.
type oidcRequest struct {
	Nonce string
	// AuthTime is when the user signed in. Zero leaves auth_time out.
	AuthTime time.Time
}

// idTokenInput is everything one id_token is built from.
type idTokenInput struct {
	Client      *OAuth2Client
	User        *user.User
	Scopes      []string
	AccessToken string
	Nonce       string
	// AuthTime is when the user signed in. Zero leaves auth_time out.
	AuthTime time.Time
}

// idTokenClaims is the id_token payload (OpenID Connect Core 2): the
// registered envelope, the standard claims the scopes cover, and the
// OpenID-specific members.
type idTokenClaims struct {
	jwt.RegisteredClaims
	StandardClaims
	Nonce    string           `json:"nonce,omitempty"`
	AuthTime *jwt.NumericDate `json:"auth_time,omitempty"`
	AtHash   string           `json:"at_hash"`
	Azp      string           `json:"azp"`
}

// idTokenFor mints the id_token for a user grant, or returns "" when the
// grant gets none: oidc is nil, or openid is not among the granted scopes.
func (p *Plugin) idTokenFor(ctx context.Context, client *OAuth2Client, userID id.UserID, scopes []string, accessToken string, oidc *oidcRequest) (string, error) {
	if oidc == nil || !slices.Contains(scopes, "openid") {
		return "", nil
	}
	u, err := p.store.GetUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("oauth2: load id_token subject: %w", err)
	}
	return p.mintIDToken(ctx, idTokenInput{
		Client:      client,
		User:        u,
		Scopes:      scopes,
		AccessToken: accessToken,
		Nonce:       oidc.Nonce,
		AuthTime:    oidc.AuthTime,
	})
}

// mintIDToken signs an id_token for in with the active key of the client's
// issuer.
func (p *Plugin) mintIDToken(ctx context.Context, in idTokenInput) (string, error) {
	if p.keys == nil {
		return "", errNoSigningKey
	}
	issuer, scope, err := p.issuerFor(ctx, in.Client.AppID)
	if err != nil {
		return "", err
	}
	s, err := p.keys.Active(ctx, scope)
	if err != nil {
		return "", err
	}
	hash, err := atHash(in.AccessToken, s.alg)
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := idTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   in.User.ID.String(),
			Audience:  jwt.ClaimStrings{in.Client.ClientID},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(p.config.IDTokenTTL)),
		},
		StandardClaims: standardClaims(in.User, func(scope string) bool {
			return slices.Contains(in.Scopes, scope)
		}),
		Nonce:  in.Nonce,
		AtHash: hash,
		Azp:    in.Client.ClientID,
	}
	if !in.AuthTime.IsZero() {
		claims.AuthTime = jwt.NewNumericDate(in.AuthTime)
	}
	return signIDToken(s, claims)
}

// signIDToken signs claims with s. The key's concrete type has to match the
// alg: golang-jwt then produces the JWS form of the signature, which for
// ES256 is the raw 64-byte r||s rather than the ASN.1 DER crypto.Signer
// returns.
func signIDToken(s *signer, claims jwt.Claims) (string, error) {
	var (
		method jwt.SigningMethod
		key    any
	)
	switch s.alg {
	case "RS256":
		k, ok := s.key.(*rsa.PrivateKey)
		if !ok {
			return "", fmt.Errorf("oauth2: RS256 signing key %s is a %T", s.kid, s.key)
		}
		method, key = jwt.SigningMethodRS256, k
	case "ES256":
		k, ok := s.key.(*ecdsa.PrivateKey)
		if !ok {
			return "", fmt.Errorf("oauth2: ES256 signing key %s is a %T", s.kid, s.key)
		}
		method, key = jwt.SigningMethodES256, k
	default:
		return "", fmt.Errorf("oauth2: unsupported id_token alg %q", s.alg)
	}
	tok := jwt.NewWithClaims(method, claims)
	tok.Header["kid"] = s.kid
	tok.Header["typ"] = "JWT"
	return tok.SignedString(key)
}

// atHash is the access token hash (OpenID Connect Core 3.1.3.6): the left
// half of the token's hash under the hash function of the id_token's alg,
// base64url encoded. An alg it does not know is an error rather than a
// guess, since a wrong hash is a token no relying party will accept.
func atHash(accessToken, alg string) (string, error) {
	switch alg {
	case "RS256", "ES256":
		sum := sha256.Sum256([]byte(accessToken))
		return base64.RawURLEncoding.EncodeToString(sum[:len(sum)/2]), nil
	default:
		return "", fmt.Errorf("oauth2: no at_hash function for alg %q", alg)
	}
}

// idTokenServerError logs why an id_token could not be minted and returns the
// server_error the client sees. The cause stays in the log.
func (p *Plugin) idTokenServerError(err error) error {
	p.logger.Error("oauth2: mint id_token", log.Error(err))
	return newOAuth2Error(http.StatusInternalServerError, "server_error", "the id_token could not be issued")
}
