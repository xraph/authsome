package providers

import (
	"context"
	"fmt"
	"net"
	"time"
)

// RequestContext contains ephemeral request-specific data.
type RequestContext struct {
	IP          string            `json:"ip"`
	UserAgent   string            `json:"user_agent"`
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	Timestamp   time.Time         `json:"timestamp"`
	Geolocation *Geolocation      `json:"geolocation,omitempty"`
	Device      *DeviceInfo       `json:"device,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Metadata    map[string]any    `json:"metadata,omitempty"`
}

// Geolocation contains geographic information about the request.
type Geolocation struct {
	Country   string  `json:"country"`
	Region    string  `json:"region"`
	City      string  `json:"city"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timezone  string  `json:"timezone"`
}

// DeviceInfo contains device-specific information.
type DeviceInfo struct {
	Type      string `json:"type"`    // mobile, desktop, tablet
	OS        string `json:"os"`      // iOS, Android, Windows, macOS, Linux
	Browser   string `json:"browser"` // Chrome, Firefox, Safari, etc.
	IsMobile  bool   `json:"is_mobile"`
	IsDesktop bool   `json:"is_desktop"`
}

// ContextAttributeProvider provides request context attributes.
type ContextAttributeProvider struct {
	// No external dependencies - uses the data provided in the request
}

// NewContextAttributeProvider creates a new context attribute provider.
func NewContextAttributeProvider() *ContextAttributeProvider {
	return &ContextAttributeProvider{}
}

// Name returns the provider name.
func (p *ContextAttributeProvider) Name() string {
	return "context"
}

// GetAttributes returns the request context attributes
// key is ignored as context is typically set directly in the evaluation context.
func (p *ContextAttributeProvider) GetAttributes(ctx context.Context, key string) (map[string]any, error) {
	// For context provider, we typically don't fetch by key
	// Instead, the request context should be set directly in the evaluation context
	// This method returns current time-based attributes
	return contextToAttributes(&RequestContext{
		Timestamp: time.Now().UTC(),
	}), nil
}

// GetBatchAttributes returns context attributes for multiple keys
// For context, batch operations don't make much sense, so we return individual contexts.
func (p *ContextAttributeProvider) GetBatchAttributes(ctx context.Context, keys []string) (map[string]map[string]any, error) {
	result := make(map[string]map[string]any)
	attrs := contextToAttributes(&RequestContext{
		Timestamp: time.Now().UTC(),
	})

	for _, key := range keys {
		result[key] = attrs
	}

	return result, nil
}

// contextToAttributes converts RequestContext to attributes map.
func contextToAttributes(reqCtx *RequestContext) map[string]any {
	if reqCtx == nil {
		return make(map[string]any)
	}

	attrs := map[string]any{
		"ip":         reqCtx.IP,
		"user_agent": reqCtx.UserAgent,
		"method":     reqCtx.Method,
		"path":       reqCtx.Path,
		"timestamp":  reqCtx.Timestamp,
		"time":       reqCtx.Timestamp, // Alias for easier access
	}

	// Add time-based attributes
	now := reqCtx.Timestamp
	if now.IsZero() {
		now = time.Now().UTC()
	}

	attrs["hour"] = now.Hour()
	attrs["day_of_week"] = int(now.Weekday())
	attrs["day"] = now.Day()
	attrs["month"] = int(now.Month())
	attrs["year"] = now.Year()
	attrs["is_weekday"] = now.Weekday() >= time.Monday && now.Weekday() <= time.Friday
	attrs["is_weekend"] = now.Weekday() == time.Saturday || now.Weekday() == time.Sunday

	// Add geolocation if present
	if reqCtx.Geolocation != nil {
		attrs["country"] = reqCtx.Geolocation.Country
		attrs["region"] = reqCtx.Geolocation.Region
		attrs["city"] = reqCtx.Geolocation.City
		attrs["latitude"] = reqCtx.Geolocation.Latitude
		attrs["longitude"] = reqCtx.Geolocation.Longitude
		attrs["timezone"] = reqCtx.Geolocation.Timezone
	}

	// Add device info if present
	if reqCtx.Device != nil {
		attrs["device_type"] = reqCtx.Device.Type
		attrs["device_os"] = reqCtx.Device.OS
		attrs["device_browser"] = reqCtx.Device.Browser
		attrs["is_mobile"] = reqCtx.Device.IsMobile
		attrs["is_desktop"] = reqCtx.Device.IsDesktop
	}

	// Add headers
	if reqCtx.Headers != nil {
		for k, v := range reqCtx.Headers {
			attrs["header_"+k] = v
		}
	}

	// Merge metadata
	if reqCtx.Metadata != nil {
		for k, v := range reqCtx.Metadata {
			attrs["meta_"+k] = v
		}
	}

	return attrs
}

// Helper functions for request context

// IPInRange checks if an IP address is in any of the given CIDR ranges.
func IPInRange(ip string, cidrs []string) bool {
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return false
	}

	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}

		if network.Contains(ipAddr) {
			return true
		}
	}

	return false
}

// InTimeRange checks if current time is between start and end times (24-hour format)
// start and end are in format "HH:MM" (e.g., "09:00", "17:00").
func InTimeRange(now time.Time, start, end string) bool {
	// Parse start time
	startHour, startMin, err := parseTime(start)
	if err != nil {
		return false
	}

	// Parse end time
	endHour, endMin, err := parseTime(end)
	if err != nil {
		return false
	}

	// Get current time components
	nowHour := now.Hour()
	nowMin := now.Minute()

	// Convert to minutes since midnight for easier comparison
	nowMinutes := nowHour*60 + nowMin
	startMinutes := startHour*60 + startMin
	endMinutes := endHour*60 + endMin

	// Handle overnight ranges (e.g., "22:00" to "06:00")
	if endMinutes < startMinutes {
		return nowMinutes >= startMinutes || nowMinutes <= endMinutes
	}

	return nowMinutes >= startMinutes && nowMinutes <= endMinutes
}

// parseTime parses a time string in format "HH:MM".
func parseTime(timeStr string) (hour, min int, err error) {
	var h, m int

	_, err = fmt.Sscanf(timeStr, "%d:%d", &h, &m)
	if err != nil {
		return 0, 0, err
	}

	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, fmt.Errorf("invalid time: %s", timeStr)
	}

	return h, m, nil
}

// IsWeekday returns true if the given time is a weekday.
func IsWeekday(t time.Time) bool {
	return t.Weekday() >= time.Monday && t.Weekday() <= time.Friday
}

// DaysSince returns the number of days since the given time.
func DaysSince(t time.Time) int {
	duration := time.Since(t)

	return int(duration.Hours() / 24)
}

// HoursSince returns the number of hours since the given time.
func HoursSince(t time.Time) int {
	duration := time.Since(t)

	return int(duration.Hours())
}
