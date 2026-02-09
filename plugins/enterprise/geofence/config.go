package geofence

import (
	"errors"
	"time"
)

// Config holds the geofencing plugin configuration.
type Config struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Geographic Restrictions
	Restrictions RestrictionConfig `json:"restrictions" yaml:"restrictions"`

	// IP Geolocation
	Geolocation GeolocationConfig `json:"geolocation" yaml:"geolocation"`

	// GPS-Based Authentication
	GPS GPSConfig `json:"gps" yaml:"gps"`

	// VPN/Proxy Detection
	Detection DetectionConfig `json:"detection" yaml:"detection"`

	// Corporate Network Detection
	Corporate CorporateConfig `json:"corporate" yaml:"corporate"`

	// Travel Notifications
	Travel TravelConfig `json:"travel" yaml:"travel"`

	// Session Management
	Session SessionConfig `json:"session" yaml:"session"`

	// API Endpoints
	API APIConfig `json:"api" yaml:"api"`

	// Security & Audit
	Security SecurityConfig `json:"security" yaml:"security"`

	// Session Security Notifications
	Notifications NotificationConfig `json:"notifications" yaml:"notifications"`
}

// RestrictionConfig configures geographic restrictions.
type RestrictionConfig struct {
	// Country/Region Controls
	AllowedCountries []string `json:"allowedCountries" yaml:"allowedCountries"` // ISO 3166-1 alpha-2
	BlockedCountries []string `json:"blockedCountries" yaml:"blockedCountries"` // ISO 3166-1 alpha-2
	AllowedRegions   []string `json:"allowedRegions"   yaml:"allowedRegions"`   // US: state codes, etc.
	BlockedRegions   []string `json:"blockedRegions"   yaml:"blockedRegions"`

	// City-Level Controls
	AllowedCities []string `json:"allowedCities" yaml:"allowedCities"`
	BlockedCities []string `json:"blockedCities" yaml:"blockedCities"`

	// Time-Based Restrictions
	TimeRestrictions []TimeRestriction `json:"timeRestrictions" yaml:"timeRestrictions"`

	// Distance-Based Restrictions
	MaxDistanceKm float64 `json:"maxDistanceKm" yaml:"maxDistanceKm"` // Max distance from reference point

	// Behavior
	DefaultAction string `json:"defaultAction" yaml:"defaultAction"` // "allow" or "deny"
	StrictMode    bool   `json:"strictMode"    yaml:"strictMode"`    // Deny on lookup failure
}

// TimeRestriction defines time-based access rules.
type TimeRestriction struct {
	Countries   []string `json:"countries"   yaml:"countries"`
	AllowedDays []string `json:"allowedDays" yaml:"allowedDays"` // Monday, Tuesday, etc.
	StartHour   int      `json:"startHour"   yaml:"startHour"`   // 0-23
	EndHour     int      `json:"endHour"     yaml:"endHour"`     // 0-23
	Timezone    string   `json:"timezone"    yaml:"timezone"`    // IANA timezone
}

// GeolocationConfig configures IP geolocation services.
type GeolocationConfig struct {
	// Primary Provider
	Provider         string            `json:"provider"         yaml:"provider"` // maxmind, ipapi, ipinfo, ipgeolocation
	ProviderConfig   map[string]string `json:"providerConfig"   yaml:"providerConfig"`
	FallbackProvider string            `json:"fallbackProvider" yaml:"fallbackProvider"`

	// MaxMind GeoIP2
	MaxMindLicenseKey   string `json:"maxmindLicenseKey"   yaml:"maxmindLicenseKey"`
	MaxMindDatabasePath string `json:"maxmindDatabasePath" yaml:"maxmindDatabasePath"`
	MaxMindAutoUpdate   bool   `json:"maxmindAutoUpdate"   yaml:"maxmindAutoUpdate"`

	// IPInfo.io
	IPInfoToken string `json:"ipinfoToken" yaml:"ipinfoToken"`

	// ipapi.com
	IPAPIKey string `json:"ipapiKey" yaml:"ipapiKey"`

	// ipgeolocation.io
	IPGeolocationKey string `json:"ipgeolocationKey" yaml:"ipgeolocationKey"`

	// Caching
	CacheDuration time.Duration `json:"cacheDuration" yaml:"cacheDuration"`
	CacheMaxSize  int           `json:"cacheMaxSize"  yaml:"cacheMaxSize"`

	// Performance
	Timeout    time.Duration `json:"timeout"    yaml:"timeout"`
	MaxRetries int           `json:"maxRetries" yaml:"maxRetries"`

	// Accuracy Requirements
	MinAccuracyKm float64 `json:"minAccuracyKm" yaml:"minAccuracyKm"` // Minimum accuracy in km
}

