package engine

import (
	"time"

	"github.com/google/cel-go/cel"
	"github.com/rs/xid"
)

// CompiledPolicy represents a policy compiled to executable CEL bytecode
type CompiledPolicy struct {
	// Policy metadata
	PolicyID    xid.ID
	OrgID       xid.ID
	NamespaceID xid.ID
	Name        string
	Description string
	
	// Compiled CEL program
	Program cel.Program
	AST     *cel.Ast
	
	// Indexing keys for fast lookup
	ResourceType string
	Actions      []string
	Priority     int
	
	// Metadata
	Version    int
	CompiledAt time.Time
	
	// Performance tracking
	EvaluationCount int64
	AvgLatencyMs    float64
}

// EvaluationContext contains all data available to policy expressions
type EvaluationContext struct {
	// Principal (user making the request)
	Principal map[string]interface{} `json:"principal"`
	
	// Resource being accessed
	Resource map[string]interface{} `json:"resource"`
	
	// Request context (IP, time, method, etc.)
	Request map[string]interface{} `json:"request"`
	
	// Action being performed
	Action string `json:"action"`
}

// Decision represents the result of an authorization evaluation
type Decision struct {
	// Allowed indicates if access is granted
	Allowed bool `json:"allowed"`
	
	// MatchedPolicies lists policies that allowed access
	MatchedPolicies []string `json:"matchedPolicies,omitempty"`
	
	// EvaluatedPolicies is the total number of policies checked
	EvaluatedPolicies int `json:"evaluatedPolicies"`
	
	// EvaluationTime is how long evaluation took
	EvaluationTime time.Duration `json:"evaluationTime"`
	
	// CacheHit indicates if result came from cache
	CacheHit bool `json:"cacheHit"`
	
	// Error if evaluation failed
	Error string `json:"error,omitempty"`
}

// IndexKey represents a multi-dimensional index key for fast policy lookup
type IndexKey struct {
	OrgID        string
	ResourceType string
	Action       string
}

// String returns the string representation of the index key
func (k IndexKey) String() string {
	return k.OrgID + ":" + k.ResourceType + ":" + k.Action
}

// EvaluationStats tracks performance metrics for policies
type EvaluationStats struct {
	PolicyID         string
	EvaluationCount  int64
	TotalLatencyMs   float64
	AvgLatencyMs     float64
	P50LatencyMs     float64
	P99LatencyMs     float64
	AllowCount       int64
	DenyCount        int64
	ErrorCount       int64
	LastEvaluated    time.Time
}

