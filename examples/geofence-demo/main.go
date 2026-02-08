package main

import (
	"log"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/enterprise/geofence"
)

func main() {

	// Create a new plugin
	_ = geofence.NewPlugin()

	// Test configuration
	config := geofence.DefaultConfig()

	// Validate configuration
	if err := config.Validate(); err != nil {
		log.Fatal("Config validation failed:", err)
	}

	// Test haversine distance calculation

	// San Francisco to New York
	sfLat, sfLon := 37.7749, -122.4194
	nyLat, nyLon := 40.7128, -74.0060
	distance := calculateDistance(sfLat, sfLon, nyLat, nyLon)

	// San Francisco to London
	londonLat, londonLon := 51.5074, -0.1278
	distance = calculateDistance(sfLat, sfLon, londonLat, londonLon)

	// Nearby points
	distance = calculateDistance(sfLat, sfLon, 37.8199, -122.4783)

	// Test geofence rule creation

	// Country blocking rule
	_ = &geofence.GeofenceRule{
		ID:               xid.New(),
		AppID:            xid.New(),
		Name:             "Block Sanctioned Countries",
		Description:      "Prevent access from high-risk locations",
		Enabled:          true,
		Priority:         100,
		RuleType:         "country",
		BlockedCountries: []string{"KP", "IR", "SY"},
		Action:           "deny",
		NotifyAdmin:      true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// VPN detection rule
	_ = &geofence.GeofenceRule{
		ID:          xid.New(),
		AppID:       xid.New(),
		Name:        "Block VPNs",
		Description: "Prevent anonymous connections",
		Enabled:     true,
		Priority:    90,
		RuleType:    "detection",
		BlockVPN:    true,
		BlockProxy:  true,
		Action:      "deny",
		NotifyAdmin: true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Circular geofence rule
	centerLat := 37.7749
	centerLon := -122.4194
	radiusKm := 10.0
	_ = &geofence.GeofenceRule{
		ID:           xid.New(),
		AppID:        xid.New(),
		Name:         "Office Geofence",
		Description:  "Access only within 10km of office",
		Enabled:      true,
		Priority:     85,
		RuleType:     "geofence",
		GeofenceType: "circle",
		CenterLat:    &centerLat,
		CenterLon:    &centerLon,
		RadiusKm:     &radiusKm,
		Action:       "deny",
		RequireMFA:   true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Test point-in-polygon

	polygon := [][2]float64{
		{37.8, -122.5},
		{37.8, -122.3},
		{37.7, -122.3},
		{37.7, -122.5},
	}

	testPoint1 := [2]float64{37.75, -122.4}
	_ = pointInPolygon(testPoint1[0], testPoint1[1], polygon)

	testPoint2 := [2]float64{37.9, -122.4}
	_ = pointInPolygon(testPoint2[0], testPoint2[1], polygon)

	// Test location event

	lat := 37.7749
	lon := -122.4194
	_ = &geofence.LocationEvent{
		ID:          xid.New(),
		UserID:      xid.New(),
		AppID:       xid.New(),
		IPAddress:   "8.8.8.8",
		Country:     "United States",
		CountryCode: "US",
		Region:      "California",
		City:        "San Francisco",
		Latitude:    &lat,
		Longitude:   &lon,
		EventType:   "login",
		EventResult: "allowed",
		Timestamp:   time.Now(),
	}

	// Test travel alert

	// Simulate travel from SF to NY in 2 hours (impossible)
	travelTime := 2 * time.Hour
	speed := distance / travelTime.Hours()

	_ = &geofence.TravelAlert{
		ID:               xid.New(),
		UserID:           xid.New(),
		AppID:            xid.New(),
		AlertType:        "impossible_travel",
		Severity:         "critical",
		FromCountry:      "United States",
		FromCity:         "San Francisco",
		ToCountry:        "United States",
		ToCity:           "New York",
		DistanceKm:       4130.5, // Approximate SF to NY distance
		TimeDifference:   travelTime,
		CalculatedSpeed:  speed,
		Status:           "pending",
		RequiresApproval: true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Test trusted location

	_ = &geofence.TrustedLocation{
		ID:          xid.New(),
		UserID:      xid.New(),
		AppID:       xid.New(),
		Name:        "Home",
		Description: "Primary residence",
		Country:     "United States",
		CountryCode: "US",
		City:        "San Francisco",
		Latitude:    37.7749,
		Longitude:   -122.4194,
		RadiusKm:    2.0,
		AutoApprove: true,
		SkipMFA:     false,
		UsageCount:  15,
		FirstUsedAt: time.Now().Add(-30 * 24 * time.Hour),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

}

// Helper function to calculate distance (same as haversineDistance)
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0
	const pi = 3.14159265358979323846

	dLat := (lat2 - lat1) * pi / 180.0
	dLon := (lon2 - lon1) * pi / 180.0

	lat1Rad := lat1 * pi / 180.0
	lat2Rad := lat2 * pi / 180.0

	a := (1-cosine(dLat))/2 + cosine(lat1Rad)*cosine(lat2Rad)*(1-cosine(dLon))/2
	c := 2 * asin(sqrt(a))

	return earthRadiusKm * c
}

// Helper functions for distance calculation
func sine(x float64) float64 {
	// Taylor series approximation
	return x - (x*x*x)/6 + (x*x*x*x*x)/120
}

func cosine(x float64) float64 {
	return 1 - (x*x)/2 + (x*x*x*x)/24
}

func asin(x float64) float64 {
	return x + (x*x*x)/6 + (3*x*x*x*x*x)/40
}

func sqrt(x float64) float64 {
	if x == 0 {
		return 0
	}
	guess := x / 2.0
	for i := 0; i < 10; i++ {
		guess = (guess + x/guess) / 2.0
	}
	return guess
}

// pointInPolygon checks if a point is inside a polygon
func pointInPolygon(lat, lon float64, polygon [][2]float64) bool {
	inside := false
	j := len(polygon) - 1

	for i := 0; i < len(polygon); i++ {
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
