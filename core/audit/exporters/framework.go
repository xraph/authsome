package exporters

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/xraph/authsome/core/audit"
)

// =============================================================================
// SIEM EXPORTER FRAMEWORK - Extensible framework for exporting to SIEM systems
// =============================================================================

// Exporter defines the interface for SIEM exporters
type Exporter interface {
	// Name returns the exporter name (e.g., "splunk", "datadog")
	Name() string

	// Export exports a batch of events
	Export(ctx context.Context, events []*audit.Event) error

	// HealthCheck checks if the exporter is healthy
	HealthCheck(ctx context.Context) error

	// Close closes the exporter and releases resources
	Close() error
}

// ExporterConfig contains common configuration for all exporters
type ExporterConfig struct {
	Name           string        `json:"name"`
	Enabled        bool          `json:"enabled"`
	BatchSize      int           `json:"batchSize"`      // Number of events per batch
	FlushInterval  time.Duration `json:"flushInterval"`  // Max time between flushes
	RetryAttempts  int           `json:"retryAttempts"`  // Number of retry attempts
	RetryBackoff   time.Duration `json:"retryBackoff"`   // Initial retry backoff
	BufferSize     int           `json:"bufferSize"`     // Event buffer size
	ErrorThreshold int           `json:"errorThreshold"` // Consecutive errors before circuit breaker
}

// DefaultExporterConfig returns default exporter configuration
func DefaultExporterConfig(name string) *ExporterConfig {
	return &ExporterConfig{
		Name:           name,
		Enabled:        true,
		BatchSize:      100,
		FlushInterval:  10 * time.Second,
		RetryAttempts:  3,
		RetryBackoff:   1 * time.Second,
		BufferSize:     10000,
		ErrorThreshold: 5,
	}
}

// =============================================================================
// EXPORT MANAGER - Manages multiple SIEM exporters
// =============================================================================

// ExportManager manages multiple SIEM exporters
type ExportManager struct {
	exporters      map[string]*ManagedExporter
	eventBuffer    chan *audit.Event
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	mu             sync.RWMutex
}

// ManagedExporter wraps an exporter with buffering, retries, and circuit breaker
type ManagedExporter struct {
	exporter       Exporter
	config         *ExporterConfig
	buffer         []*audit.Event
	mu             sync.Mutex
	consecutiveErrors int
	circuitOpen    bool
	circuitOpenedAt time.Time
	stats          *ExporterStats
}

// ExporterStats tracks exporter statistics
type ExporterStats struct {
	EventsExported int64     `json:"eventsExported"`
	EventsFailed   int64     `json:"eventsFailed"`
	BatchesExported int64    `json:"batchesExported"`
	BatchesFailed  int64     `json:"batchesFailed"`
	LastExportAt   time.Time `json:"lastExportAt"`
	LastErrorAt    time.Time `json:"lastErrorAt"`
	LastError      string    `json:"lastError"`
	CircuitOpen    bool      `json:"circuitOpen"`
}

// NewExportManager creates a new export manager
func NewExportManager() *ExportManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &ExportManager{
		exporters:   make(map[string]*ManagedExporter),
		eventBuffer: make(chan *audit.Event, 50000), // Large buffer for high throughput
		ctx:         ctx,
		cancel:      cancel,
	}
}

// RegisterExporter registers a new SIEM exporter
func (em *ExportManager) RegisterExporter(exporter Exporter, config *ExporterConfig) error {
	if config == nil {
		config = DefaultExporterConfig(exporter.Name())
	}

	em.mu.Lock()
	defer em.mu.Unlock()

	if _, exists := em.exporters[exporter.Name()]; exists {
		return fmt.Errorf("exporter '%s' already registered", exporter.Name())
	}

	managed := &ManagedExporter{
		exporter: exporter,
		config:   config,
		buffer:   make([]*audit.Event, 0, config.BatchSize),
		stats:    &ExporterStats{},
	}

	em.exporters[exporter.Name()] = managed

	// Start exporter worker
	if config.Enabled {
		em.wg.Add(1)
		go em.exporterWorker(managed)
	}

	return nil
}

// Export queues an event for export to all registered exporters
func (em *ExportManager) Export(event *audit.Event) error {
	select {
	case em.eventBuffer <- event:
		return nil
	case <-em.ctx.Done():
		return fmt.Errorf("export manager shutting down")
	default:
		// Buffer full - would log warning in production
		return fmt.Errorf("export buffer full, event dropped")
	}
}

// exporterWorker processes events for a single exporter
func (em *ExportManager) exporterWorker(managed *ManagedExporter) {
	defer em.wg.Done()

	ticker := time.NewTicker(managed.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-em.ctx.Done():
			// Final flush on shutdown
			managed.flush(em.ctx)
			return

		case event := <-em.eventBuffer:
			// Add to buffer
			managed.addToBuffer(event)

			// Flush if batch size reached
			if len(managed.buffer) >= managed.config.BatchSize {
				managed.flush(em.ctx)
			}

		case <-ticker.C:
			// Periodic flush
			if len(managed.buffer) > 0 {
				managed.flush(em.ctx)
			}

			// Check circuit breaker
			managed.checkCircuitBreaker()
		}
	}
}

// addToBuffer adds an event to the exporter's buffer
func (me *ManagedExporter) addToBuffer(event *audit.Event) {
	me.mu.Lock()
	defer me.mu.Unlock()

	// Check if circuit is open
	if me.circuitOpen {
		me.stats.EventsFailed++
		return
	}

	me.buffer = append(me.buffer, event)
}

