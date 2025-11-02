package geofence

import (
	"context"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository implements Repository for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateRule(ctx context.Context, rule *GeofenceRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockRepository) GetRule(ctx context.Context, id xid.ID) (*GeofenceRule, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GeofenceRule), args.Error(1)
}

func (m *MockRepository) GetRulesByOrganization(ctx context.Context, orgID xid.ID) ([]*GeofenceRule, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*GeofenceRule), args.Error(1)
}

func (m *MockRepository) GetRulesByUser(ctx context.Context, userID xid.ID) ([]*GeofenceRule, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*GeofenceRule), args.Error(1)
}

func (m *MockRepository) UpdateRule(ctx context.Context, rule *GeofenceRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockRepository) DeleteRule(ctx context.Context, id xid.ID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) ListEnabledRules(ctx context.Context, orgID xid.ID, userID *xid.ID) ([]*GeofenceRule, error) {
	args := m.Called(ctx, orgID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*GeofenceRule), args.Error(1)
}

func (m *MockRepository) CreateLocationEvent(ctx context.Context, event *LocationEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockRepository) GetLocationEvent(ctx context.Context, id xid.ID) (*LocationEvent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LocationEvent), args.Error(1)
}

func (m *MockRepository) GetUserLocationHistory(ctx context.Context, userID xid.ID, limit int) ([]*LocationEvent, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*LocationEvent), args.Error(1)
}

func (m *MockRepository) GetLastLocationEvent(ctx context.Context, userID xid.ID) (*LocationEvent, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LocationEvent), args.Error(1)
}

func (m *MockRepository) DeleteOldLocationEvents(ctx context.Context, before time.Time) (int64, error) {
	args := m.Called(ctx, before)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRepository) CreateTravelAlert(ctx context.Context, alert *TravelAlert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *MockRepository) GetTravelAlert(ctx context.Context, id xid.ID) (*TravelAlert, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TravelAlert), args.Error(1)
}

func (m *MockRepository) GetUserTravelAlerts(ctx context.Context, userID xid.ID, status string) ([]*TravelAlert, error) {
	args := m.Called(ctx, userID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*TravelAlert), args.Error(1)
}

func (m *MockRepository) GetPendingTravelAlerts(ctx context.Context, orgID xid.ID) ([]*TravelAlert, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*TravelAlert), args.Error(1)
}

func (m *MockRepository) UpdateTravelAlert(ctx context.Context, alert *TravelAlert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *MockRepository) ApproveTravel(ctx context.Context, alertID xid.ID, approvedBy xid.ID) error {
	args := m.Called(ctx, alertID, approvedBy)
	return args.Error(0)
}

func (m *MockRepository) DenyTravel(ctx context.Context, alertID xid.ID, deniedBy xid.ID) error {
	args := m.Called(ctx, alertID, deniedBy)
	return args.Error(0)
}

func (m *MockRepository) CreateTrustedLocation(ctx context.Context, location *TrustedLocation) error {
	args := m.Called(ctx, location)
	return args.Error(0)
}

func (m *MockRepository) GetTrustedLocation(ctx context.Context, id xid.ID) (*TrustedLocation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TrustedLocation), args.Error(1)
}

func (m *MockRepository) GetUserTrustedLocations(ctx context.Context, userID xid.ID) ([]*TrustedLocation, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*TrustedLocation), args.Error(1)
}

func (m *MockRepository) UpdateTrustedLocation(ctx context.Context, location *TrustedLocation) error {
	args := m.Called(ctx, location)
	return args.Error(0)
}

func (m *MockRepository) DeleteTrustedLocation(ctx context.Context, id xid.ID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) IsLocationTrusted(ctx context.Context, userID xid.ID, lat, lon float64) (bool, *TrustedLocation, error) {
	args := m.Called(ctx, userID, lat, lon)
	if args.Get(1) == nil {
		return args.Bool(0), nil, args.Error(2)
	}
	return args.Bool(0), args.Get(1).(*TrustedLocation), args.Error(2)
}

