# Geofencing Plugin

Enterprise-grade geographic fencing and location-based security for AuthSome. Control access based on user location, detect VPNs/proxies, monitor travel patterns, and enforce GPS-based authentication.

## Features

### 🌍 Geographic Restrictions
- **Country/Region Controls**: Allowlist or blocklist specific countries and regions
- **City-Level Restrictions**: Fine-grained access control at city level
- **Geofencing**: Define circular or polygonal geographic boundaries
- **Distance-Based Controls**: Maximum distance from reference points
- **Time-Based Rules**: Different restrictions by time of day/week

### 🔍 IP Geolocation
- **Multiple Providers**: MaxMind, IPInfo.io, ipapi.com, ipgeolocation.io
- **Automatic Fallback**: Switch to backup provider on failure
- **Caching**: Efficient caching to minimize API calls
- **High Accuracy**: Support for multiple accuracy levels

### 📱 GPS-Based Authentication
- **Device GPS**: Require GPS coordinates from mobile devices
- **Accuracy Validation**: Enforce minimum/maximum accuracy thresholds
- **Geofence Matching**: Verify coordinates within defined geofences
- **Timestamp Validation**: Prevent replay attacks with timestamp checks

### 🛡️ VPN/Proxy Detection
- **VPN Detection**: Identify and block VPN connections
- **Proxy Detection**: Detect HTTP/SOCKS proxies
- **Tor Detection**: Block Tor exit nodes
- **Datacenter Detection**: Identify datacenter IPs
- **Fraud Scoring**: Risk scoring for suspicious connections

### 🏢 Corporate Network Detection
- **Network Ranges**: Define corporate IP ranges (CIDR)
- **DNS-Based Detection**: Verify expected DNS servers
- **Certificate-Based**: Require trusted certificates
- **Hybrid Mode**: Allow external with additional verification

### ✈️ Travel Notifications
- **Impossible Travel Detection**: Flag physically impossible travel speeds
- **Distance Thresholds**: Alert on significant location changes
- **Travel Approval**: Require admin approval for suspicious travel
- **Notification Channels**: Email, SMS, push, webhooks
- **Trusted Destinations**: Whitelist frequent locations

## Installation

### 1. Enable the Plugin

```go
package main

import (
    "github.com/xraph/authsome"
    "github.com/xraph/authsome/plugins/enterprise/geofence"
)

func main() {
    auth := authsome.New(
        authsome.WithDatabase(db),
        authsome.WithForgeApp(app),
    )

    // Register geofence plugin
    geofencePlugin := geofence.NewPlugin()
    auth.RegisterPlugin(geofencePlugin)

    // Initialize
    auth.Initialize(context.Background())
}
```

### 2. Configuration

Add to your configuration file (`config.yaml`):

