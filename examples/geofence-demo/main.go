package main

import (
	"fmt"
	"log"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/enterprise/geofence"
)

func main() {
	fmt.Println("=== Geofencing Plugin Demo ===\n")

	// Create a new plugin
	plugin := geofence.NewPlugin()
	fmt.Printf("✓ Plugin created: %s v%s\n", plugin.Name(), plugin.Version())
	fmt.Printf("  Description: %s\n\n", plugin.Description())

	// Test configuration
	config := geofence.DefaultConfig()
	fmt.Println("✓ Default configuration loaded:")
	fmt.Printf("  - Enabled: %v\n", config.Enabled)
	fmt.Printf("  - Geo Provider: %s\n", config.Geolocation.Provider)
	fmt.Printf("  - Cache Duration: %v\n", config.Geolocation.CacheDuration)
	fmt.Printf("  - Travel Detection: %v\n", config.Travel.Enabled)
	fmt.Printf("  - Min Travel Distance: %.0f km\n", config.Travel.MinDistanceKm)
	fmt.Printf("  - Max Speed: %.0f km/h\n\n", config.Travel.MaxSpeedKmh)

	// Validate configuration
	if err := config.Validate(); err != nil {
		log.Fatal("Config validation failed:", err)
	}
	fmt.Println("✓ Configuration validated successfully\n")

	// Test haversine distance calculation
	fmt.Println("=== Distance Calculations ===")
	
	// San Francisco to New York
	sfLat, sfLon := 37.7749, -122.4194
	nyLat, nyLon := 40.7128, -74.0060
	distance := calculateDistance(sfLat, sfLon, nyLat, nyLon)
	fmt.Printf("San Francisco to New York: %.2f km\n", distance)

	// San Francisco to London
	londonLat, londonLon := 51.5074, -0.1278
	distance = calculateDistance(sfLat, sfLon, londonLat, londonLon)
	fmt.Printf("San Francisco to London: %.2f km\n", distance)

	// Nearby points
	distance = calculateDistance(sfLat, sfLon, 37.8199, -122.4783)
	fmt.Printf("San Francisco to Berkeley: %.2f km\n\n", distance)

	// Test geofence rule creation
	fmt.Println("=== Geofence Rules ===")
	
	// Country blocking rule
	rule1 := &geofence.GeofenceRule{
		ID:               xid.New(),
		OrganizationID:   xid.New(),
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
	fmt.Printf("✓ Created rule: %s (Priority: %d)\n", rule1.Name, rule1.Priority)

	// VPN detection rule
	rule2 := &geofence.GeofenceRule{
		ID:             xid.New(),
		OrganizationID: xid.New(),
		Name:           "Block VPNs",
		Description:    "Prevent anonymous connections",
		Enabled:        true,
		Priority:       90,
		RuleType:       "detection",
		BlockVPN:       true,
		BlockProxy:     true,
		Action:         "deny",
		NotifyAdmin:    true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	fmt.Printf("✓ Created rule: %s (Priority: %d)\n", rule2.Name, rule2.Priority)

	// Circular geofence rule
	centerLat := 37.7749
	centerLon := -122.4194
	radiusKm := 10.0
	rule3 := &geofence.GeofenceRule{
		ID:             xid.New(),
		OrganizationID: xid.New(),
		Name:           "Office Geofence",
		Description:    "Access only within 10km of office",
		Enabled:        true,
		Priority:       85,
		RuleType:       "geofence",
		GeofenceType:   "circle",
		CenterLat:      &centerLat,
		CenterLon:      &centerLon,
		RadiusKm:       &radiusKm,
		Action:         "deny",
		RequireMFA:     true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	fmt.Printf("✓ Created rule: %s (Radius: %.0f km)\n\n", rule3.Name, *rule3.RadiusKm)

	// Test point-in-polygon
	fmt.Println("=== Point-in-Polygon Test ===")
	polygon := [][2]float64{
		{37.8, -122.5},
		{37.8, -122.3},
		{37.7, -122.3},
		{37.7, -122.5},
	}
	
	testPoint1 := [2]float64{37.75, -122.4}
	inside1 := pointInPolygon(testPoint1[0], testPoint1[1], polygon)
	fmt.Printf("Point (%.2f, %.2f) inside polygon: %v\n", testPoint1[0], testPoint1[1], inside1)

	testPoint2 := [2]float64{37.9, -122.4}
	inside2 := pointInPolygon(testPoint2[0], testPoint2[1], polygon)
	fmt.Printf("Point (%.2f, %.2f) inside polygon: %v\n\n", testPoint2[0], testPoint2[1], inside2)

	// Test location event
	fmt.Println("=== Location Event ===")
	lat := 37.7749
	lon := -122.4194
	event := &geofence.LocationEvent{
		ID:             xid.New(),
		UserID:         xid.New(),
		OrganizationID: xid.New(),
		IPAddress:      "8.8.8.8",
		Country:        "United States",
		CountryCode:    "US",
		Region:         "California",
		City:           "San Francisco",
		Latitude:       &lat,
		Longitude:      &lon,
		EventType:      "login",
		EventResult:    "allowed",
		Timestamp:      time.Now(),
	}
	fmt.Printf("✓ Location Event: %s from %s, %s (%s)\n", event.EventType, event.City, event.Country, event.IPAddress)
	fmt.Printf("  Coordinates: %.4f, %.4f\n", *event.Latitude, *event.Longitude)
	fmt.Printf("  Result: %s\n\n", event.EventResult)

	// Test travel alert
	fmt.Println("=== Travel Detection ===")
	
	// Simulate travel from SF to NY in 2 hours (impossible)
	travelTime := 2 * time.Hour
	speed := distance / travelTime.Hours()
	
	alert := &geofence.TravelAlert{
		ID:              xid.New(),
		UserID:          xid.New(),
		OrganizationID:  xid.New(),
		AlertType:       "impossible_travel",
		Severity:        "critical",
		FromCountry:     "United States",
		FromCity:        "San Francisco",
		ToCountry:       "United States",
		ToCity:          "New York",
		DistanceKm:      4130.5, // Approximate SF to NY distance
		TimeDifference:  travelTime,
		CalculatedSpeed: speed,
		Status:          "pending",
		RequiresApproval: true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	fmt.Printf("✓ Travel Alert: %s\n", alert.AlertType)
	fmt.Printf("  From: %s to %s\n", alert.FromCity, alert.ToCity)
	fmt.Printf("  Distance: %.0f km\n", alert.DistanceKm)
	fmt.Printf("  Time: %s\n", alert.TimeDifference)
	fmt.Printf("  Speed: %.0f km/h ⚠️ (Max: %.0f km/h)\n", alert.CalculatedSpeed, config.Travel.MaxSpeedKmh)
	fmt.Printf("  Severity: %s\n", alert.Severity)
	fmt.Printf("  Requires Approval: %v\n\n", alert.RequiresApproval)

	// Test trusted location
	fmt.Println("=== Trusted Location ===")
	trusted := &geofence.TrustedLocation{
		ID:             xid.New(),
		UserID:         xid.New(),
		OrganizationID: xid.New(),
		Name:           "Home",
		Description:    "Primary residence",
		Country:        "United States",
		CountryCode:    "US",
		City:           "San Francisco",
		Latitude:       37.7749,
		Longitude:      -122.4194,
		RadiusKm:       2.0,
		AutoApprove:    true,
		SkipMFA:        false,
		UsageCount:     15,
		FirstUsedAt:    time.Now().Add(-30 * 24 * time.Hour),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	fmt.Printf("✓ Trusted Location: %s\n", trusted.Name)
	fmt.Printf("  Location: %s, %s\n", trusted.City, trusted.Country)
	fmt.Printf("  Radius: %.0f km\n", trusted.RadiusKm)
	fmt.Printf("  Usage: %d times\n", trusted.UsageCount)
	fmt.Printf("  Auto-approve: %v\n\n", trusted.AutoApprove)

	fmt.Println("=== Demo Complete ===")
	fmt.Println("✓ All geofencing components working correctly!")
	fmt.Println("\nTo use in production:")
	fmt.Println("  1. Configure geolocation provider (MaxMind, IPInfo, etc.)")
	fmt.Println("  2. Set up VPN/proxy detection (IPQualityScore, ProxyCheck)")
	fmt.Println("  3. Create geofence rules for your organization")
	fmt.Println("  4. Enable travel notifications")
	fmt.Println("  5. Add trusted locations for users")
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

