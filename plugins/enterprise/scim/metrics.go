package scim

import (
	"expvar"
	"fmt"
	"sync"
	"time"
)

// Metrics collects SCIM plugin metrics using Go's built-in expvar
// These metrics are automatically exposed via the /debug/vars endpoint
type Metrics struct {
	// Operations counter by type and status
	operations *expvar.Map // scim_operations_total{operation,status,org_id}
	
	// Request duration histogram
	requestDuration *expvar.Map // scim_request_duration_seconds{endpoint,quantile}
	
	// Rate limit hits
	rateLimitHits *expvar.Int // scim_rate_limit_hits_total
	
	// Token operations
	tokenValidations *expvar.Int // scim_token_validations_total
	tokenFailures    *expvar.Int // scim_token_validation_failures_total
	tokenCreations   *expvar.Int // scim_token_creations_total
	tokenRevocations *expvar.Int // scim_token_revocations_total
	
	// Provisioning operations
	userCreations *expvar.Int // scim_user_creations_total
	userUpdates   *expvar.Int // scim_user_updates_total
	userDeletions *expvar.Int // scim_user_deletions_total
	userReads     *expvar.Int // scim_user_reads_total
	
	groupCreations *expvar.Int // scim_group_creations_total
	groupUpdates   *expvar.Int // scim_group_updates_total
	groupDeletions *expvar.Int // scim_group_deletions_total
	
	bulkOperations *expvar.Int // scim_bulk_operations_total
	
	// Errors by type
	errors *expvar.Map // scim_errors_total{type}
	
	// Latency tracking (for percentile calculation)
	latencyMu sync.RWMutex
	latencies map[string][]float64 // endpoint -> duration in milliseconds
	
	// Active requests gauge
	activeRequests *expvar.Int // scim_active_requests
	
	// Webhook metrics
	webhooksSent    *expvar.Int // scim_webhooks_sent_total
	webhooksFailed  *expvar.Int // scim_webhooks_failed_total
	webhooksRetried *expvar.Int // scim_webhooks_retried_total
}

var (
	metrics     *Metrics
	metricsOnce sync.Once
)

// GetMetrics returns the singleton metrics instance
func GetMetrics() *Metrics {
	metricsOnce.Do(func() {
		metrics = &Metrics{
			operations:       expvar.NewMap("scim_operations_total"),
			requestDuration:  expvar.NewMap("scim_request_duration_seconds"),
			rateLimitHits:    expvar.NewInt("scim_rate_limit_hits_total"),
			tokenValidations: expvar.NewInt("scim_token_validations_total"),
			tokenFailures:    expvar.NewInt("scim_token_validation_failures_total"),
			tokenCreations:   expvar.NewInt("scim_token_creations_total"),
			tokenRevocations: expvar.NewInt("scim_token_revocations_total"),
			userCreations:    expvar.NewInt("scim_user_creations_total"),
			userUpdates:      expvar.NewInt("scim_user_updates_total"),
			userDeletions:    expvar.NewInt("scim_user_deletions_total"),
			userReads:        expvar.NewInt("scim_user_reads_total"),
			groupCreations:   expvar.NewInt("scim_group_creations_total"),
			groupUpdates:     expvar.NewInt("scim_group_updates_total"),
			groupDeletions:   expvar.NewInt("scim_group_deletions_total"),
			bulkOperations:   expvar.NewInt("scim_bulk_operations_total"),
			errors:           expvar.NewMap("scim_errors_total"),
			latencies:        make(map[string][]float64),
			activeRequests:   expvar.NewInt("scim_active_requests"),
			webhooksSent:     expvar.NewInt("scim_webhooks_sent_total"),
			webhooksFailed:   expvar.NewInt("scim_webhooks_failed_total"),
			webhooksRetried:  expvar.NewInt("scim_webhooks_retried_total"),
		}
		
		// Register custom functions for percentile calculations
		expvar.Publish("scim_request_duration_p50", expvar.Func(func() interface{} {
			return metrics.getPercentile(50)
		}))
		expvar.Publish("scim_request_duration_p95", expvar.Func(func() interface{} {
			return metrics.getPercentile(95)
		}))
		expvar.Publish("scim_request_duration_p99", expvar.Func(func() interface{} {
			return metrics.getPercentile(99)
		}))
	})
	return metrics
}

// RecordOperation records a SCIM operation
func (m *Metrics) RecordOperation(operation, status, orgID string) {
	key := fmt.Sprintf("%s.%s.%s", operation, status, orgID)
	m.operations.Add(key, 1)
}

// RecordRequestDuration records the duration of a SCIM request
func (m *Metrics) RecordRequestDuration(endpoint string, duration time.Duration) {
	durationMs := float64(duration.Milliseconds())
	
	m.latencyMu.Lock()
	defer m.latencyMu.Unlock()
	
	if m.latencies[endpoint] == nil {
		m.latencies[endpoint] = make([]float64, 0, 1000)
	}
	
	m.latencies[endpoint] = append(m.latencies[endpoint], durationMs)
	
	// Keep only last 1000 requests per endpoint
	if len(m.latencies[endpoint]) > 1000 {
		m.latencies[endpoint] = m.latencies[endpoint][1:]
	}
}

// RecordRateLimitHit records a rate limit hit
func (m *Metrics) RecordRateLimitHit() {
	m.rateLimitHits.Add(1)
}

