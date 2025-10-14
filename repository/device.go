package repository

import (
    "context"
    "github.com/rs/xid"
    "github.com/uptrace/bun"
    core "github.com/xraph/authsome/core/device"
    "github.com/xraph/authsome/schema"
)

// DeviceRepository implements core device repository using Bun
type DeviceRepository struct{ db *bun.DB }

func NewDeviceRepository(db *bun.DB) *DeviceRepository { return &DeviceRepository{db: db} }

func (r *DeviceRepository) toSchema(d *core.Device) *schema.Device {
    return &schema.Device{
        ID:          d.ID,
        UserID:      d.UserID,
        Fingerprint: d.Fingerprint,
        UserAgent:   d.UserAgent,
        IPAddress:   d.IPAddress,
        LastActive:  d.LastActive,
    }
}

func (r *DeviceRepository) fromSchema(sd *schema.Device) *core.Device {
    if sd == nil { return nil }
    return &core.Device{
        ID:          sd.ID,
        UserID:      sd.UserID,
        Fingerprint: sd.Fingerprint,
        UserAgent:   sd.UserAgent,
        IPAddress:   sd.IPAddress,
        LastActive:  sd.LastActive,
        CreatedAt:   sd.CreatedAt,
        UpdatedAt:   sd.UpdatedAt.Time,
    }
}

func (r *DeviceRepository) Create(ctx context.Context, d *core.Device) error {
    sd := r.toSchema(d)
    _, err := r.db.NewInsert().Model(sd).Exec(ctx)
    return err
}

func (r *DeviceRepository) Update(ctx context.Context, d *core.Device) error {
    sd := r.toSchema(d)
    _, err := r.db.NewUpdate().Model(sd).Where("fingerprint = ?", d.Fingerprint).Exec(ctx)
    return err
}

func (r *DeviceRepository) FindByFingerprint(ctx context.Context, userID xid.ID, fingerprint string) (*core.Device, error) {
    sd := new(schema.Device)
    err := r.db.NewSelect().Model(sd).Where("user_id = ? AND fingerprint = ?", userID, fingerprint).Scan(ctx)
    if err != nil { return nil, err }
    return r.fromSchema(sd), nil
}

func (r *DeviceRepository) ListByUser(ctx context.Context, userID xid.ID, limit, offset int) ([]*core.Device, error) {
    var sds []schema.Device
    err := r.db.NewSelect().Model(&sds).Where("user_id = ?", userID).OrderExpr("last_active DESC").Limit(limit).Offset(offset).Scan(ctx)
    if err != nil { return nil, err }
    res := make([]*core.Device, 0, len(sds))
    for i := range sds { res = append(res, r.fromSchema(&sds[i])) }
    return res, nil
}

func (r *DeviceRepository) DeleteByFingerprint(ctx context.Context, userID xid.ID, fingerprint string) error {
    _, err := r.db.NewDelete().Model((*schema.Device)(nil)).Where("user_id = ? AND fingerprint = ?", userID, fingerprint).Exec(ctx)
    return err
}