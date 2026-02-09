package geofence

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/notification"
)

// Service handles geofencing operations.
type Service struct {
	config              *Config
	repo                Repository
	geoProvider         GeoProvider
	fallbackGeoProvider GeoProvider
	detectionProvider   DetectionProvider
	auditService        *audit.Service
	notificationService *notification.Service
	authInst            any // Auth instance for service registry access
	geoCache            map[string]*CachedGeo
	detectionCache      map[string]*CachedDetection
}

// CachedGeo represents cached geolocation data.
type CachedGeo struct {
	Data      *GeoData
	ExpiresAt time.Time
}

// CachedDetection represents cached detection data.
type CachedDetection struct {
	Data      *DetectionResult
	ExpiresAt time.Time
}

// NewService creates a new geofencing service.
func NewService(
	config *Config,
	repo Repository,
	geoProvider GeoProvider,
	detectionProvider DetectionProvider,
	auditService *audit.Service,
	notificationService *notification.Service,
	authInst any,
) *Service {
	svc := &Service{
		config:              config,
		repo:                repo,
		geoProvider:         geoProvider,
		detectionProvider:   detectionProvider,
		auditService:        auditService,
		notificationService: notificationService,
		authInst:            authInst,
		geoCache:            make(map[string]*CachedGeo),
		detectionCache:      make(map[string]*CachedDetection),
	}

	// Initialize fallback provider if configured
	if config.Geolocation.FallbackProvider != "" {
		svc.fallbackGeoProvider = svc.createGeoProvider(config.Geolocation.FallbackProvider)
	}

	return svc
}

// createGeoProvider creates a geolocation provider based on name.
func (s *Service) createGeoProvider(name string) GeoProvider {
	switch strings.ToLower(name) {
	case "maxmind":
		return NewMaxMindProvider(
			s.config.Geolocation.MaxMindLicenseKey,
			s.config.Geolocation.MaxMindDatabasePath,
		)
	case "ipapi":
		return NewIPAPIProvider(s.config.Geolocation.IPAPIKey)
	case "ipinfo":
		return NewIPInfoProvider(s.config.Geolocation.IPInfoToken)
	case "ipgeolocation":
		return NewIPGeolocationProvider(s.config.Geolocation.IPGeolocationKey)
	default:
		return nil
	}
}

// CheckLocation performs a comprehensive geofence check.
func (s *Service) CheckLocation(ctx context.Context, req *LocationCheckRequest) (*LocationCheckResult, error) {
	result := &LocationCheckResult{
		Allowed:    true,
		Violations: []string{},
	}

	// Get geolocation data
	geoData, err := s.GetGeolocation(ctx, req.IPAddress)
	if err != nil {
		if s.config.Restrictions.StrictMode {
			result.Allowed = false
			result.Reason = "geolocation_lookup_failed"

			return result, nil
		}
		// Log error but allow if not in strict mode
		_ = s.auditLog(ctx, "geolocation_error", req.UserID, req.AppID, map[string]any{
			"error": err.Error(),
			"ip":    req.IPAddress,
		})
	}

	// Get detection data
	detection, err := s.GetDetection(ctx, req.IPAddress)
	if err != nil {
		// Detection failure is not critical, log and continue
		_ = s.auditLog(ctx, "detection_error", req.UserID, req.AppID, map[string]any{
			"error": err.Error(),
			"ip":    req.IPAddress,
		})
	}

	// Get applicable rules
	rules, err := s.repo.ListEnabledRules(ctx, req.AppID, &req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules: %w", err)
	}

	// Evaluate rules in priority order
	for _, rule := range rules {
		violation := s.evaluateRule(ctx, rule, geoData, detection, req)
		if violation != nil {
			result.Violations = append(result.Violations, violation.ViolationType)

			// Store violation
			_ = s.repo.CreateViolation(ctx, violation)

			// Handle violation based on rule action
			switch rule.Action {
			case "deny":
				result.Allowed = false
				result.Reason = violation.ViolationType
				result.RuleName = rule.Name

				return result, nil
			case "mfa_required":
				result.RequireMFA = true
				result.RuleName = rule.Name
			case "notify":
				result.Notify = true
				result.RuleName = rule.Name
			}
		}
	}

	// Create location event
	event := s.createLocationEvent(req, geoData, detection, result)
	if err := s.repo.CreateLocationEvent(ctx, event); err != nil {
		// Log error but don't fail the check
		_ = s.auditLog(ctx, "location_event_error", req.UserID, req.AppID, map[string]any{
			"error": err.Error(),
		})
	}

	// Check for travel anomalies
	if s.config.Travel.Enabled {
		alert := s.checkTravelAnomaly(ctx, req.UserID, event)
		if alert != nil {
			result.TravelAlert = true
			result.TravelAlertID = &alert.ID

			if alert.RequiresApproval {
				result.Allowed = false
				result.Reason = "travel_approval_required"
			}
		}
	}

	return result, nil
}

