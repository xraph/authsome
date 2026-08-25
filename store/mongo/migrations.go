package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/xraph/grove/drivers/mongodriver/mongomigrate"
	"github.com/xraph/grove/migrate"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/user"
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
		// Migration: Create service_accounts collection with indexes.
		&migrate.Migration{
			Name:    "create_authsome_service_accounts",
			Version: "20260505000002",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}

				if err := mexec.CreateCollection(ctx, (*serviceAccountModel)(nil)); err != nil {
					return err
				}

				return mexec.CreateIndexes(ctx, colServiceAccounts, []mongo.IndexModel{
					{
						Keys:    bson.D{{Key: "app_id", Value: 1}, {Key: "name", Value: 1}},
						Options: options.Index().SetUnique(true),
					},
					{Keys: bson.D{{Key: "app_id", Value: 1}}},
				})
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				return mexec.DropCollection(ctx, (*serviceAccountModel)(nil))
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
				// This deletes across all tenants, but that's acceptable because authsome
				// controls role creation and enforces kebab-case slugs. No tenant should
				// ever have a legitimate "platform_admin" (underscore) role.
				if _, err := coll.DeleteMany(ctx, bson.M{"slug": "platform_admin"}); err != nil {
					return fmt.Errorf("remove platform_admin (underscore) roles: %w", err)
				}

				// 2. Find all rows with the kebab slug.
				cursor, err := coll.Find(ctx, bson.M{"slug": "platform-admin"})
				if err != nil {
					return fmt.Errorf("find platform-admin roles: %w", err)
				}
				defer cursor.Close(ctx)

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
		// Migration: Backfill namespace_path="" on warden_assignments documents
		// that were created before warden commit 566f0e1 added namespace support.
		// The new ListRolesForSubject filters by namespace_path IN [""] when
		// querying the tenant root; documents missing the field don't match,
		// causing 403s on all permission checks.
		&migrate.Migration{
			Name:    "backfill_warden_assignment_namespace_path",
			Version: "20260505000003",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				coll := mexec.DB().Collection("warden_assignments")
				_, err := coll.UpdateMany(
					ctx,
					bson.M{"namespace_path": bson.M{"$exists": false}},
					bson.M{"$set": bson.M{"namespace_path": ""}},
				)
				return err
			},
			Down: func(_ context.Context, _ migrate.Executor) error {
				// Backfilled empty strings are equivalent to the absent field; no-op down.
				return nil
			},
		},
		// Migration: User emails collection (multiple emails per account).
		// Each user owns >=1 email; exactly one is primary and mirrors the
		// user's email/email_verified. Addresses are unique per
		// (app_id, env_id, email) so an address belongs to one account.
		// Backfills one primary doc per existing user with a non-empty email.
		&migrate.Migration{
			Name:    "create_user_emails",
			Version: "20260601000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				if err := mexec.CreateCollection(ctx, (*userEmailModel)(nil)); err != nil {
					return err
				}
				// MongoDB partial-index filters reject $exists:false ($not), so
				// (unlike the SQL backends, which use WHERE deleted_at IS NULL)
				// the email-uniqueness index is non-partial. To keep "deleting an
				// email frees the address" observable, DeleteUserEmail hard-deletes
				// on mongo. The single-primary index uses an equality-only partial
				// filter (is_primary: true), which mongo does allow.
				if err := mexec.CreateIndexes(ctx, colUserEmails, []mongo.IndexModel{
					{
						Keys:    bson.D{{Key: "app_id", Value: 1}, {Key: "env_id", Value: 1}, {Key: "email", Value: 1}},
						Options: options.Index().SetUnique(true),
					},
					{
						Keys: bson.D{{Key: "user_id", Value: 1}},
						Options: options.Index().SetUnique(true).
							SetPartialFilterExpression(bson.M{"is_primary": true}),
					},
					{
						Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: 1}},
					},
				}); err != nil {
					return err
				}

				// Backfill: one primary doc per existing non-deleted user with
				// a non-empty email. {deleted_at: nil} matches missing-or-null.
				usersColl := mexec.DB().Collection(colUsers)
				cur, err := usersColl.Find(ctx, bson.M{"deleted_at": nil, "email": bson.M{"$ne": ""}})
				if err != nil {
					return fmt.Errorf("list users for email backfill: %w", err)
				}
				defer cur.Close(ctx)

				emailsColl := mexec.DB().Collection(colUserEmails)
				for cur.Next(ctx) {
					var u struct {
						ID            string `bson:"_id"`
						AppID         string `bson:"app_id"`
						EnvID         string `bson:"env_id"`
						Email         string `bson:"email"`
						EmailVerified bool   `bson:"email_verified"`
					}
					if err := cur.Decode(&u); err != nil {
						return fmt.Errorf("decode user for backfill: %w", err)
					}
					t := now()
					if _, err := emailsColl.InsertOne(ctx, bson.M{
						"_id":        id.NewUserEmailID().String(),
						"user_id":    u.ID,
						"app_id":     u.AppID,
						"env_id":     u.EnvID,
						"email":      user.NormalizeEmail(u.Email),
						"verified":   u.EmailVerified,
						"is_primary": true,
						"source":     "backfill",
						"created_at": t,
						"updated_at": t,
					}); err != nil {
						return fmt.Errorf("backfill email for user %s: %w", u.ID, err)
					}
				}
				return cur.Err()
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				return mexec.DropCollection(ctx, (*userEmailModel)(nil))
			},
		},

		// Migration: session principal identity. A session may be owned by a
		// user or by a service account (session.Session), and until
		// sessionModel carried these fields the store silently dropped which
		// one — a service-account session came back looking like an ordinary
		// user session whose user_id happened to be empty.
		//
		// Mongo needs no column added, but the collection's $jsonSchema
		// validator is generated from the model struct, so an existing
		// deployment's validator does not know principal_kind or
		// service_account_id. RefreshValidator reapplies the schema via
		// collMod, the same way refresh_validators_for_nullable_pointers does.
		// Fresh installs are unaffected: they pick up the current schema when
		// the collection is created.
		//
		// The index is partial on {$gt: ""} rather than sparse. Sparse would be
		// pointless here: grove writes every mapped field whatever the bson
		// omitempty tag says, so a user session stores service_account_id: ""
		// rather than omitting it, and a sparse index would cover every session
		// in the collection. $gt: "" selects the non-empty values — the
		// service-account sessions — and is one of the operators
		// partialFilterExpression actually accepts ($ne is not). This mirrors
		// the postgres partial index on service_account_id <> ''.
		&migrate.Migration{
			Name:    "add_session_principal_identity",
			Version: "20260620000002",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				if err := mexec.RefreshValidator(ctx, (*sessionModel)(nil)); err != nil {
					return fmt.Errorf("refresh session validator: %w", err)
				}
				return mexec.CreateIndexes(ctx, colSessions, []mongo.IndexModel{
					{
						Keys: bson.D{{Key: "service_account_id", Value: 1}},
						Options: options.Index().SetPartialFilterExpression(
							bson.D{{Key: "service_account_id", Value: bson.D{{Key: "$gt", Value: ""}}}},
						),
					},
				})
			},
			// Forward-only: rolling the validator back would reject the
			// service-account sessions this migration makes storable, and
			// dropping the index only costs a scan.
			Down: func(_ context.Context, _ migrate.Executor) error { return nil },
		},

		// Migration: session agent principal index. A session may now be
		// owned by an AI agent acting under a human's grant (AgentID/GrantID
		// on session.Session), mirroring postgres/sqlite's
		// add_session_agent_principal migration — same version
		// (20260824000100) on all three backends, since this is one logical
		// change.
		//
		// Mongo needs no column added and no CHECK constraint: sessionModel
		// already carries AgentID/GrantID (bson agent_id/grant_id,
		// omitempty), and the store already round-trips them (see
		// toSessionModel/fromSessionModel and DeleteSessionsByGrant in
		// session.go). What is missing is what DeleteSessionsByGrant needs:
		// an index on grant_id. Without one, revoking a grant is a full
		// collection scan on the one path where revocation latency matters
		// most.
		//
		// RefreshValidator follows the same precedent as
		// add_session_principal_identity above: agent_id/grant_id are new
		// fields on sessionModel, so an existing deployment's validator does
		// not know them until the schema is reapplied via collMod. Fresh
		// installs are unaffected.
		//
		// The index is partial on {$gt: ""}, not sparse, for the same reason
		// as service_account_id above: grove writes grant_id: "" on every
		// non-agent session rather than omitting it, so a sparse index would
		// cover the whole collection. $gt: "" selects only the agent
		// sessions. This mirrors the postgres partial index
		// (idx_authsome_sessions_grant_id, WHERE grant_id <> '').
		&migrate.Migration{
			Name:    "add_session_agent_principal_index",
			Version: "20260824000110",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				if err := mexec.RefreshValidator(ctx, (*sessionModel)(nil)); err != nil {
					return fmt.Errorf("refresh session validator: %w", err)
				}
				return mexec.CreateIndexes(ctx, colSessions, []mongo.IndexModel{
					{
						Keys: bson.D{{Key: "grant_id", Value: 1}},
						Options: options.Index().SetPartialFilterExpression(
							bson.D{{Key: "grant_id", Value: bson.D{{Key: "$gt", Value: ""}}}},
						),
					},
				})
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				// Drop the index by its auto-derived name. Mongo names a
				// single-key index after its key — grant_id_1 — regardless
				// of the partial filter. Tolerate IndexNotFound (code 27)
				// so this is safe to re-run.
				coll := mexec.DB().Collection(colSessions)
				if err := coll.Indexes().DropOne(ctx, "grant_id_1"); err != nil {
					if !mongoIsIndexNotFound(err) {
						return fmt.Errorf("drop grant_id index: %w", err)
					}
				}
				// Forward-only on the validator, matching
				// add_session_principal_identity: rolling it back would
				// reject the agent sessions this migration makes storable.
				return nil
			},
		},

		// Migration: session actor chain. sessionModel gained actors,
		// actor_grant and delegation_id, which carry the principals acting on
		// the subject's behalf. Only impersonated_by was stored before, and
		// that is a projection of the chain that is empty for a delegation, so
		// a delegated session round-tripped with no actors at all.
		//
		// Mongo adds no columns, but the collection's $jsonSchema validator is
		// generated from the model struct, so an existing deployment's
		// validator does not know the three new fields. RefreshValidator
		// reapplies the schema through collMod, the same way
		// add_session_principal_identity does.
		//
		// No backfill of actors from impersonated_by. fromSessionModel rebuilds
		// the impersonation chain from that field whenever actors is absent, so
		// documents written before this migration already read back correctly;
		// backfilling them is Task 7's job, alongside the delegation collection.
		//
		// The index is over actors.kind and actors.id, which mongo applies to
		// each subdocument in the array. That answers "what has this agent
		// acted on", which is the question the chain exists to make answerable.
		&migrate.Migration{
			Name:    "add_session_actor_chain",
			Version: "20260824000049",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				if err := mexec.RefreshValidator(ctx, (*sessionModel)(nil)); err != nil {
					return fmt.Errorf("refresh session validator: %w", err)
				}
				return mexec.CreateIndexes(ctx, colSessions, []mongo.IndexModel{
					{Keys: bson.D{
						{Key: "actors.kind", Value: 1},
						{Key: "actors.id", Value: 1},
					}},
					{Keys: bson.D{{Key: "delegation_id", Value: 1}}},
				})
			},
			// Forward-only, matching add_session_principal_identity: rolling the
			// validator back would reject the delegated sessions this makes
			// storable.
			Down: func(_ context.Context, _ migrate.Executor) error { return nil },
		},

		// Migration: the delegation grants collection. A grant is the durable,
		// revocable record that one principal may act for another. The chain on
		// a session says who is acting; this says they were allowed to.
		&migrate.Migration{
			Name:    "create_authsome_delegations",
			Version: "20260824000070",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				if err := mexec.CreateCollection(ctx, (*delegationModel)(nil)); err != nil {
					return err
				}
				return mexec.CreateIndexes(ctx, colDelegations, []mongo.IndexModel{
					{
						// One live grant per (app, actor, subject, kind). The
						// partial filter keys on revoked_at being null rather
						// than absent, which is the whole point: grove writes
						// every mapped field whatever the bson omitempty tag
						// says, so a live grant stores revoked_at: null and
						// never omits the key. Filtering on $exists would
						// match no document at all and quietly enforce
						// nothing, and mongo rejects $exists: false in a
						// partial filter regardless. Revoking sets a date,
						// which drops the row out of the index and frees the
						// slot for a fresh grant.
						Keys: bson.D{
							{Key: "app_id", Value: 1},
							{Key: "actor_kind", Value: 1}, {Key: "actor_id", Value: 1},
							{Key: "subject_kind", Value: 1}, {Key: "subject_id", Value: 1},
							{Key: "grant_kind", Value: 1},
						},
						Options: options.Index().SetUnique(true).
							SetPartialFilterExpression(bson.D{
								{Key: "revoked_at", Value: bson.D{{Key: "$type", Value: "null"}}},
							}),
					},
					{Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "subject_kind", Value: 1}, {Key: "subject_id", Value: 1}}},
					{Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "actor_kind", Value: 1}, {Key: "actor_id", Value: 1}}},
				})
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				return mexec.DropCollection(ctx, (*delegationModel)(nil))
			},
		},

		// Migration: principal fields on service accounts, and the actor-chain
		// backfill. serviceAccountModel gained kind, env_id, org_id,
		// owner_user_id, parent_id and expires_at, so an existing deployment's
		// generated validator has to be reapplied before a row carrying them
		// can be written.
		//
		// The backfill converts the legacy impersonation projection into a real
		// chain. fromSessionModel already reads those documents correctly
		// through its impersonated_by fallback, so this changes no behaviour;
		// it means an operator querying actors.id sees impersonations too,
		// rather than only sessions written since the chain landed.
		&migrate.Migration{
			Name:    "add_principal_fields_and_backfill_chain",
			Version: "20260824000071",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				if err := mexec.RefreshValidator(ctx, (*serviceAccountModel)(nil)); err != nil {
					return fmt.Errorf("refresh service account validator: %w", err)
				}
				if err := mexec.CreateIndexes(ctx, colServiceAccounts, []mongo.IndexModel{
					{Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "kind", Value: 1}}},
					{Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "parent_id", Value: 1}}},
					{Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "owner_user_id", Value: 1}}},
				}); err != nil {
					return fmt.Errorf("create service account indexes: %w", err)
				}

				// Only documents that carry the legacy field and no chain yet,
				// so a re-run cannot overwrite a chain written since.
				_, err := mexec.DB().Collection(colSessions).UpdateMany(ctx,
					bson.M{
						"impersonated_by": bson.M{"$gt": ""},
						"actor_grant":     bson.M{"$in": bson.A{nil, ""}},
					},
					mongo.Pipeline{{{Key: "$set", Value: bson.D{
						{Key: "actors", Value: bson.A{bson.D{
							{Key: "kind", Value: string(principal.KindUser)},
							{Key: "id", Value: "$impersonated_by"},
						}}},
						{Key: "actor_grant", Value: string(principal.GrantImpersonation)},
					}}}},
				)
				if err != nil {
					return fmt.Errorf("backfill session actor chain: %w", err)
				}
				return nil
			},
			// Forward-only. Undoing the backfill would strip chains this store
			// now depends on, and the read path treats a chain as the truth.
			Down: func(_ context.Context, _ migrate.Executor) error { return nil },
		},

		// Migration: stop a lapsed grant from blocking a fresh one. Mongo's
		// counterpart to postgres's delegation_live_index_excludes_expired.
		//
		// Mongo has the same limitation as the SQL backends here, for its own
		// reason: a partialFilterExpression admits only equality, $exists,
		// $type, the range operators against a CONSTANT, $and, $or and $in.
		// There is no way to name "now" inside one, so expiry cannot live in
		// the filter any more than it can live in a postgres index predicate.
		//
		// The filter therefore adds expires_at being null alongside revoked_at
		// being null: never-revoked AND never-expiring, the rows whose
		// liveness mongo can decide by itself. Grants that carry an expiry are
		// checked in CreateDelegation instead, against
		// principal.Delegation.IsActive, the same method every other backend
		// now agrees with. See the postgres migration for the concurrency
		// trade that leaves.
		//
		// $type: "null" rather than $exists, for the reason spelled out on
		// create_authsome_delegations above: grove writes every mapped field
		// whatever omitempty says, so an unset expiry reaches mongo as an
		// explicit null and never as an absent key.
		//
		// The reshape goes through createIndexesReshapeConflicting because the
		// keys are unchanged and only the options move. Mongo refuses that as
		// an index conflict, and the helper's whole job is to drop the
		// conflicting index by name and retry.
		&migrate.Migration{
			Name:    "delegation_live_index_excludes_expired",
			Version: "20260824000100",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				return createIndexesReshapeConflicting(ctx, mexec.DB().Collection(colDelegations),
					[]mongo.IndexModel{delegationLiveIndex(true)})
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				return createIndexesReshapeConflicting(ctx, mexec.DB().Collection(colDelegations),
					[]mongo.IndexModel{delegationLiveIndex(false)})
			},
		},

		// Bring the users collection in line with postgres. Its unique
		// indexes were still (app_id, email) and (app_id, username), so mongo
		// forbade what postgres allows: the same address or handle in two
		// environments of one app. Existing docs also never had env_id
		// backfilled, so they belonged to no environment at all.
		&migrate.Migration{
			Name:    "env_scope_user_unique_indexes",
			Version: "20260824000090",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				if err := backfillDefaultEnvironments(ctx, mexec); err != nil {
					return err
				}

				coll := mexec.DB().Collection(colUsers)
				for _, name := range []string{"app_id_1_email_1", "app_id_1_username_1"} {
					if err := coll.Indexes().DropOne(ctx, name); err != nil {
						if !mongoIsIndexNotFound(err) {
							return fmt.Errorf("drop %s: %w", name, err)
						}
					}
				}
				// The email index stays non-partial for the reason given on
				// the user_emails collection: mongo rejects $exists:false in a
				// partial filter, so "deleted_at IS NULL" cannot be expressed
				// here the way the SQL backends express it.
				return mexec.CreateIndexes(ctx, colUsers, []mongo.IndexModel{
					{
						Keys: bson.D{
							{Key: "app_id", Value: 1},
							{Key: "env_id", Value: 1},
							{Key: "email", Value: 1},
						},
						Options: options.Index().SetUnique(true),
					},
					{
						Keys: bson.D{
							{Key: "app_id", Value: 1},
							{Key: "env_id", Value: 1},
							{Key: "username", Value: 1},
						},
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
				// Indexes only. The backfilled env_id values stay: stripping
				// them would strand docs the running code now depends on.
				coll := mexec.DB().Collection(colUsers)
				for _, name := range []string{
					"app_id_1_env_id_1_email_1", "app_id_1_env_id_1_username_1",
				} {
					if err := coll.Indexes().DropOne(ctx, name); err != nil {
						if !mongoIsIndexNotFound(err) {
							return err
						}
					}
				}
				return mexec.CreateIndexes(ctx, colUsers, []mongo.IndexModel{
					{
						Keys:    bson.D{{Key: "app_id", Value: 1}, {Key: "email", Value: 1}},
						Options: options.Index().SetUnique(true),
					},
					{
						Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "username", Value: 1}},
						Options: options.Index().
							SetUnique(true).
							SetPartialFilterExpression(bson.M{"username": bson.M{"$gt": ""}}),
					},
				})
			},
		},
		&migrate.Migration{
			Name:    "add_session_audience",
			Version: "20260824000080",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				// The collection's $jsonSchema is generated from sessionModel,
				// so an existing deployment's validator does not know the
				// audience field and rejects every document carrying it.
				return mexec.RefreshValidator(ctx, (*sessionModel)(nil))
			},
			Down: func(_ context.Context, _ migrate.Executor) error { return nil },
		},
	)
}

