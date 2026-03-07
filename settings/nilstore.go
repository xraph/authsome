package settings

import "context"

// NilStore is a no-op settings store used when the backing store
// does not support settings. All reads return nil (causing the manager
// to fall through to code defaults), and all writes are silently discarded.
type NilStore struct{}

var _ Store = NilStore{}

func (NilStore) GetSetting(_ context.Context, _ string, _ Scope, _ string) (*Setting, error) {
	return nil, ErrNotFound
}

func (NilStore) SetSetting(_ context.Context, _ *Setting) error {
	return nil
}

func (NilStore) DeleteSetting(_ context.Context, _ string, _ Scope, _ string) error {
	return nil
}

func (NilStore) ListSettings(_ context.Context, _ ListOpts) ([]*Setting, error) {
	return nil, nil
}

func (NilStore) ResolveSettings(_ context.Context, _ string, _ ResolveOpts) ([]*Setting, error) {
	return nil, nil
}

func (NilStore) BatchResolve(_ context.Context, _ []string, _ ResolveOpts) (map[string][]*Setting, error) {
	return nil, nil
}

func (NilStore) DeleteSettingsByNamespace(_ context.Context, _ string) error {
	return nil
}
