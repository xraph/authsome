package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rs/xid"
)

// RetryConfig holds configuration for the retry service.
type RetryConfig struct {
	// Enabled enables retry functionality
	Enabled bool `json:"enabled"`
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int `json:"maxRetries"`
	// BackoffDurations are the delays between retries (e.g., 1m, 5m, 15m)
	BackoffDurations []time.Duration `json:"backoffDurations"`
	// PersistFailures persists permanently failed notifications to DB
	PersistFailures bool `json:"persistFailures"`
}

// DefaultRetryConfig returns sensible defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		Enabled:    true,
		MaxRetries: 3,
		BackoffDurations: []time.Duration{
			1 * time.Minute,
			5 * time.Minute,
			15 * time.Minute,
		},
		PersistFailures: true,
	}
}

// RetryItem represents a notification queued for retry.
type RetryItem struct {
	ID          xid.ID               `json:"id"`
	AppID       xid.ID               `json:"appId"`
	Type        NotificationType     `json:"type"`
	Priority    NotificationPriority `json:"priority"`
	Recipient   string               `json:"recipient"`
	Subject     string               `json:"subject,omitempty"`
	Body        string               `json:"body,omitempty"`
	TemplateKey string               `json:"templateKey,omitempty"`
	Variables   map[string]any       `json:"variables,omitempty"`
	Metadata    map[string]any       `json:"metadata,omitempty"`
	Attempts    int                  `json:"attempts"`
	LastError   string               `json:"lastError"`
	NextRetry   time.Time            `json:"nextRetry"`
	CreatedAt   time.Time            `json:"createdAt"`
}

// RetryStorage defines the interface for retry queue storage.
type RetryStorage interface {
	// Enqueue adds an item to the retry queue
	Enqueue(ctx context.Context, item *RetryItem) error
	// Dequeue retrieves items ready for retry
	Dequeue(ctx context.Context, limit int) ([]*RetryItem, error)
	// Update updates an item's retry state
	Update(ctx context.Context, item *RetryItem) error
	// Delete removes an item from the queue
	Delete(ctx context.Context, id xid.ID) error
	// MarkFailed marks an item as permanently failed
	MarkFailed(ctx context.Context, item *RetryItem) error
	// GetStats returns queue statistics
	GetStats(ctx context.Context) (*RetryStats, error)
}

// RetryStats holds retry queue statistics.
type RetryStats struct {
	PendingCount   int64 `json:"pendingCount"`
	FailedCount    int64 `json:"failedCount"`
	ProcessedCount int64 `json:"processedCount"`
}

// InMemoryRetryStorage provides an in-memory implementation of RetryStorage
// Used when Redis is not available.
type InMemoryRetryStorage struct {
	mu        sync.RWMutex
	items     map[xid.ID]*RetryItem
	failed    map[xid.ID]*RetryItem
	processed int64
}

// NewInMemoryRetryStorage creates a new in-memory retry storage.
func NewInMemoryRetryStorage() *InMemoryRetryStorage {
	return &InMemoryRetryStorage{
		items:  make(map[xid.ID]*RetryItem),
		failed: make(map[xid.ID]*RetryItem),
	}
}

func (s *InMemoryRetryStorage) Enqueue(ctx context.Context, item *RetryItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items[item.ID] = item

	return nil
}

func (s *InMemoryRetryStorage) Dequeue(ctx context.Context, limit int) ([]*RetryItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	result := make([]*RetryItem, 0, limit)

	for _, item := range s.items {
		if item.NextRetry.Before(now) || item.NextRetry.Equal(now) {
			result = append(result, item)
			if len(result) >= limit {
				break
			}
		}
	}

	return result, nil
}

func (s *InMemoryRetryStorage) Update(ctx context.Context, item *RetryItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items[item.ID] = item

	return nil
}

func (s *InMemoryRetryStorage) Delete(ctx context.Context, id xid.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.items, id)
	s.processed++

	return nil
}

func (s *InMemoryRetryStorage) MarkFailed(ctx context.Context, item *RetryItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.items, item.ID)
	s.failed[item.ID] = item

	return nil
}

func (s *InMemoryRetryStorage) GetStats(ctx context.Context) (*RetryStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &RetryStats{
		PendingCount:   int64(len(s.items)),
		FailedCount:    int64(len(s.failed)),
		ProcessedCount: s.processed,
	}, nil
}

