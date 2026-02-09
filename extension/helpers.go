package extension

import (
	"github.com/xraph/authsome"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// ResolveExtension resolves the AuthSome extension from a Forge app
// This allows you to access the extension instance after registration.
func ResolveExtension(app forge.App) (*Extension, error) {
	// Extensions are typically stored in the app, get it from extensions list
	exts := app.Extensions()
	for _, ext := range exts {
		if authExt, ok := ext.(*Extension); ok {
			return authExt, nil
		}
	}

	return nil, errs.NotFound("authsome extension not registered")
}

// ResolveAuth resolves the AuthSome instance from a Forge app
// Note: This only works after the app has been started.
func ResolveAuth(app forge.App) (*authsome.Auth, error) {
	ext, err := ResolveExtension(app)
	if err != nil {
		return nil, err
	}

	auth := ext.Auth()
	if auth == nil {
		return nil, errs.InternalServerErrorWithMessage("authsome not initialized yet - call app.Start() first")
	}

	return auth, nil
}