```yaml
auth:
  geofence:
    enabled: true
    
    # Geographic Restrictions
    restrictions:
      defaultAction: allow  # or "deny"
      strictMode: false
      allowedCountries: []
      blockedCountries: ["KP", "IR", "SY"]  # Example: sanctioned countries
      maxDistanceKm: 0  # 0 = unlimited
      
    # Geolocation Provider
    geolocation:
      provider: maxmind  # maxmind, ipapi, ipinfo, ipgeolocation
      maxmindLicenseKey: "your-license-key"
      maxmindDatabasePath: "/path/to/GeoLite2-City.mmdb"
      fallbackProvider: ipapi
      ipapiKey: "your-api-key"
      cacheDuration: 24h
      timeout: 5s
      
    # VPN/Proxy Detection
    detection:
      detectVpn: true
      blockVpn: false  # Set to true to block VPNs
      detectProxy: true
      blockProxy: false
      detectTor: true
      blockTor: false
      provider: ipqs  # ipqs, proxycheck, vpnapi
      ipqsKey: "your-ipqs-key"
      ipqsStrictness: 1  # 0-3
      ipqsMinScore: 75.0
      cacheDuration: 1h
      
    # GPS Authentication
    gps:
      enabled: false
      requireGps: false
      maxAccuracyMeters: 1000
      maxSpeedKmh: 1000
      validateTimestamp: true
      maxTimestampAge: 5m
      
    # Travel Notifications
    travel:
      enabled: true
      minDistanceKm: 500  # Trigger at 500km
      maxSpeedKmh: 900    # Flag impossible travel
      notifyUser: true
      notifyAdmin: false
      requireApproval: false
      emailNotify: true
      autoApproveAfter: 24h
      trustFrequentDest: true
      
    # Session Management
    session:
      trackLocation: true
      updateInterval: 5m
      validateOnRequest: true
      invalidateOnViolation: false
      gracePeriod: 10m
      maxViolations: 3
      
    # API Configuration
    api:
      basePath: /auth/geofence
      enableManagement: true
      enableValidation: true
      enableMetrics: true
      
    # Security
    security:
      rateLimitEnabled: true
      maxChecksPerMinute: 60
      auditViolations: true
      storeLocations: true
      locationRetention: 2160h  # 90 days
      anonymizeOldData: true
```

## Usage

### Creating Geofence Rules

#### Block Specific Countries

```go
rule := &geofence.GeofenceRule{
    AppID: orgID,
    Name: "Block Sanctioned Countries",
    Description: "Prevent access from sanctioned countries",
    Enabled: true,
    Priority: 100,
    RuleType: "country",
    BlockedCountries: []string{"KP", "IR", "SY", "CU"},
    Action: "deny",
    NotifyAdmin: true,
}

err := service.repo.CreateRule(ctx, rule)
```

#### Allow Only Specific Regions

```go
rule := &geofence.GeofenceRule{
    AppID: orgID,
    Name: "US Only",
    Description: "Allow access only from United States",
    Enabled: true,
    Priority: 90,
    RuleType: "country",
    AllowedCountries: []string{"US"},
    Action: "deny",
    RequireMFA: true,  # Require MFA for violations
}
```

#### Circular Geofence (Office Radius)

```go
centerLat := 37.7749
centerLon := -122.4194
radiusKm := 10.0

rule := &geofence.GeofenceRule{
    AppID: orgID,
    Name: "Office Geofence",
    Description: "Access only within 10km of San Francisco office",
    Enabled: true,
    Priority: 80,
    RuleType: "geofence",
    GeofenceType: "circle",
    CenterLat: &centerLat,
    CenterLon: &centerLon,
    RadiusKm: &radiusKm,
    Action: "deny",
}
```

#### Polygonal Geofence (Campus)

```go
// Define polygon coordinates (latitude, longitude pairs)
coordinates := [][2]float64{
    {37.7749, -122.4194},
    {37.7750, -122.4180},
    {37.7735, -122.4175},
    {37.7730, -122.4190},
}

rule := &geofence.GeofenceRule{
    AppID: orgID,
    Name: "Campus Boundary",
    Description: "Access only within campus boundaries",
    Enabled: true,
    Priority: 85,
    RuleType: "geofence",
    GeofenceType: "polygon",
    Coordinates: coordinates,
    Action: "deny",
}
```

#### Block VPNs and Proxies

```go
rule := &geofence.GeofenceRule{
    AppID: orgID,
    Name: "Block Anonymous Connections",
    Description: "Prevent VPN, proxy, and Tor access",
    Enabled: true,
    Priority: 95,
    RuleType: "detection",
    BlockVPN: true,
    BlockProxy: true,
    BlockTor: true,
    Action: "deny",
    NotifyAdmin: true,
}
```

#### Time-Based Restrictions

