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

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/organization"
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

// Verifications, password resets and invitations follow the session
// contract: token_hash carries store.HashToken of the plaintext, the token
// field stays empty on new documents and holds plaintext only on documents
// older than hash_credential_tokens, which the first lookup rewrites.

// credentialTokenIndexes is the index shape hash_credential_tokens installs
// and ensureBaselineIndexes keeps for one of the three collections.
// Verification codes may repeat across users (6-digit OTPs), so their hash
// index is plain; reset and invitation tokens are long and random, so theirs
// are unique over present, non-empty hashes.
func credentialTokenIndexes(unique bool) []mongo.IndexModel {
	hashOpts := options.Index()
	if unique {
		hashOpts = hashOpts.SetUnique(true).
			SetPartialFilterExpression(bson.M{"token_hash": bson.M{"$gt": ""}})
	}
	return []mongo.IndexModel{
		{Keys: bson.D{{Key: "token", Value: 1}}},
		{Keys: bson.D{{Key: "token_hash", Value: 1}}, Options: hashOpts},
	}
}

// findByToken loads one document by the hash of the presented plaintext,
// falling back to the legacy plaintext field. It reports whether the
// plaintext path matched so the caller can upgrade the document.
func (s *Store) findByToken(ctx context.Context, m any, what, plaintext string) (legacy bool, err error) {
	if plaintext == "" {
		return false, store.ErrNotFound
	}
	err = s.mdb.NewFind(m).Filter(bson.M{"token_hash": store.HashToken(plaintext)}).Scan(ctx)
	if err == nil {
		return false, nil
	}
	if !isNoDocuments(err) {
		return false, fmt.Errorf("authsome/mongo: get %s by token: %w", what, err)
	}
	if plainErr := s.mdb.NewFind(m).Filter(bson.M{"token": plaintext}).Scan(ctx); plainErr != nil {
		if isNoDocuments(plainErr) {
			return false, store.ErrNotFound
		}
		return false, fmt.Errorf("authsome/mongo: get %s by token: %w", what, plainErr)
	}
	return true, nil
}

// upgradeLegacyToken rewrites one plaintext document of model's collection
// as a hash. A document another request upgraded first is left alone.
func (s *Store) upgradeLegacyToken(ctx context.Context, model any, docID, plaintext string) error {
	_, err := s.mdb.NewUpdate(model).
		Filter(bson.M{"_id": docID, "token_hash": bson.M{"$in": bson.A{nil, ""}}}).
		Set("token_hash", store.HashToken(plaintext)).
		Set("token", "").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: hash legacy token: %w", err)
	}
	return nil
}

// consumeByToken flips consumed on the document behind plaintext, hashed or
// legacy, and reports whether one was still unconsumed.
func (s *Store) consumeByToken(ctx context.Context, model any, what, plaintext string) (bool, error) {
	if plaintext == "" {
		return false, nil
	}
	res, err := s.mdb.NewUpdate(model).
		Filter(bson.M{
			"$or":      bson.A{bson.M{"token_hash": store.HashToken(plaintext)}, bson.M{"token": plaintext}},
			"consumed": false,
		}).
		Set("consumed", true).
		Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("authsome/mongo: consume %s: %w", what, err)
	}
	return res.MatchedCount() > 0, nil
}

// SeedLegacyVerification, SeedLegacyPasswordReset and SeedLegacyInvitation
// write documents the way releases before token hashing did: plaintext and
// no hash. They exist for the conformance suite and are refused outside
// `go test`.
func (s *Store) SeedLegacyVerification(ctx context.Context, v *account.Verification) error {
	if !testing.Testing() {
		return errors.New("authsome/mongo: SeedLegacyVerification is a test seam")
	}
	m := toVerificationModel(v)
	m.Token, m.TokenHash = v.Token, ""
	if _, err := s.mdb.NewInsert(m).Exec(ctx); err != nil {
		return fmt.Errorf("authsome/mongo: seed legacy verification: %w", err)
	}
	return nil
}

func (s *Store) SeedLegacyPasswordReset(ctx context.Context, pr *account.PasswordReset) error {
	if !testing.Testing() {
		return errors.New("authsome/mongo: SeedLegacyPasswordReset is a test seam")
	}
	m := toPasswordResetModel(pr)
	m.Token, m.TokenHash = pr.Token, ""
	if _, err := s.mdb.NewInsert(m).Exec(ctx); err != nil {
		return fmt.Errorf("authsome/mongo: seed legacy password reset: %w", err)
	}
	return nil
}

func (s *Store) SeedLegacyInvitation(ctx context.Context, inv *organization.Invitation) error {
	if !testing.Testing() {
		return errors.New("authsome/mongo: SeedLegacyInvitation is a test seam")
	}
	m := toInvitationModel(inv)
	m.Token, m.TokenHash = inv.Token, ""
	if _, err := s.mdb.NewInsert(m).Exec(ctx); err != nil {
		return fmt.Errorf("authsome/mongo: seed legacy invitation: %w", err)
	}
	return nil
}

