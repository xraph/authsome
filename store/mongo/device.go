package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/xraph/authsome/device"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store"
)

// CreateDevice persists a new device.
func (s *Store) CreateDevice(ctx context.Context, d *device.Device) error {
	m := toDeviceModel(d)

	_, err := s.mdb.NewInsert(m).Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: create device: %w", err)
	}

	return nil
}

// GetDevice returns a device by ID.
func (s *Store) GetDevice(ctx context.Context, deviceID id.DeviceID) (*device.Device, error) {
	var m deviceModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"_id": deviceID.String()}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get device: %w", err)
	}

	return fromDeviceModel(&m)
}

// GetDeviceByFingerprint returns a device by user ID and fingerprint.
func (s *Store) GetDeviceByFingerprint(ctx context.Context, userID id.UserID, fingerprint string) (*device.Device, error) {
	var m deviceModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{
			"user_id":     userID.String(),
			"fingerprint": fingerprint,
		}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get device by fingerprint: %w", err)
	}

	return fromDeviceModel(&m)
}

// UpdateDevice modifies an existing device.
func (s *Store) UpdateDevice(ctx context.Context, d *device.Device) error {
	m := toDeviceModel(d)
	m.UpdatedAt = now()

	res, err := s.mdb.NewUpdate(m).
		Filter(bson.M{"_id": m.ID}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: update device: %w", err)
	}

	if res.MatchedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// DeleteDevice removes a device.
func (s *Store) DeleteDevice(ctx context.Context, deviceID id.DeviceID) error {
	res, err := s.mdb.NewDelete((*deviceModel)(nil)).
		Filter(bson.M{"_id": deviceID.String()}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: delete device: %w", err)
	}

	if res.DeletedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// ListUserDevices returns all devices for a user, ordered by last seen descending.
func (s *Store) ListUserDevices(ctx context.Context, userID id.UserID) ([]*device.Device, error) {
	var models []deviceModel

	err := s.mdb.NewFind(&models).
		Filter(bson.M{"user_id": userID.String()}).
		Sort(bson.D{{Key: "last_seen_at", Value: -1}}).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("authsome/mongo: list user devices: %w", err)
	}

	result := make([]*device.Device, 0, len(models))

	for i := range models {
		d, err := fromDeviceModel(&models[i])
		if err != nil {
			return nil, err
		}

		result = append(result, d)
	}

	return result, nil
}