// evaluateRule evaluates a single geofence rule.
func (s *Service) evaluateRule(
	ctx context.Context,
	rule *GeofenceRule,
	geoData *GeoData,
	detection *DetectionResult,
	req *LocationCheckRequest,
) *GeofenceViolation {
	if geoData == nil {
		return nil
	}

	// Check country restrictions
	if len(rule.AllowedCountries) > 0 {
		if !contains(rule.AllowedCountries, geoData.CountryCode) {
			return s.createViolation(rule, req, geoData, "country_not_allowed")
		}
	}

	if len(rule.BlockedCountries) > 0 {
		if contains(rule.BlockedCountries, geoData.CountryCode) {
			return s.createViolation(rule, req, geoData, "country_blocked")
		}
	}

	// Check region restrictions
	if len(rule.AllowedRegions) > 0 {
		if !contains(rule.AllowedRegions, geoData.Region) {
			return s.createViolation(rule, req, geoData, "region_not_allowed")
		}
	}

	if len(rule.BlockedRegions) > 0 {
		if contains(rule.BlockedRegions, geoData.Region) {
			return s.createViolation(rule, req, geoData, "region_blocked")
		}
	}

	// Check city restrictions
	if len(rule.AllowedCities) > 0 {
		if !contains(rule.AllowedCities, geoData.City) {
			return s.createViolation(rule, req, geoData, "city_not_allowed")
		}
	}

	if len(rule.BlockedCities) > 0 {
		if contains(rule.BlockedCities, geoData.City) {
			return s.createViolation(rule, req, geoData, "city_blocked")
		}
	}

	// Check geofence (circle or polygon)
	if rule.GeofenceType != "" && geoData.Latitude != nil && geoData.Longitude != nil {
		inside := s.isInsideGeofence(rule, *geoData.Latitude, *geoData.Longitude)
		if !inside {
			return s.createViolation(rule, req, geoData, "outside_geofence")
		}
	}

	// Check detection rules
	if detection != nil {
		if rule.BlockVPN && detection.IsVPN {
			return s.createViolation(rule, req, geoData, "vpn_detected")
		}

		if rule.BlockProxy && detection.IsProxy {
			return s.createViolation(rule, req, geoData, "proxy_detected")
		}

		if rule.BlockTor && detection.IsTor {
			return s.createViolation(rule, req, geoData, "tor_detected")
		}

		if rule.BlockDatacenter && detection.IsDatacenter {
			return s.createViolation(rule, req, geoData, "datacenter_detected")
		}
	}

	return nil
}

// isInsideGeofence checks if coordinates are inside a geofence.
func (s *Service) isInsideGeofence(rule *GeofenceRule, lat, lon float64) bool {
	if rule.GeofenceType == "circle" && rule.CenterLat != nil && rule.CenterLon != nil && rule.RadiusKm != nil {
		distance := haversineDistance(lat, lon, *rule.CenterLat, *rule.CenterLon)

		return distance <= *rule.RadiusKm
	}

	if rule.GeofenceType == "polygon" && len(rule.Coordinates) >= 3 {
		return pointInPolygon(lat, lon, rule.Coordinates)
	}

	return true // If geofence type not recognized, allow
}

