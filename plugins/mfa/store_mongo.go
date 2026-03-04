package mfa

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/mongodriver"

	"github.com/xraph/authsome/id"
)

// MongoStore implements mfa.Store using the Grove MongoDB driver.
type MongoStore struct {
	db  *grove.DB
	mdb *mongodriver.MongoDB
}

// NewMongoStore creates a new MongoDB-backed MFA store.
func NewMongoStore(db *grove.DB) *MongoStore {
	return &MongoStore{
		db:  db,
		mdb: mongodriver.Unwrap(db),
	}
}

// Compile-time interface check.
var _ Store = (*MongoStore)(nil)

// ──────────────────────────────────────────────────
// Mongo document models
// ──────────────────────────────────────────────────

type enrollmentDoc struct {
	ID        string    `bson:"_id"`
	UserID    string    `bson:"user_id"`
	Method    string    `bson:"method"`
	Secret    string    `bson:"secret"`
	Verified  bool      `bson:"verified"`
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}

type recoveryCodeDoc struct {
	ID        string     `bson:"_id"`
	UserID    string     `bson:"user_id"`
	CodeHash  string     `bson:"code_hash"`
	Used      bool       `bson:"used"`
	UsedAt    *time.Time `bson:"used_at,omitempty"`
	CreatedAt time.Time  `bson:"created_at"`
}

// ──────────────────────────────────────────────────
// Converters
// ──────────────────────────────────────────────────