// backfillDefaultEnvironments gives every app a default environment if it has
// none, then stamps every scoped doc whose env_id is missing or empty.
//
// Both halves are idempotent, so the migration is safe to re-run: an app that
// already has a default environment keeps it, and a doc whose env_id is
// already set is not touched.
func backfillDefaultEnvironments(ctx context.Context, mexec *mongomigrate.Executor) error {
	db := mexec.DB()

	appCur, err := db.Collection(colApps).Find(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("list apps for env backfill: %w", err)
	}
	var appIDs []string
	for appCur.Next(ctx) {
		var a struct {
			ID string `bson:"_id"`
		}
		if decErr := appCur.Decode(&a); decErr != nil {
			appCur.Close(ctx)
			return fmt.Errorf("decode app: %w", decErr)
		}
		appIDs = append(appIDs, a.ID)
	}
	if curErr := appCur.Err(); curErr != nil {
		appCur.Close(ctx)
		return fmt.Errorf("iterate apps: %w", curErr)
	}
	appCur.Close(ctx)

	envs := db.Collection(colEnvironments)
	for _, appID := range appIDs {
		var existing struct {
			ID string `bson:"_id"`
		}
		findErr := envs.FindOne(ctx, bson.M{"app_id": appID, "is_default": true}).Decode(&existing)
		envID := existing.ID
		switch {
		case findErr == nil:
			// Already has one; reuse it below.
		case isNoDocuments(findErr):
			envID = id.NewEnvironmentID().String()
			t := now()
			if _, insErr := envs.InsertOne(ctx, bson.M{
				"_id": envID, "app_id": appID,
				"name": "Production", "slug": "production", "type": "production",
				"is_default": true, "color": "#ef4444",
				"created_at": t, "updated_at": t,
			}); insErr != nil {
				return fmt.Errorf("create default env for app %s: %w", appID, insErr)
			}
		default:
			return fmt.Errorf("look up default env for app %s: %w", appID, findErr)
		}

		for _, col := range envScopedCollections {
			if _, updErr := db.Collection(col).UpdateMany(ctx,
				bson.M{"app_id": appID, "$or": []bson.M{
					{"env_id": bson.M{"$exists": false}},
					{"env_id": ""},
				}},
				bson.M{"$set": bson.M{"env_id": envID}},
			); updErr != nil {
				return fmt.Errorf("backfill env_id for %s: %w", col, updErr)
			}
		}
	}
	return nil
}

