package notification

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/xid"
)

// DispatcherConfig holds configuration for the notification dispatcher
type DispatcherConfig struct {
	// AsyncEnabled enables async processing for non-critical notifications
	AsyncEnabled bool `json:"asyncEnabled"`
	// WorkerPoolSize is the number of workers per priority level
	WorkerPoolSize int `json:"workerPoolSize"`
	// QueueSize is the buffer size for each priority queue
	QueueSize int `json:"queueSize"`
	// ShutdownTimeout is the max time to wait for graceful shutdown
	ShutdownTimeout time.Duration `json:"shutdownTimeout"`
}

// DefaultDispatcherConfig returns sensible defaults
func DefaultDispatcherConfig() DispatcherConfig {
	return DispatcherConfig{
		AsyncEnabled:    true,
		WorkerPoolSize:  5,
		QueueSize:       1000,
		ShutdownTimeout: 30 * time.Second,
	}
}

// DispatchRequest represents a notification dispatch request
type DispatchRequest struct {
	AppID       xid.ID
	Type        NotificationType
	Priority    NotificationPriority
	Recipient   string
	Subject     string
	Body        string
	TemplateKey string
	Variables   map[string]interface{}
	Metadata    map[string]interface{}
}

// DispatchResult represents the result of a dispatch operation
type DispatchResult struct {
	NotificationID xid.ID
	Status         NotificationStatus
	Error          error
	Queued         bool // true if queued for async processing
}

// NotificationSender is the interface for sending notifications
type NotificationSender interface {
	Send(ctx context.Context, req *SendRequest) (*Notification, error)
}

// Dispatcher handles async notification dispatching with priority-based queuing
type Dispatcher struct {
	config  DispatcherConfig
	sender  NotificationSender
	retry   *RetryService
	queues  map[NotificationPriority]chan *dispatchJob
	wg      sync.WaitGroup
	mu      sync.RWMutex
	running bool
	stopCh  chan struct{}
}

// dispatchJob represents a job in the queue
type dispatchJob struct {
	ctx     context.Context
	request *DispatchRequest
	result  chan *DispatchResult
}

// NewDispatcher creates a new notification dispatcher
func NewDispatcher(config DispatcherConfig, sender NotificationSender, retry *RetryService) *Dispatcher {
	if config.WorkerPoolSize <= 0 {
		config.WorkerPoolSize = 5
	}
	if config.QueueSize <= 0 {
		config.QueueSize = 1000
	}
	if config.ShutdownTimeout <= 0 {
		config.ShutdownTimeout = 30 * time.Second
	}

	d := &Dispatcher{
		config: config,
		sender: sender,
		retry:  retry,
		queues: make(map[NotificationPriority]chan *dispatchJob),
		stopCh: make(chan struct{}),
	}

	// Create queues for each priority
	priorities := []NotificationPriority{PriorityCritical, PriorityHigh, PriorityNormal, PriorityLow}
	for _, p := range priorities {
		d.queues[p] = make(chan *dispatchJob, config.QueueSize)
	}

	return d
}

// Start starts the dispatcher workers
func (d *Dispatcher) Start() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.running {
		return
	}
	d.running = true

	// Start workers for each priority level
	// Critical gets more workers since it's synchronous
	workerCounts := map[NotificationPriority]int{
		PriorityCritical: d.config.WorkerPoolSize * 2, // More workers for critical
		PriorityHigh:     d.config.WorkerPoolSize,
		PriorityNormal:   d.config.WorkerPoolSize,
		PriorityLow:      d.config.WorkerPoolSize / 2, // Fewer for low priority
	}

	for priority, count := range workerCounts {
		if count < 1 {
			count = 1
		}
		for i := 0; i < count; i++ {
			d.wg.Add(1)
			go d.worker(priority)
		}
	}

	fmt.Printf("[Dispatcher] Started with %d workers per priority level\n", d.config.WorkerPoolSize)
}

// Stop gracefully stops the dispatcher
func (d *Dispatcher) Stop() {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return
	}
	d.running = false
	d.mu.Unlock()

	close(d.stopCh)

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		fmt.Println("[Dispatcher] Shutdown complete")
	case <-time.After(d.config.ShutdownTimeout):
		fmt.Println("[Dispatcher] Shutdown timed out, some notifications may not have been sent")
	}
}

