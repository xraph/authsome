package migrations

import (
	"github.com/uptrace/bun/migrate"
)

// Migrations is the global migration registry
var Migrations = migrate.NewMigrations()