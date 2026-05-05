package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/xraph/grove/drivers/mongodriver/mongomigrate"
	"github.com/xraph/grove/migrate"
)

// Migrations is the grove migration group for the AuthSome mongo store.
var Migrations = migrate.NewGroup("authsome")

func init() {
	Migrations.MustRegister(
		&migrate.Migration{
			Name:    "create_authsome_apps",
			Version: "20240101000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}

				if err := mexec.CreateCollection(ctx, (*appModel)(nil)); err != nil {
					return err
				}

				return mexec.CreateIndexes(ctx, colApps, []mongo.IndexModel{
					{
						Keys:    bson.D{{Key: "slug", Value: 1}},
						Options: options.Index().SetUnique(true),
					},
				})
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				return mexec.DropCollection(ctx, (*appModel)(nil))
			},
		},
		&migrate.Migration{
			Name:    "create_authsome_users",
			Version: "20240101000002",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}

				if err := mexec.CreateCollection(ctx, (*userModel)(nil)); err != nil {
					return err
				}

				// Use createIndexesReshapeConflicting (not the bare
				// mexec.CreateIndexes) because this migration may run
				// against a deployment that already had an OLD-shape
				// app_id_1_username_1 index from a prior install. The
				// helper drops the conflicting index by name and retries
				// once. See store/mongo/store.go for the recovery
				// rationale.
				return createIndexesReshapeConflicting(ctx, mexec.DB().Collection(colUsers), []mongo.IndexModel{
					{
						Keys:    bson.D{{Key: "app_id", Value: 1}, {Key: "email", Value: 1}},
						Options: options.Index().SetUnique(true),
					},
					{
						// PartialFilterExpression — NOT SetSparse — because
						// SetSparse only excludes documents where the field
						// is missing entirely, and writes always include
						// username: "" for users without one. PartialFilter
						// excludes by VALUE so empty strings don't collide.
						Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "username", Value: 1}},
						Options: options.Index().
							SetUnique(true).
							SetPartialFilterExpression(bson.M{"username": bson.M{"$gt": ""}}),
					},
					{Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "created_at", Value: -1}}},
				})
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				return mexec.DropCollection(ctx, (*userModel)(nil))
			},
		},
		&migrate.Migration{
			Name:    "create_authsome_sessions",
			Version: "20240101000003",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}

				if err := mexec.CreateCollection(ctx, (*sessionModel)(nil)); err != nil {
					return err
				}

				return mexec.CreateIndexes(ctx, colSessions, []mongo.IndexModel{
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
				})
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				return mexec.DropCollection(ctx, (*sessionModel)(nil))
			},
		},
		&migrate.Migration{
			Name:    "create_authsome_verifications",
			Version: "20240101000004",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}

				if err := mexec.CreateCollection(ctx, (*verificationModel)(nil)); err != nil {
					return err
				}

				return mexec.CreateIndexes(ctx, colVerifications, []mongo.IndexModel{
					{
						Keys:    bson.D{{Key: "token", Value: 1}},
						Options: options.Index().SetUnique(true),
					},
				})
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				return mexec.DropCollection(ctx, (*verificationModel)(nil))
			},
		},
		&migrate.Migration{
			Name:    "create_authsome_password_resets",
			Version: "20240101000005",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}

				if err := mexec.CreateCollection(ctx, (*passwordResetModel)(nil)); err != nil {
					return err
				}

				return mexec.CreateIndexes(ctx, colPasswordResets, []mongo.IndexModel{
					{
						Keys:    bson.D{{Key: "token", Value: 1}},
						Options: options.Index().SetUnique(true),
					},
				})
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				return mexec.DropCollection(ctx, (*passwordResetModel)(nil))
			},
		},
		&migrate.Migration{
			Name:    "create_authsome_organizations",
			Version: "20240101000006",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}

				if err := mexec.CreateCollection(ctx, (*organizationModel)(nil)); err != nil {
					return err
				}

				return mexec.CreateIndexes(ctx, colOrganizations, []mongo.IndexModel{
					{
						Keys:    bson.D{{Key: "app_id", Value: 1}, {Key: "slug", Value: 1}},
						Options: options.Index().SetUnique(true),
					},
					{Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "created_at", Value: -1}}},
				})
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				return mexec.DropCollection(ctx, (*organizationModel)(nil))
			},
		},
		&migrate.Migration{
			Name:    "create_authsome_members",
			Version: "20240101000007",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}

				if err := mexec.CreateCollection(ctx, (*memberModel)(nil)); err != nil {
					return err
				}

				return mexec.CreateIndexes(ctx, colMembers, []mongo.IndexModel{
					{
						Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "org_id", Value: 1}},
						Options: options.Index().SetUnique(true),
					},
					{Keys: bson.D{{Key: "org_id", Value: 1}}},
				})
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				return mexec.DropCollection(ctx, (*memberModel)(nil))
			},
		},
		&migrate.Migration{
			Name:    "create_authsome_invitations",
			Version: "20240101000008",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}

				if err := mexec.CreateCollection(ctx, (*invitationModel)(nil)); err != nil {
					return err
				}

				return mexec.CreateIndexes(ctx, colInvitations, []mongo.IndexModel{
					{
						Keys:    bson.D{{Key: "token", Value: 1}},
						Options: options.Index().SetUnique(true),
					},
					{Keys: bson.D{{Key: "org_id", Value: 1}, {Key: "status", Value: 1}}},
				})
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				return mexec.DropCollection(ctx, (*invitationModel)(nil))
			},
		},
		&migrate.Migration{
			Name:    "create_authsome_teams",
			Version: "20240101000009",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}

				if err := mexec.CreateCollection(ctx, (*teamModel)(nil)); err != nil {
					return err
				}

				return mexec.CreateIndexes(ctx, colTeams, []mongo.IndexModel{
					{
						Keys:    bson.D{{Key: "org_id", Value: 1}, {Key: "slug", Value: 1}},
						Options: options.Index().SetUnique(true),
					},
					{Keys: bson.D{{Key: "org_id", Value: 1}}},
				})
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				return mexec.DropCollection(ctx, (*teamModel)(nil))
			},
		},
		&migrate.Migration{
			Name:    "create_authsome_devices",
			Version: "20240101000010",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}

				if err := mexec.CreateCollection(ctx, (*deviceModel)(nil)); err != nil {
					return err
				}

				return mexec.CreateIndexes(ctx, colDevices, []mongo.IndexModel{
					{Keys: bson.D{{Key: "user_id", Value: 1}}},
					{
						Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "fingerprint", Value: 1}},
						Options: options.Index().SetUnique(true),
					},
				})
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				return mexec.DropCollection(ctx, (*deviceModel)(nil))
			},
		},
		&migrate.Migration{
			Name:    "create_authsome_webhooks",
			Version: "20240101000011",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}

				if err := mexec.CreateCollection(ctx, (*webhookModel)(nil)); err != nil {
					return err
				}

				return mexec.CreateIndexes(ctx, colWebhooks, []mongo.IndexModel{
					{Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "active", Value: 1}}},
				})
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				return mexec.DropCollection(ctx, (*webhookModel)(nil))
			},
		},
		&migrate.Migration{
			Name:    "create_authsome_notifications",
			Version: "20240101000012",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}

				if err := mexec.CreateCollection(ctx, (*notificationModel)(nil)); err != nil {
					return err
				}

				return mexec.CreateIndexes(ctx, colNotifications, []mongo.IndexModel{
					{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
				})
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				return mexec.DropCollection(ctx, (*notificationModel)(nil))
			},
		},
		&migrate.Migration{
			Name:    "create_authsome_api_keys",
			Version: "20240101000013",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}

				if err := mexec.CreateCollection(ctx, (*apiKeyModel)(nil)); err != nil {
					return err
				}

				return mexec.CreateIndexes(ctx, colAPIKeys, []mongo.IndexModel{
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
				})
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				return mexec.DropCollection(ctx, (*apiKeyModel)(nil))
			},
		},
		// RBAC migrations kept as no-ops — roles, permissions, and assignments
		// are now managed exclusively by Warden.
		&migrate.Migration{
			Name:    "create_authsome_roles",
			Version: "20240101000014",
			Up:      func(_ context.Context, _ migrate.Executor) error { return nil },
			Down:    func(_ context.Context, _ migrate.Executor) error { return nil },
		},
		&migrate.Migration{
			Name:    "create_authsome_permissions",
			Version: "20240101000015",
			Up:      func(_ context.Context, _ migrate.Executor) error { return nil },
			Down:    func(_ context.Context, _ migrate.Executor) error { return nil },
		},
		&migrate.Migration{
			Name:    "create_authsome_user_roles",
			Version: "20240101000016",
			Up:      func(_ context.Context, _ migrate.Executor) error { return nil },
			Down:    func(_ context.Context, _ migrate.Executor) error { return nil },
		},
		&migrate.Migration{
			Name:    "create_authsome_environments",
			Version: "20240101000017",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}

				if err := mexec.CreateCollection(ctx, (*environmentModel)(nil)); err != nil {
					return err
				}

				return mexec.CreateIndexes(ctx, colEnvironments, []mongo.IndexModel{
					{
						Keys:    bson.D{{Key: "app_id", Value: 1}, {Key: "slug", Value: 1}},
						Options: options.Index().SetUnique(true),
					},
					{Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "is_default", Value: 1}}},
					{Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "created_at", Value: 1}}},
				})
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				return mexec.DropCollection(ctx, (*environmentModel)(nil))
			},
		},

		// Migration 18: Add last_activity_at field to sessions (no-op for MongoDB, field is added dynamically).
		&migrate.Migration{
			Name:    "add_session_last_activity_at",
			Version: "20240101000018",
			Up:      func(_ context.Context, _ migrate.Executor) error { return nil },
			Down:    func(_ context.Context, _ migrate.Executor) error { return nil },
		},
		// Migration 19: Add family_id index on sessions for refresh-token
		// replay detection.
		&migrate.Migration{
			Name:    "add_session_family_id_index",
			Version: "20260502000002",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				return mexec.CreateIndexes(ctx, colSessions, []mongo.IndexModel{
					{Keys: bson.D{{Key: "family_id", Value: 1}}},
				})
			},
			Down: func(_ context.Context, _ migrate.Executor) error { return nil },
		},
		// Migration 20: Create revoked_refresh_tokens collection with indices
		// for refresh-token replay detection.
		&migrate.Migration{
			// Fix the username index that was created with SetSparse(true)
			// in migration 20240101000002. SetSparse only excludes documents
			// where the field is *missing*, but every write to authsome_users
			// includes `username: ""` because Go's zero value serializes that
			// way. The result: any second user without a username collides
			// with E11000 dup key error on app_id_1_username_1, surfacing as
			// a 500 from POST /v1/signup.
			//
			// Drop the index by name and recreate it with a
			// PartialFilterExpression that excludes by VALUE — empty string
			// stops being a key.
			Name:    "fix_username_index_partial_filter",
			Version: "20260502000004",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				// Drop the bad index by its auto-derived name. Mongo
				// names indexes after their keys — app_id_1_username_1.
				// Tolerate IndexNotFound (code 27) so the migration is
				// safe to re-run on a deployment that already fixed the
				// index manually.
				coll := mexec.DB().Collection(colUsers)
				if err := coll.Indexes().DropOne(ctx, "app_id_1_username_1"); err != nil {
					if !mongoIsIndexNotFound(err) {
						return fmt.Errorf("drop bad username index: %w", err)
					}
				}
				return mexec.CreateIndexes(ctx, colUsers, []mongo.IndexModel{
					{
						Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "username", Value: 1}},
						Options: options.Index().
							SetUnique(true).
							SetPartialFilterExpression(bson.M{"username": bson.M{"$gt": ""}}),
					},
				})
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				// Recreate the (broken) sparse index for completeness on
				// rollback. Operators rolling back will hit the original
				// E11000 bug again.
				coll := mexec.DB().Collection(colUsers)
				if err := coll.Indexes().DropOne(ctx, "app_id_1_username_1"); err != nil {
					if !mongoIsIndexNotFound(err) {
						return err
					}
				}
				return mexec.CreateIndexes(ctx, colUsers, []mongo.IndexModel{
					{
						Keys:    bson.D{{Key: "app_id", Value: 1}, {Key: "username", Value: 1}},
						Options: options.Index().SetUnique(true).SetSparse(true),
					},
				})
			},
		},
		&migrate.Migration{
			Name:    "create_revoked_refresh_tokens",
			Version: "20260502000003",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				if err := mexec.CreateCollection(ctx, (*revokedRefreshTokenModel)(nil)); err != nil {
					return err
				}
				return mexec.CreateIndexes(ctx, colRevokedRefreshTokens, []mongo.IndexModel{
					{Keys: bson.D{{Key: "family_id", Value: 1}}},
				})
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				return mexec.DropCollection(ctx, (*revokedRefreshTokenModel)(nil))
			},
		},
		// Refresh $jsonSchema validators on existing collections so that
		// nullable pointer fields (e.g. *time.Time, *bool, *int, *string)
		// accept null. The earlier grove builder emitted a strict bsonType
		// for pointer fields, which made any insert with a nil pointer fail
		// with "Document failed validation" — see CreateUser when both
		// ban_expires and deleted_at default to nil. This migration reapplies
		// the (now nullable-aware) schema via collMod for every collection
		// the store creates. It is a no-op for fresh installs because new
		// collections already pick up the corrected schema.
		&migrate.Migration{
			Name:    "refresh_validators_for_nullable_pointers",
			Version: "20260504000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				models := []any{
					(*appModel)(nil),
					(*environmentModel)(nil),
					(*userModel)(nil),
					(*sessionModel)(nil),
					(*verificationModel)(nil),
					(*passwordResetModel)(nil),
					(*organizationModel)(nil),
					(*memberModel)(nil),
					(*invitationModel)(nil),
					(*teamModel)(nil),
					(*deviceModel)(nil),
					(*webhookModel)(nil),
					(*notificationModel)(nil),
					(*apiKeyModel)(nil),
					(*revokedRefreshTokenModel)(nil),
				}
				for _, m := range models {
					if err := mexec.RefreshValidator(ctx, m); err != nil {
						return fmt.Errorf("refresh validator: %w", err)
					}
				}
				return nil
			},
			Down: func(_ context.Context, _ migrate.Executor) error {
				// Rolling back would re-introduce the original bug; we leave
				// the corrected validator in place. No-op down.
				return nil
			},
		},
		// Migration: Remove duplicate / stale platform-admin role rows that
		// were created when authsome transitioned from a programmatic
		// DefaultRoles bootstrap approach to the warden DSL approach. Both
		// code paths ran against the same database, leaving orphaned rows.
		//
		// Strategy (idempotent):
		//  1. Delete ALL rows with slug "platform_admin" (underscore) — these
		//     are always stale; the new code uses the kebab form exclusively.
		//  2. For slug "platform-admin": if more than one row exists, delete
		//     those that have no parent_slug (the orphaned / programmatic
		//     ones). The warden-DSL row has parent_slug = "platform-user" and
		//     is kept. We only delete parentless rows when a parented row also
		//     exists, so the last surviving row is never accidentally removed.
		&migrate.Migration{
			Name:    "remove_duplicate_platform_admin_roles",
			Version: "20260505000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				coll := mexec.DB().Collection("warden_roles")

				// 1. Remove all underscore-slug rows — always stale.
				if _, err := coll.DeleteMany(ctx, bson.M{"slug": "platform_admin"}); err != nil {
					return fmt.Errorf("remove platform_admin (underscore) roles: %w", err)
				}

				// 2. Find all rows with the kebab slug.
				cursor, err := coll.Find(ctx, bson.M{"slug": "platform-admin"})
				if err != nil {
					return fmt.Errorf("find platform-admin roles: %w", err)
				}
				defer cursor.Close(ctx) //nolint:errcheck

				type minimalRole struct {
					ID         string  `bson:"_id"`
					ParentSlug *string `bson:"parent_slug"`
				}
				var rows []minimalRole
				if err := cursor.All(ctx, &rows); err != nil {
					return fmt.Errorf("decode platform-admin roles: %w", err)
				}

				// Nothing to clean up if ≤1 row.
				if len(rows) <= 1 {
					return nil
				}

				// Check that at least one parented (warden-DSL) row exists
				// before we start deleting parentless rows, to be conservative.
				hasParented := false
				for _, r := range rows {
					if r.ParentSlug != nil && *r.ParentSlug != "" {
						hasParented = true
						break
					}
				}
				if !hasParented {
					// All rows are parentless — unusual state; leave them alone
					// and let an operator investigate rather than deleting
					// everything.
					return nil
				}

				// Collect IDs of parentless (orphaned) rows to delete.
				var orphanIDs []string
				for _, r := range rows {
					if r.ParentSlug == nil || *r.ParentSlug == "" {
						orphanIDs = append(orphanIDs, r.ID)
					}
				}
				if len(orphanIDs) == 0 {
					return nil
				}

				if _, err := coll.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": orphanIDs}}); err != nil {
					return fmt.Errorf("remove orphaned platform-admin roles: %w", err)
				}
				return nil
			},
			Down: func(_ context.Context, _ migrate.Executor) error {
				// Deleted orphan rows cannot be safely re-created; no-op down.
				return nil
			},
		},
	)
}