```go
timeRestrictions := []geofence.TimeRestrictionRule{
    {
        AllowedDays: []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"},
        StartHour: 9,   # 9 AM
        EndHour: 17,    # 5 PM
        Timezone: "America/New_York",
    },
}

rule := &geofence.GeofenceRule{
    AppID: orgID,
    Name: "Business Hours Only",
    Description: "Access only during business hours",
    Enabled: true,
    Priority: 70,
    RuleType: "time",
    TimeRestrictions: timeRestrictions,
    Action: "deny",
}
```

### Using Middleware

#### Automatic Geofence Checking

```go
// Apply geofence middleware to protected routes
protected := router.Group("/api")
protected.Use(geofencePlugin.Middleware())

protected.GET("/data", handlers.GetData)
protected.POST("/action", handlers.PerformAction)
```

#### Country-Specific Middleware

```go
// Require requests from specific countries
usOnly := geofencePlugin.Middleware().RequireCountry("US", "CA")
router.Group("/us-only").Use(usOnly)
```

#### Block VPN Middleware

```go
// Block all VPN connections
noVPN := geofencePlugin.Middleware().BlockVPN
router.Group("/sensitive").Use(noVPN)
```

### Manual Location Checks

```go
// Perform ad-hoc location check
req := &geofence.LocationCheckRequest{
    UserID: userID,
    AppID: orgID,
    IPAddress: "8.8.8.8",
    EventType: "login",
}

result, err := geofencePlugin.Service().CheckLocation(ctx, req)
if err != nil {
    // Handle error
}

if !result.Allowed {
    // Access denied
    log.Printf("Access denied: %s", result.Reason)
}

if result.RequireMFA {
    // Enforce MFA
}

if result.TravelAlert {
    // Handle travel alert
}
```

### Managing Trusted Locations

```go
// Add a trusted location
trustedLoc := &geofence.TrustedLocation{
    UserID: userID,
    AppID: orgID,
    Name: "Home",
    Country: "United States",
    CountryCode: "US",
    City: "San Francisco",
    Latitude: 37.7749,
    Longitude: -122.4194,
    RadiusKm: 2.0,
    AutoApprove: true,
    SkipMFA: false,
}

err := service.repo.CreateTrustedLocation(ctx, trustedLoc)
```

### Handling Travel Alerts

```go
// Get pending travel alerts for organization
alerts, err := service.repo.GetPendingTravelAlerts(ctx, appID)

for _, alert := range alerts {
    if alert.Severity == "critical" {
        // Notify admin
        log.Printf("Critical travel alert for user %s: %.0f km in %s", 
            alert.UserID, alert.DistanceKm, alert.TimeDifference)
    }
}

// Approve travel
err = service.repo.ApproveTravel(ctx, alert.ID, adminUserID)

// Or deny travel
err = service.repo.DenyTravel(ctx, alert.ID, adminUserID)
```

## API Endpoints

### Rule Management

```bash
# Create rule
POST /auth/geofence/rules

# List rules
GET /auth/geofence/rules

# Get specific rule
GET /auth/geofence/rules/:id

# Update rule
PUT /auth/geofence/rules/:id

# Delete rule
DELETE /auth/geofence/rules/:id
```

### Location Validation

```bash
# Check location
POST /auth/geofence/check
{
  "ipAddress": "8.8.8.8",
  "userId": "user-id",
  "eventType": "login"
}

# IP lookup
GET /auth/geofence/lookup/8.8.8.8
```

### Travel Alerts

```bash
# List travel alerts
GET /auth/geofence/travel-alerts?status=pending

# Get specific alert
GET /auth/geofence/travel-alerts/:id

# Approve travel
POST /auth/geofence/travel-alerts/:id/approve

# Deny travel
POST /auth/geofence/travel-alerts/:id/deny
```

### Trusted Locations

```bash
# Create trusted location
POST /auth/geofence/trusted-locations

# List trusted locations
GET /auth/geofence/trusted-locations

# Update trusted location
PUT /auth/geofence/trusted-locations/:id

# Delete trusted location
DELETE /auth/geofence/trusted-locations/:id
```

### Violations