func (m *MockRepository) CreateViolation(ctx context.Context, violation *GeofenceViolation) error {
	args := m.Called(ctx, violation)
	return args.Error(0)
}

func (m *MockRepository) GetViolation(ctx context.Context, id xid.ID) (*GeofenceViolation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GeofenceViolation), args.Error(1)
}

func (m *MockRepository) GetUserViolations(ctx context.Context, userID xid.ID, limit int) ([]*GeofenceViolation, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*GeofenceViolation), args.Error(1)
}

func (m *MockRepository) GetOrganizationViolations(ctx context.Context, orgID xid.ID, limit int) ([]*GeofenceViolation, error) {
	args := m.Called(ctx, orgID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*GeofenceViolation), args.Error(1)
}

func (m *MockRepository) GetUnresolvedViolations(ctx context.Context, orgID xid.ID) ([]*GeofenceViolation, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*GeofenceViolation), args.Error(1)
}

func (m *MockRepository) ResolveViolation(ctx context.Context, id xid.ID, resolvedBy xid.ID, resolution string) error {
	args := m.Called(ctx, id, resolvedBy, resolution)
	return args.Error(0)
}

func (m *MockRepository) GetCachedGeoData(ctx context.Context, ip string) (*GeoCache, error) {
	args := m.Called(ctx, ip)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GeoCache), args.Error(1)
}

func (m *MockRepository) SetCachedGeoData(ctx context.Context, cache *GeoCache) error {
	args := m.Called(ctx, cache)
	return args.Error(0)
}

func (m *MockRepository) DeleteExpiredCache(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

// MockGeoProvider implements GeoProvider for testing
type MockGeoProvider struct {
	mock.Mock
}

func (m *MockGeoProvider) Lookup(ctx context.Context, ip string) (*GeoData, error) {
	args := m.Called(ctx, ip)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GeoData), args.Error(1)
}

func (m *MockGeoProvider) Name() string {
	return "mock"
}

// MockDetectionProvider implements DetectionProvider for testing
type MockDetectionProvider struct {
	mock.Mock
}

func (m *MockDetectionProvider) Check(ctx context.Context, ip string) (*DetectionResult, error) {
	args := m.Called(ctx, ip)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DetectionResult), args.Error(1)
}

func (m *MockDetectionProvider) Name() string {
	return "mock"
}

// Test Service Creation
func TestNewService(t *testing.T) {
	config := DefaultConfig()
	repo := new(MockRepository)
	geoProvider := new(MockGeoProvider)
	detectionProvider := new(MockDetectionProvider)

	service := NewService(config, repo, geoProvider, detectionProvider, nil, nil)

	assert.NotNil(t, service)
	assert.Equal(t, config, service.config)
	assert.Equal(t, repo, service.repo)
}

// Test Country Blocking
func TestCheckLocation_BlockedCountry(t *testing.T) {
	config := DefaultConfig()
	config.Restrictions.BlockedCountries = []string{"KP", "IR"}

	repo := new(MockRepository)
	geoProvider := new(MockGeoProvider)

	service := NewService(config, repo, geoProvider, nil, nil, nil)

	// Mock geolocation response
	lat := 39.0392
	lon := 125.7625
	geoData := &GeoData{
		IPAddress:   "1.2.3.4",
		Country:     "North Korea",
		CountryCode: "KP",
		City:        "Pyongyang",
		Latitude:    &lat,
		Longitude:   &lon,
	}
	geoProvider.On("Lookup", mock.Anything, "1.2.3.4").Return(geoData, nil)

	// Mock rule retrieval
	rule := &GeofenceRule{
		ID:               xid.New(),
		OrganizationID:   xid.New(),
		Name:             "Block Sanctioned",
		Enabled:          true,
		Priority:         100,
		RuleType:         "country",
		BlockedCountries: []string{"KP", "IR"},
		Action:           "deny",
	}
	userID := xid.New()
	repo.On("ListEnabledRules", mock.Anything, mock.Anything, &userID).Return([]*GeofenceRule{rule}, nil)
	repo.On("CreateViolation", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateLocationEvent", mock.Anything, mock.Anything).Return(nil)
	repo.On("GetCachedGeoData", mock.Anything, "1.2.3.4").Return(nil, assert.AnError)
	repo.On("SetCachedGeoData", mock.Anything, mock.Anything).Return(nil)

	// Check location
	req := &LocationCheckRequest{
		UserID:         userID,
		OrganizationID: xid.New(),
		IPAddress:      "1.2.3.4",
		EventType:      "login",
	}

	result, err := service.CheckLocation(context.Background(), req)

	assert.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, "country_blocked", result.Reason)
	assert.Contains(t, result.Violations, "country_blocked")
}