// RecordTokenValidation records a token validation attempt
func (m *Metrics) RecordTokenValidation(success bool) {
	m.tokenValidations.Add(1)
	if !success {
		m.tokenFailures.Add(1)
	}
}

// RecordTokenCreation records a token creation
func (m *Metrics) RecordTokenCreation() {
	m.tokenCreations.Add(1)
}

// RecordTokenRevocation records a token revocation
func (m *Metrics) RecordTokenRevocation() {
	m.tokenRevocations.Add(1)
}

// RecordUserOperation records a user provisioning operation
func (m *Metrics) RecordUserOperation(operation string) {
	switch operation {
	case "create":
		m.userCreations.Add(1)
	case "update":
		m.userUpdates.Add(1)
	case "delete":
		m.userDeletions.Add(1)
	case "read":
		m.userReads.Add(1)
	}
	m.RecordOperation(fmt.Sprintf("user_%s", operation), "success", "all")
}

// RecordGroupOperation records a group operation
func (m *Metrics) RecordGroupOperation(operation string) {
	switch operation {
	case "create":
		m.groupCreations.Add(1)
	case "update":
		m.groupUpdates.Add(1)
	case "delete":
		m.groupDeletions.Add(1)
	}
	m.RecordOperation(fmt.Sprintf("group_%s", operation), "success", "all")
}

// RecordBulkOperation records a bulk operation
func (m *Metrics) RecordBulkOperation(operationCount int) {
	m.bulkOperations.Add(1)
	m.RecordOperation("bulk", "success", "all")
}

// RecordError records an error by type
func (m *Metrics) RecordError(errorType string) {
	m.errors.Add(errorType, 1)
}

// IncrementActiveRequests increments the active request counter
func (m *Metrics) IncrementActiveRequests() {
	m.activeRequests.Add(1)
}

// DecrementActiveRequests decrements the active request counter
func (m *Metrics) DecrementActiveRequests() {
	m.activeRequests.Add(-1)
}

// RecordWebhook records a webhook operation
func (m *Metrics) RecordWebhook(success bool, retried bool) {
	m.webhooksSent.Add(1)
	if !success {
		m.webhooksFailed.Add(1)
	}
	if retried {
		m.webhooksRetried.Add(1)
	}
}

// getPercentile calculates the percentile across all endpoints
func (m *Metrics) getPercentile(percentile int) map[string]float64 {
	m.latencyMu.RLock()
	defer m.latencyMu.RUnlock()
	
	result := make(map[string]float64)
	
	for endpoint, latencies := range m.latencies {
		if len(latencies) == 0 {
			continue
		}
		
		// Simple percentile calculation (sorted copy)
		sorted := make([]float64, len(latencies))
		copy(sorted, latencies)
		
		// Bubble sort (good enough for 1000 elements)
		for i := 0; i < len(sorted); i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[i] > sorted[j] {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}
		
		index := (len(sorted) * percentile) / 100
		if index >= len(sorted) {
			index = len(sorted) - 1
		}
		
		result[endpoint] = sorted[index]
	}
	
	return result
}

// GetStats returns current statistics
func (m *Metrics) GetStats() map[string]interface{} {
	m.latencyMu.RLock()
	defer m.latencyMu.RUnlock()
	
	totalLatencies := 0
	for _, latencies := range m.latencies {
		totalLatencies += len(latencies)
	}
	
	return map[string]interface{}{
		"token_validations":   m.tokenValidations.Value(),
		"token_failures":      m.tokenFailures.Value(),
		"token_creations":     m.tokenCreations.Value(),
		"token_revocations":   m.tokenRevocations.Value(),
		"user_creations":      m.userCreations.Value(),
		"user_updates":        m.userUpdates.Value(),
		"user_deletions":      m.userDeletions.Value(),
		"user_reads":          m.userReads.Value(),
		"group_creations":     m.groupCreations.Value(),
		"group_updates":       m.groupUpdates.Value(),
		"group_deletions":     m.groupDeletions.Value(),
		"bulk_operations":     m.bulkOperations.Value(),
		"rate_limit_hits":     m.rateLimitHits.Value(),
		"active_requests":     m.activeRequests.Value(),
		"webhooks_sent":       m.webhooksSent.Value(),
		"webhooks_failed":     m.webhooksFailed.Value(),
		"webhooks_retried":    m.webhooksRetried.Value(),
		"total_latency_samples": totalLatencies,
		"p50_latencies":       m.getPercentile(50),
		"p95_latencies":       m.getPercentile(95),
		"p99_latencies":       m.getPercentile(99),
	}
}

// Reset resets all metrics (useful for testing)
func (m *Metrics) Reset() {
	m.operations = expvar.NewMap("scim_operations_total")
	m.rateLimitHits.Set(0)
	m.tokenValidations.Set(0)
	m.tokenFailures.Set(0)
	m.tokenCreations.Set(0)
	m.tokenRevocations.Set(0)
	m.userCreations.Set(0)
	m.userUpdates.Set(0)
	m.userDeletions.Set(0)
	m.userReads.Set(0)
	m.groupCreations.Set(0)
	m.groupUpdates.Set(0)
	m.groupDeletions.Set(0)
	m.bulkOperations.Set(0)
	m.errors = expvar.NewMap("scim_errors_total")
	m.activeRequests.Set(0)
	m.webhooksSent.Set(0)
	m.webhooksFailed.Set(0)
	m.webhooksRetried.Set(0)
	
	m.latencyMu.Lock()
	m.latencies = make(map[string][]float64)
	m.latencyMu.Unlock()
}

