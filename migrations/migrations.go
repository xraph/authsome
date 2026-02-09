package migrations

import (
	forgemigrate "github.com/xraph/forge/extensions/database/migrate"
)

// Migrations is the global migration registry
// Now using Forge's database extension migration registry for better integration.
var Migrations = forgemigrate.Migrations
