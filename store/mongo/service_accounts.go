package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/serviceaccount"
	"github.com/xraph/authsome/store"
)

// CreateServiceAccount persists a new service account.
func (s *Store) CreateServiceAccount(ctx context.Context, svc *serviceaccount.ServiceAccount) error {
	m := toServiceAccountModel(svc)

	_, err := s.mdb.NewInsert(m).Exec(ctx)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return store.ErrConflict
		}
		return fmt.Errorf("authsome/mongo: create service account: %w", err)
	}

	return nil
}

// GetServiceAccount returns a service account by ID.
func (s *Store) GetServiceAccount(ctx context.Context, svcID id.ServiceAccountID) (*serviceaccount.ServiceAccount, error) {
	var m serviceAccountModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"_id": svcID.String()}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("authsome/mongo: get service account: %w", err)
	}

	return fromServiceAccountModel(&m)
}

// ListServiceAccounts returns service accounts matching the query.
func (s *Store) ListServiceAccounts(ctx context.Context, q *serviceaccount.Query) (*serviceaccount.List, error) {
	filter := bson.M{"app_id": q.AppID.String()}
	if q.Active != nil {
		filter["active"] = *q.Active
	}

	var models []serviceAccountModel

	finder := s.mdb.NewFind(&models).
		Filter(filter).
		Sort(bson.D{{Key: "created_at", Value: -1}})

	if q.Limit > 0 {
		finder = finder.Limit(int64(q.Limit))
	}

	if err := finder.Scan(ctx); err != nil {
		return nil, fmt.Errorf("authsome/mongo: list service accounts: %w", err)
	}

	result := make([]*serviceaccount.ServiceAccount, 0, len(models))

	for i := range models {
		svc, err := fromServiceAccountModel(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, svc)
	}

	return &serviceaccount.List{
		ServiceAccounts: result,
		Total:           len(result),
	}, nil
}

// UpdateServiceAccount modifies an existing service account.
func (s *Store) UpdateServiceAccount(ctx context.Context, svc *serviceaccount.ServiceAccount) error {
	m := toServiceAccountModel(svc)
	m.UpdatedAt = now()

	res, err := s.mdb.NewUpdate(m).
		Filter(bson.M{"_id": m.ID}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: update service account: %w", err)
	}

	if res.MatchedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// DeleteServiceAccount removes a service account.
func (s *Store) DeleteServiceAccount(ctx context.Context, svcID id.ServiceAccountID) error {
	res, err := s.mdb.NewDelete((*serviceAccountModel)(nil)).
		Filter(bson.M{"_id": svcID.String()}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: delete service account: %w", err)
	}

	if res.DeletedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}
