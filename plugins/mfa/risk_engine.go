package mfa

import (
	"context"
	"fmt"
	"math"
	"net"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/repository"
)

// RiskEngine assesses authentication risk and recommends factors.
type RiskEngine struct {
	config *AdaptiveMFAConfig
	repo   *repository.MFARepository
}

// NewRiskEngine creates a new risk assessment engine.
func NewRiskEngine(config *AdaptiveMFAConfig, repo *repository.MFARepository) *RiskEngine {
	return &RiskEngine{
		config: config,
		repo:   repo,
	}
}

// RiskFactor represents an identified risk factor.
type RiskFactor struct {
	Name        string
	Description string
	Score       float64 // 0-100
	Weight      float64 // 0-1
}

// RiskContext contains contextual information for risk assessment.
type RiskContext struct {
	UserID    xid.ID
	IPAddress string
	UserAgent string
	Location  string
	DeviceID  string
	Timestamp time.Time
}

// AssessRisk performs a comprehensive risk assessment.
func (e *RiskEngine) AssessRisk(ctx context.Context, riskCtx *RiskContext) (*RiskAssessment, error) {
	if !e.config.Enabled {
		// Return low risk if adaptive MFA is disabled
		return &RiskAssessment{
			Level:       RiskLevelLow,
			Score:       0,
			Factors:     []string{},
			Recommended: []FactorType{FactorTypeTOTP},
			Metadata:    make(map[string]any),
		}, nil
	}

	var riskFactors []RiskFactor

	// Assess location change risk
	if e.config.FactorLocationChange {
		factor, err := e.assessLocationChange(ctx, riskCtx)
		if err == nil && factor != nil {
			riskFactors = append(riskFactors, *factor)
		}
	}

	// Assess new device risk
	if e.config.FactorNewDevice {
		factor, err := e.assessNewDevice(ctx, riskCtx)
		if err == nil && factor != nil {
			riskFactors = append(riskFactors, *factor)
		}
	}

	// Assess velocity risk (rapid authentication attempts)
	if e.config.FactorVelocity {
		factor, err := e.assessVelocity(ctx, riskCtx)
		if err == nil && factor != nil {
			riskFactors = append(riskFactors, *factor)
		}
	}

	// Assess IP reputation (if enabled)
	if e.config.FactorIPReputation {
		factor := e.assessIPReputation(riskCtx)
		if factor != nil {
			riskFactors = append(riskFactors, *factor)
		}
	}

	// Calculate overall risk score
	totalScore := e.calculateRiskScore(riskFactors)
	riskLevel := e.determineRiskLevel(totalScore)
	recommended := e.getRecommendedFactors(riskLevel, totalScore)

	// Extract factor names for response
	factorNames := make([]string, len(riskFactors))
	for i, rf := range riskFactors {
		factorNames[i] = rf.Name
	}

	return &RiskAssessment{
		Level:       riskLevel,
		Score:       totalScore,
		Factors:     factorNames,
		Recommended: recommended,
		Metadata: map[string]any{
			"factors_detail": riskFactors,
			"timestamp":      riskCtx.Timestamp,
		},
	}, nil
}

// assessLocationChange checks if the user is logging in from a new location.
func (e *RiskEngine) assessLocationChange(ctx context.Context, riskCtx *RiskContext) (*RiskFactor, error) {
	// Get the most recent successful login
	recent, err := e.repo.GetRecentAttempts(ctx, riskCtx.UserID, time.Now().Add(-30*24*time.Hour))
	if err != nil || len(recent) == 0 {
		// No recent attempts, low risk
		return nil, nil
	}

	// Check if location has changed
	// In production, this would use IP geolocation
	lastLocation := extractLocationFromIP(recent[0].IPAddress)
	currentLocation := extractLocationFromIP(riskCtx.IPAddress)

	if lastLocation != currentLocation && lastLocation != "" {
		return &RiskFactor{
			Name:        "location_change",
			Description: fmt.Sprintf("Login from new location: %s (was: %s)", currentLocation, lastLocation),
			Score:       e.config.LocationChangeRisk,
			Weight:      1.0,
		}, nil
	}

	return nil, nil
}

// assessNewDevice checks if the user is using a new device.
func (e *RiskEngine) assessNewDevice(ctx context.Context, riskCtx *RiskContext) (*RiskFactor, error) {
	if riskCtx.DeviceID == "" {
		// No device ID provided, can't assess
		return nil, nil
	}

	// Check if device is trusted
	device, err := e.repo.GetTrustedDevice(ctx, riskCtx.UserID, riskCtx.DeviceID)
	if err != nil {
		return nil, err
	}

	if device == nil {
		// New/untrusted device
		return &RiskFactor{
			Name:        "new_device",
			Description: "Login from new or untrusted device",
			Score:       e.config.NewDeviceRisk,
			Weight:      1.0,
		}, nil
	}

	return nil, nil
}

// assessVelocity checks for rapid authentication attempts.
func (e *RiskEngine) assessVelocity(ctx context.Context, riskCtx *RiskContext) (*RiskFactor, error) {
	// Look for attempts in the last hour
	since := time.Now().Add(-1 * time.Hour)

	attempts, err := e.repo.GetRecentAttempts(ctx, riskCtx.UserID, since)
	if err != nil {
		return nil, err
	}

	// High velocity if more than 5 attempts in an hour
	if len(attempts) > 5 {
		velocityScore := math.Min(e.config.VelocityRisk*float64(len(attempts))/5.0, 100.0)

		return &RiskFactor{
			Name:        "high_velocity",
			Description: fmt.Sprintf("High authentication velocity: %d attempts in last hour", len(attempts)),
			Score:       velocityScore,
			Weight:      1.0,
		}, nil
	}

	return nil, nil
}

