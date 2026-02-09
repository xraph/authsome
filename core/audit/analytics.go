package audit

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
)

// =============================================================================
// ADVANCED ANALYTICS - Anomaly detection, risk scoring, behavioral baselines
// =============================================================================

// AnalyticsService provides advanced security analytics.
type AnalyticsService struct {
	repo            Repository
	baselineCache   *BaselineCache
	anomalyDetector *AnomalyDetector
	riskEngine      *RiskEngine
	mu              sync.RWMutex
}

// NewAnalyticsService creates a new analytics service.
func NewAnalyticsService(repo Repository) *AnalyticsService {
	return &AnalyticsService{
		repo:            repo,
		baselineCache:   NewBaselineCache(),
		anomalyDetector: NewAnomalyDetector(),
		riskEngine:      NewRiskEngine(),
	}
}

// =============================================================================
// BASELINE CALCULATION - Statistical baselines for normal behavior
// =============================================================================

// Baseline represents statistical baseline for user behavior.
type Baseline struct {
	UserID           xid.ID                 `json:"userId"`
	OrganizationID   *xid.ID                `json:"organizationId,omitempty"` // Optional org scope
	Period           time.Duration          `json:"period"`
	EventsPerHour    float64                `json:"eventsPerHour"`
	TopActions       map[string]int         `json:"topActions"`
	TopResources     map[string]int         `json:"topResources"`
	TopLocations     []string               `json:"topLocations"`
	TypicalHours     []int                  `json:"typicalHours"` // Hours of day (0-23)
	TypicalDays      []time.Weekday         `json:"typicalDays"`  // Days of week
	UniqueIPCount    int                    `json:"uniqueIpCount"`
	AvgSessionLength time.Duration          `json:"avgSessionLength"`
	CalculatedAt     time.Time              `json:"calculatedAt"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// BaselineCalculator calculates behavioral baselines.
type BaselineCalculator struct {
	repo Repository
}

// NewBaselineCalculator creates a new baseline calculator.
func NewBaselineCalculator(repo Repository) *BaselineCalculator {
	return &BaselineCalculator{repo: repo}
}

// Calculate calculates baseline for a user over a period.
func (bc *BaselineCalculator) Calculate(ctx context.Context, userID xid.ID, period time.Duration) (*Baseline, error) {
	return bc.CalculateWithOptions(ctx, userID, period, nil)
}

// CalculateWithOptions calculates baseline for a user over a period with optional organization scope.
func (bc *BaselineCalculator) CalculateWithOptions(ctx context.Context, userID xid.ID, period time.Duration, organizationID *xid.ID) (*Baseline, error) {
	// Calculate time range
	until := time.Now()
	since := until.Add(-period)

	// Fetch events for the period
	filter := &ListEventsFilter{
		UserID:         &userID,
		OrganizationID: organizationID, // Optional org scope
		Since:          &since,
		Until:          &until,
	}

	resp, err := bc.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events for baseline: %w", err)
	}

	events := FromSchemaEvents(resp.Data)
	if len(events) == 0 {
		return nil, errs.New(errs.CodeInvalidInput, "insufficient data for baseline calculation", http.StatusBadRequest)
	}

	baseline := &Baseline{
		UserID:         userID,
		OrganizationID: organizationID,
		Period:         period,
		TopActions:     make(map[string]int),
		TopResources:   make(map[string]int),
		CalculatedAt:   time.Now(),
		Metadata:       make(map[string]interface{}),
	}

	// Calculate statistics
	baseline.calculateEventRate(events, period)
	baseline.calculateTopActions(events)
	baseline.calculateTopResources(events)
	baseline.calculateTemporalPatterns(events)
	baseline.calculateIPPatterns(events)

	return baseline, nil
}

func (b *Baseline) calculateEventRate(events []*Event, period time.Duration) {
	hours := period.Hours()
	if hours == 0 {
		hours = 1
	}

	b.EventsPerHour = float64(len(events)) / hours
}

func (b *Baseline) calculateTopActions(events []*Event) {
	actionCounts := make(map[string]int)
	for _, event := range events {
		actionCounts[event.Action]++
	}

	b.TopActions = actionCounts
}

func (b *Baseline) calculateTopResources(events []*Event) {
	resourceCounts := make(map[string]int)
	for _, event := range events {
		resourceCounts[event.Resource]++
	}

	b.TopResources = resourceCounts
}

func (b *Baseline) calculateTemporalPatterns(events []*Event) {
	hourCounts := make(map[int]int)
	dayCounts := make(map[time.Weekday]int)

	for _, event := range events {
		hourCounts[event.CreatedAt.Hour()]++
		dayCounts[event.CreatedAt.Weekday()]++
	}

	// Get top hours (hours with > 5% of total events)
	threshold := len(events) / 20
	for hour, count := range hourCounts {
		if count > threshold {
			b.TypicalHours = append(b.TypicalHours, hour)
		}
	}

	sort.Ints(b.TypicalHours)

	// Get top days
	for day, count := range dayCounts {
		if count > threshold {
			b.TypicalDays = append(b.TypicalDays, day)
		}
	}
}

func (b *Baseline) calculateIPPatterns(events []*Event) {
	uniqueIPs := make(map[string]bool)

	for _, event := range events {
		if event.IPAddress != "" {
			uniqueIPs[event.IPAddress] = true

			// Extract location (would use IP geolocation service in production)
			// For now, just count unique IPs
		}
	}

	b.UniqueIPCount = len(uniqueIPs)
	b.TopLocations = []string{} // Populated by geolocation service
}

// =============================================================================
// BASELINE CACHE - In-memory cache for baselines
// =============================================================================

// BaselineCache caches user baselines in memory.
type BaselineCache struct {
	baselines map[xid.ID]*Baseline
	mu        sync.RWMutex
	ttl       time.Duration
}

// NewBaselineCache creates a new baseline cache.
func NewBaselineCache() *BaselineCache {
	return &BaselineCache{
		baselines: make(map[xid.ID]*Baseline),
		ttl:       24 * time.Hour, // Recalculate daily
	}
}

// Get retrieves a baseline from cache.
func (bc *BaselineCache) Get(userID xid.ID) (*Baseline, bool) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	baseline, exists := bc.baselines[userID]
	if !exists {
		return nil, false
	}

	// Check if baseline is stale
	if time.Since(baseline.CalculatedAt) > bc.ttl {
		return nil, false
	}

	return baseline, true
}

// Set stores a baseline in cache.
func (bc *BaselineCache) Set(userID xid.ID, baseline *Baseline) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.baselines[userID] = baseline
}

// =============================================================================
// ANOMALY DETECTION - Detects abnormal behavior
// =============================================================================

// Anomaly represents a detected anomaly.
type Anomaly struct {
	Type        string                 `json:"type"`     // geo_velocity, unusual_action, frequency_spike, etc.
	Severity    string                 `json:"severity"` // low, medium, high, critical
	Score       float64                `json:"score"`    // 0-100
	Event       *Event                 `json:"event"`
	Baseline    *Baseline              `json:"baseline,omitempty"`
	Description string                 `json:"description"`
	Evidence    map[string]interface{} `json:"evidence"`
	DetectedAt  time.Time              `json:"detectedAt"`
}

// AnomalyDetector detects anomalies in audit events.
type AnomalyDetector struct {
	baselineCalc *BaselineCalculator
}

// NewAnomalyDetector creates a new anomaly detector.
func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{}
}

// SetBaselineCalculator sets the baseline calculator.
func (ad *AnomalyDetector) SetBaselineCalculator(calc *BaselineCalculator) {
	ad.baselineCalc = calc
}

// DetectAnomalies detects anomalies in an event against baseline.
func (ad *AnomalyDetector) DetectAnomalies(ctx context.Context, event *Event, baseline *Baseline) ([]*Anomaly, error) {
	anomalies := make([]*Anomaly, 0)

	// Check for unusual actions
	if anomaly := ad.detectUnusualAction(event, baseline); anomaly != nil {
		anomalies = append(anomalies, anomaly)
	}

	// Check for temporal anomalies
	if anomaly := ad.detectTemporalAnomaly(event, baseline); anomaly != nil {
		anomalies = append(anomalies, anomaly)
	}

	// Check for frequency spikes
	if anomaly := ad.detectFrequencySpike(event, baseline); anomaly != nil {
		anomalies = append(anomalies, anomaly)
	}

	return anomalies, nil
}

// detectUnusualAction checks if action is outside normal behavior.
func (ad *AnomalyDetector) detectUnusualAction(event *Event, baseline *Baseline) *Anomaly {
	if baseline == nil || baseline.TopActions == nil {
		return nil
	}

	// Check if action exists in baseline
	count, exists := baseline.TopActions[event.Action]
	if !exists || count < 5 { // Action never seen or rarely seen
		score := 70.0 // High score for never-seen action
		if exists {
			// Reduce score based on frequency
			score = math.Max(40.0, 70.0-float64(count)*5)
		}

		return &Anomaly{
			Type:        "unusual_action",
			Severity:    ad.calculateSeverity(score),
			Score:       score,
			Event:       event,
			Baseline:    baseline,
			Description: fmt.Sprintf("Action '%s' is unusual for this user", event.Action),
			Evidence: map[string]interface{}{
				"action":        event.Action,
				"normalActions": baseline.TopActions,
			},
			DetectedAt: time.Now(),
		}
	}

	return nil
}

// detectTemporalAnomaly checks if event time is outside normal hours.
func (ad *AnomalyDetector) detectTemporalAnomaly(event *Event, baseline *Baseline) *Anomaly {
	if baseline == nil || len(baseline.TypicalHours) == 0 {
		return nil
	}

	hour := event.CreatedAt.Hour()
	day := event.CreatedAt.Weekday()

	// Check if hour is typical
	hourTypical := false

	for _, typicalHour := range baseline.TypicalHours {
		if hour == typicalHour {
			hourTypical = true

			break
		}
	}

	// Check if day is typical
	dayTypical := false

	for _, typicalDay := range baseline.TypicalDays {
		if day == typicalDay {
			dayTypical = true

			break
		}
	}

	if !hourTypical || !dayTypical {
		score := 50.0
		if !hourTypical && !dayTypical {
			score = 65.0 // Both hour and day unusual
		}

		return &Anomaly{
			Type:        "temporal_anomaly",
			Severity:    ad.calculateSeverity(score),
			Score:       score,
			Event:       event,
			Baseline:    baseline,
			Description: "Activity at unusual time: " + event.CreatedAt.Format(time.RFC3339),
			Evidence: map[string]interface{}{
				"hour":         hour,
				"day":          day.String(),
				"typicalHours": baseline.TypicalHours,
				"typicalDays":  baseline.TypicalDays,
			},
			DetectedAt: time.Now(),
		}
	}

	return nil
}

// detectFrequencySpike checks for sudden increase in activity.
func (ad *AnomalyDetector) detectFrequencySpike(event *Event, baseline *Baseline) *Anomaly {
	// This would require recent event count - simplified version
	// In production, would compare last hour's event count to baseline.EventsPerHour
	return nil
}

// calculateSeverity calculates severity from score.
func (ad *AnomalyDetector) calculateSeverity(score float64) string {
	switch {
	case score >= 80:
		return "critical"
	case score >= 60:
		return "high"
	case score >= 40:
		return "medium"
	default:
		return "low"
	}
}

// =============================================================================
// GEO-VELOCITY DETECTION - Detects impossible travel
// =============================================================================

// GeoVelocityDetector detects impossible travel based on IP geolocation.
type GeoVelocityDetector struct {
	// Would integrate with IP geolocation service (MaxMind, IP2Location, etc.)
}

// DetectImpossibleTravel checks if travel between two locations is physically impossible.
func (gvd *GeoVelocityDetector) DetectImpossibleTravel(event1, event2 *Event) (*Anomaly, error) {
	// Calculate distance between IPs (requires geolocation service)
	// Calculate time difference
	// Calculate required velocity
	// If velocity > threshold (e.g., 1000 km/h), flag as impossible travel

	// Placeholder implementation
	timeDiff := event2.CreatedAt.Sub(event1.CreatedAt)
	if timeDiff < 1*time.Hour {
		// Check if IPs are from different countries/continents
		// This would use a geolocation database
		if event1.IPAddress != event2.IPAddress {
			return &Anomaly{
				Type:        "geo_velocity",
				Severity:    "critical",
				Score:       95.0,
				Event:       event2,
				Description: "Impossible travel detected",
				Evidence: map[string]interface{}{
					"ip1":      event1.IPAddress,
					"ip2":      event2.IPAddress,
					"timeDiff": timeDiff.String(),
				},
				DetectedAt: time.Now(),
			}, nil
		}
	}

	return nil, nil
}

// =============================================================================
// RISK ENGINE - Multi-factor risk scoring
// =============================================================================

// RiskEngine calculates risk scores for events.
type RiskEngine struct {
	weights map[string]float64
}

// NewRiskEngine creates a new risk engine.
func NewRiskEngine() *RiskEngine {
	return &RiskEngine{
		weights: map[string]float64{
			"anomaly_score": 0.4,
			"user_risk":     0.2,
			"resource_risk": 0.2,
			"context_risk":  0.2,
		},
	}
}

// RiskScore represents a calculated risk score.
type RiskScore struct {
	Score        float64            `json:"score"` // 0-100
	Level        string             `json:"level"` // low, medium, high, critical
	Factors      map[string]float64 `json:"factors"`
	Anomalies    []*Anomaly         `json:"anomalies,omitempty"`
	Event        *Event             `json:"event"`
	CalculatedAt time.Time          `json:"calculatedAt"`
}

// Calculate calculates risk score for an event.
func (re *RiskEngine) Calculate(ctx context.Context, event *Event, anomalies []*Anomaly, baseline *Baseline) (*RiskScore, error) {
	factors := make(map[string]float64)

	// Factor 1: Anomaly score (average of all detected anomalies)
	anomalyScore := 0.0

	if len(anomalies) > 0 {
		for _, anomaly := range anomalies {
			anomalyScore += anomaly.Score
		}

		anomalyScore /= float64(len(anomalies))
	}

	factors["anomaly_score"] = anomalyScore

	// Factor 2: User risk (based on historical violations, privileges)
	userRisk := re.calculateUserRisk(event)
	factors["user_risk"] = userRisk

	// Factor 3: Resource risk (sensitivity of accessed resource)
	resourceRisk := re.calculateResourceRisk(event)
	factors["resource_risk"] = resourceRisk

	// Factor 4: Context risk (IP reputation, time of day, etc.)
	contextRisk := re.calculateContextRisk(event, baseline)
	factors["context_risk"] = contextRisk

	// Calculate weighted total
	totalScore := 0.0

	for factor, score := range factors {
		weight := re.weights[factor]
		totalScore += score * weight
	}

	return &RiskScore{
		Score:        totalScore,
		Level:        re.calculateRiskLevel(totalScore),
		Factors:      factors,
		Anomalies:    anomalies,
		Event:        event,
		CalculatedAt: time.Now(),
	}, nil
}

func (re *RiskEngine) calculateUserRisk(event *Event) float64 {
	// Would check user's historical violations, privilege level, MFA status
	// Placeholder: return moderate risk for all users
	return 30.0
}

func (re *RiskEngine) calculateResourceRisk(event *Event) float64 {
	// Would check resource sensitivity classification
	// High-risk resources: user deletion, role assignment, config changes
	highRiskActions := map[string]bool{
		"user.delete":      true,
		"role.assign":      true,
		"config.update":    true,
		"permission.grant": true,
	}

	if highRiskActions[event.Action] {
		return 80.0
	}

	return 20.0
}

func (re *RiskEngine) calculateContextRisk(event *Event, baseline *Baseline) float64 {
	risk := 0.0

	// Check IP reputation (would use threat intelligence feed)
	// Placeholder: check if IP is in known malicious list

	// Check time of day against baseline
	if baseline != nil && len(baseline.TypicalHours) > 0 {
		hour := event.CreatedAt.Hour()
		isTypical := false

		for _, typicalHour := range baseline.TypicalHours {
			if hour == typicalHour {
				isTypical = true

				break
			}
		}

		if !isTypical {
			risk += 30.0
		}
	}

	return risk
}

func (re *RiskEngine) calculateRiskLevel(score float64) string {
	switch {
	case score >= 75:
		return "critical"
	case score >= 50:
		return "high"
	case score >= 25:
		return "medium"
	default:
		return "low"
	}
}
