// Package mongo implements the AuthSome store interface using MongoDB via Grove ORM.
package mongo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/mongodriver"
	"github.com/xraph/grove/migrate"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/appsessionconfig"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/store"
)

// Collection name constants.
const (
	colApps              = "authsome_apps"
	colUsers             = "authsome_users"
	colSessions          = "authsome_sessions"
	colVerifications     = "authsome_verifications"
	colPasswordResets    = "authsome_password_resets"
	colOrganizations     = "authsome_organizations"
	colMembers           = "authsome_members"
	colInvitations       = "authsome_invitations"
	colTeams             = "authsome_teams"
	colDevices           = "authsome_devices"
	colWebhooks          = "authsome_webhooks"
	colNotifications     = "authsome_notifications"
	colAPIKeys              = "authsome_api_keys" //nolint:gosec // G101: not a credential
	colEnvironments         = "authsome_environments"
	colFormConfigs          = "authsome_form_configs"
	colBrandingConfigs      = "authsome_branding_configs"
	colAppSessionConfigs    = "authsome_app_session_configs"
	colRevokedRefreshTokens = "authsome_revoked_refresh_tokens" //nolint:gosec // G101: not a credential
	colServiceAccounts      = "authsome_service_accounts"
)

// Compile-time interface checks.
var (
	_ store.Store            = (*Store)(nil)
	_ environment.Store      = (*Store)(nil)
	_ appsessionconfig.Store = (*Store)(nil)
)

// Store implements store.Store using MongoDB via Grove ORM.
type Store struct {
	db  *grove.DB
	mdb *mongodriver.MongoDB
}

// New creates a new MongoDB store backed by Grove ORM.
func New(db *grove.DB) *Store {
	return &Store{
		db:  db,
		mdb: mongodriver.Unwrap(db),
	}
}

// DB returns the underlying grove database for direct access.
func (s *Store) DB() *grove.DB { return s.db }

// Migrate runs the mongo migration orchestrator (the authoritative
// path for index/schema changes — every entry in Migrations runs
// exactly once and is tracked) and then ensures baseline indexes
// exist for any collection that doesn't yet have them.
//
// The two paths coexist deliberately:
//   - The orchestrator is the only path that can RESHAPE an existing
//     index (drop + recreate with different options). Repair migrations
//     like fix_username_index_partial_filter live here.
//   - The eager ensureBaselineIndexes path is the only path that runs
//     when an operator boots with WithDisableMigrate() (used in tests
//     and some custom-control deployments). It tolerates index-shape
//     conflicts so a stale index from a prior deployment doesn't block
//     boot — the migration system is expected to repair it.
//
// extraGroups are concatenated after the mongo Migrations group so
// plugins can register their own migration groups via the standard
// MigrationProvider interface (mirrors the postgres + sqlite stores).
func (s *Store) Migrate(ctx context.Context, extraGroups ...*migrate.Group) error {
	groups := append([]*migrate.Group{Migrations}, extraGroups...)

	if err := s.runMigrationsWithSelfHeal(ctx, groups); err != nil {
		return err
	}

	return s.ensureBaselineIndexes(ctx)
}

