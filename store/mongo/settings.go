package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
)

// ──────────────────────────────────────────────────
// Settings store
// ──────────────────────────────────────────────────

func (s *Store) GetSetting(ctx context.Context, key string, scope settings.Scope, scopeID string) (*settings.Setting, error) {
	var m settingModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{
			"key":      key,
			"scope":    string(scope),
			"scope_id": scopeID,
		}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("authsome/mongo: get setting: %w", err)
	}

	return fromSettingModel(&m)
}

func (s *Store) SetSetting(ctx context.Context, st *settings.Setting) error {
	m := toSettingModel(st)

	// Try update first; if no match, insert.
	res, err := s.mdb.NewUpdate(m).
		Filter(bson.M{
			"key":      m.Key,
			"scope":    m.Scope,
			"scope_id": m.ScopeID,
		}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: set setting (update): %w", err)
	}

	if res.MatchedCount() > 0 {
		return nil
	}

	_, err = s.mdb.NewInsert(m).Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: set setting (insert): %w", err)
	}

	return nil
}

func (s *Store) DeleteSetting(ctx context.Context, key string, scope settings.Scope, scopeID string) error {
	_, err := s.mdb.NewDelete((*settingModel)(nil)).
		Filter(bson.M{
			"key":      key,
			"scope":    string(scope),
			"scope_id": scopeID,
		}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: delete setting: %w", err)
	}

	return nil
}

func (s *Store) ListSettings(ctx context.Context, opts settings.ListOpts) ([]*settings.Setting, error) {
	var models []settingModel

	filter := bson.M{}
	if opts.Namespace != "" {
		filter["namespace"] = opts.Namespace
	}
	if opts.Scope != "" {
		filter["scope"] = string(opts.Scope)
	}
	if opts.ScopeID != "" {
		filter["scope_id"] = opts.ScopeID
	}
	if opts.AppID != "" {
		filter["app_id"] = opts.AppID
	}
	if opts.OrgID != "" {
		filter["org_id"] = opts.OrgID
	}

	q := s.mdb.NewFind(&models).
		Filter(filter).
		Sort(bson.D{{Key: "created_at", Value: -1}})

	if opts.Limit > 0 {
		q = q.Limit(int64(opts.Limit))
	}
	if opts.Offset > 0 {
		q = q.Skip(int64(opts.Offset))
	}

	err := q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("authsome/mongo: list settings: %w", err)
	}

	result := make([]*settings.Setting, 0, len(models))
	for i := range models {
		st, err := fromSettingModel(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, st)
	}

	return result, nil
}

func (s *Store) ResolveSettings(ctx context.Context, key string, opts settings.ResolveOpts) ([]*settings.Setting, error) {
	var models []settingModel

	// Build OR conditions for each scope level.
	orConditions := bson.A{
		bson.M{"key": key, "scope": "global", "scope_id": ""},
	}

	if opts.AppID != "" {
		orConditions = append(orConditions, bson.M{
			"key": key, "scope": "app", "scope_id": opts.AppID,
		})
	}

	if opts.OrgID != "" {
		orConditions = append(orConditions, bson.M{
			"key": key, "scope": "org", "scope_id": opts.OrgID, "app_id": opts.AppID,
		})
	}

	if opts.UserID != "" {
		orConditions = append(orConditions, bson.M{
			"key": key, "scope": "user", "scope_id": opts.UserID,
			"app_id": opts.AppID, "org_id": opts.OrgID,
		})
	}

	err := s.mdb.NewFind(&models).
		Filter(bson.M{"$or": orConditions}).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("authsome/mongo: resolve settings: %w", err)
	}

	// Sort by scope priority (global=1, app=2, org=3, user=4).
	result := make([]*settings.Setting, 0, len(models))
	for i := range models {
		st, err := fromSettingModel(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, st)
	}

	// Sort by scope priority.
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if settings.ScopePriority(result[i].Scope) > settings.ScopePriority(result[j].Scope) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result, nil
}

func (s *Store) BatchResolve(ctx context.Context, keys []string, opts settings.ResolveOpts) (map[string][]*settings.Setting, error) {
	result := make(map[string][]*settings.Setting, len(keys))
	for _, key := range keys {
		resolved, err := s.ResolveSettings(ctx, key, opts)
		if err != nil {
			return nil, err
		}
		if len(resolved) > 0 {
			result[key] = resolved
		}
	}
	return result, nil
}

func (s *Store) DeleteSettingsByNamespace(ctx context.Context, namespace string) error {
	_, err := s.mdb.NewDelete((*settingModel)(nil)).
		Filter(bson.M{"namespace": namespace}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: delete settings by namespace: %w", err)
	}
	return nil
}