func enrollmentDocToEnrollment(d *enrollmentDoc) (*Enrollment, error) {
	mfaID, err := id.ParseMFAID(d.ID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(d.UserID)
	if err != nil {
		return nil, err
	}
	return &Enrollment{
		ID:        mfaID,
		UserID:    userID,
		Method:    d.Method,
		Secret:    d.Secret,
		Verified:  d.Verified,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}, nil
}

func enrollmentToDoc(e *Enrollment) *enrollmentDoc {
	return &enrollmentDoc{
		ID:        e.ID.String(),
		UserID:    e.UserID.String(),
		Method:    e.Method,
		Secret:    e.Secret,
		Verified:  e.Verified,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

func recoveryCodeDocToRecoveryCode(d *recoveryCodeDoc) (*RecoveryCode, error) {
	rcID, err := id.ParseRecoveryCodeID(d.ID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(d.UserID)
	if err != nil {
		return nil, err
	}
	return &RecoveryCode{
		ID:        rcID,
		UserID:    userID,
		CodeHash:  d.CodeHash,
		Used:      d.Used,
		UsedAt:    d.UsedAt,
		CreatedAt: d.CreatedAt,
	}, nil
}

// ──────────────────────────────────────────────────
// Collection helpers
// ──────────────────────────────────────────────────

const (
	mfaEnrollmentsColl   = "authsome_mfa_enrollments"
	mfaRecoveryCodesColl = "authsome_mfa_recovery_codes"
)

// ──────────────────────────────────────────────────
// Store methods
// ──────────────────────────────────────────────────

func (s *MongoStore) CreateEnrollment(ctx context.Context, e *Enrollment) error {
	now := time.Now()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	if e.UpdatedAt.IsZero() {
		e.UpdatedAt = now
	}
	doc := enrollmentToDoc(e)
	_, err := s.mdb.Collection(mfaEnrollmentsColl).InsertOne(ctx, doc)
	return mfaMongoError(err)
}

func (s *MongoStore) GetEnrollment(ctx context.Context, userID id.UserID, method string) (*Enrollment, error) {
	doc := new(enrollmentDoc)
	err := s.mdb.Collection(mfaEnrollmentsColl).FindOne(ctx, bson.M{
		"user_id": userID.String(),
		"method":  method,
	}).Decode(doc)
	if err != nil {
		return nil, mfaMongoError(err)
	}
	return enrollmentDocToEnrollment(doc)
}

func (s *MongoStore) GetEnrollmentByID(ctx context.Context, mfaID id.MFAID) (*Enrollment, error) {
	doc := new(enrollmentDoc)
	err := s.mdb.Collection(mfaEnrollmentsColl).FindOne(ctx, bson.M{
		"_id": mfaID.String(),
	}).Decode(doc)
	if err != nil {
		return nil, mfaMongoError(err)
	}
	return enrollmentDocToEnrollment(doc)
}

func (s *MongoStore) UpdateEnrollment(ctx context.Context, e *Enrollment) error {
	e.UpdatedAt = time.Now()
	_, err := s.mdb.Collection(mfaEnrollmentsColl).UpdateOne(ctx,
		bson.M{"_id": e.ID.String()},
		bson.M{"$set": bson.M{
			"method":     e.Method,
			"secret":     e.Secret,
			"verified":   e.Verified,
			"updated_at": e.UpdatedAt,
		}},
	)
	return mfaMongoError(err)
}

func (s *MongoStore) DeleteEnrollment(ctx context.Context, mfaID id.MFAID) error {
	_, err := s.mdb.Collection(mfaEnrollmentsColl).DeleteOne(ctx, bson.M{
		"_id": mfaID.String(),
	})
	return mfaMongoError(err)
}

func (s *MongoStore) ListEnrollments(ctx context.Context, userID id.UserID) ([]*Enrollment, error) {
	cursor, err := s.mdb.Collection(mfaEnrollmentsColl).Find(ctx, bson.M{
		"user_id": userID.String(),
	})
	if err != nil {
		return nil, mfaMongoError(err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []enrollmentDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, mfaMongoError(err)
	}

	result := make([]*Enrollment, 0, len(docs))
	for i := range docs {
		e, err := enrollmentDocToEnrollment(&docs[i])
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// Recovery code store methods
// ──────────────────────────────────────────────────

func (s *MongoStore) CreateRecoveryCodes(ctx context.Context, codes []*RecoveryCode) error {
	for _, c := range codes {
		doc := &recoveryCodeDoc{
			ID:        c.ID.String(),
			UserID:    c.UserID.String(),
			CodeHash:  c.CodeHash,
			Used:      false,
			CreatedAt: c.CreatedAt,
		}
		if _, err := s.mdb.Collection(mfaRecoveryCodesColl).InsertOne(ctx, doc); err != nil {
			return mfaMongoError(err)
		}
	}
	return nil
}

func (s *MongoStore) GetRecoveryCodes(ctx context.Context, userID id.UserID) ([]*RecoveryCode, error) {
	cursor, err := s.mdb.Collection(mfaRecoveryCodesColl).Find(ctx, bson.M{
		"user_id": userID.String(),
	})
	if err != nil {
		return nil, mfaMongoError(err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []recoveryCodeDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, mfaMongoError(err)
	}

	result := make([]*RecoveryCode, 0, len(docs))
	for i := range docs {
		rc, err := recoveryCodeDocToRecoveryCode(&docs[i])
		if err != nil {
			return nil, err
		}
		result = append(result, rc)
	}
	return result, nil
}

func (s *MongoStore) ConsumeRecoveryCode(ctx context.Context, codeID id.RecoveryCodeID) error {
	now := time.Now()
	_, err := s.mdb.Collection(mfaRecoveryCodesColl).UpdateOne(ctx,
		bson.M{"_id": codeID.String(), "used": false},
		bson.M{"$set": bson.M{
			"used":    true,
			"used_at": now,
		}},
	)
	return mfaMongoError(err)
}

func (s *MongoStore) DeleteRecoveryCodes(ctx context.Context, userID id.UserID) error {
	_, err := s.mdb.Collection(mfaRecoveryCodesColl).DeleteMany(ctx, bson.M{
		"user_id": userID.String(),
	})
	return mfaMongoError(err)
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func isNoDocuments(err error) bool {
	return errors.Is(err, mongo.ErrNoDocuments) || strings.Contains(err.Error(), "no documents")
}

func mfaMongoError(err error) error {
	if err == nil {
		return nil
	}
	if isNoDocuments(err) {
		return ErrEnrollmentNotFound
	}
	return err
}