// pointInPolygon checks if a point is inside a polygon using ray casting algorithm.
func pointInPolygon(lat, lon float64, polygon [][2]float64) bool {
	inside := false
	j := len(polygon) - 1

	for i := range polygon {
		xi, yi := polygon[i][0], polygon[i][1]
		xj, yj := polygon[j][0], polygon[j][1]

		intersect := ((yi > lon) != (yj > lon)) &&
			(lat < (xj-xi)*(lon-yi)/(yj-yi)+xi)

		if intersect {
			inside = !inside
		}

		j = i
	}

	return inside
}

// createViolation creates a geofence violation record.
func (s *Service) createViolation(
	rule *GeofenceRule,
	req *LocationCheckRequest,
	geoData *GeoData,
	violationType string,
) *GeofenceViolation {
	violation := &GeofenceViolation{
		UserID:        req.UserID,
		AppID:         req.AppID,
		RuleID:        rule.ID,
		ViolationType: violationType,
		Severity:      s.determineSeverity(violationType),
		Action:        rule.Action,
		IPAddress:     req.IPAddress,
		Blocked:       rule.Action == "deny",
		UserNotified:  rule.NotifyUser,
		AdminNotified: rule.NotifyAdmin,
	}

	if geoData != nil {
		violation.Country = geoData.Country
		violation.CountryCode = geoData.CountryCode
		violation.City = geoData.City
		violation.Latitude = geoData.Latitude
		violation.Longitude = geoData.Longitude
	}

	return violation
}

// determineSeverity determines the severity level of a violation.
func (s *Service) determineSeverity(violationType string) string {
	switch violationType {
	case "country_blocked", "tor_detected":
		return "critical"
	case "vpn_detected", "proxy_detected":
		return "high"
	case "region_blocked", "datacenter_detected":
		return "medium"
	default:
		return "low"
	}
}

// createLocationEvent creates a location event record.
func (s *Service) createLocationEvent(
	req *LocationCheckRequest,
	geoData *GeoData,
	detection *DetectionResult,
	result *LocationCheckResult,
) *LocationEvent {
	event := &LocationEvent{
		UserID:    req.UserID,
		AppID:     req.AppID,
		SessionID: req.SessionID,
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
		EventType: req.EventType,
	}

	if result.Allowed {
		event.EventResult = "allowed"
	} else {
		event.EventResult = "denied"
	}

	if result.RuleName != "" {
		event.RuleName = result.RuleName
	}

	if geoData != nil {
		event.Country = geoData.Country
		event.CountryCode = geoData.CountryCode
		event.Region = geoData.Region
		event.City = geoData.City
		event.Latitude = geoData.Latitude
		event.Longitude = geoData.Longitude
		event.ASN = geoData.ASN
		event.ISP = geoData.ISP
		event.Organization = geoData.Organization
	}

	if detection != nil {
		event.IsVPN = detection.IsVPN
		event.IsProxy = detection.IsProxy
		event.IsTor = detection.IsTor
		event.IsDatacenter = detection.IsDatacenter
		event.VPNProvider = detection.VPNProvider
		event.FraudScore = detection.FraudScore
	}

	if req.GPS != nil {
		event.GPSLatitude = &req.GPS.Latitude
		event.GPSLongitude = &req.GPS.Longitude
		event.GPSAccuracy = &req.GPS.AccuracyMeters
		event.GPSTimestamp = &req.GPS.Timestamp
	}

	return event
}