// GPSConfig configures GPS-based authentication.
type GPSConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Coordinate Requirements
	RequireGPS        bool    `json:"requireGps"        yaml:"requireGps"`
	MaxAccuracyMeters float64 `json:"maxAccuracyMeters" yaml:"maxAccuracyMeters"`
	MinAccuracyMeters float64 `json:"minAccuracyMeters" yaml:"minAccuracyMeters"`

	// Geofencing
	Geofences          []Geofence `json:"geofences"          yaml:"geofences"`
	RequireInsideFence bool       `json:"requireInsideFence" yaml:"requireInsideFence"`

	// Movement Detection
	MaxSpeedKmh    float64       `json:"maxSpeedKmh"    yaml:"maxSpeedKmh"` // Alert on impossible travel speed
	MinTimeBetween time.Duration `json:"minTimeBetween" yaml:"minTimeBetween"`

	// Validation
	ValidateTimestamp bool          `json:"validateTimestamp" yaml:"validateTimestamp"`
	MaxTimestampAge   time.Duration `json:"maxTimestampAge"   yaml:"maxTimestampAge"`
}

// Geofence defines a geographic boundary.
type Geofence struct {
	ID          string `json:"id"          yaml:"id"`
	Name        string `json:"name"        yaml:"name"`
	Description string `json:"description" yaml:"description"`
	Type        string `json:"type"        yaml:"type"` // "circle", "polygon"

	// Circle
	CenterLat float64 `json:"centerLat" yaml:"centerLat"`
	CenterLon float64 `json:"centerLon" yaml:"centerLon"`
	RadiusKm  float64 `json:"radiusKm"  yaml:"radiusKm"`

	// Polygon (array of [lat, lon] pairs)
	Coordinates [][2]float64 `json:"coordinates" yaml:"coordinates"`

	// Rules
	Action string   `json:"action" yaml:"action"` // "allow" or "deny"
	Users  []string `json:"users"  yaml:"users"`  // Specific user IDs (empty = all)
	Roles  []string `json:"roles"  yaml:"roles"`  // Specific roles (empty = all)
}

// DetectionConfig configures VPN/proxy detection.
type DetectionConfig struct {
	// VPN Detection
	DetectVPN   bool     `json:"detectVpn"   yaml:"detectVpn"`
	BlockVPN    bool     `json:"blockVpn"    yaml:"blockVpn"`
	AllowedVPNs []string `json:"allowedVpns" yaml:"allowedVpns"` // Whitelisted VPN providers

	// Proxy Detection
	DetectProxy    bool     `json:"detectProxy"    yaml:"detectProxy"`
	BlockProxy     bool     `json:"blockProxy"     yaml:"blockProxy"`
	AllowedProxies []string `json:"allowedProxies" yaml:"allowedProxies"`

	// Tor Detection
	DetectTor bool `json:"detectTor" yaml:"detectTor"`
	BlockTor  bool `json:"blockTor"  yaml:"blockTor"`

	// Datacenter Detection
	DetectDatacenter bool `json:"detectDatacenter" yaml:"detectDatacenter"`
	BlockDatacenter  bool `json:"blockDatacenter"  yaml:"blockDatacenter"`

	// Detection Services
	Provider       string            `json:"provider"       yaml:"provider"` // ipqs, proxycheck, vpnapi
	ProviderConfig map[string]string `json:"providerConfig" yaml:"providerConfig"`

	// IPQualityScore
	IPQSKey        string  `json:"ipqsKey"        yaml:"ipqsKey"`
	IPQSStrictness int     `json:"ipqsStrictness" yaml:"ipqsStrictness"` // 0-3
	IPQSMinScore   float64 `json:"ipqsMinScore"   yaml:"ipqsMinScore"`   // 0-100

	// ProxyCheck.io
	ProxyCheckKey string `json:"proxycheckKey" yaml:"proxycheckKey"`

	// VPNapi.io
	VPNAPIKey string `json:"vpnapiKey" yaml:"vpnapiKey"`

	// Caching
	CacheDuration time.Duration `json:"cacheDuration" yaml:"cacheDuration"`
	CacheMaxSize  int           `json:"cacheMaxSize"  yaml:"cacheMaxSize"`
}

