package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
)

// AuditServiceAdapter adapts AuthSome's audit service to compliance service expectations.
type AuditServiceAdapter struct {
	svc *audit.Service
}

// NewAuditServiceAdapter creates a new audit service adapter.
func NewAuditServiceAdapter(svc *audit.Service) *AuditServiceAdapter {
	return &AuditServiceAdapter{svc: svc}
}

// LogEvent logs a compliance audit event.
func (a *AuditServiceAdapter) LogEvent(ctx context.Context, event *AuditEvent) error {
	if a.svc == nil {
		return errs.InternalServerErrorWithMessage("audit service not available")
	}

	// Convert metadata map to JSON string
	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	// Log to core audit service
	// Note: core audit.Service doesn't have app scoping yet, so we include it in metadata
	return a.svc.Log(
		ctx,
		nil, // userID - not available in AuditEvent
		event.Action,
		event.ResourceID,
		"", // IP address - not in compliance AuditEvent
		"", // User agent - not in compliance AuditEvent
		string(metadataJSON),
	)
}

// GetOldestLog retrieves the oldest audit log for data retention checks.
func (a *AuditServiceAdapter) GetOldestLog(ctx context.Context, appID string) (*AuditLog, error) {
	if a.svc == nil {
		return nil, errs.InternalServerErrorWithMessage("audit service not available")
	}

	// Query oldest audit event
	// AuthSome's audit service doesn't have app filtering yet
	// TODO: Update when app scoping is fully integrated
	filter := &audit.ListEventsFilter{
		PaginationParams: pagination.PaginationParams{Limit: 1, Offset: 0},
	}

	eventsResp, err := a.svc.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	if len(eventsResp.Data) == 0 {
		return nil, nil
	}

	// Convert to compliance AuditLog
	return &AuditLog{
		CreatedAt: eventsResp.Data[0].CreatedAt,
	}, nil
}

// UserServiceAdapter adapts AuthSome's user service to compliance service expectations.
type UserServiceAdapter struct {
	svc user.ServiceInterface // Use interface to support decorated services
}

// NewUserServiceAdapter creates a new user service adapter.
func NewUserServiceAdapter(svc user.ServiceInterface) *UserServiceAdapter {
	return &UserServiceAdapter{svc: svc}
}

// ListByApp retrieves all users in an app.
func (a *UserServiceAdapter) ListByApp(ctx context.Context, appID string) ([]*User, error) {
	if a.svc == nil {
		return nil, errs.InternalServerErrorWithMessage("user service not available")
	}

	// TODO: Update when multiapp plugin provides app-scoped user listing
	// For now, return empty list
	return []*User{}, nil
}

// GetMFAStatus checks if a user has MFA enabled.
func (a *UserServiceAdapter) GetMFAStatus(ctx context.Context, userID string) (bool, error) {
	if a.svc == nil {
		return false, errs.InternalServerErrorWithMessage("user service not available")
	}

	// Parse user ID
	uid, err := xid.FromString(userID)
	if err != nil {
		return false, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get user
	u, err := a.svc.FindByID(ctx, uid)
	if err != nil {
		return false, err
	}

	// Check if MFA is enabled (requires 2FA plugin integration)
	// For now, return false
	// TODO: Check u.MFAEnabled or similar field when available
	_ = u

	return false, nil
}

// AppServiceAdapter adapts the app service (from multiapp plugin).
type AppServiceAdapter struct {
	svc any // Will be app.Service when multiapp plugin is loaded
}

// NewAppServiceAdapter creates a new app service adapter.
func NewAppServiceAdapter(svc any) *AppServiceAdapter {
	return &AppServiceAdapter{svc: svc}
}

// Get retrieves an app by ID.
func (a *AppServiceAdapter) Get(ctx context.Context, id string) (*App, error) {
	if a.svc == nil {
		// Multiapp plugin not loaded - return default app
		return &App{
			ID:   "platform",
			Name: "Platform",
		}, nil
	}

	// TODO: Cast to actual app service and call Get
	// This requires the multiapp plugin to be implemented
	return &App{
		ID:   id,
		Name: "App",
	}, nil
}

// EmailServiceAdapter adapts the notification service for email sending.
type EmailServiceAdapter struct {
	svc *notification.Service
}

// NewEmailServiceAdapter creates a new email service adapter.
func NewEmailServiceAdapter(svc *notification.Service) *EmailServiceAdapter {
	return &EmailServiceAdapter{svc: svc}
}

// SendEmail sends an email using the notification service.
func (a *EmailServiceAdapter) SendEmail(ctx context.Context, email *Email) error {
	if a.svc == nil {
		// Email service not available - log and continue
		// Don't block compliance operations if email fails
		return nil
	}

	// TODO: Use notification service to send email
	// The notification service needs to expose email sending capability
	// For now, this is a no-op
	return nil
}

// SendCompliance sends a compliance-related email (convenience method).
func (a *EmailServiceAdapter) SendCompliance(ctx context.Context, to []string, subject, body string) error {
	// Join recipients into a comma-separated string
	toStr := ""
	if len(to) > 0 {
		toStr = to[0]

		var toStrSb182 strings.Builder
		for i := 1; i < len(to); i++ {
			toStrSb182.WriteString(", " + to[i])
		}

		toStr += toStrSb182.String()
	}

	return a.SendEmail(ctx, &Email{
		To:      toStr,
		Subject: subject,
		Body:    body,
	})
}

// SendViolationAlert sends an alert about a compliance violation.
func (a *EmailServiceAdapter) SendViolationAlert(ctx context.Context, violation *ComplianceViolation, recipients []string) error {
	if a.svc == nil {
		return nil
	}

	subject := "Compliance Violation: " + violation.ViolationType
	body := fmt.Sprintf(`
A compliance violation has been detected:

Type: %s
Description: %s
Severity: %s
Time: %s

Please review and take action.
`, violation.ViolationType, violation.Description, violation.Severity, violation.CreatedAt.Format(time.RFC3339))

	return a.SendCompliance(ctx, recipients, subject, body)
}

// SendCheckFailure sends an alert about a failed compliance check.
func (a *EmailServiceAdapter) SendCheckFailure(ctx context.Context, check *ComplianceCheck, recipients []string) error {
	if a.svc == nil {
		return nil
	}

	subject := "Compliance Check Failed: " + check.CheckType
	body := fmt.Sprintf(`
A compliance check has failed:

Check Type: %s
Status: %s
Time: %s

Please investigate and remediate.
`, check.CheckType, check.Status, check.CreatedAt.Format(time.RFC3339))

	return a.SendCompliance(ctx, recipients, subject, body)
}

// Note: Types like AuditLog, User, Organization, AuditEvent, and Email
// are defined in service.go as helper types for the service interfaces
