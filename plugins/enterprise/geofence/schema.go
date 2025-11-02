package geofence

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// GeofenceRule represents a geographic restriction rule
type GeofenceRule struct {
	bun.BaseModel `bun:"table:geofence_rules,alias:gr"`

	ID             xid.ID    `bun:"id,pk,type:varchar(20)" json:"id"`
	OrganizationID xid.ID    `bun:"organization_id,type:varchar(20),notnull" json:"organizationId"`
	UserID         *xid.ID   `bun:"user_id,type:varchar(20)" json:"userId,omitempty"` // Null = org-wide
	Name           string    `bun:"name,notnull" json:"name"`
	Description    string    `bun:"description" json:"description"`
	Enabled        bool      `bun:"enabled,notnull" json:"enabled"`
	Priority       int       `bun:"priority,notnull" json:"priority"` // Higher = evaluated first

	// Rule Type
	RuleType string `bun:"rule_type,notnull" json:"ruleType"` // country, region, city, geofence, distance

	// Geographic Criteria (JSON)
	AllowedCountries []string `bun:"allowed_countries,type:jsonb" json:"allowedCountries,omitempty"`
	BlockedCountries []string `bun:"blocked_countries,type:jsonb" json:"blockedCountries,omitempty"`
	AllowedRegions   []string `bun:"allowed_regions,type:jsonb" json:"allowedRegions,omitempty"`
	BlockedRegions   []string `bun:"blocked_regions,type:jsonb" json:"blockedRegions,omitempty"`
	AllowedCities    []string `bun:"allowed_cities,type:jsonb" json:"allowedCities,omitempty"`
	BlockedCities    []string `bun:"blocked_cities,type:jsonb" json:"blockedCities,omitempty"`

	// Geofence Data (JSON)
	GeofenceType   string       `bun:"geofence_type" json:"geofenceType,omitempty"` // circle, polygon
	CenterLat      *float64     `bun:"center_lat" json:"centerLat,omitempty"`
	CenterLon      *float64     `bun:"center_lon" json:"centerLon,omitempty"`
	RadiusKm       *float64     `bun:"radius_km" json:"radiusKm,omitempty"`
	Coordinates    [][2]float64 `bun:"coordinates,type:jsonb" json:"coordinates,omitempty"`

	// Distance Restrictions
	MaxDistanceKm *float64 `bun:"max_distance_km" json:"maxDistanceKm,omitempty"`
	ReferencePoint *[2]float64 `bun:"reference_point,type:jsonb" json:"referencePoint,omitempty"` // [lat, lon]

	// Time Restrictions (JSON)
	TimeRestrictions []TimeRestrictionRule `bun:"time_restrictions,type:jsonb" json:"timeRestrictions,omitempty"`

	// Detection Settings
	BlockVPN        bool `bun:"block_vpn" json:"blockVpn"`
	BlockProxy      bool `bun:"block_proxy" json:"blockProxy"`
	BlockTor        bool `bun:"block_tor" json:"blockTor"`
	BlockDatacenter bool `bun:"block_datacenter" json:"blockDatacenter"`

	// Actions
	Action       string `bun:"action,notnull" json:"action"` // allow, deny, mfa_required, notify
	RequireMFA   bool   `bun:"require_mfa" json:"requireMfa"`
	NotifyUser   bool   `bun:"notify_user" json:"notifyUser"`
	NotifyAdmin  bool   `bun:"notify_admin" json:"notifyAdmin"`

	// Metadata
	CreatedAt time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time  `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
	CreatedBy xid.ID     `bun:"created_by,type:varchar(20)" json:"createdBy"`
	UpdatedBy *xid.ID    `bun:"updated_by,type:varchar(20)" json:"updatedBy,omitempty"`
}

// TimeRestrictionRule defines time-based access rules
type TimeRestrictionRule struct {
	AllowedDays []string `json:"allowedDays"` // Monday, Tuesday, etc.
	StartHour   int      `json:"startHour"`   // 0-23
	EndHour     int      `json:"endHour"`     // 0-23
	Timezone    string   `json:"timezone"`    // IANA timezone
}

// LocationEvent represents a recorded location event
type LocationEvent struct {
	bun.BaseModel `bun:"table:location_events,alias:le"`

	ID             xid.ID    `bun:"id,pk,type:varchar(20)" json:"id"`
	UserID         xid.ID    `bun:"user_id,type:varchar(20),notnull" json:"userId"`
	OrganizationID xid.ID    `bun:"organization_id,type:varchar(20),notnull" json:"organizationId"`
	SessionID      *xid.ID   `bun:"session_id,type:varchar(20)" json:"sessionId,omitempty"`

	// Location Data
	IPAddress   string   `bun:"ip_address,notnull" json:"ipAddress"`
	Country     string   `bun:"country" json:"country"`
	CountryCode string   `bun:"country_code" json:"countryCode"` // ISO 3166-1 alpha-2
	Region      string   `bun:"region" json:"region"`
	City        string   `bun:"city" json:"city"`
	Latitude    *float64 `bun:"latitude" json:"latitude,omitempty"`
	Longitude   *float64 `bun:"longitude" json:"longitude,omitempty"`
	AccuracyKm  *float64 `bun:"accuracy_km" json:"accuracyKm,omitempty"`

	// GPS Data (if available)
	GPSLatitude  *float64  `bun:"gps_latitude" json:"gpsLatitude,omitempty"`
	GPSLongitude *float64  `bun:"gps_longitude" json:"gpsLongitude,omitempty"`
	GPSAccuracy  *float64  `bun:"gps_accuracy" json:"gpsAccuracy,omitempty"` // meters
	GPSTimestamp *time.Time `bun:"gps_timestamp" json:"gpsTimestamp,omitempty"`

	// Detection Results
	IsVPN        bool   `bun:"is_vpn" json:"isVpn"`
	IsProxy      bool   `bun:"is_proxy" json:"isProxy"`
	IsTor        bool   `bun:"is_tor" json:"isTor"`
	IsDatacenter bool   `bun:"is_datacenter" json:"isDatacenter"`
	VPNProvider  string `bun:"vpn_provider" json:"vpnProvider,omitempty"`
	FraudScore   *float64 `bun:"fraud_score" json:"fraudScore,omitempty"`

	// Network Info
	ASN          string `bun:"asn" json:"asn,omitempty"`
	ISP          string `bun:"isp" json:"isp,omitempty"`
	Organization string `bun:"organization" json:"organization,omitempty"`
	ConnectionType string `bun:"connection_type" json:"connectionType,omitempty"` // cable, cellular, etc.

	// Context
	UserAgent   string `bun:"user_agent" json:"userAgent,omitempty"`
	EventType   string `bun:"event_type,notnull" json:"eventType"` // login, request, manual_check
	EventResult string `bun:"event_result,notnull" json:"eventResult"` // allowed, denied, flagged
	RuleName    string `bun:"rule_name" json:"ruleName,omitempty"` // Which rule triggered

	// Distance from previous location
	DistanceKm   *float64       `bun:"distance_km" json:"distanceKm,omitempty"`
	TimeFromPrev *time.Duration `bun:"time_from_prev" json:"timeFromPrev,omitempty"`
	SpeedKmh     *float64       `bun:"speed_kmh" json:"speedKmh,omitempty"` // Calculated speed

	// Metadata
	Timestamp time.Time `bun:"timestamp,notnull,default:current_timestamp" json:"timestamp"`
	Metadata  map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
}

// TravelAlert represents a travel notification/alert
type TravelAlert struct {
	bun.BaseModel `bun:"table:travel_alerts,alias:ta"`

	ID             xid.ID  `bun:"id,pk,type:varchar(20)" json:"id"`
	UserID         xid.ID  `bun:"user_id,type:varchar(20),notnull" json:"userId"`
	OrganizationID xid.ID  `bun:"organization_id,type:varchar(20),notnull" json:"organizationId"`
	
	// Alert Type
	AlertType   string `bun:"alert_type,notnull" json:"alertType"` // impossible_travel, new_location, anomaly
	Severity    string `bun:"severity,notnull" json:"severity"` // low, medium, high, critical

	// Location Context
	FromCountry     string   `bun:"from_country" json:"fromCountry"`
	FromCity        string   `bun:"from_city" json:"fromCity"`
	FromLat         *float64 `bun:"from_lat" json:"fromLat,omitempty"`
	FromLon         *float64 `bun:"from_lon" json:"fromLon,omitempty"`
	ToCountry       string   `bun:"to_country" json:"toCountry"`
	ToCity          string   `bun:"to_city" json:"toCity"`
	ToLat           *float64 `bun:"to_lat" json:"toLat,omitempty"`
	ToLon           *float64 `bun:"to_lon" json:"toLon,omitempty"`
	
	// Travel Metrics
	DistanceKm      float64       `bun:"distance_km,notnull" json:"distanceKm"`
	TimeDifference  time.Duration `bun:"time_difference,notnull" json:"timeDifference"`
	CalculatedSpeed float64       `bun:"calculated_speed,notnull" json:"calculatedSpeed"` // km/h

	// Status
	Status          string     `bun:"status,notnull" json:"status"` // pending, approved, denied, auto_approved
	RequiresApproval bool      `bun:"requires_approval" json:"requiresApproval"`
	ApprovedBy      *xid.ID    `bun:"approved_by,type:varchar(20)" json:"approvedBy,omitempty"`
	ApprovedAt      *time.Time `bun:"approved_at" json:"approvedAt,omitempty"`

	// Notifications
	UserNotified  bool       `bun:"user_notified" json:"userNotified"`
	AdminNotified bool       `bun:"admin_notified" json:"adminNotified"`
	NotifiedAt    *time.Time `bun:"notified_at" json:"notifiedAt,omitempty"`

	// Resolution
	ResolvedAt *time.Time `bun:"resolved_at" json:"resolvedAt,omitempty"`
	Resolution string     `bun:"resolution" json:"resolution,omitempty"`
	
	// References
	LocationEventID xid.ID `bun:"location_event_id,type:varchar(20)" json:"locationEventId"`

	// Metadata
	CreatedAt time.Time              `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time              `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
	Metadata  map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
}

// TrustedLocation represents a user's trusted location
type TrustedLocation struct {
	bun.BaseModel `bun:"table:trusted_locations,alias:tl"`

	ID             xid.ID  `bun:"id,pk,type:varchar(20)" json:"id"`
	UserID         xid.ID  `bun:"user_id,type:varchar(20),notnull" json:"userId"`
	OrganizationID xid.ID  `bun:"organization_id,type:varchar(20),notnull" json:"organizationId"`
	
	// Location
	Name        string   `bun:"name,notnull" json:"name"` // e.g., "Home", "Office"
	Description string   `bun:"description" json:"description"`
	Country     string   `bun:"country,notnull" json:"country"`
	CountryCode string   `bun:"country_code,notnull" json:"countryCode"`
	Region      string   `bun:"region" json:"region"`
	City        string   `bun:"city" json:"city"`
	Latitude    float64  `bun:"latitude" json:"latitude"`
	Longitude   float64  `bun:"longitude" json:"longitude"`
	RadiusKm    float64  `bun:"radius_km,notnull" json:"radiusKm"` // Trust radius

	// Trust Settings
	AutoApprove     bool `bun:"auto_approve" json:"autoApprove"`
	SkipMFA         bool `bun:"skip_mfa" json:"skipMfa"`
	
	// Usage Statistics
	UsageCount      int        `bun:"usage_count" json:"usageCount"`
	FirstUsedAt     time.Time  `bun:"first_used_at" json:"firstUsedAt"`
	LastUsedAt      *time.Time `bun:"last_used_at" json:"lastUsedAt,omitempty"`
	
	// Metadata
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
	ExpiresAt *time.Time `bun:"expires_at" json:"expiresAt,omitempty"`
}

// GeofenceViolation represents a geofence policy violation
type GeofenceViolation struct {
	bun.BaseModel `bun:"table:geofence_violations,alias:gv"`

	ID             xid.ID  `bun:"id,pk,type:varchar(20)" json:"id"`
	UserID         xid.ID  `bun:"user_id,type:varchar(20),notnull" json:"userId"`
	OrganizationID xid.ID  `bun:"organization_id,type:varchar(20),notnull" json:"organizationId"`
	RuleID         xid.ID  `bun:"rule_id,type:varchar(20),notnull" json:"ruleId"`
	
	// Violation Details
	ViolationType   string `bun:"violation_type,notnull" json:"violationType"` // blocked_country, vpn_detected, etc.
	Severity        string `bun:"severity,notnull" json:"severity"` // low, medium, high, critical
	Action          string `bun:"action,notnull" json:"action"` // blocked, flagged, mfa_required
	
	// Location Context
	IPAddress   string   `bun:"ip_address,notnull" json:"ipAddress"`
	Country     string   `bun:"country" json:"country"`
	CountryCode string   `bun:"country_code" json:"countryCode"`
	City        string   `bun:"city" json:"city"`
	Latitude    *float64 `bun:"latitude" json:"latitude,omitempty"`
	Longitude   *float64 `bun:"longitude" json:"longitude,omitempty"`

	// Detection Info
	IsVPN        bool   `bun:"is_vpn" json:"isVpn"`
	IsProxy      bool   `bun:"is_proxy" json:"isProxy"`
	IsTor        bool   `bun:"is_tor" json:"isTor"`
	IsDatacenter bool   `bun:"is_datacenter" json:"isDatacenter"`
	
	// Response
	Blocked         bool       `bun:"blocked" json:"blocked"`
	UserNotified    bool       `bun:"user_notified" json:"userNotified"`
	AdminNotified   bool       `bun:"admin_notified" json:"adminNotified"`
	
	// Resolution
	Resolved        bool       `bun:"resolved" json:"resolved"`
	ResolvedAt      *time.Time `bun:"resolved_at" json:"resolvedAt,omitempty"`
	ResolvedBy      *xid.ID    `bun:"resolved_by,type:varchar(20)" json:"resolvedBy,omitempty"`
	Resolution      string     `bun:"resolution" json:"resolution,omitempty"`
	
	// References
	LocationEventID *xid.ID `bun:"location_event_id,type:varchar(20)" json:"locationEventId,omitempty"`
	SessionID       *xid.ID `bun:"session_id,type:varchar(20)" json:"sessionId,omitempty"`
	
	// Metadata
	CreatedAt time.Time              `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time              `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
	Metadata  map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
}

// GeoCache represents cached geolocation data
type GeoCache struct {
	bun.BaseModel `bun:"table:geo_cache,alias:gc"`

	IPAddress   string    `bun:"ip_address,pk,notnull" json:"ipAddress"`
	
	// Geolocation Data
	Country     string   `bun:"country" json:"country"`
	CountryCode string   `bun:"country_code" json:"countryCode"`
	Region      string   `bun:"region" json:"region"`
	City        string   `bun:"city" json:"city"`
	Latitude    *float64 `bun:"latitude" json:"latitude,omitempty"`
	Longitude   *float64 `bun:"longitude" json:"longitude,omitempty"`
	AccuracyKm  *float64 `bun:"accuracy_km" json:"accuracyKm,omitempty"`
	
	// Detection Data
	IsVPN        bool     `bun:"is_vpn" json:"isVpn"`
	IsProxy      bool     `bun:"is_proxy" json:"isProxy"`
	IsTor        bool     `bun:"is_tor" json:"isTor"`
	IsDatacenter bool     `bun:"is_datacenter" json:"isDatacenter"`
	VPNProvider  string   `bun:"vpn_provider" json:"vpnProvider,omitempty"`
	FraudScore   *float64 `bun:"fraud_score" json:"fraudScore,omitempty"`
	
	// Network Info
	ASN          string `bun:"asn" json:"asn,omitempty"`
	ISP          string `bun:"isp" json:"isp,omitempty"`
	Organization string `bun:"organization" json:"organization,omitempty"`
	
	// Cache Metadata
	Provider  string    `bun:"provider,notnull" json:"provider"` // Which provider gave us this data
	CachedAt  time.Time `bun:"cached_at,notnull,default:current_timestamp" json:"cachedAt"`
	ExpiresAt time.Time `bun:"expires_at,notnull" json:"expiresAt"`
	HitCount  int       `bun:"hit_count" json:"hitCount"`
}

