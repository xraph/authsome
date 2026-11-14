package mfa

import (
	"context"
	"errors"
	"sort"
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

type trustedDeviceDoc struct {
	ID         string    `bson:"_id"`
	UserID     string    `bson:"user_id"`
	TokenHash  string    `bson:"token_hash"`
	Label      string    `bson:"label"`
	UserAgent  string    `bson:"user_agent"`
	IPAddress  string    `bson:"ip_address"`
	CreatedAt  time.Time `bson:"created_at"`
	LastUsedAt time.Time `bson:"last_used_at"`
	ExpiresAt  time.Time `bson:"expires_at"`
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
	mfaEnrollmentsColl    = "authsome_mfa_enrollments"
	mfaRecoveryCodesColl  = "authsome_mfa_recovery_codes"
	mfaTrustedDevicesColl = "authsome_mfa_trusted_devices"
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
// Trusted device store methods
// ──────────────────────────────────────────────────

func (s *MongoStore) CreateTrustedDevice(ctx context.Context, d *TrustedDevice) error {
	_, err := s.mdb.Collection(mfaTrustedDevicesColl).InsertOne(ctx, &trustedDeviceDoc{
		ID:         d.ID.String(),
		UserID:     d.UserID.String(),
		TokenHash:  d.TokenHash,
		Label:      d.Label,
		UserAgent:  d.UserAgent,
		IPAddress:  d.IPAddress,
		CreatedAt:  d.CreatedAt,
		LastUsedAt: d.LastUsedAt,
		ExpiresAt:  d.ExpiresAt,
	})
	return mfaMongoError(err)
}

func (s *MongoStore) GetTrustedDeviceByTokenHash(ctx context.Context, tokenHash string) (*TrustedDevice, error) {
	doc := new(trustedDeviceDoc)
	err := s.mdb.Collection(mfaTrustedDevicesColl).FindOne(ctx, bson.M{"token_hash": tokenHash}).Decode(doc)
	if err != nil {
		if isNoDocuments(err) {
			return nil, ErrTrustedDeviceNotFound
		}
		return nil, err
	}
	return trustedDeviceDocToTrustedDevice(doc)
}

func (s *MongoStore) ListTrustedDevices(ctx context.Context, userID id.UserID) ([]*TrustedDevice, error) {
	cursor, err := s.mdb.Collection(mfaTrustedDevicesColl).Find(ctx, bson.M{"user_id": userID.String()})
	if err != nil {
		return nil, mfaMongoError(err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []trustedDeviceDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, mfaMongoError(err)
	}
	result := make([]*TrustedDevice, 0, len(docs))
	for i := range docs {
		d, err := trustedDeviceDocToTrustedDevice(&docs[i])
		if err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return result, nil
}

func (s *MongoStore) TouchTrustedDevice(ctx context.Context, deviceID id.TrustedDeviceID, at time.Time) error {
	_, err := s.mdb.Collection(mfaTrustedDevicesColl).UpdateOne(ctx,
		bson.M{"_id": deviceID.String()},
		bson.M{"$set": bson.M{"last_used_at": at}},
	)
	return mfaMongoError(err)
}

func (s *MongoStore) DeleteTrustedDevice(ctx context.Context, userID id.UserID, deviceID id.TrustedDeviceID) error {
	_, err := s.mdb.Collection(mfaTrustedDevicesColl).DeleteOne(ctx, bson.M{
		"_id":     deviceID.String(),
		"user_id": userID.String(),
	})
	return mfaMongoError(err)
}

func (s *MongoStore) DeleteTrustedDevices(ctx context.Context, userID id.UserID) error {
	_, err := s.mdb.Collection(mfaTrustedDevicesColl).DeleteMany(ctx, bson.M{"user_id": userID.String()})
	return mfaMongoError(err)
}

func trustedDeviceDocToTrustedDevice(d *trustedDeviceDoc) (*TrustedDevice, error) {
	deviceID, err := id.ParseTrustedDeviceID(d.ID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(d.UserID)
	if err != nil {
		return nil, err
	}
	return &TrustedDevice{
		ID:         deviceID,
		UserID:     userID,
		TokenHash:  d.TokenHash,
		Label:      d.Label,
		UserAgent:  d.UserAgent,
		IPAddress:  d.IPAddress,
		CreatedAt:  d.CreatedAt,
		LastUsedAt: d.LastUsedAt,
		ExpiresAt:  d.ExpiresAt,
	}, nil
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
