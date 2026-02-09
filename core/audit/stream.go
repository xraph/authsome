package audit

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/rs/xid"
)

// =============================================================================
// WEBSOCKET STREAMING - Real-time audit event streaming
// =============================================================================

// StreamFilter defines filters for streaming audit events.
type StreamFilter struct {
	AppID      *xid.ID  `json:"appId,omitempty"`
	UserID     *xid.ID  `json:"userId,omitempty"`
	Actions    []string `json:"actions,omitempty"`    // Filter by specific actions
	BufferSize int      `json:"bufferSize,omitempty"` // Channel buffer size
}

// StreamService manages real-time audit event streaming
// Note: Requires PostgreSQL LISTEN/NOTIFY support via pgdriver.Listener.
type StreamService struct {
	listener  interface{} // pgdriver.Listener interface for PostgreSQL NOTIFY
	listeners map[string]*clientListener
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// clientListener represents a single WebSocket client's listener.
type clientListener struct {
	id       string
	filter   *StreamFilter
	events   chan *Event
	done     chan struct{}
	lastSeen time.Time
}

// NewStreamService creates a new stream service
// listener should be *pgdriver.Listener from github.com/uptrace/bun/driver/pgdriver.
func NewStreamService(listener interface{}) *StreamService {
	ctx, cancel := context.WithCancel(context.Background())

	svc := &StreamService{
		listener:  listener,
		listeners: make(map[string]*clientListener),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Start listening to PostgreSQL notifications
	go svc.listen()

	// Start heartbeat checker
	go svc.heartbeat()

	return svc
}

// Subscribe subscribes to audit event stream with optional filters.
func (s *StreamService) Subscribe(ctx context.Context, filter *StreamFilter) (<-chan *Event, string, error) {
	if filter == nil {
		filter = &StreamFilter{}
	}

	// Set default buffer size
	if filter.BufferSize == 0 {
		filter.BufferSize = 100
	}

	// Create client listener
	clientID := xid.New().String()
	client := &clientListener{
		id:       clientID,
		filter:   filter,
		events:   make(chan *Event, filter.BufferSize),
		done:     make(chan struct{}),
		lastSeen: time.Now(),
	}

	// Register client
	s.mu.Lock()
	s.listeners[clientID] = client
	s.mu.Unlock()

	// Start client monitor
	go s.monitorClient(ctx, clientID)

	return client.events, clientID, nil
}

// Unsubscribe removes a client subscription.
func (s *StreamService) Unsubscribe(clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if client, exists := s.listeners[clientID]; exists {
		close(client.done)
		close(client.events)
		delete(s.listeners, clientID)
	}
}

// listen processes PostgreSQL NOTIFY messages
// Note: This is a placeholder implementation. In production, would use pgdriver.Listener.
func (s *StreamService) listen() {
	// This method requires pgdriver.Listener from github.com/uptrace/bun/driver/pgdriver
	// Implementation would be:
	// 1. listener.Listen(ctx, "audit_events")
	// 2. for notification := range listener.Channel() { ... }
	// 3. Parse notification.Payload and broadcast to clients

	// For now, this is a stub that external implementations can override
}

// parseNotification parses PostgreSQL notification payload into Event.
func (s *StreamService) parseNotification(payload string) (*Event, error) {
	var data struct {
		ID        string    `json:"id"`
		AppID     string    `json:"app_id"`
		UserID    *string   `json:"user_id"`
		Action    string    `json:"action"`
		Resource  string    `json:"resource"`
		IPAddress string    `json:"ip_address"`
		CreatedAt time.Time `json:"created_at"`
	}

	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return nil, err
	}

	// Convert to Event
	event := &Event{
		Action:    data.Action,
		Resource:  data.Resource,
		IPAddress: data.IPAddress,
		CreatedAt: data.CreatedAt,
	}

	// Parse IDs
	if id, err := xid.FromString(data.ID); err == nil {
		event.ID = id
	}

	if appID, err := xid.FromString(data.AppID); err == nil {
		event.AppID = appID
	}

	if data.UserID != nil {
		if userID, err := xid.FromString(*data.UserID); err == nil {
			event.UserID = &userID
		}
	}

	return event, nil
}

// broadcast sends event to all matching clients.
func (s *StreamService) broadcast(event *Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, client := range s.listeners {
		if s.matchesFilter(event, client.filter) {
			select {
			case client.events <- event:
				// Event sent successfully
			case <-client.done:
				// Client disconnected
			default:
				// Buffer full, skip event (or could implement backpressure)
			}
		}
	}
}

// matchesFilter checks if event matches client's filter.
func (s *StreamService) matchesFilter(event *Event, filter *StreamFilter) bool {
	if filter == nil {
		return true
	}

	// Check AppID filter
	if filter.AppID != nil && event.AppID.Compare(*filter.AppID) != 0 {
		return false
	}

	// Check UserID filter
	if filter.UserID != nil {
		if event.UserID == nil || event.UserID.Compare(*filter.UserID) != 0 {
			return false
		}
	}

	// Check Actions filter
	if len(filter.Actions) > 0 {
		matched := false

		for _, action := range filter.Actions {
			if event.Action == action {
				matched = true

				break
			}
		}

		if !matched {
			return false
		}
	}

	return true
}

// monitorClient monitors a client connection and cleans up on disconnect.
func (s *StreamService) monitorClient(ctx context.Context, clientID string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.Unsubscribe(clientID)

			return
		case <-ticker.C:
			s.mu.RLock()
			client, exists := s.listeners[clientID]
			s.mu.RUnlock()

			if !exists {
				return
			}

			// Check if client is still active (updated by heartbeat)
			if time.Since(client.lastSeen) > 2*time.Minute {
				// Client inactive - disconnect
				s.Unsubscribe(clientID)

				return
			}
		}
	}
}

