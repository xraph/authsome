package geofence

import (
	"context"
	"math"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Repository defines the interface for geofence data storage
type Repository interface {
	// Rules
	CreateRule(ctx context.Context, rule *GeofenceRule) error
	GetRule(ctx context.Context, id xid.ID) (*GeofenceRule, error)
	GetRulesByOrganization(ctx context.Context, orgID xid.ID) ([]*GeofenceRule, error)
	GetRulesByUser(ctx context.Context, userID xid.ID) ([]*GeofenceRule, error)
	UpdateRule(ctx context.Context, rule *GeofenceRule) error
	DeleteRule(ctx context.Context, id xid.ID) error
	ListEnabledRules(ctx context.Context, orgID xid.ID, userID *xid.ID) ([]*GeofenceRule, error)

	// Location Events
	CreateLocationEvent(ctx context.Context, event *LocationEvent) error
	GetLocationEvent(ctx context.Context, id xid.ID) (*LocationEvent, error)
	GetUserLocationHistory(ctx context.Context, userID xid.ID, limit int) ([]*LocationEvent, error)
	GetLastLocationEvent(ctx context.Context, userID xid.ID) (*LocationEvent, error)
	DeleteOldLocationEvents(ctx context.Context, before time.Time) (int64, error)

	// Travel Alerts
	CreateTravelAlert(ctx context.Context, alert *TravelAlert) error
	GetTravelAlert(ctx context.Context, id xid.ID) (*TravelAlert, error)
	GetUserTravelAlerts(ctx context.Context, userID xid.ID, status string) ([]*TravelAlert, error)
	GetPendingTravelAlerts(ctx context.Context, orgID xid.ID) ([]*TravelAlert, error)
	UpdateTravelAlert(ctx context.Context, alert *TravelAlert) error
	ApproveTravel(ctx context.Context, alertID xid.ID, approvedBy xid.ID) error
	DenyTravel(ctx context.Context, alertID xid.ID, deniedBy xid.ID) error

	// Trusted Locations
	CreateTrustedLocation(ctx context.Context, location *TrustedLocation) error
	GetTrustedLocation(ctx context.Context, id xid.ID) (*TrustedLocation, error)
	GetUserTrustedLocations(ctx context.Context, userID xid.ID) ([]*TrustedLocation, error)
	UpdateTrustedLocation(ctx context.Context, location *TrustedLocation) error
	DeleteTrustedLocation(ctx context.Context, id xid.ID) error
	IsLocationTrusted(ctx context.Context, userID xid.ID, lat, lon float64) (bool, *TrustedLocation, error)

	// Violations
	CreateViolation(ctx context.Context, violation *GeofenceViolation) error
	GetViolation(ctx context.Context, id xid.ID) (*GeofenceViolation, error)
	GetUserViolations(ctx context.Context, userID xid.ID, limit int) ([]*GeofenceViolation, error)
	GetOrganizationViolations(ctx context.Context, orgID xid.ID, limit int) ([]*GeofenceViolation, error)
	GetUnresolvedViolations(ctx context.Context, orgID xid.ID) ([]*GeofenceViolation, error)
	ResolveViolation(ctx context.Context, id xid.ID, resolvedBy xid.ID, resolution string) error

	// Geo Cache
	GetCachedGeoData(ctx context.Context, ip string) (*GeoCache, error)
	SetCachedGeoData(ctx context.Context, cache *GeoCache) error
	DeleteExpiredCache(ctx context.Context) (int64, error)
}

// BunRepository implements Repository using Bun ORM
type BunRepository struct {
	db *bun.DB
}

// NewBunRepository creates a new Bun-based repository
func NewBunRepository(db *bun.DB) Repository {
	return &BunRepository{db: db}
}

// Rules
func (r *BunRepository) CreateRule(ctx context.Context, rule *GeofenceRule) error {
	if rule.ID.IsNil() {
		rule.ID = xid.New()
	}
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	_, err := r.db.NewInsert().Model(rule).Exec(ctx)
	return err
}

func (r *BunRepository) GetRule(ctx context.Context, id xid.ID) (*GeofenceRule, error) {
	rule := new(GeofenceRule)
	err := r.db.NewSelect().Model(rule).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return rule, nil
}

func (r *BunRepository) GetRulesByOrganization(ctx context.Context, orgID xid.ID) ([]*GeofenceRule, error) {
	var rules []*GeofenceRule
	err := r.db.NewSelect().
		Model(&rules).
		Where("organization_id = ?", orgID).
		Order("priority DESC").
		Scan(ctx)
	return rules, err
}

func (r *BunRepository) GetRulesByUser(ctx context.Context, userID xid.ID) ([]*GeofenceRule, error) {
	var rules []*GeofenceRule
	err := r.db.NewSelect().
		Model(&rules).
		Where("user_id = ?", userID).
		Order("priority DESC").
		Scan(ctx)
	return rules, err
}

func (r *BunRepository) UpdateRule(ctx context.Context, rule *GeofenceRule) error {
	rule.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().Model(rule).WherePK().Exec(ctx)
	return err
}

func (r *BunRepository) DeleteRule(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().Model((*GeofenceRule)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

func (r *BunRepository) ListEnabledRules(ctx context.Context, orgID xid.ID, userID *xid.ID) ([]*GeofenceRule, error) {
	var rules []*GeofenceRule
	query := r.db.NewSelect().
		Model(&rules).
		Where("organization_id = ?", orgID).
		Where("enabled = ?", true)

	if userID != nil {
		// Get both org-wide and user-specific rules
		query = query.Where("user_id IS NULL OR user_id = ?", userID)
	} else {
		// Only org-wide rules
		query = query.Where("user_id IS NULL")
	}

	err := query.Order("priority DESC").Scan(ctx)
	return rules, err
}

// Location Events
func (r *BunRepository) CreateLocationEvent(ctx context.Context, event *LocationEvent) error {
	if event.ID.IsNil() {
		event.ID = xid.New()
	}
	event.Timestamp = time.Now()
	_, err := r.db.NewInsert().Model(event).Exec(ctx)
	return err
}

func (r *BunRepository) GetLocationEvent(ctx context.Context, id xid.ID) (*LocationEvent, error) {
	event := new(LocationEvent)
	err := r.db.NewSelect().Model(event).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return event, nil
}

func (r *BunRepository) GetUserLocationHistory(ctx context.Context, userID xid.ID, limit int) ([]*LocationEvent, error) {
	var events []*LocationEvent
	err := r.db.NewSelect().
		Model(&events).
		Where("user_id = ?", userID).
		Order("timestamp DESC").
		Limit(limit).
		Scan(ctx)
	return events, err
}

func (r *BunRepository) GetLastLocationEvent(ctx context.Context, userID xid.ID) (*LocationEvent, error) {
	event := new(LocationEvent)
	err := r.db.NewSelect().
		Model(event).
		Where("user_id = ?", userID).
		Order("timestamp DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return event, nil
}

func (r *BunRepository) DeleteOldLocationEvents(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.NewDelete().
		Model((*LocationEvent)(nil)).
		Where("timestamp < ?", before).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Travel Alerts
func (r *BunRepository) CreateTravelAlert(ctx context.Context, alert *TravelAlert) error {
	if alert.ID.IsNil() {
		alert.ID = xid.New()
	}
	alert.CreatedAt = time.Now()
	alert.UpdatedAt = time.Now()
	_, err := r.db.NewInsert().Model(alert).Exec(ctx)
	return err
}

func (r *BunRepository) GetTravelAlert(ctx context.Context, id xid.ID) (*TravelAlert, error) {
	alert := new(TravelAlert)
	err := r.db.NewSelect().Model(alert).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return alert, nil
}

func (r *BunRepository) GetUserTravelAlerts(ctx context.Context, userID xid.ID, status string) ([]*TravelAlert, error) {
	var alerts []*TravelAlert
	query := r.db.NewSelect().
		Model(&alerts).
		Where("user_id = ?", userID).
		Order("created_at DESC")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.Scan(ctx)
	return alerts, err
}

func (r *BunRepository) GetPendingTravelAlerts(ctx context.Context, orgID xid.ID) ([]*TravelAlert, error) {
	var alerts []*TravelAlert
	err := r.db.NewSelect().
		Model(&alerts).
		Where("organization_id = ?", orgID).
		Where("status = ?", "pending").
		Where("requires_approval = ?", true).
		Order("created_at DESC").
		Scan(ctx)
	return alerts, err
}

func (r *BunRepository) UpdateTravelAlert(ctx context.Context, alert *TravelAlert) error {
	alert.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().Model(alert).WherePK().Exec(ctx)
	return err
}

func (r *BunRepository) ApproveTravel(ctx context.Context, alertID xid.ID, approvedBy xid.ID) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*TravelAlert)(nil)).
		Set("status = ?", "approved").
		Set("approved_by = ?", approvedBy).
		Set("approved_at = ?", now).
		Set("resolved_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", alertID).
		Exec(ctx)
	return err
}

func (r *BunRepository) DenyTravel(ctx context.Context, alertID xid.ID, deniedBy xid.ID) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*TravelAlert)(nil)).
		Set("status = ?", "denied").
		Set("approved_by = ?", deniedBy).
		Set("approved_at = ?", now).
		Set("resolved_at = ?", now).
		Set("resolution = ?", "denied_by_admin").
		Set("updated_at = ?", now).
		Where("id = ?", alertID).
		Exec(ctx)
	return err
}