// checkTravelAnomaly checks for impossible travel or travel alerts.
func (s *Service) checkTravelAnomaly(ctx context.Context, userID xid.ID, current *LocationEvent) *TravelAlert {
	if current.Latitude == nil || current.Longitude == nil {
		return nil
	}

	// Get last location event
	lastEvent, err := s.repo.GetLastLocationEvent(ctx, userID)
	if err != nil || lastEvent == nil || lastEvent.Latitude == nil || lastEvent.Longitude == nil {
		return nil
	}

	// Calculate distance and speed
	distance := haversineDistance(*current.Latitude, *current.Longitude, *lastEvent.Latitude, *lastEvent.Longitude)
	timeDiff := current.Timestamp.Sub(lastEvent.Timestamp)

	if timeDiff <= 0 {
		return nil
	}

	speedKmh := distance / (timeDiff.Hours())
	current.DistanceKm = &distance
	current.TimeFromPrev = &timeDiff
	current.SpeedKmh = &speedKmh

	// Check thresholds
	if distance < s.config.Travel.MinDistanceKm {
		return nil // Too close to be interesting
	}

	alertType := "new_location"
	severity := "low"

	if speedKmh > s.config.Travel.MaxSpeedKmh {
		alertType = "impossible_travel"
		severity = "critical"
	} else if distance > s.config.Travel.MinDistanceKm*2 {
		severity = "medium"
	}

	// Check if destination is trusted
	isTrusted, _, _ := s.repo.IsLocationTrusted(ctx, userID, *current.Latitude, *current.Longitude)
	if isTrusted && s.config.Travel.TrustFrequentDest {
		return nil // Trusted location, no alert
	}

	// Create alert
	alert := &TravelAlert{
		UserID:           userID,
		AppID:            current.AppID,
		AlertType:        alertType,
		Severity:         severity,
		FromCountry:      lastEvent.Country,
		FromCity:         lastEvent.City,
		FromLat:          lastEvent.Latitude,
		FromLon:          lastEvent.Longitude,
		ToCountry:        current.Country,
		ToCity:           current.City,
		ToLat:            current.Latitude,
		ToLon:            current.Longitude,
		DistanceKm:       distance,
		TimeDifference:   timeDiff,
		CalculatedSpeed:  speedKmh,
		Status:           "pending",
		RequiresApproval: s.config.Travel.RequireApproval && severity == "critical",
		UserNotified:     false,
		AdminNotified:    false,
		LocationEventID:  current.ID,
	}

	if err := s.repo.CreateTravelAlert(ctx, alert); err != nil {
		return nil
	}

	// Send notifications
	s.sendTravelNotifications(ctx, alert)

	return alert
}

// sendTravelNotifications sends travel alert notifications.
func (s *Service) sendTravelNotifications(ctx context.Context, alert *TravelAlert) {
	if s.notificationService == nil {
		return
	}

	message := fmt.Sprintf(
		"Travel detected: %s (%s) to %s (%s). Distance: %.0fkm, Speed: %.0fkm/h",
		alert.FromCity, alert.FromCountry,
		alert.ToCity, alert.ToCountry,
		alert.DistanceKm, alert.CalculatedSpeed,
	)

	if s.config.Travel.NotifyUser {
		// TODO: Send user notification
		_ = s.auditLog(ctx, "travel_alert_user_notification", alert.UserID, alert.AppID, map[string]any{
			"alert_id": alert.ID,
			"message":  message,
		})
	}

	if s.config.Travel.NotifyAdmin {
		// TODO: Send admin notification
		_ = s.auditLog(ctx, "travel_alert_admin_notification", alert.UserID, alert.AppID, map[string]any{
			"alert_id": alert.ID,
			"message":  message,
		})
	}
}

// GetGeolocation gets geolocation data for an IP address.
func (s *Service) GetGeolocation(ctx context.Context, ip string) (*GeoData, error) {
	// Check memory cache
	if cached, ok := s.geoCache[ip]; ok && time.Now().Before(cached.ExpiresAt) {
		return cached.Data, nil
	}

	// Check database cache
	dbCache, err := s.repo.GetCachedGeoData(ctx, ip)
	if err == nil && dbCache != nil {
		data := &GeoData{
			IPAddress:    dbCache.IPAddress,
			Country:      dbCache.Country,
			CountryCode:  dbCache.CountryCode,
			Region:       dbCache.Region,
			City:         dbCache.City,
			Latitude:     dbCache.Latitude,
			Longitude:    dbCache.Longitude,
			AccuracyKm:   dbCache.AccuracyKm,
			ASN:          dbCache.ASN,
			ISP:          dbCache.ISP,
			Organization: dbCache.Organization,
			Provider:     dbCache.Provider,
		}

		// Update memory cache
		s.geoCache[ip] = &CachedGeo{
			Data:      data,
			ExpiresAt: dbCache.ExpiresAt,
		}

		return data, nil
	}

	// Query provider
	data, err := s.geoProvider.Lookup(ctx, ip)
	if err != nil && s.fallbackGeoProvider != nil {
		// Try fallback provider
		data, err = s.fallbackGeoProvider.Lookup(ctx, ip)
	}

	if err != nil {
		return nil, fmt.Errorf("geolocation lookup failed: %w", err)
	}

	// Cache the result
	expiresAt := time.Now().Add(s.config.Geolocation.CacheDuration)
	cache := &GeoCache{
		IPAddress:    ip,
		Country:      data.Country,
		CountryCode:  data.CountryCode,
		Region:       data.Region,
		City:         data.City,
		Latitude:     data.Latitude,
		Longitude:    data.Longitude,
		AccuracyKm:   data.AccuracyKm,
		ASN:          data.ASN,
		ISP:          data.ISP,
		Organization: data.Organization,
		Provider:     data.Provider,
		ExpiresAt:    expiresAt,
	}

	_ = s.repo.SetCachedGeoData(ctx, cache)

	s.geoCache[ip] = &CachedGeo{
		Data:      data,
		ExpiresAt: expiresAt,
	}

	return data, nil
}