```bash
# List violations
GET /auth/geofence/violations?limit=50

# Get specific violation
GET /auth/geofence/violations/:id

# Resolve violation
POST /auth/geofence/violations/:id/resolve
{
  "resolution": "approved_exception"
}
```

## Use Cases

### 1. Compliance with Data Residency Laws

```go
// Only allow access from EU for GDPR compliance
rule := &geofence.GeofenceRule{
    Name: "GDPR Data Residency",
    AllowedCountries: []string{"DE", "FR", "IT", "ES", "NL", "BE", "AT", "IE"},
    Action: "deny",
}
```

### 2. Prevent Fraud from High-Risk Countries

```go
// Block countries known for high fraud rates
rule := &geofence.GeofenceRule{
    Name: "Fraud Prevention",
    BlockedCountries: []string{"NG", "PK", "BD", "RO"},
    BlockVPN: true,
    BlockProxy: true,
    Action: "deny",
    NotifyAdmin: true,
}
```

### 3. Office-Only Access for Sensitive Operations

```go
// Require physical presence at office for admin operations
rule := &geofence.GeofenceRule{
    Name: "Admin Office Only",
    GeofenceType: "circle",
    CenterLat: &officeLat,
    CenterLon: &officeLon,
    RadiusKm: &radius,
    Action: "deny",
    RequireMFA: true,
}
```

### 4. Detect Account Takeover

```go
// Monitor for impossible travel patterns
config.Travel.Enabled = true
config.Travel.MaxSpeedKmh = 900  # Commercial aircraft speed
config.Travel.RequireApproval = true
config.Travel.NotifyUser = true
config.Travel.NotifyAdmin = true
```

### 5. Regional Licensing Restrictions

```go
// Enforce software licensing by region
rule := &geofence.GeofenceRule{
    Name: "North America License",
    AllowedCountries: []string{"US", "CA", "MX"},
    Action: "deny",
}
```

## Session Security Notifications

The geofence plugin integrates with the notification system to automatically alert users about security-relevant session events.

### New Location Login Notifications

Automatically notify users when they log in from a significantly different location:

```yaml
auth:
  geofence:
    notifications:
      enabled: true
      newLocationEnabled: true
      newLocationThresholdKm: 100  # Trigger at 100km distance from last location
```

**How it works:**
1. User signs in from a new location
2. Geofence plugin detects location change > threshold
3. Email sent to user with location details
4. Location stored for future comparisons

**Example notification:**
> "We noticed a new sign-in to your account from San Francisco, CA (500 km from previous location: Los Angeles, CA) at 2025-12-14 10:30 AM. IP: 8.8.8.8"

### Suspicious Login Detection

Automatically detect and notify users about suspicious login patterns:

```yaml
auth:
  geofence:
    notifications:
      enabled: true
      suspiciousLoginEnabled: true
      suspiciousLoginScoreThreshold: 75.0  # IPQS fraud score threshold
      impossibleTravelEnabled: true
      vpnDetectionEnabled: true
      proxyDetectionEnabled: true
      torDetectionEnabled: true
```

**Detection triggers:**
- **Impossible Travel**: e.g., 5000km in 1 hour (faster than aircraft)
- **VPN Usage**: VPN connection detected (configurable providers)
- **Proxy Detection**: HTTP/SOCKS proxy detected
- **Tor Exit Node**: Connection from Tor network
- **High Fraud Score**: IPQS score above threshold (default: 75/100)

**Example notifications:**
> "Suspicious login detected: Impossible travel - 8500 km in 2.5 hours (3400 km/h, max: 900 km/h). Please verify this was you."

> "Suspicious login detected: VPN detected - NordVPN. If this wasn't you, secure your account immediately."

### Configuration Reference