// CorporateConfig configures corporate network detection.
type CorporateConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Network Ranges
	Networks       []string `json:"networks"       yaml:"networks"` // CIDR ranges
	RequireNetwork bool     `json:"requireNetwork" yaml:"requireNetwork"`

	// DNS-Based Detection
	RequiredDNS []string `json:"requiredDns" yaml:"requiredDns"` // Expected DNS servers

	// Certificate-Based Detection
	RequireCert  bool     `json:"requireCert"  yaml:"requireCert"`
	TrustedCerts []string `json:"trustedCerts" yaml:"trustedCerts"` // Cert fingerprints

	// Hybrid Detection
	AllowExternal bool `json:"allowExternal" yaml:"allowExternal"` // Allow external if other auth strong
	RequireMFA    bool `json:"requireMfa"    yaml:"requireMfa"`    // Require MFA for external
}

// TravelConfig configures travel notifications.
type TravelConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Detection Thresholds
	MinDistanceKm  float64       `json:"minDistanceKm"  yaml:"minDistanceKm"`  // Trigger distance
	MinTimeBetween time.Duration `json:"minTimeBetween" yaml:"minTimeBetween"` // Minimum time between locations
	MaxSpeedKmh    float64       `json:"maxSpeedKmh"    yaml:"maxSpeedKmh"`    // Impossible travel speed

	// Notification Settings
	NotifyUser      bool          `json:"notifyUser"      yaml:"notifyUser"`
	NotifyAdmin     bool          `json:"notifyAdmin"     yaml:"notifyAdmin"`
	RequireApproval bool          `json:"requireApproval" yaml:"requireApproval"` // Block until approved
	ApprovalTimeout time.Duration `json:"approvalTimeout" yaml:"approvalTimeout"`

	// Channels
	EmailNotify   bool `json:"emailNotify"   yaml:"emailNotify"`
	SMSNotify     bool `json:"smsNotify"     yaml:"smsNotify"`
	PushNotify    bool `json:"pushNotify"    yaml:"pushNotify"`
	WebhookNotify bool `json:"webhookNotify" yaml:"webhookNotify"`

	// Auto-Approval
	AutoApproveAfter  time.Duration `json:"autoApproveAfter"  yaml:"autoApproveAfter"`
	TrustFrequentDest bool          `json:"trustFrequentDest" yaml:"trustFrequentDest"` // Trust frequent destinations
}

// SessionConfig configures geofence session management.
type SessionConfig struct {
	// Location Tracking
	TrackLocation  bool          `json:"trackLocation"  yaml:"trackLocation"`
	UpdateInterval time.Duration `json:"updateInterval" yaml:"updateInterval"`

	// Session Validation
	ValidateOnRequest     bool `json:"validateOnRequest"     yaml:"validateOnRequest"`
	InvalidateOnViolation bool `json:"invalidateOnViolation" yaml:"invalidateOnViolation"`

	// Grace Period
	GracePeriod   time.Duration `json:"gracePeriod"   yaml:"gracePeriod"` // Allow brief violations
	MaxViolations int           `json:"maxViolations" yaml:"maxViolations"`
}

// APIConfig configures geofencing API endpoints.
type APIConfig struct {
	BasePath         string `json:"basePath"         yaml:"basePath"`
	EnableManagement bool   `json:"enableManagement" yaml:"enableManagement"`
	EnableValidation bool   `json:"enableValidation" yaml:"enableValidation"`
	EnableMetrics    bool   `json:"enableMetrics"    yaml:"enableMetrics"`
	EnableRealtime   bool   `json:"enableRealtime"   yaml:"enableRealtime"` // WebSocket for live tracking
}