// runMigrationsWithSelfHeal runs the orchestrator. If the run fails
// with "cannot decode objectID into an integer type" (an upstream
// inconsistency in grove/mongodriver/mongomigrate where RecordApplied
// writes ObjectID _ids but ListApplied decodes _id as int64), the
// grove_migrations collection is unreadable and no further progress
// can be made.
//
// Recovery: drop the grove_migrations + grove_migration_locks
// collections so the orchestrator treats nothing as applied, then
// retry once. Every migration in Migrations is idempotent
// (CreateCollection no-ops if present; CreateIndexes returns
// IndexKeySpecsConflict for shape conflicts which ensureBaselineIndexes
// already swallows; the fix_username_index_partial_filter repair
// tolerates IndexNotFound on its drop step) so the re-run is safe.
//
// We log a warning at recovery time so operators know history was
// reset; if any non-idempotent plugin migration was previously
// applied, the operator must investigate via the audit log.
func (s *Store) runMigrationsWithSelfHeal(ctx context.Context, groups []*migrate.Group) error {
	executor, err := migrate.NewExecutorFor(s.mdb)
	if err != nil {
		return fmt.Errorf("authsome/mongo: create migration executor: %w", err)
	}

	orch := migrate.NewOrchestrator(executor, groups...)
	if _, err := orch.Migrate(ctx); err != nil {
		if !mongoIsMigrationDecodeCorruption(err) {
			return fmt.Errorf("authsome/mongo: migration failed: %w", err)
		}

		// Self-heal: the grove_migrations collection has incompatible
		// docs (ObjectID _ids vs the library's int64 expectation).
		// Drop the tracking collections so a fresh orchestrator run
		// can rebuild them.
		if dropErr := s.resetMigrationTracking(ctx); dropErr != nil {
			return fmt.Errorf("authsome/mongo: migration tracking corrupted (%w) and reset failed: %v", err, dropErr)
		}

		// Build a fresh executor + orchestrator after the reset and
		// retry once.
		retryExec, retryErr := migrate.NewExecutorFor(s.mdb)
		if retryErr != nil {
			return fmt.Errorf("authsome/mongo: rebuild executor after tracking reset: %w", retryErr)
		}
		retryOrch := migrate.NewOrchestrator(retryExec, groups...)
		if _, err := retryOrch.Migrate(ctx); err != nil {
			return fmt.Errorf("authsome/mongo: migration retry after tracking reset failed: %w", err)
		}
	}
	return nil
}

// resetMigrationTracking drops the grove_migrations and
// grove_migration_locks collections used by mongomigrate. Called as
// a one-shot recovery from runMigrationsWithSelfHeal when the
// existing tracking docs can't be decoded.
//
// Tolerates collection-not-found so the helper is safe to invoke on
// a deployment that never had the tracking collections at all.
func (s *Store) resetMigrationTracking(ctx context.Context) error {
	for _, col := range []string{"grove_migrations", "grove_migration_locks"} {
		if err := s.mdb.Collection(col).Drop(ctx); err != nil {
			// NamespaceNotFound (code 26) means the collection didn't
			// exist — that's fine.
			var cmdErr mongo.CommandError
			if errors.As(err, &cmdErr) && cmdErr.Code == 26 {
				continue
			}
			if strings.Contains(err.Error(), "ns not found") {
				continue
			}
			return fmt.Errorf("drop %s: %w", col, err)
		}
	}
	return nil
}

// createIndexesReshapeConflicting calls CreateMany for the supplied
// models. On IndexKeySpecsConflict it derives the conflicting index
// name from the error message (Mongo includes "name: \"<name>\"" in
// the IndexKeySpecsConflict text), drops that single index, and
// retries CreateMany once. Other errors are returned unchanged.
//
// Used by migrations that re-shape an existing index after the
// authoritative shape changed — most commonly when the migration
// orchestrator self-heals from grove_migrations corruption (see
// runMigrationsWithSelfHeal) and re-runs create_authsome_users
// against a deployment that already had the OLD-shape sparse index.
func createIndexesReshapeConflicting(ctx context.Context, coll *mongo.Collection, models []mongo.IndexModel) error {
	if len(models) == 0 {
		return nil
	}
	_, err := coll.Indexes().CreateMany(ctx, models)
	if err == nil {
		return nil
	}
	if !mongoIsIndexConflict(err) {
		return err
	}
	conflicting := mongoExtractConflictingIndexName(err)
	if conflicting == "" {
		// Couldn't parse the name out — return the original error so
		// operators see what went wrong instead of silently swallowing.
		return err
	}
	if dropErr := coll.Indexes().DropOne(ctx, conflicting); dropErr != nil {
		if !mongoIsIndexNotFound(dropErr) {
			return fmt.Errorf("drop conflicting index %q: %w (original: %v)", conflicting, dropErr, err)
		}
	}
	if _, retryErr := coll.Indexes().CreateMany(ctx, models); retryErr != nil {
		return fmt.Errorf("recreate after drop %q: %w", conflicting, retryErr)
	}
	return nil
}

