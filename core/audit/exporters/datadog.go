package exporters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/internal/errs"
)

// =============================================================================
// DATADOG EXPORTER - Exports audit events to Datadog Logs API
// =============================================================================

// DatadogExporter exports audit events to Datadog.
type DatadogExporter struct {
	config *DatadogConfig
	client *http.Client
}

// DatadogConfig contains Datadog configuration.
type DatadogConfig struct {
	APIKey  string        `json:"apiKey"`  // Datadog API key
	Site    string        `json:"site"`    // Datadog site (e.g., datadoghq.com, datadoghq.eu)
	Service string        `json:"service"` // Service name
	Source  string        `json:"source"`  // Log source
	Tags    []string      `json:"tags"`    // Additional tags
	Timeout time.Duration `json:"timeout"`
}

// DefaultDatadogConfig returns default Datadog configuration.
func DefaultDatadogConfig() *DatadogConfig {
	return &DatadogConfig{
		Site:    "datadoghq.com",
		Service: "authsome",
		Source:  "audit",
		Timeout: 30 * time.Second,
		Tags:    []string{"env:production"},
	}
}

// NewDatadogExporter creates a new Datadog exporter.
func NewDatadogExporter(config *DatadogConfig) (*DatadogExporter, error) {
	if config.APIKey == "" {
		return nil, errs.New(errs.CodeInvalidInput, "datadog API key is required", http.StatusBadRequest)
	}

	client := &http.Client{
		Timeout: config.Timeout,
	}

	return &DatadogExporter{
		config: config,
		client: client,
	}, nil
}

// Name returns the exporter name.
func (e *DatadogExporter) Name() string {
	return "datadog"
}

// Export exports a batch of events to Datadog.
func (e *DatadogExporter) Export(ctx context.Context, events []*audit.Event) error {
	if len(events) == 0 {
		return nil
	}

	// Convert events to Datadog format
	logs := e.convertToDatadogFormat(events)

	// Marshal to JSON
	payload, err := json.Marshal(logs)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	// Create HTTP request
	endpoint := fmt.Sprintf("https://http-intake.logs.%s/api/v2/logs", e.config.Site)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Dd-Api-Key", e.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("datadog returned status %d", resp.StatusCode)
	}

	return nil
}

// convertToDatadogFormat converts audit events to Datadog format.
func (e *DatadogExporter) convertToDatadogFormat(events []*audit.Event) []map[string]any {
	logs := make([]map[string]any, len(events))

	for i, event := range events {
		// Build tags
		tags := append([]string{
			"action:" + event.Action,
			"resource:" + event.Resource,
			"app_id:" + event.AppID.String(),
		}, e.config.Tags...)

		if event.UserID != nil {
			tags = append(tags, "user_id:"+event.UserID.String())
		}

		logs[i] = map[string]any{
			"ddsource": e.config.Source,
			"ddtags":   tags,
			"hostname": event.IPAddress,
			"service":  e.config.Service,
			"message": fmt.Sprintf("%s performed %s on %s",
				e.getUserID(event),
				event.Action,
				event.Resource,
			),
			"timestamp": event.CreatedAt.UnixNano() / 1000000, // Milliseconds
			"attributes": map[string]any{
				"id":         event.ID.String(),
				"app_id":     event.AppID.String(),
				"user_id":    e.getUserID(event),
				"action":     event.Action,
				"resource":   event.Resource,
				"ip_address": event.IPAddress,
				"user_agent": event.UserAgent,
				"metadata":   event.Metadata,
			},
		}
	}

	return logs
}

func (e *DatadogExporter) getUserID(event *audit.Event) string {
	if event.UserID != nil {
		return event.UserID.String()
	}

	return "system"
}

// HealthCheck checks if Datadog API is reachable.
func (e *DatadogExporter) HealthCheck(ctx context.Context) error {
	// Send a minimal log entry to test connectivity
	testLog := []map[string]any{
		{
			"ddsource": e.config.Source,
			"service":  e.config.Service,
			"message":  "health_check",
			"ddtags":   []string{"test:health_check"},
		},
	}

	payload, _ := json.Marshal(testLog)
	endpoint := fmt.Sprintf("https://http-intake.logs.%s/api/v2/logs", e.config.Site)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Dd-Api-Key", e.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("datadog health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("datadog health check returned status %d", resp.StatusCode)
	}

	return nil
}

// Close closes the exporter.
func (e *DatadogExporter) Close() error {
	e.client.CloseIdleConnections()

	return nil
}