// Trusted Locations
func (r *BunRepository) CreateTrustedLocation(ctx context.Context, location *TrustedLocation) error {
	if location.ID.IsNil() {
		location.ID = xid.New()
	}
	location.CreatedAt = time.Now()
	location.UpdatedAt = time.Now()
	location.FirstUsedAt = time.Now()
	_, err := r.db.NewInsert().Model(location).Exec(ctx)
	return err
}

func (r *BunRepository) GetTrustedLocation(ctx context.Context, id xid.ID) (*TrustedLocation, error) {
	location := new(TrustedLocation)
	err := r.db.NewSelect().Model(location).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return location, nil
}

func (r *BunRepository) GetUserTrustedLocations(ctx context.Context, userID xid.ID) ([]*TrustedLocation, error) {
	var locations []*TrustedLocation
	err := r.db.NewSelect().
		Model(&locations).
		Where("user_id = ?", userID).
		Where("expires_at IS NULL OR expires_at > ?", time.Now()).
		Order("name").
		Scan(ctx)
	return locations, err
}

func (r *BunRepository) UpdateTrustedLocation(ctx context.Context, location *TrustedLocation) error {
	location.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().Model(location).WherePK().Exec(ctx)
	return err
}

func (r *BunRepository) DeleteTrustedLocation(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().Model((*TrustedLocation)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

func (r *BunRepository) IsLocationTrusted(ctx context.Context, userID xid.ID, lat, lon float64) (bool, *TrustedLocation, error) {
	var locations []*TrustedLocation
	err := r.db.NewSelect().
		Model(&locations).
		Where("user_id = ?", userID).
		Where("expires_at IS NULL OR expires_at > ?", time.Now()).
		Scan(ctx)
	
	if err != nil {
		return false, nil, err
	}

	for _, loc := range locations {
		distance := haversineDistance(lat, lon, loc.Latitude, loc.Longitude)
		if distance <= loc.RadiusKm {
			// Update usage stats
			now := time.Now()
			loc.LastUsedAt = &now
			loc.UsageCount++
			_ = r.UpdateTrustedLocation(ctx, loc)
			return true, loc, nil
		}
	}

	return false, nil, nil
}

// Violations
func (r *BunRepository) CreateViolation(ctx context.Context, violation *GeofenceViolation) error {
	if violation.ID.IsNil() {
		violation.ID = xid.New()
	}
	violation.CreatedAt = time.Now()
	violation.UpdatedAt = time.Now()
	_, err := r.db.NewInsert().Model(violation).Exec(ctx)
	return err
}

func (r *BunRepository) GetViolation(ctx context.Context, id xid.ID) (*GeofenceViolation, error) {
	violation := new(GeofenceViolation)
	err := r.db.NewSelect().Model(violation).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return violation, nil
}

func (r *BunRepository) GetUserViolations(ctx context.Context, userID xid.ID, limit int) ([]*GeofenceViolation, error) {
	var violations []*GeofenceViolation
	err := r.db.NewSelect().
		Model(&violations).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Scan(ctx)
	return violations, err
}

func (r *BunRepository) GetOrganizationViolations(ctx context.Context, orgID xid.ID, limit int) ([]*GeofenceViolation, error) {
	var violations []*GeofenceViolation
	err := r.db.NewSelect().
		Model(&violations).
		Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Limit(limit).
		Scan(ctx)
	return violations, err
}

func (r *BunRepository) GetUnresolvedViolations(ctx context.Context, orgID xid.ID) ([]*GeofenceViolation, error) {
	var violations []*GeofenceViolation
	err := r.db.NewSelect().
		Model(&violations).
		Where("organization_id = ?", orgID).
		Where("resolved = ?", false).
		Order("created_at DESC").
		Scan(ctx)
	return violations, err
}

func (r *BunRepository) ResolveViolation(ctx context.Context, id xid.ID, resolvedBy xid.ID, resolution string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*GeofenceViolation)(nil)).
		Set("resolved = ?", true).
		Set("resolved_at = ?", now).
		Set("resolved_by = ?", resolvedBy).
		Set("resolution = ?", resolution).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// Geo Cache
func (r *BunRepository) GetCachedGeoData(ctx context.Context, ip string) (*GeoCache, error) {
	cache := new(GeoCache)
	err := r.db.NewSelect().
		Model(cache).
		Where("ip_address = ?", ip).
		Where("expires_at > ?", time.Now()).
		Scan(ctx)
	
	if err != nil {
		return nil, err
	}

	// Update hit count
	cache.HitCount++
	_, _ = r.db.NewUpdate().
		Model(cache).
		Set("hit_count = ?", cache.HitCount).
		WherePK().
		Exec(ctx)

	return cache, nil
}

func (r *BunRepository) SetCachedGeoData(ctx context.Context, cache *GeoCache) error {
	cache.CachedAt = time.Now()
	cache.HitCount = 0

	// Upsert
	_, err := r.db.NewInsert().
		Model(cache).
		On("CONFLICT (ip_address) DO UPDATE").
		Set("country = EXCLUDED.country").
		Set("country_code = EXCLUDED.country_code").
		Set("region = EXCLUDED.region").
		Set("city = EXCLUDED.city").
		Set("latitude = EXCLUDED.latitude").
		Set("longitude = EXCLUDED.longitude").
		Set("accuracy_km = EXCLUDED.accuracy_km").
		Set("is_vpn = EXCLUDED.is_vpn").
		Set("is_proxy = EXCLUDED.is_proxy").
		Set("is_tor = EXCLUDED.is_tor").
		Set("is_datacenter = EXCLUDED.is_datacenter").
		Set("vpn_provider = EXCLUDED.vpn_provider").
		Set("fraud_score = EXCLUDED.fraud_score").
		Set("asn = EXCLUDED.asn").
		Set("isp = EXCLUDED.isp").
		Set("organization = EXCLUDED.organization").
		Set("provider = EXCLUDED.provider").
		Set("cached_at = EXCLUDED.cached_at").
		Set("expires_at = EXCLUDED.expires_at").
		Exec(ctx)

	return err
}

func (r *BunRepository) DeleteExpiredCache(ctx context.Context) (int64, error) {
	result, err := r.db.NewDelete().
		Model((*GeoCache)(nil)).
		Where("expires_at < ?", time.Now()).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// haversineDistance calculates distance between two coordinates in kilometers
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0

	lat1Rad := lat1 * math.Pi / 180.0
	lat2Rad := lat2 * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

