package examples

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
)

// =============================================================================
// EXAMPLE: WebSocket Streaming Handler for Fiber Framework
// =============================================================================
// This is a reference implementation showing how to integrate audit streaming
// with the Fiber web framework. External projects can adapt this for their
// preferred framework (Gin, Echo, Chi, net/http, etc.)

// FiberStreamHandler handles WebSocket connections using Fiber framework
type FiberStreamHandler struct {
	streamService *audit.StreamService
}

// NewFiberStreamHandler creates a new Fiber stream handler
func NewFiberStreamHandler(streamService *audit.StreamService) *FiberStreamHandler {
	return &FiberStreamHandler{
		streamService: streamService,
	}
}

// HandleWebSocket handles WebSocket connections
func (h *FiberStreamHandler) HandleWebSocket(c *websocket.Conn) {
	// Parse filter from initial message
	filter, err := h.parseFilter(c)
	if err != nil {
		h.sendError(c, fmt.Errorf("invalid filter: %w", err))
		return
	}

	// Subscribe to event stream
	ctx := context.Background()
	events, clientID, err := h.streamService.Subscribe(ctx, filter)
	if err != nil {
		h.sendError(c, fmt.Errorf("subscription failed: %w", err))
		return
	}

	defer h.streamService.Unsubscribe(clientID)

	// Send subscription confirmation
	h.sendMessage(c, map[string]interface{}{
		"type":     "subscribed",
		"clientId": clientID,
	})

	// Handle bidirectional communication
	done := make(chan struct{})

	// Read messages from client (heartbeats, close)
	go func() {
		defer close(done)
		for {
			var msg map[string]interface{}
			if err := c.ReadJSON(&msg); err != nil {
				return
			}

			msgType, _ := msg["type"].(string)
			switch msgType {
			case "ping":
				h.streamService.UpdateHeartbeat(clientID)
				h.sendMessage(c, map[string]interface{}{"type": "pong"})
			case "close":
				return
			}
		}
	}()

	// Stream events to client
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			h.sendEvent(c, event)

		case <-ticker.C:
			h.streamService.UpdateHeartbeat(clientID)

		case <-done:
			return
		}
	}
}

// parseFilter parses stream filter from WebSocket message
func (h *FiberStreamHandler) parseFilter(c *websocket.Conn) (*audit.StreamFilter, error) {
	var req struct {
		AppID      *string  `json:"appId"`
		UserID     *string  `json:"userId"`
		Actions    []string `json:"actions"`
		BufferSize int      `json:"bufferSize"`
	}

	if err := c.ReadJSON(&req); err != nil {
		return nil, err
	}

	filter := &audit.StreamFilter{
		BufferSize: 100,
	}

	if req.AppID != nil {
		appID, err := xid.FromString(*req.AppID)
		if err != nil {
			return nil, fmt.Errorf("invalid appId: %w", err)
		}
		filter.AppID = &appID
	}

	if req.UserID != nil {
		userID, err := xid.FromString(*req.UserID)
		if err != nil {
			return nil, fmt.Errorf("invalid userId: %w", err)
		}
		filter.UserID = &userID
	}

	filter.Actions = req.Actions

	if req.BufferSize > 0 {
		filter.BufferSize = req.BufferSize
	}

	return filter, nil
}

// sendEvent sends an audit event to the client
func (h *FiberStreamHandler) sendEvent(c *websocket.Conn, event *audit.Event) error {
	return c.WriteJSON(map[string]interface{}{
		"type":  "event",
		"event": event,
	})
}

// sendMessage sends a JSON message to the client
func (h *FiberStreamHandler) sendMessage(c *websocket.Conn, msg interface{}) error {
	return c.WriteJSON(msg)
}

// sendError sends an error and closes the connection
func (h *FiberStreamHandler) sendError(c *websocket.Conn, err error) {
	c.WriteJSON(map[string]interface{}{
		"type":  "error",
		"error": err.Error(),
	})
	c.Close()
}

// RegisterRoutes registers WebSocket routes with Fiber app
func (h *FiberStreamHandler) RegisterRoutes(app *fiber.App) {
	// WebSocket endpoint
	app.Get("/api/audit/stream", websocket.New(h.HandleWebSocket))

	// Stats endpoint
	app.Get("/api/audit/stream/stats", func(c *fiber.Ctx) error {
		stats := h.streamService.Stats()
		return c.JSON(stats)
	})
}

// =============================================================================
// EXAMPLE: Server-Sent Events (SSE) Handler
// =============================================================================

// HandleSSE handles Server-Sent Events for browsers without WebSocket support
func (h *FiberStreamHandler) HandleSSE(c *fiber.Ctx) error {
	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	// Parse filter from query params
	filter := &audit.StreamFilter{
		BufferSize: 100,
	}

	if appIDStr := c.Query("appId"); appIDStr != "" {
		appID, err := xid.FromString(appIDStr)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid appId")
		}
		filter.AppID = &appID
	}

	// Subscribe to stream
	ctx := c.Context()
	events, clientID, err := h.streamService.Subscribe(ctx, filter)
	if err != nil {
		return err
	}
	defer h.streamService.Unsubscribe(clientID)

	// Stream events
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case event, ok := <-events:
				if !ok {
					return
				}

				// Send event as SSE
				data, _ := json.Marshal(event)
				fmt.Fprintf(w, "event: audit\ndata: %s\n\n", data)
				w.Flush()

			case <-ticker.C:
				// Heartbeat
				fmt.Fprintf(w, ": heartbeat\n\n")
				w.Flush()

			case <-ctx.Done():
				return
			}
		}
	})

	return nil
}
