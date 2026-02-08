package exporters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/xraph/authsome/core/audit"
)

// =============================================================================
// SPLUNK HEC EXPORTER - Exports audit events to Splunk via HTTP Event Collector
// =============================================================================

// SplunkExporter exports audit events to Splunk HEC
type SplunkExporter struct {
	config *SplunkConfig
	client *http.Client
}

// SplunkConfig contains Splunk HEC configuration
type SplunkConfig struct {
	Endpoint   string        `json:"endpoint"`   // Splunk HEC endpoint (e.g., https://splunk:8088/services/collector)
	Token      string        `json:"token"`      // HEC token
	Index      string        `json:"index"`      // Splunk index
	Source     string        `json:"source"`     // Event source
	SourceType string        `json:"sourceType"` // Source type
	VerifySSL  bool          `json:"verifySSL"`  // Verify SSL certificates
	Timeout    time.Duration `json:"timeout"`
}

// DefaultSplunkConfig returns default Splunk configuration
func DefaultSplunkConfig() *SplunkConfig {
	return &SplunkConfig{
		Source:     "authsome",
		SourceType: "audit:event",
		VerifySSL:  true,
		Timeout:    30 * time.Second,
	}
}

// NewSplunkExporter creates a new Splunk HEC exporter
func NewSplunkExporter(config *SplunkConfig) (*SplunkExporter, error) {
	if config.Endpoint == "" {
		return nil, fmt.Errorf("splunk endpoint is required")
	}
	if config.Token == "" {
		return nil, fmt.Errorf("splunk token is required")
	}

	client := &http.Client{
		Timeout: config.Timeout,
		// Would configure TLS based on VerifySSL in production
	}

	return &SplunkExporter{
		config: config,
		client: client,
	}, nil
}

// Name returns the exporter name
func (e *SplunkExporter) Name() string {
	return "splunk"
}

// Export exports a batch of events to Splunk HEC
func (e *SplunkExporter) Export(ctx context.Context, events []*audit.Event) error {
	if len(events) == 0 {
		return nil
	}

	// Convert events to Splunk HEC format
	hecEvents := e.convertToHECFormat(events)

	// Marshal to JSON
	payload, err := json.Marshal(hecEvents)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", e.config.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Splunk "+e.config.Token)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("splunk returned status %d", resp.StatusCode)
	}

	return nil
}

// convertToHECFormat converts audit events to Splunk HEC format
func (e *SplunkExporter) convertToHECFormat(events []*audit.Event) []map[string]interface{} {
	hecEvents := make([]map[string]interface{}, len(events))

	for i, event := range events {
		hecEvents[i] = map[string]interface{}{
			"time":       event.CreatedAt.Unix(),
			"host":       event.IPAddress,
			"source":     e.config.Source,
			"sourcetype": e.config.SourceType,
			"index":      e.config.Index,
			"event": map[string]interface{}{
				"id":         event.ID.String(),
				"app_id":     event.AppID.String(),
				"user_id":    e.getUserID(event),
				"action":     event.Action,
				"resource":   event.Resource,
				"ip_address": event.IPAddress,
				"user_agent": event.UserAgent,
				"metadata":   event.Metadata,
				"timestamp":  event.CreatedAt.Format(time.RFC3339),
			},
		}
	}

	return hecEvents
}

func (e *SplunkExporter) getUserID(event *audit.Event) string {
	if event.UserID != nil {
		return event.UserID.String()
	}
	return ""
}

// HealthCheck checks if Splunk HEC is reachable
func (e *SplunkExporter) HealthCheck(ctx context.Context) error {
	// Send a test request to Splunk HEC health endpoint
	req, err := http.NewRequestWithContext(ctx, "GET", e.config.Endpoint+"/health", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Splunk "+e.config.Token)

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("splunk health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("splunk health check returned status %d", resp.StatusCode)
	}

	return nil
}

// Close closes the exporter
func (e *SplunkExporter) Close() error {
	// Close HTTP client connections
	e.client.CloseIdleConnections()
	return nil
}
