// Package mongo implements the AuthSome store interface using MongoDB via Grove ORM.
package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/mongodriver"
	"github.com/xraph/grove/migrate"

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
	colAPIKeys           = "authsome_api_keys" //nolint:gosec // G101: not a credential
	colEnvironments      = "authsome_environments"
	colFormConfigs       = "authsome_form_configs"
	colBrandingConfigs   = "authsome_branding_configs"
	colAppSessionConfigs = "authsome_app_session_configs"
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

// Migrate creates indexes for all authsome collections.
// The extraGroups parameter is accepted for interface compatibility but is not
// used for MongoDB (Mongo does not use SQL migrations).
func (s *Store) Migrate(ctx context.Context, _ ...*migrate.Group) error {
	indexes := migrationIndexes()

	for col, models := range indexes {
		if len(models) == 0 {
			continue
		}

		_, err := s.mdb.Collection(col).Indexes().CreateMany(ctx, models)
		if err != nil {
			return fmt.Errorf("authsome/mongo: migrate %s indexes: %w", col, err)
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
				Keys:    bson.D{{Key: "app_id", Value: 1}, {Key: "username", Value: 1}},
				Options: options.Index().SetUnique(true).SetSparse(true),
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
	}
}
