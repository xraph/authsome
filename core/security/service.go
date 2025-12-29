package security

import (
	"context"
	"net"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
)

// Config for security checks
type Config struct {
	Enabled          bool     `json:"enabled"`
	IPWhitelist      []string `json:"ipWhitelist"`
	IPBlacklist      []string `json:"ipBlacklist"`
	AllowedCountries []string `json:"allowedCountries"`
	BlockedCountries []string `json:"blockedCountries"`
	// TrustProxyHeaders enables honoring X-Forwarded-For/X-Real-IP/Forwarded
	TrustProxyHeaders bool `json:"trustProxyHeaders"`
	// TrustedProxies restricts which proxy IPs are trusted for headers (exact or CIDR).
	// If empty and TrustProxyHeaders=true, all proxies are trusted.
	TrustedProxies []string `json:"trustedProxies"`
}

// Service handles security checks and event logging
type Service struct {
	repo   Repository
	config Config
	// in-memory counters for lockout; can be moved to persistent storage later
	failedCounts map[string]int
	lockoutUntil map[string]time.Time
	// optional GeoIP provider for country lookups
	geo GeoIPProvider
}

func NewService(repo Repository, cfg Config) *Service {
	return &Service{
		repo:         repo,
		config:       cfg,
		failedCounts: make(map[string]int),
		lockoutUntil: make(map[string]time.Time),
	}
}

// CheckIPAllowed verifies IP against whitelist/blacklist
func (s *Service) CheckIPAllowed(_ context.Context, ip string) bool {
	if !s.config.Enabled {
		return true
	}
	// Whitelist overrides
	if len(s.config.IPWhitelist) > 0 {
		for _, w := range s.config.IPWhitelist {
			if ipMatches(w, ip) {
				return true
			}
		}
		return false
	}
	for _, b := range s.config.IPBlacklist {
		if ipMatches(b, ip) {
			return false
		}
	}
	return true
}

// LogEvent logs a security event
func (s *Service) LogEvent(ctx context.Context, typ string, userID *xid.ID, ip, ua, geo string) error {
	// Extract AppID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		// Graceful degradation: skip event if no app context
		return nil
	}

	e := &SecurityEvent{
		ID:        xid.New(),
		AppID:     appID,
		UserID:    userID,
		Type:      typ,
		IPAddress: ip,
		UserAgent: ua,
		Geo:       geo,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	return s.repo.Create(ctx, e)
}

// ShouldTrustForwardedHeaders returns true if forwarded headers should be honored
// based on TrustProxyHeaders and TrustedProxies matching the remote IP.
func (s *Service) ShouldTrustForwardedHeaders(remoteIP string) bool {
	if !s.config.TrustProxyHeaders {
		return false
	}
	if len(s.config.TrustedProxies) == 0 {
		return true
	}
	for _, e := range s.config.TrustedProxies {
		if ipMatches(e, remoteIP) {
			return true
		}
	}
	return false
}

// CheckCountryAllowed enforces geo-based restrictions using AllowedCountries/BlockedCountries.
// If a GeoIP provider is not set and lists are configured, enforcement is skipped (allowed).
func (s *Service) CheckCountryAllowed(ctx context.Context, ip string) bool {
	if !s.config.Enabled {
		return true
	}
	if len(s.config.AllowedCountries) == 0 && len(s.config.BlockedCountries) == 0 {
		return true
	}
	if s.geo == nil {
		return true
	}
	country, err := s.geo.CountryForIP(ctx, ip)
	if err != nil || country == "" {
		return true
	}
	if len(s.config.AllowedCountries) > 0 {
		for _, c := range s.config.AllowedCountries {
			if equalCountry(c, country) {
				return true
			}
		}
		return false
	}
	for _, c := range s.config.BlockedCountries {
		if equalCountry(c, country) {
			return false
		}
	}
	return true
}

// SetGeoIPProvider sets the GeoIP provider used for country lookups
func (s *Service) SetGeoIPProvider(p GeoIPProvider) { s.geo = p }

// IsLockedOut returns true if key (email or IP) is under lockout
func (s *Service) IsLockedOut(_ context.Context, key string) bool {
	until, ok := s.lockoutUntil[key]
	return ok && time.Now().Before(until)
}

// GetLockoutTime returns the lockout expiration time for a key if locked out
// Returns zero time if not locked out
func (s *Service) GetLockoutTime(_ context.Context, key string) time.Time {
	until, ok := s.lockoutUntil[key]
	if !ok {
		return time.Time{}
	}
	if time.Now().Before(until) {
		return until
	}
	return time.Time{}
}

// RecordFailedAttempt increments failed attempt count and applies lockout if threshold reached
func (s *Service) RecordFailedAttempt(_ context.Context, key string) {
	// default thresholds
	const defaultMax = 5
	const defaultWindow = time.Minute * 15
	const defaultLockout = time.Minute * 15
	// increment
	s.failedCounts[key]++
	max := defaultMax
	// if reached threshold, set lockout
	if s.failedCounts[key] >= max {
		s.lockoutUntil[key] = time.Now().Add(defaultLockout)
		// reset counter after window
		go func(k string) {
			time.Sleep(defaultWindow)
			s.failedCounts[k] = 0
		}(key)
	}
}

// GetFailedAttemptCount returns the current number of failed attempts for a key
func (s *Service) GetFailedAttemptCount(_ context.Context, key string) int {
	count, ok := s.failedCounts[key]
	if !ok {
		return 0
	}
	return count
}

// GetAttemptsRemaining returns the number of attempts remaining before lockout
func (s *Service) GetAttemptsRemaining(_ context.Context, key string) int {
	const defaultMax = 5
	count := s.GetFailedAttemptCount(context.Background(), key)
	remaining := defaultMax - count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// ResetFailedAttempts clears counters and lockout for a key
func (s *Service) ResetFailedAttempts(_ context.Context, key string) {
	delete(s.failedCounts, key)
	delete(s.lockoutUntil, key)
}

// ipMatches checks if ip matches entry which may be exact or CIDR
func ipMatches(entry, ip string) bool {
	if entry == ip {
		return true
	}
	// Try CIDR
	if _, cidr, err := net.ParseCIDR(entry); err == nil && cidr != nil {
		parsed := net.ParseIP(ip)
		if parsed != nil {
			return cidr.Contains(parsed)
		}
	}
	return false
}