// worker processes jobs from a specific priority queue
func (d *Dispatcher) worker(priority NotificationPriority) {
	defer d.wg.Done()

	queue := d.queues[priority]
	for {
		select {
		case <-d.stopCh:
			return
		case job, ok := <-queue:
			if !ok {
				return
			}
			d.processJob(job)
		}
	}
}

// processJob processes a single dispatch job
func (d *Dispatcher) processJob(job *dispatchJob) {
	ctx := job.ctx
	req := job.request

	result := &DispatchResult{}

	// Build send request
	sendReq := &SendRequest{
		AppID:        req.AppID,
		Type:         req.Type,
		Recipient:    req.Recipient,
		Subject:      req.Subject,
		Body:         req.Body,
		TemplateName: req.TemplateKey,
		Variables:    req.Variables,
		Metadata:     req.Metadata,
	}

	// Send the notification
	notification, err := d.sender.Send(ctx, sendReq)
	if err != nil {
		result.Error = err
		result.Status = NotificationStatusFailed

		// Queue for retry based on priority
		if d.retry != nil && req.Priority != PriorityLow {
			if retryErr := d.retry.QueueForRetry(ctx, req, err); retryErr != nil {
				fmt.Printf("[Dispatcher] Failed to queue for retry: %v\n", retryErr)
			}
		}
	} else {
		result.NotificationID = notification.ID
		result.Status = notification.Status
	}

	// Send result if channel is provided (for sync dispatch)
	if job.result != nil {
		select {
		case job.result <- result:
		default:
			// Result channel not listening, skip
		}
	}
}

// Dispatch dispatches a notification with the specified priority
// Critical notifications are processed synchronously
// Non-critical notifications are queued for async processing
func (d *Dispatcher) Dispatch(ctx context.Context, req *DispatchRequest) *DispatchResult {
	d.mu.RLock()
	running := d.running
	d.mu.RUnlock()

	// If not running or async disabled, process synchronously
	if !running || !d.config.AsyncEnabled {
		return d.dispatchSync(ctx, req)
	}

	// Critical notifications are always synchronous
	if req.Priority == PriorityCritical {
		return d.dispatchSync(ctx, req)
	}

	// Queue for async processing
	return d.dispatchAsync(ctx, req)
}

// dispatchSync sends notification synchronously and waits for result
func (d *Dispatcher) dispatchSync(ctx context.Context, req *DispatchRequest) *DispatchResult {
	resultCh := make(chan *DispatchResult, 1)
	job := &dispatchJob{
		ctx:     ctx,
		request: req,
		result:  resultCh,
	}

	// Process immediately in current goroutine
	d.processJob(job)

	select {
	case result := <-resultCh:
		return result
	case <-ctx.Done():
		return &DispatchResult{
			Status: NotificationStatusFailed,
			Error:  ctx.Err(),
		}
	}
}

// dispatchAsync queues notification for async processing
func (d *Dispatcher) dispatchAsync(ctx context.Context, req *DispatchRequest) *DispatchResult {
	queue, ok := d.queues[req.Priority]
	if !ok {
		queue = d.queues[PriorityNormal] // Fallback to normal queue
	}

	job := &dispatchJob{
		ctx:     ctx,
		request: req,
		result:  nil, // No result channel for async
	}

	select {
	case queue <- job:
		return &DispatchResult{
			Status: NotificationStatusPending,
			Queued: true,
		}
	default:
		// Queue is full, process synchronously as fallback
		fmt.Printf("[Dispatcher] Queue full for priority %s, falling back to sync\n", req.Priority)
		return d.dispatchSync(ctx, req)
	}
}

// DispatchWithPriority is a convenience method for dispatching with a specific priority
func (d *Dispatcher) DispatchWithPriority(ctx context.Context, req *SendRequest, priority NotificationPriority) *DispatchResult {
	dispatchReq := &DispatchRequest{
		AppID:       req.AppID,
		Type:        req.Type,
		Priority:    priority,
		Recipient:   req.Recipient,
		Subject:     req.Subject,
		Body:        req.Body,
		TemplateKey: req.TemplateName,
		Variables:   req.Variables,
		Metadata:    req.Metadata,
	}
	return d.Dispatch(ctx, dispatchReq)
}

// QueueLength returns the current queue length for a priority
func (d *Dispatcher) QueueLength(priority NotificationPriority) int {
	queue, ok := d.queues[priority]
	if !ok {
		return 0
	}
	return len(queue)
}

// IsRunning returns whether the dispatcher is running
func (d *Dispatcher) IsRunning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running
}