// flush exports buffered events
func (me *ManagedExporter) flush(ctx context.Context) {
	me.mu.Lock()
	if len(me.buffer) == 0 {
		me.mu.Unlock()
		return
	}

	// Copy buffer and clear
	batch := make([]*audit.Event, len(me.buffer))
	copy(batch, me.buffer)
	me.buffer = me.buffer[:0]
	me.mu.Unlock()

	// Export with retries
	err := me.exportWithRetry(ctx, batch)
	if err != nil {
		me.handleExportError(err, len(batch))
	} else {
		me.handleExportSuccess(len(batch))
	}
}

// exportWithRetry exports events with exponential backoff retry
func (me *ManagedExporter) exportWithRetry(ctx context.Context, events []*audit.Event) error {
	var lastErr error
	backoff := me.config.RetryBackoff

	for attempt := 0; attempt <= me.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
			backoff *= 2
		}

		// Attempt export
		err := me.exporter.Export(ctx, events)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryable(err) {
			break
		}
	}

	return fmt.Errorf("export failed after %d attempts: %w", me.config.RetryAttempts, lastErr)
}

// handleExportSuccess updates stats after successful export
func (me *ManagedExporter) handleExportSuccess(count int) {
	me.mu.Lock()
	defer me.mu.Unlock()

	me.stats.EventsExported += int64(count)
	me.stats.BatchesExported++
	me.stats.LastExportAt = time.Now()
	me.consecutiveErrors = 0
}

// handleExportError updates stats after failed export
func (me *ManagedExporter) handleExportError(err error, count int) {
	me.mu.Lock()
	defer me.mu.Unlock()

	me.stats.EventsFailed += int64(count)
	me.stats.BatchesFailed++
	me.stats.LastErrorAt = time.Now()
	me.stats.LastError = err.Error()
	me.consecutiveErrors++

	// Open circuit breaker if threshold reached
	if me.consecutiveErrors >= me.config.ErrorThreshold {
		me.circuitOpen = true
		me.circuitOpenedAt = time.Now()
		me.stats.CircuitOpen = true
	}
}

// checkCircuitBreaker checks if circuit breaker should be reset
func (me *ManagedExporter) checkCircuitBreaker() {
	me.mu.Lock()
	defer me.mu.Unlock()

	if !me.circuitOpen {
		return
	}

	// Reset circuit after 1 minute
	if time.Since(me.circuitOpenedAt) > 1*time.Minute {
		me.circuitOpen = false
		me.consecutiveErrors = 0
		me.stats.CircuitOpen = false
	}
}

// GetStats returns statistics for all exporters
func (em *ExportManager) GetStats() map[string]*ExporterStats {
	em.mu.RLock()
	defer em.mu.RUnlock()

	stats := make(map[string]*ExporterStats)
	for name, managed := range em.exporters {
		managed.mu.Lock()
		statsCopy := *managed.stats
		managed.mu.Unlock()
		stats[name] = &statsCopy
	}

	return stats
}

// HealthCheck checks health of all exporters
func (em *ExportManager) HealthCheck(ctx context.Context) map[string]error {
	em.mu.RLock()
	defer em.mu.RUnlock()

	results := make(map[string]error)
	for name, managed := range em.exporters {
		err := managed.exporter.HealthCheck(ctx)
		results[name] = err
	}

	return results
}

// Shutdown gracefully shuts down the export manager
func (em *ExportManager) Shutdown(timeout time.Duration) error {
	em.cancel()

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		em.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Clean shutdown
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout exceeded")
	}

	// Close all exporters
	em.mu.Lock()
	defer em.mu.Unlock()

	for name, managed := range em.exporters {
		if err := managed.exporter.Close(); err != nil {
			// Log error but continue
			fmt.Printf("Error closing exporter '%s': %v\n", name, err)
		}
	}

	return nil
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// isRetryable checks if an error is retryable
func isRetryable(err error) bool {
	// Check for common retryable errors
	// - Network timeouts
	// - Temporary failures
	// - Rate limits (429)
	// - Server errors (5xx)
	
	// Simplified version - would check specific error types in production
	return true
}

// =============================================================================
// EVENT FORMATTER - Formats events for different SIEM systems
// =============================================================================

// EventFormatter formats audit events for different SIEM systems
type EventFormatter interface {
	Format(event *audit.Event) ([]byte, error)
}

// JSONFormatter formats events as JSON
type JSONFormatter struct{}

// Format formats an event as JSON
func (f *JSONFormatter) Format(event *audit.Event) ([]byte, error) {
	// Would use json.Marshal in production
	// Simplified version
	return []byte(fmt.Sprintf(`{"action":"%s","resource":"%s"}`, event.Action, event.Resource)), nil
}

// CEFFormatter formats events as Common Event Format (used by ArcSight, QRadar)
type CEFFormatter struct{}

// Format formats an event as CEF
func (f *CEFFormatter) Format(event *audit.Event) ([]byte, error) {
	// CEF format: CEF:Version|Device Vendor|Device Product|Device Version|Signature ID|Name|Severity|Extension
	cef := fmt.Sprintf(
		"CEF:0|Authsome|Audit|1.0|%s|%s|5|src=%s act=%s",
		event.Action,
		event.Action,
		event.IPAddress,
		event.Action,
	)
	return []byte(cef), nil
}

// LEEFFormatter formats events as Log Event Extended Format (used by QRadar)
type LEEFFormatter struct{}

// Format formats an event as LEEF
func (f *LEEFFormatter) Format(event *audit.Event) ([]byte, error) {
	// LEEF format: LEEF:Version|Vendor|Product|Version|EventID|
	leef := fmt.Sprintf(
		"LEEF:2.0|Authsome|Audit|1.0|%s|\tsrc=%s\tact=%s",
		event.Action,
		event.IPAddress,
		event.Action,
	)
	return []byte(leef), nil
}

