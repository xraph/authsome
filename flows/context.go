package flows

import (
	"context"
	"sync"
	"time"
)

// FlowContext provides execution context for flows with state management
type FlowContext struct {
	ctx         context.Context
	flowID      string
	sessionID   string
	userID      string
	orgID       string
	data        map[string]interface{}
	metadata    map[string]interface{}
	startTime   time.Time
	currentStep string
	errors      []string
	mutex       sync.RWMutex
}

// NewFlowContext creates a new flow context
func NewFlowContext(ctx context.Context, flowID string) *FlowContext {
	return &FlowContext{
		ctx:       ctx,
		flowID:    flowID,
		data:      make(map[string]interface{}),
		metadata:  make(map[string]interface{}),
		startTime: time.Now(),
		errors:    make([]string, 0),
	}
}

// Context returns the underlying context.Context
func (fc *FlowContext) Context() context.Context {
	return fc.ctx
}

// FlowID returns the flow ID
func (fc *FlowContext) FlowID() string {
	return fc.flowID
}

// SessionID returns the session ID
func (fc *FlowContext) SessionID() string {
	fc.mutex.RLock()
	defer fc.mutex.RUnlock()
	return fc.sessionID
}

// SetSessionID sets the session ID
func (fc *FlowContext) SetSessionID(sessionID string) {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()
	fc.sessionID = sessionID
}

// UserID returns the user ID
func (fc *FlowContext) UserID() string {
	fc.mutex.RLock()
	defer fc.mutex.RUnlock()
	return fc.userID
}

// SetUserID sets the user ID
func (fc *FlowContext) SetUserID(userID string) {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()
	fc.userID = userID
}

// OrgID returns the organization ID
func (fc *FlowContext) OrgID() string {
	fc.mutex.RLock()
	defer fc.mutex.RUnlock()
	return fc.orgID
}

// SetOrgID sets the organization ID
func (fc *FlowContext) SetOrgID(orgID string) {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()
	fc.orgID = orgID
}

// Get retrieves a value from the flow data
func (fc *FlowContext) Get(key string) (interface{}, bool) {
	fc.mutex.RLock()
	defer fc.mutex.RUnlock()
	value, exists := fc.data[key]
	return value, exists
}

// Set stores a value in the flow data
func (fc *FlowContext) Set(key string, value interface{}) {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()
	fc.data[key] = value
}

// GetString retrieves a string value from the flow data
func (fc *FlowContext) GetString(key string) string {
	if value, exists := fc.Get(key); exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// GetBool retrieves a boolean value from the flow data
func (fc *FlowContext) GetBool(key string) bool {
	if value, exists := fc.Get(key); exists {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return false
}

// GetInt retrieves an integer value from the flow data
func (fc *FlowContext) GetInt(key string) int {
	if value, exists := fc.Get(key); exists {
		if i, ok := value.(int); ok {
			return i
		}
	}
	return 0
}

// GetData returns a copy of all flow data
func (fc *FlowContext) GetData() map[string]interface{} {
	fc.mutex.RLock()
	defer fc.mutex.RUnlock()

	data := make(map[string]interface{})
	for k, v := range fc.data {
		data[k] = v
	}
	return data
}

// SetData replaces all flow data
func (fc *FlowContext) SetData(data map[string]interface{}) {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()
	fc.data = make(map[string]interface{})
	for k, v := range data {
		fc.data[k] = v
	}
}

// MergeData merges new data into existing flow data
func (fc *FlowContext) MergeData(data map[string]interface{}) {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()
	for k, v := range data {
		fc.data[k] = v
	}
}

// GetMetadata retrieves a metadata value
func (fc *FlowContext) GetMetadata(key string) (interface{}, bool) {
	fc.mutex.RLock()
	defer fc.mutex.RUnlock()
	value, exists := fc.metadata[key]
	return value, exists
}

// SetMetadata stores a metadata value
func (fc *FlowContext) SetMetadata(key string, value interface{}) {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()
	fc.metadata[key] = value
}

// CurrentStep returns the current step ID
func (fc *FlowContext) CurrentStep() string {
	fc.mutex.RLock()
	defer fc.mutex.RUnlock()
	return fc.currentStep
}

// SetCurrentStep sets the current step ID
func (fc *FlowContext) SetCurrentStep(stepID string) {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()
	fc.currentStep = stepID
}

// AddError adds an error to the context
func (fc *FlowContext) AddError(err string) {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()
	fc.errors = append(fc.errors, err)
}

// GetErrors returns all errors
func (fc *FlowContext) GetErrors() []string {
	fc.mutex.RLock()
	defer fc.mutex.RUnlock()
	errors := make([]string, len(fc.errors))
	copy(errors, fc.errors)
	return errors
}

// HasErrors returns true if there are any errors
func (fc *FlowContext) HasErrors() bool {
	fc.mutex.RLock()
	defer fc.mutex.RUnlock()
	return len(fc.errors) > 0
}

// ClearErrors removes all errors
func (fc *FlowContext) ClearErrors() {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()
	fc.errors = make([]string, 0)
}

// StartTime returns when the flow context was created
func (fc *FlowContext) StartTime() time.Time {
	return fc.startTime
}

// Duration returns how long the flow has been running
func (fc *FlowContext) Duration() time.Duration {
	return time.Since(fc.startTime)
}

// Clone creates a copy of the flow context
func (fc *FlowContext) Clone() *FlowContext {
	fc.mutex.RLock()
	defer fc.mutex.RUnlock()

	clone := &FlowContext{
		ctx:         fc.ctx,
		flowID:      fc.flowID,
		sessionID:   fc.sessionID,
		userID:      fc.userID,
		orgID:       fc.orgID,
		data:        make(map[string]interface{}),
		metadata:    make(map[string]interface{}),
		startTime:   fc.startTime,
		currentStep: fc.currentStep,
		errors:      make([]string, len(fc.errors)),
	}

	// Deep copy data
	for k, v := range fc.data {
		clone.data[k] = v
	}

	// Deep copy metadata
	for k, v := range fc.metadata {
		clone.metadata[k] = v
	}

	// Copy errors
	copy(clone.errors, fc.errors)

	return clone
}