// Test Allowed Country
func TestCheckLocation_AllowedCountry(t *testing.T) {
	config := DefaultConfig()

	repo := new(MockRepository)
	geoProvider := new(MockGeoProvider)

	service := NewService(config, repo, geoProvider, nil, nil, nil)

	// Mock geolocation response
	lat := 37.7749
	lon := -122.4194
	geoData := &GeoData{
		IPAddress:   "8.8.8.8",
		Country:     "United States",
		CountryCode: "US",
		City:        "San Francisco",
		Latitude:    &lat,
		Longitude:   &lon,
	}
	geoProvider.On("Lookup", mock.Anything, "8.8.8.8").Return(geoData, nil)

	// Mock no rules
	userID := xid.New()
	repo.On("ListEnabledRules", mock.Anything, mock.Anything, &userID).Return([]*GeofenceRule{}, nil)
	repo.On("CreateLocationEvent", mock.Anything, mock.Anything).Return(nil)
	repo.On("GetCachedGeoData", mock.Anything, "8.8.8.8").Return(nil, assert.AnError)
	repo.On("SetCachedGeoData", mock.Anything, mock.Anything).Return(nil)
	repo.On("GetLastLocationEvent", mock.Anything, userID).Return(nil, assert.AnError)

	// Check location
	req := &LocationCheckRequest{
		UserID:         userID,
		OrganizationID: xid.New(),
		IPAddress:      "8.8.8.8",
		EventType:      "login",
	}

	result, err := service.CheckLocation(context.Background(), req)

	assert.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Empty(t, result.Violations)
}

// Test VPN Detection
func TestCheckLocation_VPNDetected(t *testing.T) {
	config := DefaultConfig()
	config.Detection.BlockVPN = true

	repo := new(MockRepository)
	geoProvider := new(MockGeoProvider)
	detectionProvider := new(MockDetectionProvider)

	service := NewService(config, repo, geoProvider, detectionProvider, nil, nil)

	// Mock geolocation
	lat := 37.7749
	lon := -122.4194
	geoData := &GeoData{
		IPAddress:   "1.2.3.4",
		Country:     "United States",
		CountryCode: "US",
		Latitude:    &lat,
		Longitude:   &lon,
	}
	geoProvider.On("Lookup", mock.Anything, "1.2.3.4").Return(geoData, nil)

	// Mock VPN detection
	detectionResult := &DetectionResult{
		IPAddress:   "1.2.3.4",
		IsVPN:       true,
		VPNProvider: "NordVPN",
	}
	detectionProvider.On("Check", mock.Anything, "1.2.3.4").Return(detectionResult, nil)

	// Mock rule with VPN blocking
	rule := &GeofenceRule{
		ID:             xid.New(),
		OrganizationID: xid.New(),
		Name:           "Block VPNs",
		Enabled:        true,
		Priority:       100,
		RuleType:       "detection",
		BlockVPN:       true,
		Action:         "deny",
	}
	userID := xid.New()
	repo.On("ListEnabledRules", mock.Anything, mock.Anything, &userID).Return([]*GeofenceRule{rule}, nil)
	repo.On("CreateViolation", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateLocationEvent", mock.Anything, mock.Anything).Return(nil)
	repo.On("GetCachedGeoData", mock.Anything, "1.2.3.4").Return(nil, assert.AnError)
	repo.On("SetCachedGeoData", mock.Anything, mock.Anything).Return(nil)

	// Check location
	req := &LocationCheckRequest{
		UserID:         userID,
		OrganizationID: xid.New(),
		IPAddress:      "1.2.3.4",
		EventType:      "login",
	}

	result, err := service.CheckLocation(context.Background(), req)

	assert.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, "vpn_detected", result.Reason)
}