// mongoExtractConflictingIndexName parses the Mongo
// IndexKeySpecsConflict message to recover the auto-derived name
// of the existing index that blocks the create. Mongo's message is:
//
//	An existing index has the same name as the requested index. ...
//	existing index: { v: 2, unique: true, key: {...}, name: "<name>", sparse: true }
//
// Best-effort substring match — returns empty when the format
// changes (newer driver versions, locale variants).
func mongoExtractConflictingIndexName(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	// Look for `existing index:` then the first `name: "<...>"` after it.
	const sentinel = "existing index:"
	idx := strings.Index(msg, sentinel)
	if idx < 0 {
		return ""
	}
	tail := msg[idx+len(sentinel):]
	const namePrefix = `name: "`
	nameIdx := strings.Index(tail, namePrefix)
	if nameIdx < 0 {
		return ""
	}
	rest := tail[nameIdx+len(namePrefix):]
	end := strings.IndexByte(rest, '"')
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// ensureBaselineIndexes calls Indexes().CreateMany per collection
// for every index in migrationIndexes(). Tolerates IndexKeySpecsConflict
// (code 86) and IndexOptionsConflict (code 85) — the migration system
// is the authoritative path for reshaping; the eager call's job is
// only to ensure the baseline exists.
//
// Other errors abort boot.
func (s *Store) ensureBaselineIndexes(ctx context.Context) error {
	for col, models := range migrationIndexes() {
		if len(models) == 0 {
			continue
		}
		if _, err := s.mdb.Collection(col).Indexes().CreateMany(ctx, models); err != nil {
			if mongoIsIndexConflict(err) {
				// Stale index shape — migration system repairs it.
				// Don't abort boot.
				continue
			}
			return fmt.Errorf("authsome/mongo: ensure baseline indexes for %s: %w", col, err)
		}
	}
	return nil
}

// Ping checks database connectivity.
func (s *Store) Ping(ctx context.Context) error {
	return s.db.Ping(ctx)
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// now returns the current UTC time.
func now() time.Time {
	return time.Now().UTC()
}

// isNoDocuments checks if an error wraps mongo.ErrNoDocuments.
func isNoDocuments(err error) bool {
	return errors.Is(err, mongo.ErrNoDocuments)
}

// mongoIsMigrationDecodeCorruption returns true when the error
// matches the upstream grove/mongodriver/mongomigrate
// inconsistency: RecordApplied writes the grove_migrations doc via
// bson.M{} (so MongoDB auto-generates an ObjectID _id) but
// ListApplied decodes _id as int64. The first time the orchestrator
// tries to read a previously-written record back, it fails with
// "decode applied: error decoding key _id: cannot decode objectID
// into an integer type."
//
// We detect this so runMigrationsWithSelfHeal can drop the broken
// tracking collections and retry — the upstream library doesn't
// expose a way to repair the docs in place (mongo _id is immutable).
func mongoIsMigrationDecodeCorruption(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "decode applied") &&
		strings.Contains(msg, "cannot decode objectID into an integer")
}

// mongoIsIndexConflict returns true when the error reports
// IndexKeySpecsConflict (code 86) or IndexOptionsConflict (code 85)
// — i.e. an index with the same name exists but has a different
// spec. The eager index-creation path treats these as recoverable:
// the migration system is the authoritative repair path; the eager
// call's job is only "ensure baseline."
func mongoIsIndexConflict(err error) bool {
	if err == nil {
		return false
	}
	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) && (cmdErr.Code == 85 || cmdErr.Code == 86) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "IndexKeySpecsConflict") ||
		strings.Contains(msg, "IndexOptionsConflict")
}

// mongoIsIndexNotFound returns true when the error reports
// "IndexNotFound" (mongo error code 27). Used by migrations that
// drop-then-recreate an index — re-running the migration on a
// deployment that already lacks the old index shouldn't fail.
func mongoIsIndexNotFound(err error) bool {
	if err == nil {
		return false
	}
	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) && cmdErr.Code == 27 {
		return true
	}
	// Some driver versions surface IndexNotFound as a plain string.
	return strings.Contains(err.Error(), "IndexNotFound") ||
		strings.Contains(err.Error(), "index not found")
}