// SecurityConfig configures security settings.
type SecurityConfig struct {
	// Rate Limiting
	RateLimitEnabled   bool `json:"rateLimitEnabled"   yaml:"rateLimitEnabled"`
	MaxChecksPerMinute int  `json:"maxChecksPerMinute" yaml:"maxChecksPerMinute"`
	MaxChecksPerHour   int  `json:"maxChecksPerHour"   yaml:"maxChecksPerHour"`

	// Audit Logging
	AuditAllChecks  bool `json:"auditAllChecks"  yaml:"auditAllChecks"`
	AuditViolations bool `json:"auditViolations" yaml:"auditViolations"`
	AuditTravel     bool `json:"auditTravel"     yaml:"auditTravel"`

	// Data Storage
	StoreLocations    bool          `json:"storeLocations"    yaml:"storeLocations"`
	LocationRetention time.Duration `json:"locationRetention" yaml:"locationRetention"`
	AnonymizeOldData  bool          `json:"anonymizeOldData"  yaml:"anonymizeOldData"`

	// Privacy
	ConsentRequired bool `json:"consentRequired" yaml:"consentRequired"`
	AllowOptOut     bool `json:"allowOptOut"     yaml:"allowOptOut"`

	// Notifications
	NotifyOnViolation bool `json:"notifyOnViolation" yaml:"notifyOnViolation"`
	NotifyOnAnomaly   bool `json:"notifyOnAnomaly"   yaml:"notifyOnAnomaly"`
}

// DefaultConfig returns the default geofencing configuration.
func DefaultConfig() *Config {
	return &Config{
		Enabled: true,
		Restrictions: RestrictionConfig{
			AllowedCountries: []string{}, // Empty = allow all
			BlockedCountries: []string{}, // Sanctioned countries can be added
			AllowedRegions:   []string{},
			BlockedRegions:   []string{},
			AllowedCities:    []string{},
			BlockedCities:    []string{},
			TimeRestrictions: []TimeRestriction{},
			MaxDistanceKm:    0, // 0 = unlimited
			DefaultAction:    "allow",
			StrictMode:       false,
		},
		Geolocation: GeolocationConfig{
			Provider:          "maxmind",
			FallbackProvider:  "ipapi",
			MaxMindAutoUpdate: true,
			CacheDuration:     24 * time.Hour,
			CacheMaxSize:      10000,
			Timeout:           5 * time.Second,
			MaxRetries:        3,
			MinAccuracyKm:     100, // 100km accuracy minimum
		},
		GPS: GPSConfig{
			Enabled:            false,
			RequireGPS:         false,
			MaxAccuracyMeters:  1000,
			MinAccuracyMeters:  10,
			Geofences:          []Geofence{},
			RequireInsideFence: false,
			MaxSpeedKmh:        1000, // Speed of sound ~1235 km/h, planes ~900 km/h
			MinTimeBetween:     1 * time.Minute,
			ValidateTimestamp:  true,
			MaxTimestampAge:    5 * time.Minute,
		},
		Detection: DetectionConfig{
			DetectVPN:        true,
			BlockVPN:         false, // Don't block by default
			AllowedVPNs:      []string{},
			DetectProxy:      true,
			BlockProxy:       false,
			AllowedProxies:   []string{},
			DetectTor:        true,
			BlockTor:         false,
			DetectDatacenter: true,
			BlockDatacenter:  false,
			Provider:         "ipqs",
			IPQSStrictness:   1, // Medium strictness
			IPQSMinScore:     75.0,
			CacheDuration:    1 * time.Hour,
			CacheMaxSize:     5000,
		},
		Corporate: CorporateConfig{
			Enabled:        false,
			Networks:       []string{},
			RequireNetwork: false,
			RequiredDNS:    []string{},
			RequireCert:    false,
			TrustedCerts:   []string{},
			AllowExternal:  true,
			RequireMFA:     false,
		},
		Travel: TravelConfig{
			Enabled:           true,
			MinDistanceKm:     500, // 500km triggers notification
			MinTimeBetween:    1 * time.Hour,
			MaxSpeedKmh:       900, // Commercial aircraft speed
			NotifyUser:        true,
			NotifyAdmin:       false,
			RequireApproval:   false,
			ApprovalTimeout:   15 * time.Minute,
			EmailNotify:       true,
			SMSNotify:         false,
			PushNotify:        true,
			WebhookNotify:     false,
			AutoApproveAfter:  24 * time.Hour,
			TrustFrequentDest: true,
		},
		Session: SessionConfig{
			TrackLocation:         true,
			UpdateInterval:        5 * time.Minute,
			ValidateOnRequest:     true,
			InvalidateOnViolation: false, // Don't kick out by default
			GracePeriod:           10 * time.Minute,
			MaxViolations:         3,
		},
		API: APIConfig{
			BasePath:         "/auth/geofence",
			EnableManagement: true,
			EnableValidation: true,
			EnableMetrics:    true,
			EnableRealtime:   false,
		},
		Security: SecurityConfig{
			RateLimitEnabled:   true,
			MaxChecksPerMinute: 60,
			MaxChecksPerHour:   1000,
			AuditAllChecks:     false,
			AuditViolations:    true,
			AuditTravel:        true,
			StoreLocations:     true,
			LocationRetention:  90 * 24 * time.Hour, // 90 days
			AnonymizeOldData:   true,
			ConsentRequired:    false, // Depends on jurisdiction
			AllowOptOut:        false,
			NotifyOnViolation:  true,
			NotifyOnAnomaly:    true,
		},
		Notifications: NotificationConfig{
			Enabled:                       true,
			NewLocationEnabled:            true,
			NewLocationThresholdKm:        100.0, // 100km triggers new location alert
			SuspiciousLoginEnabled:        true,
			SuspiciousLoginScoreThreshold: 75.0, // IPQS fraud score threshold
			ImpossibleTravelEnabled:       true,
			VpnDetectionEnabled:           true,
			ProxyDetectionEnabled:         true,
			TorDetectionEnabled:           true,
		},
	}
}

