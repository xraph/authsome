package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
)

// CreateUser persists a new user.
func (s *Store) CreateUser(ctx context.Context, u *user.User) error {
	m := toUserModel(u)

	_, err := s.mdb.NewInsert(m).Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: create user: %w", err)
	}

	return nil
}

// GetUser returns a user by ID, excluding soft-deleted users.
func (s *Store) GetUser(ctx context.Context, userID id.UserID) (*user.User, error) {
	var m userModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"_id": userID.String(), "deleted_at": nil}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get user: %w", err)
	}

	return fromUserModel(&m)
}

// GetUserByEmail returns a user by app ID and email, excluding soft-deleted users.
func (s *Store) GetUserByEmail(ctx context.Context, appID id.AppID, email string) (*user.User, error) {
	var m userModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{
			"app_id":     appID.String(),
			"email":      email,
			"deleted_at": nil,
		}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get user by email: %w", err)
	}

	return fromUserModel(&m)
}

// GetUserByPhone returns a user by app ID and phone number, excluding soft-deleted users.
func (s *Store) GetUserByPhone(ctx context.Context, appID id.AppID, phone string) (*user.User, error) {
	var m userModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{
			"app_id":     appID.String(),
			"phone":      phone,
			"deleted_at": nil,
		}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get user by phone: %w", err)
	}

	return fromUserModel(&m)
}

// GetUserByUsername returns a user by app ID and username, excluding soft-deleted users.
func (s *Store) GetUserByUsername(ctx context.Context, appID id.AppID, username string) (*user.User, error) {
	var m userModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{
			"app_id":     appID.String(),
			"username":   username,
			"deleted_at": nil,
		}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get user by username: %w", err)
	}

	return fromUserModel(&m)
}

// UpdateUser modifies an existing user.
func (s *Store) UpdateUser(ctx context.Context, u *user.User) error {
	m := toUserModel(u)
	m.UpdatedAt = now()

	res, err := s.mdb.NewUpdate(m).
		Filter(bson.M{"_id": m.ID}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: update user: %w", err)
	}

	if res.MatchedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// DeleteUser soft-deletes a user by setting the deleted_at timestamp.
func (s *Store) DeleteUser(ctx context.Context, userID id.UserID) error {
	t := now()

	res, err := s.mdb.NewUpdate((*userModel)(nil)).
		Filter(bson.M{"_id": userID.String(), "deleted_at": nil}).
		Set("deleted_at", t).
		Set("updated_at", t).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: delete user: %w", err)
	}

	if res.MatchedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// ListUsers returns a paginated list of users for an app, with optional email search.
func (s *Store) ListUsers(ctx context.Context, q *user.Query) (*user.List, error) {
	var models []userModel

	filter := bson.M{
		"app_id":     q.AppID.String(),
		"deleted_at": nil,
	}

	if !q.EnvID.IsNil() {
		filter["env_id"] = q.EnvID.String()
	}

	if q.Email != "" {
		filter["email"] = bson.M{"$regex": q.Email, "$options": "i"}
	}

	if q.Cursor != "" {
		filter["_id"] = bson.M{"$lt": q.Cursor}
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}

	err := s.mdb.NewFind(&models).
		Filter(filter).
		Sort(bson.D{{Key: "_id", Value: -1}}).
		Limit(int64(limit + 1)).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("authsome/mongo: list users: %w", err)
	}

	list := &user.List{
		Users: make([]*user.User, 0, len(models)),
	}

	for i := range models {
		if i >= limit {
			list.NextCursor = models[i].ID
			break
		}

		u, err := fromUserModel(&models[i])
		if err != nil {
			return nil, err
		}

		list.Users = append(list.Users, u)
	}

	return list, nil
}