// mapWriteErr converts low-level mongo write failures into the
// account-package sentinels the API layer maps to 4xx. Without
// this, a duplicate-key violation on the (app_id, email) or
// (app_id, username) index bubbles up as a 500 carrying the raw
// E11000 message — which leaks both the index name and the
// existence of the colliding row.
//
// Returns the original error unchanged when it isn't a recognized
// duplicate, so callers can still distinguish "not a known mapping"
// from "something we deliberately translated."
func mapWriteErr(err error) error {
	if err == nil {
		return nil
	}
	if !mongo.IsDuplicateKeyError(err) {
		return err
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "username"):
		return account.ErrUsernameTaken
	case strings.Contains(msg, "email"):
		return account.ErrEmailTaken
	default:
		// Some other unique constraint — return a generic conflict
		// rather than the raw E11000 message which leaks the index
		// name and colliding key.
		return store.ErrConflict
	}
}

// migrationIndexes returns the index definitions for all authsome collections.
func migrationIndexes() map[string][]mongo.IndexModel {
	return map[string][]mongo.IndexModel{
		colApps: {
			{
				Keys:    bson.D{{Key: "slug", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
		},
		colUsers: {
			{
				Keys:    bson.D{{Key: "app_id", Value: 1}, {Key: "email", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
			{
				// PartialFilterExpression — NOT SetSparse — because
				// SetSparse only excludes documents where the field
				// is missing entirely, but our writer always includes
				// `username: ""` for users without one. PartialFilter
				// excludes by VALUE so empty strings don't collide.
				// (See migration 20260502000004 that backfills this on
				// existing deployments.)
				Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "username", Value: 1}},
				Options: options.Index().
					SetUnique(true).
					SetPartialFilterExpression(bson.M{"username": bson.M{"$gt": ""}}),
			},
			{Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "created_at", Value: -1}}},
		},
		colSessions: {
			{
				Keys:    bson.D{{Key: "token", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
			{
				Keys:    bson.D{{Key: "refresh_token", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
			{Keys: bson.D{{Key: "expires_at", Value: 1}}},
			{Keys: bson.D{{Key: "family_id", Value: 1}}},
		},
		colRevokedRefreshTokens: {
			{Keys: bson.D{{Key: "family_id", Value: 1}}},
		},
		colVerifications: {
			{
				Keys:    bson.D{{Key: "token", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
		},
		colPasswordResets: {
			{
				Keys:    bson.D{{Key: "token", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
		},
		colOrganizations: {
			{
				Keys:    bson.D{{Key: "app_id", Value: 1}, {Key: "slug", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
			{Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "created_at", Value: -1}}},
		},
		colMembers: {
			{
				Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "org_id", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
			{Keys: bson.D{{Key: "org_id", Value: 1}}},
		},
		colInvitations: {
			{
				Keys:    bson.D{{Key: "token", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
			{Keys: bson.D{{Key: "org_id", Value: 1}, {Key: "status", Value: 1}}},
		},
		colTeams: {
			{
				Keys:    bson.D{{Key: "org_id", Value: 1}, {Key: "slug", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
			{Keys: bson.D{{Key: "org_id", Value: 1}}},
		},
		colDevices: {
			{Keys: bson.D{{Key: "user_id", Value: 1}}},
			{
				Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "fingerprint", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
		},
		colWebhooks: {
			{Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "active", Value: 1}}},
		},
		colNotifications: {
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
		},
		colAPIKeys: {
			{Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "created_at", Value: -1}}},
			{Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
			{
				Keys:    bson.D{{Key: "app_id", Value: 1}, {Key: "key_prefix", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
			{
				Keys:    bson.D{{Key: "key_hash", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
		},
		colEnvironments: {
			{
				Keys:    bson.D{{Key: "app_id", Value: 1}, {Key: "slug", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
			{Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "is_default", Value: 1}}},
			{Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "created_at", Value: 1}}},
		},
		colFormConfigs: {
			{
				Keys:    bson.D{{Key: "app_id", Value: 1}, {Key: "form_type", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
		},
		colBrandingConfigs: {
			{
				Keys:    bson.D{{Key: "org_id", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
		},
		colAppSessionConfigs: {
			{
				Keys:    bson.D{{Key: "app_id", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
		},
		colServiceAccounts: {
			{
				Keys:    bson.D{{Key: "app_id", Value: 1}, {Key: "name", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
			{Keys: bson.D{{Key: "app_id", Value: 1}}},
		},
	}
}