```yaml
auth:
  geofence:
    # Enable notification integration
    notifications:
      enabled: true
      
      # New location alerts
      newLocationEnabled: true
      newLocationThresholdKm: 100  # Distance threshold in km
      
      # Suspicious login alerts
      suspiciousLoginEnabled: true
      suspiciousLoginScoreThreshold: 75.0  # 0-100 fraud score
      
      # Detection types for suspicious login
      impossibleTravelEnabled: true
      vpnDetectionEnabled: true
      proxyDetectionEnabled: true
      torDetectionEnabled: true
    
    # Geolocation provider (required for location tracking)
    geolocation:
      provider: maxmind
      maxmindLicenseKey: "your-license-key"
      maxmindDatabasePath: "/path/to/GeoLite2-City.mmdb"
    
    # Detection provider (required for suspicious login)
    detection:
      detectVpn: true
      detectProxy: true
      detectTor: true
      provider: ipqs
      ipqsKey: "your-ipqs-key"
      ipqsStrictness: 1
```

### Disabling Notifications

To disable session security notifications while keeping geofencing active:

```yaml
auth:
  geofence:
    enabled: true  # Geofencing still active
    notifications:
      enabled: false  # Disable notifications
```

Or disable specific notification types:

```yaml
auth:
  geofence:
    notifications:
      enabled: true
      newLocationEnabled: false  # Disable new location alerts
      suspiciousLoginEnabled: true  # Keep suspicious login alerts
```

## Multi-Tenancy Support

The geofencing plugin fully supports AuthSome's multi-tenancy:

```go
// App-wide rules
rule.AppID = appID
rule.UserID = nil  // Applies to all users in app

// User-specific rules
rule.AppID = appID
rule.UserID = &specificUserID  // Only for this user
```

## Performance Considerations

### Caching

- **Geolocation**: Cached for 24h by default (configurable)
- **Detection**: Cached for 1h by default
- **Database Cache**: Persistent caching layer for geolocation data
- **Memory Cache**: In-memory LRU cache for frequently accessed IPs

### Rate Limiting

- **Per-Minute Limits**: Default 60 checks/minute
- **Per-Hour Limits**: Default 1000 checks/hour
- Configurable per organization

### Database Optimization

- Indexed tables for fast queries
- Automatic cleanup of old location events
- Optimized geospatial queries

## Privacy & Compliance

### Data Retention

```yaml
security:
  locationRetention: 2160h  # 90 days
  anonymizeOldData: true    # Auto-anonymize after retention
```

### Consent Management

```yaml
security:
  consentRequired: true     # Require user consent
  allowOptOut: false        # Allow users to opt out
```

### GDPR Compliance

- Users can request location data export
- Automatic data anonymization
- Consent tracking
- Right to be forgotten support

## Security Best Practices

1. **Use Strict Mode** for critical operations
2. **Enable VPN/Proxy Detection** for sensitive data
3. **Implement Travel Notifications** to detect account takeovers
4. **Regular Rule Audits** to ensure policies are current
5. **Monitor Violations** for security incidents
6. **Use Trusted Locations** to reduce false positives
7. **Enable Audit Logging** for compliance
8. **Test Fallback Providers** to ensure availability

## Troubleshooting

### Geolocation Provider Issues

```go
// Test provider health
err := geofencePlugin.Health(ctx)
if err != nil {
    log.Printf("Provider unhealthy: %v", err)
}

// Check provider configuration
provider := geofencePlugin.Service().geoProvider
log.Printf("Using provider: %s", provider.Name())
```

### False Positives

1. Add trusted locations for frequent users
2. Adjust accuracy thresholds
3. Enable grace periods
4. Use detection confidence scores

### Performance Issues

1. Increase cache durations
2. Use MaxMind local database instead of API
3. Enable database caching
4. Optimize rule priorities

## License

Enterprise feature - requires AuthSome Enterprise license.

## Support

For issues and questions:
- GitHub Issues: https://github.com/xraph/authsome/issues
- Documentation: https://authsome.dev/docs/plugins/geofence
- Enterprise Support: enterprise@authsome.dev

