package mongo

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/xraph/grove/drivers/mongodriver/mongomigrate"
	"github.com/xraph/grove/migrate"

	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
)

// Sessions are stored as hashes. token_hash and refresh_token_hash carry
// store.HashToken of the plaintext; the legacy token and refresh_token
// fields stay in the documents, empty on every one written after this
// release, so a lookup can fall back to them for documents written before it
// and rewrite those documents on first use.

// hashOrDerive returns the stored hash for a credential: the hash the caller
// already carries, or the digest of the plaintext when only that is known.
func hashOrDerive(hash, plaintext string) string {
	if hash == "" && plaintext != "" {
		return store.HashToken(plaintext)
	}
	return hash
}

// sessionTokenIndexes is the index shape hash_session_tokens installs and
// ensureBaselineIndexes keeps. The plaintext fields lose their uniqueness
// because every new document leaves them empty; the hash indexes ignore
// empty and missing values so legacy documents do not collide.
func sessionTokenIndexes() []mongo.IndexModel {
	return []mongo.IndexModel{
		{Keys: bson.D{{Key: "token", Value: 1}}},
		{Keys: bson.D{{Key: "refresh_token", Value: 1}}},
		{
			Keys: bson.D{{Key: "token_hash", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"token_hash": bson.M{"$gt": ""}}),
		},
		{
			Keys: bson.D{{Key: "refresh_token_hash", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"refresh_token_hash": bson.M{"$gt": ""}}),
		},
	}
}

// findSessionByCredential resolves a session by the hash of the presented
// plaintext, falling back to the legacy plaintext field and upgrading the
// document when that is where the match came from.
func (s *Store) findSessionByCredential(ctx context.Context, hashField, plainField, plaintext string) (*sessionModel, error) {
	if plaintext == "" {
		return nil, store.ErrNotFound
	}
	var m sessionModel
	err := s.mdb.NewFind(&m).Filter(bson.M{hashField: store.HashToken(plaintext)}).Scan(ctx)
	if err == nil {
		return &m, nil
	}
	if !isNoDocuments(err) {
		return nil, fmt.Errorf("authsome/mongo: get session by %s: %w", hashField, err)
	}
	m = sessionModel{}
	if err := s.mdb.NewFind(&m).Filter(bson.M{plainField: plaintext}).Scan(ctx); err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("authsome/mongo: get session by %s: %w", plainField, err)
	}
	if err := s.upgradeLegacySession(ctx, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// upgradeLegacySession rewrites a plaintext document as hashes. A document
// another request upgraded first is left alone: the filter on an empty hash
// makes the write a no-op, and m is updated the same way either way.
func (s *Store) upgradeLegacySession(ctx context.Context, m *sessionModel) error {
	tokenHash := hashOrDerive("", m.Token)
	refreshHash := hashOrDerive("", m.RefreshToken)
	_, err := s.mdb.NewUpdate((*sessionModel)(nil)).
		Filter(bson.M{"_id": m.ID, "token_hash": bson.M{"$in": bson.A{nil, ""}}}).
		Set("token_hash", tokenHash).
		Set("refresh_token_hash", refreshHash).
		Set("token", "").
		Set("refresh_token", "").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: hash legacy session: %w", err)
	}
	m.TokenHash = tokenHash
	m.RefreshTokenHash = refreshHash
	m.Token = ""
	m.RefreshToken = ""
	return nil
}

// SeedLegacySession writes sess exactly as a release before token hashing
// did: plaintext tokens and no hashes. It exists so the conformance suite can
// prove the upgrade path and is refused outside `go test`.
func (s *Store) SeedLegacySession(ctx context.Context, sess *session.Session) error {
	if !testing.Testing() {
		return errors.New("authsome/mongo: SeedLegacySession is a test seam")
	}
	m := toSessionModel(sess)
	m.Token = sess.Token
	m.RefreshToken = sess.RefreshToken
	m.TokenHash = ""
	m.RefreshTokenHash = ""
	if _, err := s.mdb.NewInsert(m).Exec(ctx); err != nil {
		return fmt.Errorf("authsome/mongo: seed legacy session: %w", err)
	}
	return nil
}

func init() {
	Migrations.MustRegister(&migrate.Migration{
		Name:    "hash_session_tokens",
		Version: "20260922000002",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			mexec, ok := exec.(*mongomigrate.Executor)
			if !ok {
				return fmt.Errorf("expected mongomigrate executor, got %T", exec)
			}
			if err := mexec.RefreshValidator(ctx, (*sessionModel)(nil)); err != nil {
				return fmt.Errorf("refresh session validator: %w", err)
			}
			// The unique plaintext indexes have to go before the plain ones
			// can take their names. Tolerate IndexNotFound so a re-run on a
			// deployment that already dropped them is safe.
			coll := mexec.DB().Collection(colSessions)
			for _, name := range []string{"token_1", "refresh_token_1"} {
				if err := coll.Indexes().DropOne(ctx, name); err != nil && !mongoIsIndexNotFound(err) {
					return fmt.Errorf("drop unique %s index: %w", name, err)
				}
			}
			return mexec.CreateIndexes(ctx, colSessions, sessionTokenIndexes())
		},
		Down: func(ctx context.Context, exec migrate.Executor) error {
			mexec, ok := exec.(*mongomigrate.Executor)
			if !ok {
				return fmt.Errorf("expected mongomigrate executor, got %T", exec)
			}
			coll := mexec.DB().Collection(colSessions)
			for _, name := range []string{"token_hash_1", "refresh_token_hash_1"} {
				if err := coll.Indexes().DropOne(ctx, name); err != nil && !mongoIsIndexNotFound(err) {
					return fmt.Errorf("drop %s index: %w", name, err)
				}
			}
			// Forward-only on the plaintext indexes: restoring uniqueness
			// would refuse every document this migration made storable.
			return nil
		},
	})
}
