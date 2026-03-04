package api

import (
	"errors"
	"net/http"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/store"
)

// mapError converts domain errors into Forge HTTP errors.
func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, store.ErrNotFound) {
		return forge.NotFound(err.Error())
	}
	if errors.Is(err, account.ErrInvalidCredentials) {
		return forge.Unauthorized("invalid credentials")
	}
	if errors.Is(err, account.ErrEmailTaken) {
		return forge.NewHTTPError(http.StatusConflict, "email already taken")
	}
	if errors.Is(err, account.ErrUsernameTaken) {
		return forge.NewHTTPError(http.StatusConflict, "username already taken")
	}
	if errors.Is(err, account.ErrUserBanned) {
		return forge.Forbidden("user is banned")
	}
	if errors.Is(err, account.ErrSessionExpired) {
		return forge.Unauthorized("session expired")
	}
	if errors.Is(err, account.ErrWeakPassword) {
		return forge.BadRequest(err.Error())
	}
	return forge.InternalError(err)
}
