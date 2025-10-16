package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

// Server implements the MCP (Model Context Protocol) server
type Server struct {
	config Config
	plugin *Plugin

	// Resource and tool handlers
	resources *ResourceRegistry
	tools     *ToolRegistry

	// Security and authorization
	security *SecurityLayer

	mu      sync.Mutex
	running bool
}

// NewServer creates a new MCP server
func NewServer(config Config, plugin *Plugin) (*Server, error) {
	s := &Server{
		config:    config,
		plugin:    plugin,
		resources: NewResourceRegistry(),
		tools:     NewToolRegistry(),
	}

	// Initialize security layer
	s.security = NewSecurityLayer(config)

	// Register built-in resources
	s.registerResources()

	// Register built-in tools
	s.registerTools()

	return s, nil
}

// Start starts the MCP server
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("MCP server already running")
	}
	s.running = true
	s.mu.Unlock()

	switch s.config.Transport {
	case TransportStdio:
		return s.runStdioServer(ctx)
	case TransportHTTP:
		return s.runHTTPServer(ctx)
	default:
		return fmt.Errorf("unsupported transport: %s", s.config.Transport)
	}
}

// Stop stops the MCP server
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.running = false
	return nil
}

// runStdioServer runs MCP over stdio (for local CLI)
func (s *Server) runStdioServer(ctx context.Context) error {
	// MCP protocol: JSON-RPC over stdio
	// Read from stdin, write to stdout

	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var req MCPRequest
		if err := decoder.Decode(&req); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("failed to decode request: %w", err)
		}

		// Handle request
		resp := s.handleRequest(ctx, &req)

		// Send response
		if err := encoder.Encode(resp); err != nil {
			return fmt.Errorf("failed to encode response: %w", err)
		}
	}
}

// runHTTPServer runs MCP over HTTP (for remote access)
func (s *Server) runHTTPServer(ctx context.Context) error {
	// TODO: Implement HTTP transport
	// Listen on configured port, handle POST /api/mcp
	return fmt.Errorf("HTTP transport not yet implemented")
}

// handleRequest processes an MCP request
func (s *Server) handleRequest(ctx context.Context, req *MCPRequest) *MCPResponse {
	// TODO: Add authorization check
	// TODO: Add rate limiting

	switch req.Method {
	case "resources/list":
		return s.handleResourcesList(ctx, req)
	case "resources/read":
		return s.handleResourcesRead(ctx, req)
	case "tools/list":
		return s.handleToolsList(ctx, req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	case "initialize":
		return s.handleInitialize(ctx, req)
	case "ping":
		return s.handlePing(ctx, req)
	default:
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: fmt.Sprintf("method not found: %s", req.Method),
			},
		}
	}
}

// handleInitialize handles MCP initialization
func (s *Server) handleInitialize(ctx context.Context, req *MCPRequest) *MCPResponse {
	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"protocolVersion": "0.1.0",
			"capabilities": map[string]interface{}{
				"resources": map[string]bool{
					"subscribe":   false, // No streaming support yet
					"listChanged": false,
				},
				"tools": map[string]bool{},
			},
			"serverInfo": map[string]string{
				"name":    "authsome-mcp",
				"version": "0.1.0",
			},
		},
	}
}

// handlePing handles ping requests
func (s *Server) handlePing(ctx context.Context, req *MCPRequest) *MCPResponse {
	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]string{"status": "ok"},
	}
}

// handleResourcesList lists available resources
func (s *Server) handleResourcesList(ctx context.Context, req *MCPRequest) *MCPResponse {
	resources := s.resources.List()

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"resources": resources,
		},
	}
}

// handleResourcesRead reads a specific resource
func (s *Server) handleResourcesRead(ctx context.Context, req *MCPRequest) *MCPResponse {
	// Extract URI from params
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "invalid params",
			},
		}
	}

	uri, ok := params["uri"].(string)
	if !ok {
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "missing uri parameter",
			},
		}
	}

	// Get resource content
	content, err := s.resources.Read(ctx, uri, s.plugin)
	if err != nil {
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32000,
				Message: fmt.Sprintf("failed to read resource: %v", err),
			},
		}
	}

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"uri":      uri,
					"mimeType": "application/json",
					"text":     content,
				},
			},
		},
	}
}

// handleToolsList lists available tools
func (s *Server) handleToolsList(ctx context.Context, req *MCPRequest) *MCPResponse {
	tools := s.tools.List(s.config.Mode)

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}
}

// handleToolsCall executes a tool
func (s *Server) handleToolsCall(ctx context.Context, req *MCPRequest) *MCPResponse {
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "invalid params",
			},
		}
	}

	name, ok := params["name"].(string)
	if !ok {
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "missing tool name",
			},
		}
	}

	arguments, _ := params["arguments"].(map[string]interface{})

	// Execute tool
	result, err := s.tools.Execute(ctx, name, arguments, s.plugin)
	if err != nil {
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32000,
				Message: fmt.Sprintf("tool execution failed: %v", err),
			},
		}
	}

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": result,
				},
			},
		},
	}
}

// registerResources registers built-in resources
func (s *Server) registerResources() {
	s.resources.Register("authsome://config", &ConfigResource{})
	s.resources.Register("authsome://schema", &SchemaResource{})
	s.resources.Register("authsome://routes", &RoutesResource{})
	// TODO: Add more resources
}

// registerTools registers built-in tools
func (s *Server) registerTools() {
	s.tools.Register("query_user", &QueryUserTool{})
	s.tools.Register("check_permission", &CheckPermissionTool{})
	// TODO: Add more tools
}

// MCPRequest represents an MCP JSON-RPC request
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// MCPResponse represents an MCP JSON-RPC response
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an MCP error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
