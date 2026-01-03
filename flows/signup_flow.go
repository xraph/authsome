package flows

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
)

// SignupFlow implements the user registration flow
type SignupFlow struct {
	*BaseFlow
	userService    *user.Service
	sessionService *session.Service
}

// NewSignupFlow creates a new signup flow
func NewSignupFlow(userService *user.Service, sessionService *session.Service) *SignupFlow {
	flow := &SignupFlow{
		BaseFlow:       NewBaseFlow("signup", "User Registration Flow"),
		userService:    userService,
		sessionService: sessionService,
	}

	// Add default steps
	flow.addDefaultSteps()

	return flow
}

// addDefaultSteps adds the default signup flow steps
func (f *SignupFlow) addDefaultSteps() {
	// Step 1: Validate input
	f.AddStep(&Step{
		ID:       "validate_input",
		Name:     "Validate Input",
		Type:     "validation",
		Handler:  f.validateInputHandler,
		Required: true,
	})

	// Step 2: Check if user exists
	f.AddStep(&Step{
		ID:       "check_user_exists",
		Name:     "Check User Exists",
		Type:     "validation",
		Handler:  f.checkUserExistsHandler,
		Required: true,
	})

	// Step 3: Create user
	f.AddStep(&Step{
		ID:       "create_user",
		Name:     "Create User",
		Type:     "action",
		Handler:  f.createUserHandler,
		Required: true,
	})

	// Step 4: Create session
	f.AddStep(&Step{
		ID:       "create_session",
		Name:     "Create Session",
		Type:     "action",
		Handler:  f.createSessionHandler,
		Required: true,
	})

	// Step 5: Send welcome email (optional)
	f.AddStep(&Step{
		ID:       "send_welcome_email",
		Name:     "Send Welcome Email",
		Type:     "notification",
		Handler:  f.sendWelcomeEmailHandler,
		Required: false,
	})
}

// validateInputHandler validates the signup form data
func (f *SignupFlow) validateInputHandler(ctx context.Context, step *Step, data map[string]interface{}) (*StepResult, error) {
	email, _ := data["email"].(string)
	password, _ := data["password"].(string)
	name, _ := data["name"].(string)

	if email == "" {
		return &StepResult{
			Success: false,
			Error:   "Email is required",
			Data:    map[string]interface{}{"field": "email"},
		}, nil
	}

	if password == "" {
		return &StepResult{
			Success: false,
			Error:   "Password is required",
			Data:    map[string]interface{}{"field": "password"},
		}, nil
	}

	if len(password) < 8 {
		return &StepResult{
			Success: false,
			Error:   "Password must be at least 8 characters",
			Data:    map[string]interface{}{"field": "password"},
		}, nil
	}

	// Store validated data
	data["validated_email"] = email
	data["validated_password"] = password
	data["validated_name"] = name

	return &StepResult{
		Success: true,
		Data:    map[string]interface{}{"validated": true},
	}, nil
}

// checkUserExistsHandler checks if a user with the given email already exists
func (f *SignupFlow) checkUserExistsHandler(ctx context.Context, step *Step, data map[string]interface{}) (*StepResult, error) {
	email, _ := data["validated_email"].(string)

	existingUser, err := f.userService.FindByEmail(ctx, email)
	if err != nil && err.Error() != "user not found" {
		return &StepResult{
			Success: false,
			Error:   "Failed to check user existence",
			Data:    map[string]interface{}{"error": err.Error()},
		}, err
	}

	if existingUser != nil {
		return &StepResult{
			Success: false,
			Error:   "User with this email already exists",
			Data:    map[string]interface{}{"field": "email"},
		}, nil
	}

	return &StepResult{
		Success: true,
		Data:    map[string]interface{}{"user_exists": false},
	}, nil
}