// RetryService handles notification retry logic.
type RetryService struct {
	config  RetryConfig
	storage RetryStorage
	sender  NotificationSender
	mu      sync.RWMutex
	running bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewRetryService creates a new retry service.
func NewRetryService(config RetryConfig, storage RetryStorage, sender NotificationSender) *RetryService {
	if len(config.BackoffDurations) == 0 {
		config.BackoffDurations = DefaultRetryConfig().BackoffDurations
	}

	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	return &RetryService{
		config:  config,
		storage: storage,
		sender:  sender,
		stopCh:  make(chan struct{}),
	}
}

// Start starts the retry processor.
func (r *RetryService) Start() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running || !r.config.Enabled {
		return
	}

	r.running = true

	r.wg.Add(1)

	go r.processor()
}

// Stop stops the retry processor.
func (r *RetryService) Stop() {
	r.mu.Lock()

	if !r.running {
		r.mu.Unlock()

		return
	}

	r.running = false
	r.mu.Unlock()

	close(r.stopCh)
	r.wg.Wait()
}

// processor runs the retry processing loop.
func (r *RetryService) processor() {
	defer r.wg.Done()

	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.processRetries()
		}
	}
}

// processRetries processes pending retries.
func (r *RetryService) processRetries() {
	ctx := context.Background()

	// Get items ready for retry
	items, err := r.storage.Dequeue(ctx, 100)
	if err != nil {
		return
	}

	for _, item := range items {
		r.processItem(ctx, item)
	}
}

// processItem processes a single retry item.
func (r *RetryService) processItem(ctx context.Context, item *RetryItem) {
	// Build send request
	sendReq := &SendRequest{
		AppID:        item.AppID,
		Type:         item.Type,
		Recipient:    item.Recipient,
		Subject:      item.Subject,
		Body:         item.Body,
		TemplateName: item.TemplateKey,
		Variables:    item.Variables,
		Metadata:     item.Metadata,
	}

	// Attempt to send
	_, err := r.sender.Send(ctx, sendReq)
	if err == nil {
		// Success - remove from queue
		_ = r.storage.Delete(ctx, item.ID)

		return
	}

	// Failed - update retry info
	item.Attempts++
	item.LastError = err.Error()

	// Check if max retries exceeded
	if item.Attempts >= r.config.MaxRetries {
		if r.config.PersistFailures {
			_ = r.storage.MarkFailed(ctx, item)
		} else {
			_ = r.storage.Delete(ctx, item.ID)
		}

		return
	}

	// Calculate next retry time
	backoffIndex := item.Attempts - 1
	if backoffIndex >= len(r.config.BackoffDurations) {
		backoffIndex = len(r.config.BackoffDurations) - 1
	}

	item.NextRetry = time.Now().Add(r.config.BackoffDurations[backoffIndex])

	// Update in storage
	_ = r.storage.Update(ctx, item)
}

// QueueForRetry queues a failed notification for retry.
func (r *RetryService) QueueForRetry(ctx context.Context, req *DispatchRequest, originalErr error) error {
	if !r.config.Enabled {
		return nil
	}

	// Don't retry low priority notifications
	if req.Priority == PriorityLow {
		return nil
	}

	item := &RetryItem{
		ID:          xid.New(),
		AppID:       req.AppID,
		Type:        req.Type,
		Priority:    req.Priority,
		Recipient:   req.Recipient,
		Subject:     req.Subject,
		Body:        req.Body,
		TemplateKey: req.TemplateKey,
		Variables:   req.Variables,
		Metadata:    req.Metadata,
		Attempts:    1, // This is the first retry
		LastError:   originalErr.Error(),
		NextRetry:   time.Now().Add(r.config.BackoffDurations[0]),
		CreatedAt:   time.Now(),
	}

	if err := r.storage.Enqueue(ctx, item); err != nil {
		return fmt.Errorf("failed to enqueue for retry: %w", err)
	}

	return nil
}

// GetStats returns retry queue statistics.
func (r *RetryService) GetStats(ctx context.Context) (*RetryStats, error) {
	return r.storage.GetStats(ctx)
}

// SerializeRetryItem serializes a retry item to JSON.
func SerializeRetryItem(item *RetryItem) ([]byte, error) {
	return json.Marshal(item)
}

// DeserializeRetryItem deserializes a retry item from JSON.
func DeserializeRetryItem(data []byte) (*RetryItem, error) {
	var item RetryItem
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, err
	}

	return &item, nil
}