// GetDetection gets VPN/proxy detection data for an IP address.
func (s *Service) GetDetection(ctx context.Context, ip string) (*DetectionResult, error) {
	if s.detectionProvider == nil {
		return nil, nil
	}

	// Check memory cache
	if cached, ok := s.detectionCache[ip]; ok && time.Now().Before(cached.ExpiresAt) {
		return cached.Data, nil
	}

	// Query provider
	data, err := s.detectionProvider.Check(ctx, ip)
	if err != nil {
		return nil, fmt.Errorf("detection check failed: %w", err)
	}

	// Cache the result
	expiresAt := time.Now().Add(s.config.Detection.CacheDuration)
	s.detectionCache[ip] = &CachedDetection{
		Data:      data,
		ExpiresAt: expiresAt,
	}

	return data, nil
}

// auditLog logs an audit event.
func (s *Service) auditLog(ctx context.Context, eventType string, userID, orgID xid.ID, data map[string]any) error {
	if s.auditService == nil {
		return nil
	}

	// TODO: Implement audit logging via audit service
	return nil
}

// Helper function.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}

	return false
}

// LocationCheckRequest represents a geofence check request.
type LocationCheckRequest struct {
	UserID    xid.ID
	AppID     xid.ID
	SessionID *xid.ID
	IPAddress string
	UserAgent string
	EventType string // "login", "request", "manual_check"
	GPS       *GPSData
}

// GPSData represents GPS coordinates from a device.
type GPSData struct {
	Latitude       float64
	Longitude      float64
	AccuracyMeters float64
	Timestamp      time.Time
}

// LocationCheckResult represents the result of a geofence check.
type LocationCheckResult struct {
	Allowed       bool
	Reason        string
	RuleName      string
	RequireMFA    bool
	Notify        bool
	Violations    []string
	TravelAlert   bool
	TravelAlertID *xid.ID
}

// CheckSessionSecurity performs location and security checks for a session.
func (s *Service) CheckSessionSecurity(ctx context.Context, userID xid.ID, appID xid.ID, ipAddress string) error {
	if !s.config.Notifications.Enabled {
		return nil
	}

	// Get current location using existing GetGeolocation method
	currentLoc, err := s.GetGeolocation(ctx, ipAddress)
	if err != nil {
		// Don't fail auth on geolocation errors
		return nil
	}

	// Store location event
	locationEvent := &LocationEvent{
		ID:           xid.New(),
		UserID:       userID,
		AppID:        appID,
		IPAddress:    ipAddress,
		Country:      currentLoc.Country,
		CountryCode:  currentLoc.CountryCode,
		Region:       "", // Optional
		City:         currentLoc.City,
		Latitude:     currentLoc.Latitude,
		Longitude:    currentLoc.Longitude,
		AccuracyKm:   currentLoc.AccuracyKm,
		IsVPN:        false, // Will be set by detection
		IsProxy:      false,
		IsTor:        false,
		IsDatacenter: false,
		EventType:    "login",
		EventResult:  "allowed",
		Timestamp:    time.Now().UTC(),
	}

	_ = s.repo.CreateLocationEvent(ctx, locationEvent)

	// Check for new location
	if s.config.Notifications.NewLocationEnabled {
		lastLoc, err := s.repo.GetLastLocation(ctx, userID, appID)
		if err == nil && lastLoc != nil {
			distance := s.calculateDistanceBetweenLocations(lastLoc, currentLoc)
			if distance >= s.config.Notifications.NewLocationThresholdKm {
				_ = s.notifyNewLocation(ctx, userID, appID, currentLoc, lastLoc, distance)
			}
		}
	}

	// Check for suspicious patterns
	if s.config.Notifications.SuspiciousLoginEnabled {
		if suspicious, reason := s.isSuspicious(ctx, userID, currentLoc, locationEvent); suspicious {
			_ = s.notifySuspiciousLogin(ctx, userID, appID, reason, currentLoc)
		}
	}

	return nil
}

