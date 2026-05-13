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

	// Get total count using the base filter (before cursor filtering).
	total, err := s.mdb.Collection(colServiceAccounts).CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("authsome/mongo: list service accounts count: %w", err)
	}

	// Apply cursor: the cursor is the ID of the last item from the previous page.
	// We use $lt for descending ID order (matching the pattern used in user listing).
	if q.Cursor != "" {
		filter["_id"] = bson.M{"$lt": q.Cursor}
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}

	var models []serviceAccountModel

	if err := s.mdb.NewFind(&models).
		Filter(filter).
		Sort(bson.D{{Key: "_id", Value: -1}}).
		Limit(int64(limit + 1)).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("authsome/mongo: list service accounts: %w", err)
	}

	list := &serviceaccount.List{
		ServiceAccounts: make([]*serviceaccount.ServiceAccount, 0, len(models)),
		Total:           int(total),
	}

	for i := range models {
		if i >= limit {
			list.NextCursor = models[i].ID
			break
		}

		svc, err := fromServiceAccountModel(&models[i])
		if err != nil {
			return nil, err
		}
		list.ServiceAccounts = append(list.ServiceAccounts, svc)
	}

	return list, nil
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