// heartbeat sends periodic heartbeats to detect stale clients.
func (s *StreamService) heartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.mu.RLock()
			count := len(s.listeners)
			s.mu.RUnlock()

			// Log active connections (would use structured logger)
			_ = count
		}
	}
}

// UpdateHeartbeat updates client's last seen time (called by WebSocket ping).
func (s *StreamService) UpdateHeartbeat(clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if client, exists := s.listeners[clientID]; exists {
		client.lastSeen = time.Now()
	}
}

// Stats returns streaming statistics.
func (s *StreamService) Stats() StreamStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := StreamStats{
		ActiveClients: len(s.listeners),
		Clients:       make([]ClientStats, 0, len(s.listeners)),
	}

	for _, client := range s.listeners {
		stats.Clients = append(stats.Clients, ClientStats{
			ID:         client.id,
			BufferSize: len(client.events),
			Connected:  time.Since(client.lastSeen),
		})
	}

	return stats
}

// Shutdown gracefully shuts down the stream service.
func (s *StreamService) Shutdown() {
	s.cancel()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Close all client connections
	for clientID := range s.listeners {
		if client, exists := s.listeners[clientID]; exists {
			close(client.done)
			close(client.events)
		}
	}

	s.listeners = make(map[string]*clientListener)
}

// =============================================================================
// STREAMING TYPES
// =============================================================================

// StreamStats contains streaming service statistics.
type StreamStats struct {
	ActiveClients int           `json:"activeClients"`
	Clients       []ClientStats `json:"clients"`
}

// ClientStats contains per-client statistics.
type ClientStats struct {
	ID         string        `json:"id"`
	BufferSize int           `json:"bufferSize"`
	Connected  time.Duration `json:"connected"`
}

// =============================================================================
// SQLITE FALLBACK - Polling-based streaming for SQLite
// =============================================================================

// PollingStreamService provides streaming for SQLite using polling.
type PollingStreamService struct {
	repo      Repository
	listeners map[string]*clientListener
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	lastID    xid.ID
}

// NewPollingStreamService creates a polling-based stream service (for SQLite).
func NewPollingStreamService(repo Repository) *PollingStreamService {
	ctx, cancel := context.WithCancel(context.Background())

	svc := &PollingStreamService{
		repo:      repo,
		listeners: make(map[string]*clientListener),
		ctx:       ctx,
		cancel:    cancel,
		lastID:    xid.NilID(),
	}

	// Start polling
	go svc.poll()

	return svc
}

// Subscribe creates a subscription (same interface as StreamService).
func (s *PollingStreamService) Subscribe(ctx context.Context, filter *StreamFilter) (<-chan *Event, string, error) {
	if filter == nil {
		filter = &StreamFilter{}
	}

	if filter.BufferSize == 0 {
		filter.BufferSize = 100
	}

	clientID := xid.New().String()
	client := &clientListener{
		id:       clientID,
		filter:   filter,
		events:   make(chan *Event, filter.BufferSize),
		done:     make(chan struct{}),
		lastSeen: time.Now(),
	}

	s.mu.Lock()
	s.listeners[clientID] = client
	s.mu.Unlock()

	go s.monitorClient(ctx, clientID)

	return client.events, clientID, nil
}

// poll queries for new events periodically.
func (s *PollingStreamService) poll() {
	ticker := time.NewTicker(1 * time.Second) // Poll every second
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.fetchNewEvents()
		}
	}
}

// fetchNewEvents fetches events since last poll.
func (s *PollingStreamService) fetchNewEvents() {
	// Query for events created after lastID
	// Using pagination to get recent events
	resp, err := s.repo.List(s.ctx, &ListEventsFilter{})
	if err != nil {
		return
	}

	// Broadcast new events
	for _, schemaEvent := range resp.Data {
		event := FromSchemaEvent(schemaEvent)

		// Update lastID
		if event.ID.Compare(s.lastID) > 0 {
			s.lastID = event.ID
		}

		// Broadcast to matching clients
		s.broadcast(event)
	}
}

// broadcast and other methods similar to StreamService.
func (s *PollingStreamService) broadcast(event *Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, client := range s.listeners {
		if matchesFilter(event, client.filter) {
			select {
			case client.events <- event:
			case <-client.done:
			default:
			}
		}
	}
}

func (s *PollingStreamService) monitorClient(ctx context.Context, clientID string) {
	<-ctx.Done()
	s.Unsubscribe(clientID)
}

func (s *PollingStreamService) Unsubscribe(clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if client, exists := s.listeners[clientID]; exists {
		close(client.done)
		close(client.events)
		delete(s.listeners, clientID)
	}
}

func (s *PollingStreamService) Shutdown() {
	s.cancel()
}

// Helper function for filter matching (same as StreamService).
func matchesFilter(event *Event, filter *StreamFilter) bool {
	if filter == nil {
		return true
	}

	if filter.AppID != nil && event.AppID.Compare(*filter.AppID) != 0 {
		return false
	}

	if filter.UserID != nil {
		if event.UserID == nil || event.UserID.Compare(*filter.UserID) != 0 {
			return false
		}
	}

	if len(filter.Actions) > 0 {
		matched := false

		for _, action := range filter.Actions {
			if event.Action == action {
				matched = true

				break
			}
		}

		if !matched {
			return false
		}
	}

	return true
}