// delegationLiveIndex builds the unique index that holds one live grant per
// (app, actor, subject, kind).
//
// excludeExpired chooses the partial filter. True is the current shape, which
// covers only grants that neither expire nor have been revoked. False is the
// original revoked-only shape, kept so the migration that introduced the
// change has a Down that actually restores what was there.
func delegationLiveIndex(excludeExpired bool) mongo.IndexModel {
	filter := bson.D{
		{Key: "revoked_at", Value: bson.D{{Key: "$type", Value: "null"}}},
	}
	if excludeExpired {
		filter = append(filter, bson.E{
			Key: "expires_at", Value: bson.D{{Key: "$type", Value: "null"}},
		})
	}
	return mongo.IndexModel{
		Keys: bson.D{
			{Key: "app_id", Value: 1},
			{Key: "actor_kind", Value: 1}, {Key: "actor_id", Value: 1},
			{Key: "subject_kind", Value: 1}, {Key: "subject_id", Value: 1},
			{Key: "grant_kind", Value: 1},
		},
		Options: options.Index().SetUnique(true).SetPartialFilterExpression(filter),
	}
}

// envScopedCollections are the collections whose models carry env_id.
var envScopedCollections = []string{
	colUsers,
	colSessions,
	colVerifications,
	colPasswordResets,
	colOrganizations,
	colDevices,
	colWebhooks,
	colNotifications,
	colAPIKeys,
}
