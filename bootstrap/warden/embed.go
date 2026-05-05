// Package wardenseed embeds authsome's default warden DSL programs and
// provides loaders + appliers that materialise them against a warden engine
// at bootstrap time.
//
// The embedded files under embed/ are the source of truth for authsome's
// default roles and permission catalog. They're loaded with ${APP_ID}
// variable substitution so the same DSL applies to every app the engine
// bootstraps.
package wardenseed

import "embed"

// FS is the embedded filesystem containing authsome's default .warden
// files. The layout is:
//
//	embed/shared/    — applied to every app (catalog + namespace "app" roles)
//	embed/platform/  — applied only to the platform app (namespace "platform" roles)
//
//go:embed embed
var FS embed.FS

const (
	// embedSharedRoot is the FS root that gets applied to every app.
	embedSharedRoot = "embed/shared"
	// embedPlatformRoot is the FS root that gets applied only to the platform app.
	embedPlatformRoot = "embed/platform"
)