func init() {
	Migrations.MustRegister(&migrate.Migration{
		Name:    "hash_credential_tokens",
		Version: "20260922000003",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			mexec, ok := exec.(*mongomigrate.Executor)
			if !ok {
				return fmt.Errorf("expected mongomigrate executor, got %T", exec)
			}
			for _, c := range []struct {
				col    string
				model  any
				unique bool
			}{
				{colVerifications, (*verificationModel)(nil), false},
				{colPasswordResets, (*passwordResetModel)(nil), true},
				{colInvitations, (*invitationModel)(nil), true},
			} {
				if err := mexec.RefreshValidator(ctx, c.model); err != nil {
					return fmt.Errorf("refresh %s validator: %w", c.col, err)
				}
				// The unique plaintext index has to go before the plain one
				// can take its name. Tolerate IndexNotFound so a re-run is
				// safe.
				coll := mexec.DB().Collection(c.col)
				if err := coll.Indexes().DropOne(ctx, "token_1"); err != nil && !mongoIsIndexNotFound(err) {
					return fmt.Errorf("drop unique %s token index: %w", c.col, err)
				}
				if err := mexec.CreateIndexes(ctx, c.col, credentialTokenIndexes(c.unique)); err != nil {
					return err
				}
			}
			return nil
		},
		Down: func(ctx context.Context, exec migrate.Executor) error {
			mexec, ok := exec.(*mongomigrate.Executor)
			if !ok {
				return fmt.Errorf("expected mongomigrate executor, got %T", exec)
			}
			for _, col := range []string{colVerifications, colPasswordResets, colInvitations} {
				if err := mexec.DB().Collection(col).Indexes().DropOne(ctx, "token_hash_1"); err != nil && !mongoIsIndexNotFound(err) {
					return fmt.Errorf("drop %s token_hash index: %w", col, err)
				}
			}
			// Forward-only on the plaintext indexes: restoring uniqueness
			// would refuse every document this migration made storable.
			return nil
		},
	})
}

// HashLegacyTokens implements store.LegacyTokenHasher. Each collection is
// swept with a bounded find of documents whose hash is absent or empty and
// whose plaintext is present, then rewritten one by one. A document with
// neither plaintext nor hash is unusable and left alone.
func (s *Store) HashLegacyTokens(ctx context.Context, batch int) (int64, error) {
	if batch <= 0 {
		batch = 500
	}
	noHash := bson.M{"$in": bson.A{nil, ""}}
	var n int64

	var sessions []sessionModel
	if err := s.mdb.NewFind(&sessions).
		Filter(bson.M{"$or": bson.A{
			bson.M{"token_hash": noHash, "token": bson.M{"$gt": ""}},
			bson.M{"refresh_token_hash": noHash, "refresh_token": bson.M{"$gt": ""}},
		}}).
		Limit(int64(batch)).Scan(ctx); err != nil && !isNoDocuments(err) {
		return n, fmt.Errorf("authsome/mongo: list legacy sessions: %w", err)
	}
	for i := range sessions {
		m := &sessions[i]
		_, err := s.mdb.NewUpdate((*sessionModel)(nil)).
			Filter(bson.M{"_id": m.ID}).
			Set("token_hash", hashOrDerive(m.TokenHash, m.Token)).
			Set("refresh_token_hash", hashOrDerive(m.RefreshTokenHash, m.RefreshToken)).
			Set("token", "").
			Set("refresh_token", "").
			Exec(ctx)
		if err != nil {
			return n, fmt.Errorf("authsome/mongo: hash legacy session: %w", err)
		}
		n++
	}

	legacyFilter := bson.M{"token_hash": noHash, "token": bson.M{"$gt": ""}}
	for _, t := range []struct {
		name  string
		rows  func() ([]legacyRow, error)
		model any
	}{
		{"verification", func() ([]legacyRow, error) {
			var ms []verificationModel
			err := s.mdb.NewFind(&ms).Filter(legacyFilter).Limit(int64(batch) - n).Scan(ctx)
			rows := make([]legacyRow, 0, len(ms))
			for i := range ms {
				rows = append(rows, legacyRow{ms[i].ID, ms[i].Token})
			}
			return rows, err
		}, (*verificationModel)(nil)},
		{"password reset", func() ([]legacyRow, error) {
			var ms []passwordResetModel
			err := s.mdb.NewFind(&ms).Filter(legacyFilter).Limit(int64(batch) - n).Scan(ctx)
			rows := make([]legacyRow, 0, len(ms))
			for i := range ms {
				rows = append(rows, legacyRow{ms[i].ID, ms[i].Token})
			}
			return rows, err
		}, (*passwordResetModel)(nil)},
		{"invitation", func() ([]legacyRow, error) {
			var ms []invitationModel
			err := s.mdb.NewFind(&ms).Filter(legacyFilter).Limit(int64(batch) - n).Scan(ctx)
			rows := make([]legacyRow, 0, len(ms))
			for i := range ms {
				rows = append(rows, legacyRow{ms[i].ID, ms[i].Token})
			}
			return rows, err
		}, (*invitationModel)(nil)},
	} {
		if n >= int64(batch) {
			return n, nil
		}
		rows, err := t.rows()
		if err != nil && !isNoDocuments(err) {
			return n, fmt.Errorf("authsome/mongo: list legacy %ss: %w", t.name, err)
		}
		for _, r := range rows {
			if err := s.upgradeLegacyToken(ctx, t.model, r.id, r.token); err != nil {
				return n, err
			}
			n++
		}
	}
	return n, nil
}

// legacyRow is one plaintext credential document awaiting conversion.
type legacyRow struct {
	id    string
	token string
}

func init() {
	Migrations.MustRegister(&migrate.Migration{
		Name:    "session_client_id",
		Version: "20260922000004",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			mexec, ok := exec.(*mongomigrate.Executor)
			if !ok {
				return fmt.Errorf("expected mongomigrate executor, got %T", exec)
			}
			// The collection's validator is generated from the model, so the
			// new field has to be admitted before any OAuth2 token is stored.
			return mexec.RefreshValidator(ctx, (*sessionModel)(nil))
		},
		Down: func(_ context.Context, _ migrate.Executor) error {
			return nil // forward-only on the validator, as the other field additions are
		},
	})
}
