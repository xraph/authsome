// Example: AuthSome as a Forge extension with all optional bridges.
//
// This demonstrates how AuthSome plugs into a Forge application, auto-discovering
// optional dependencies (Chronicle, Warden, Keysmith, Relay) from the DI container.
//
// Run:
//
//	go run ./_examples/forge/
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/xraph/forge"

	authext "github.com/xraph/authsome/extension"
	"github.com/xraph/authsome/plugins/mfa"
	"github.com/xraph/authsome/plugins/password"
	"github.com/xraph/authsome/plugins/social"
)

func main() {
	// AuthSome extension with plugins.
	// When running as a Forge extension, the logger is provided
	// automatically from the DI container.
	auth := authext.New(
		// Plugins control which authentication strategies are available.
		authext.WithPlugins(
			password.New(),
			social.New(
			// social.WithGitHub("client-id", "client-secret", "http://localhost:8080/v1/auth/social/github/callback"),
			// social.WithGoogle("client-id", "client-secret", "http://localhost:8080/v1/auth/social/google/callback"),
			),
			mfa.New(),
		),

		// Engine options can be passed through.
		// authext.WithEngineOptions(
		//     authsome.WithBasePath("/api/auth"),
		//     authsome.WithAppID("aapp_01jf0000000000000000000000"),
		// ),
	)

	// Build the Forge application.
	// When a bun.DB, chronicle.Emitter, warden.Engine, keysmith.Engine, or relay.Relay
	// is registered in the DI container (by other extensions), AuthSome auto-discovers
	// and wires them up automatically.
	app := forge.New(
		forge.WithExtensions(
			// Register other forgery extensions first so their types are
			// available in the container when AuthSome initializes:
			//
			// chronicleext.New(...),
			// wardenext.New(...),
			// keysmithext.New(...),
			// relayext.New(...),

			auth,
		),
	)

	if err := app.Start(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}