// createUserHandler creates a new user account
func (f *SignupFlow) createUserHandler(ctx context.Context, step *Step, data map[string]interface{}) (*StepResult, error) {
	email, _ := data["validated_email"].(string)
	password, _ := data["validated_password"].(string)
	name, _ := data["validated_name"].(string)

	createReq := &user.CreateUserRequest{
		Email:    email,
		Password: password,
		Name:     name,
	}

	newUser, err := f.userService.Create(ctx, createReq)
	if err != nil {
		return &StepResult{
			Success: false,
			Error:   "Failed to create user",
			Data:    map[string]interface{}{"error": err.Error()},
		}, err
	}

	// Store created user in data for next steps
	data["created_user"] = newUser
	data["user_id"] = newUser.ID

	return &StepResult{
		Success: true,
		Data: map[string]interface{}{
			"user_id": newUser.ID,
			"email":   newUser.Email,
		},
	}, nil
}

// createSessionHandler creates a session for the new user
func (f *SignupFlow) createSessionHandler(ctx context.Context, step *Step, data map[string]interface{}) (*StepResult, error) {
	userIDStr, _ := data["user_id"].(string)
	if userIDStr == "" {
		return &StepResult{
			Success: false,
			Error:   "User ID not found in data",
		}, nil
	}

	userID, err := xid.FromString(userIDStr)
	if err != nil {
		return &StepResult{
			Success: false,
			Error:   "Invalid user ID format",
		}, nil
	}

	// Extract AppID from context
	appID, _ := contexts.GetAppID(ctx)

	// Extract OrganizationID from context (optional)
	var organizationID *xid.ID
	if orgID, ok := contexts.GetOrganizationID(ctx); ok && !orgID.IsNil() {
		organizationID = &orgID
	}

	// Extract EnvironmentID from context (optional)
	var environmentID *xid.ID
	if envID, ok := contexts.GetEnvironmentID(ctx); ok && !envID.IsNil() {
		environmentID = &envID
	}

	createReq := &session.CreateSessionRequest{
		AppID:          appID,
		EnvironmentID:  environmentID,
		OrganizationID: organizationID,
		UserID:         userID,
	}

	newSession, err := f.sessionService.Create(ctx, createReq)
	if err != nil {
		return &StepResult{
			Success: false,
			Error:   "Failed to create session",
			Data:    map[string]interface{}{"error": err.Error()},
		}, err
	}

	// Store session in data
	data["created_session"] = newSession
	data["session_id"] = newSession.ID

	return &StepResult{
		Success: true,
		Data: map[string]interface{}{
			"session_id": newSession.ID,
			"token":      newSession.Token,
		},
	}, nil
}

// sendWelcomeEmailHandler sends a welcome email to the new user
func (f *SignupFlow) sendWelcomeEmailHandler(ctx context.Context, step *Step, data map[string]interface{}) (*StepResult, error) {
	// This is a placeholder implementation
	// In a real implementation, you would integrate with an email service

	email, _ := data["validated_email"].(string)
	name, _ := data["validated_name"].(string)

	// Log the welcome email (in real implementation, send actual email)
	data["welcome_email_sent"] = true
	data["welcome_email_recipient"] = email

	return &StepResult{
		Success: true,
		Data: map[string]interface{}{
			"email_sent": true,
			"recipient":  email,
			"name":       name,
		},
	}, nil
}

// CreateSignupFlow creates a customizable signup flow with hooks
func CreateSignupFlow(userService *user.Service, sessionService *session.Service) Flow {
	flow := NewSignupFlow(userService, sessionService)

	// Add before hooks
	flow.AddBeforeHook("validate_input", func(ctx context.Context, step *Step, data map[string]interface{}) error {
		// Log signup attempt
		data["signup_started"] = true
		return nil
	})

	flow.AddBeforeHook("create_user", func(ctx context.Context, step *Step, data map[string]interface{}) error {
		// Additional validation or preprocessing
		return nil
	})

	// Add after hooks
	flow.AddAfterHook("create_user", func(ctx context.Context, step *Step, data map[string]interface{}) error {
		// Log user creation
		data["user_created"] = true
		return nil
	})

	flow.AddAfterHook("create_session", func(ctx context.Context, step *Step, data map[string]interface{}) error {
		// Log session creation
		data["session_created"] = true
		return nil
	})

	return flow
}