// isSuspicious checks for suspicious login patterns.
func (s *Service) isSuspicious(ctx context.Context, userID xid.ID, currentLoc *GeoData, currentEvent *LocationEvent) (bool, string) {
	// Check impossible travel
	if s.config.Notifications.ImpossibleTravelEnabled {
		lastEvent, err := s.repo.GetLastLocationEvent(ctx, userID)
		if err == nil && lastEvent != nil {
			timeDiff := time.Since(lastEvent.Timestamp)
			if timeDiff.Hours() > 0 && lastEvent.Latitude != nil && lastEvent.Longitude != nil && currentLoc.Latitude != nil && currentLoc.Longitude != nil {
				distance := haversineDistance(*lastEvent.Latitude, *lastEvent.Longitude, *currentLoc.Latitude, *currentLoc.Longitude)

				// Calculate required speed (km/h)
				hours := timeDiff.Hours()
				speedKmh := distance / hours
				maxSpeed := 900.0 // Commercial aircraft speed

				if speedKmh > maxSpeed {
					return true, fmt.Sprintf("Impossible travel: %.0f km in %.1f hours (%.0f km/h, max: %.0f km/h)",
						distance, hours, speedKmh, maxSpeed)
				}
			}
		}
	}

	// Check VPN/Proxy/Tor using existing GetDetection method
	if s.detectionProvider != nil {
		detection, err := s.GetDetection(ctx, currentLoc.IPAddress)
		if err == nil && detection != nil {
			// VPN Detection
			if s.config.Notifications.VpnDetectionEnabled && detection.IsVPN {
				vpnInfo := "VPN detected"
				if detection.VPNProvider != "" {
					vpnInfo = "VPN detected: " + detection.VPNProvider
				}

				return true, vpnInfo
			}

			// Proxy Detection
			if s.config.Notifications.ProxyDetectionEnabled && detection.IsProxy {
				return true, "Proxy server detected"
			}

			// Tor Detection
			if s.config.Notifications.TorDetectionEnabled && detection.IsTor {
				return true, "Tor exit node detected"
			}

			// Fraud Score
			if detection.FraudScore != nil && *detection.FraudScore >= s.config.Notifications.SuspiciousLoginScoreThreshold {
				return true, fmt.Sprintf("High fraud score: %.1f/100", *detection.FraudScore)
			}

			// Update location event with detection results
			if currentEvent != nil {
				currentEvent.IsVPN = detection.IsVPN
				currentEvent.IsProxy = detection.IsProxy
				currentEvent.IsTor = detection.IsTor
				currentEvent.IsDatacenter = detection.IsDatacenter
				currentEvent.VPNProvider = detection.VPNProvider
				currentEvent.FraudScore = detection.FraudScore
			}
		}
	}

	return false, ""
}

// calculateDistanceBetweenLocations calculates distance between two GeoData locations
// Uses the haversineDistance function from repository.go.
func (s *Service) calculateDistanceBetweenLocations(loc1, loc2 *GeoData) float64 {
	if loc1.Latitude == nil || loc1.Longitude == nil || loc2.Latitude == nil || loc2.Longitude == nil {
		return 0
	}

	return haversineDistance(*loc1.Latitude, *loc1.Longitude, *loc2.Latitude, *loc2.Longitude)
}
