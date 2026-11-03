package oauth2provider

import (
	"context"
	"errors"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"github.com/xraph/authsome/internal/telemetry"
)

// spanVerifyClientSecret is the span a confidential client's bcrypt secret
// check runs in. It is the expensive part of the token endpoint.
const spanVerifyClientSecret = "verify client secret"

// grantLabel folds a grant_type into the bounded grant_type label. The
// value arrives unchecked from the request, so anything unknown is "other".
func grantLabel(grantType string) string {
	switch grantType {
	case "authorization_code":
		return telemetry.GrantAuthorizationCode
	case "refresh_token":
		return telemetry.GrantRefreshToken
	case "client_credentials":
		return telemetry.GrantClientCredentials
	case "urn:ietf:params:oauth:grant-type:device_code":
		return telemetry.GrantDeviceCode
	case tokenExchangeGrantType:
		return telemetry.GrantTokenExchange
	default:
		return telemetry.GrantOther
	}
}

// tokenResult sorts the token endpoint's error into the bounded result
// label, by its OAuth2 error code where it carries one and by its HTTP
// status otherwise.
func tokenResult(err error) string {
	if err == nil {
		return telemetry.OAuth2OK
	}
	var oauthErr *OAuth2HTTPError
	if errors.As(err, &oauthErr) {
		switch oauthErr.oauthError {
		case "invalid_client":
			return telemetry.OAuth2InvalidClient
		case "authorization_pending", "slow_down":
			return telemetry.OAuth2Pending
		}
	}
	var coded interface{ StatusCode() int }
	if !errors.As(err, &coded) {
		return telemetry.OAuth2Error
	}
	switch code := coded.StatusCode(); {
	case code == http.StatusUnauthorized:
		return telemetry.OAuth2InvalidClient
	case code >= 400 && code < 500:
		return telemetry.OAuth2Rejected
	default:
		return telemetry.OAuth2Error
	}
}

// compareClientSecret is bcrypt.CompareHashAndPassword inside a verify
// client secret span. The secret, the client ID and the outcome stay off
// the span: a mismatch is answered by the token endpoint's own status.
func compareClientSecret(ctx context.Context, hash, secret string) error {
	_, span := telemetry.Start(ctx, spanVerifyClientSecret)
	defer span.End()
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret))
}