// NotificationConfig configures session security notifications.
type NotificationConfig struct {
	// General
	Enabled bool `json:"enabled" yaml:"enabled"`

	// New Location Notifications
	NewLocationEnabled     bool    `json:"newLocationEnabled"     yaml:"newLocationEnabled"`
	NewLocationThresholdKm float64 `json:"newLocationThresholdKm" yaml:"newLocationThresholdKm"` // Trigger at N km distance

	// Suspicious Login Notifications
	SuspiciousLoginEnabled        bool    `json:"suspiciousLoginEnabled"        yaml:"suspiciousLoginEnabled"`
	SuspiciousLoginScoreThreshold float64 `json:"suspiciousLoginScoreThreshold" yaml:"suspiciousLoginScoreThreshold"` // IPQS fraud score threshold

	// Detection Types
	ImpossibleTravelEnabled bool `json:"impossibleTravelEnabled" yaml:"impossibleTravelEnabled"`
	VpnDetectionEnabled     bool `json:"vpnDetectionEnabled"     yaml:"vpnDetectionEnabled"`
	ProxyDetectionEnabled   bool `json:"proxyDetectionEnabled"   yaml:"proxyDetectionEnabled"`
	TorDetectionEnabled     bool `json:"torDetectionEnabled"     yaml:"torDetectionEnabled"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Restrictions.MaxDistanceKm < 0 {
		return errors.New("max distance cannot be negative")
	}

	if c.GPS.Enabled {
		if c.GPS.MaxAccuracyMeters < c.GPS.MinAccuracyMeters {
			return errors.New("max accuracy cannot be less than min accuracy")
		}

		if c.GPS.MaxSpeedKmh <= 0 {
			return errors.New("max speed must be positive")
		}
	}

	if c.Geolocation.CacheDuration < 1*time.Minute {
		return errors.New("cache duration must be at least 1 minute")
	}

	if c.Session.UpdateInterval < 10*time.Second {
		return errors.New("update interval must be at least 10 seconds")
	}

	if c.Travel.MinDistanceKm < 0 {
		return errors.New("travel minimum distance cannot be negative")
	}

	if c.Security.LocationRetention < 24*time.Hour {
		return errors.New("location retention must be at least 24 hours for compliance")
	}

	return nil
}