// assessIPReputation checks IP reputation (stub for now).
func (e *RiskEngine) assessIPReputation(riskCtx *RiskContext) *RiskFactor {
	// In production, this would check against IP reputation databases
	// For now, check for obvious bad patterns
	ip := net.ParseIP(riskCtx.IPAddress)
	if ip == nil {
		return &RiskFactor{
			Name:        "invalid_ip",
			Description: "Invalid IP address",
			Score:       20.0,
			Weight:      0.5,
		}
	}

	// Check for localhost/private IPs (might be suspicious in production)
	if ip.IsLoopback() || ip.IsPrivate() {
		return &RiskFactor{
			Name:        "private_ip",
			Description: "Login from private/local IP address",
			Score:       10.0,
			Weight:      0.3,
		}
	}

	// In production, integrate with services like:
	// - AbuseIPDB
	// - IPQualityScore
	// - Cloudflare threat intelligence
	// - MaxMind GeoIP2

	return nil
}

// calculateRiskScore computes the overall risk score from individual factors.
func (e *RiskEngine) calculateRiskScore(factors []RiskFactor) float64 {
	if len(factors) == 0 {
		return 0.0
	}

	// Weighted average of risk factors
	var (
		totalWeightedScore float64
		totalWeight        float64
	)

	for _, factor := range factors {
		totalWeightedScore += factor.Score * factor.Weight
		totalWeight += factor.Weight
	}

	if totalWeight == 0 {
		return 0.0
	}

	return math.Min(totalWeightedScore/totalWeight, 100.0)
}

// determineRiskLevel converts a score to a risk level.
func (e *RiskEngine) determineRiskLevel(score float64) RiskLevel {
	if score >= 75.0 {
		return RiskLevelCritical
	} else if score >= 50.0 {
		return RiskLevelHigh
	} else if score >= 25.0 {
		return RiskLevelMedium
	}

	return RiskLevelLow
}

// getRecommendedFactors recommends authentication factors based on risk.
func (e *RiskEngine) getRecommendedFactors(level RiskLevel, score float64) []FactorType {
	// Step-up authentication required for high risk
	if score >= e.config.RequireStepUpThreshold {
		// High risk: require multiple strong factors
		return []FactorType{
			FactorTypeTOTP,
			FactorTypeWebAuthn,
			FactorTypeSMS,
		}
	}

	switch level {
	case RiskLevelCritical, RiskLevelHigh:
		// High risk: require at least 2 factors
		return []FactorType{
			FactorTypeTOTP,
			FactorTypeSMS,
			FactorTypeEmail,
		}
	case RiskLevelMedium:
		// Medium risk: require 1 strong factor
		return []FactorType{
			FactorTypeTOTP,
			FactorTypeWebAuthn,
		}
	default:
		// Low risk: any factor acceptable
		return []FactorType{
			FactorTypeTOTP,
			FactorTypeSMS,
			FactorTypeEmail,
			FactorTypeBackup,
		}
	}
}

// RequiresStepUp determines if step-up authentication is needed.
func (e *RiskEngine) RequiresStepUp(score float64) bool {
	return score >= e.config.RequireStepUpThreshold
}

// GetRequiredFactorCount returns the number of factors required based on risk.
func (e *RiskEngine) GetRequiredFactorCount(level RiskLevel) int {
	switch level {
	case RiskLevelCritical:
		return 3
	case RiskLevelHigh:
		return 2
	case RiskLevelMedium:
		return 1
	default:
		return 1
	}
}

// extractLocationFromIP extracts location from IP (simplified)
// In production, use a proper geolocation service.
func extractLocationFromIP(ip string) string {
	if ip == "" {
		return ""
	}

	// Parse IP
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "unknown"
	}

	// Check for special cases
	if parsedIP.IsLoopback() {
		return "localhost"
	}

	if parsedIP.IsPrivate() {
		return "private"
	}

	// In production, use geolocation service
	// For now, return country code from IP (stub)
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		// Simplified: use first octet to determine region (NOT accurate!)
		// This is just a placeholder for demonstration
		return "region_" + parts[0]
	}

	return "unknown"
}

// CalculateDeviceFingerprint generates a device fingerprint from user agent and other data.
func CalculateDeviceFingerprint(userAgent, ipAddress string, additionalData map[string]string) string {
	// In production, use a proper device fingerprinting library
	// For now, create a simple hash-based fingerprint
	fingerprint := userAgent
	if ipAddress != "" {
		fingerprint += "|" + ipAddress
	}

	// Add additional data if provided
	var fingerprintSb360 strings.Builder
	for key, value := range additionalData {
		fingerprintSb360.WriteString(fmt.Sprintf("|%s:%s", key, value))
	}
	fingerprint += fingerprintSb360.String()

	// In production, hash this and store the hash
	// For now, return a simplified version
	return fmt.Sprintf("fp_%x", []byte(fingerprint)[:16])
}
