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

				return mexec.CreateIndexes(ctx, colUsers, []mongo.IndexModel{
					{
						Keys:    bson.D{{Key: "app_id", Value: 1}, {Key: "email", Value: 1}},
						Options: options.Index().SetUnique(true),
					},
					{
						Keys:    bson.D{{Key: "app_id", Value: 1}, {Key: "username", Value: 1}},
						Options: options.Index().SetUnique(true).SetSparse(true),
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
	)
}