// Test Geofence Circle
func TestIsInsideGeofence_Circle(t *testing.T) {
	service := &Service{}

	centerLat := 37.7749
	centerLon := -122.4194
	radiusKm := 10.0

	rule := &GeofenceRule{
		GeofenceType: "circle",
		CenterLat:    &centerLat,
		CenterLon:    &centerLon,
		RadiusKm:     &radiusKm,
	}

	// Point inside circle (5km away)
	inside := service.isInsideGeofence(rule, 37.8199, -122.4783)
	assert.True(t, inside)

	// Point outside circle (50km away)
	outside := service.isInsideGeofence(rule, 37.3352, -121.8811)
	assert.False(t, outside)
}

// Test Point in Polygon
func TestPointInPolygon(t *testing.T) {
	// Square polygon around San Francisco
	polygon := [][2]float64{
		{37.8, -122.5},
		{37.8, -122.3},
		{37.7, -122.3},
		{37.7, -122.5},
	}

	// Point inside
	inside := pointInPolygon(37.75, -122.4, polygon)
	assert.True(t, inside)

	// Point outside
	outside := pointInPolygon(37.9, -122.4, polygon)
	assert.False(t, outside)
}

// Test Caching
func TestGetGeolocation_WithCache(t *testing.T) {
	config := DefaultConfig()
	repo := new(MockRepository)
	geoProvider := new(MockGeoProvider)

	service := NewService(config, repo, geoProvider, nil, nil, nil)

	lat := 37.7749
	lon := -122.4194
	cache := &GeoCache{
		IPAddress:   "8.8.8.8",
		Country:     "United States",
		CountryCode: "US",
		City:        "Mountain View",
		Latitude:    &lat,
		Longitude:   &lon,
		Provider:    "maxmind",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}

	// Mock cache hit
	repo.On("GetCachedGeoData", mock.Anything, "8.8.8.8").Return(cache, nil)

	// Get geolocation (should use cache, not call provider)
	result, err := service.GetGeolocation(context.Background(), "8.8.8.8")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "8.8.8.8", result.IPAddress)
	assert.Equal(t, "United States", result.Country)
	assert.Equal(t, "US", result.CountryCode)

	// Verify provider was NOT called
	geoProvider.AssertNotCalled(t, "Lookup")
}

func TestGetGeolocation_CacheMiss(t *testing.T) {
	config := DefaultConfig()
	repo := new(MockRepository)
	geoProvider := new(MockGeoProvider)

	service := NewService(config, repo, geoProvider, nil, nil, nil)

	lat := 37.7749
	lon := -122.4194
	geoData := &GeoData{
		IPAddress:   "8.8.8.8",
		Country:     "United States",
		CountryCode: "US",
		City:        "Mountain View",
		Latitude:    &lat,
		Longitude:   &lon,
		Provider:    "maxmind",
	}

	// Mock cache miss
	repo.On("GetCachedGeoData", mock.Anything, "8.8.8.8").Return(nil, assert.AnError)
	repo.On("SetCachedGeoData", mock.Anything, mock.Anything).Return(nil)

	// Mock provider lookup
	geoProvider.On("Lookup", mock.Anything, "8.8.8.8").Return(geoData, nil)

	// Get geolocation (should call provider and cache result)
	result, err := service.GetGeolocation(context.Background(), "8.8.8.8")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "8.8.8.8", result.IPAddress)

	// Verify provider was called
	geoProvider.AssertCalled(t, "Lookup", mock.Anything, "8.8.8.8")
	repo.AssertCalled(t, "SetCachedGeoData", mock.Anything, mock.Anything)
}

