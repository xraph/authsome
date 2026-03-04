package bridge

import (
	"context"
	"strings"
	"time"

	log "github.com/xraph/go-utils/log"
)

// ──────────────────────────────────────────────────
// Standalone stubs — minimal fallbacks for standalone mode
// ──────────────────────────────────────────────────

// SlogChronicle is a standalone Chronicle stub that logs audit events to slog.
type SlogChronicle struct {
	Logger log.Logger
}

// NewSlogChronicle creates a Chronicle that logs to the given logger.
func NewSlogChronicle(logger log.Logger) *SlogChronicle {
	return &SlogChronicle{Logger: logger}
}

// Record implements Chronicle by logging the event.
func (c *SlogChronicle) Record(_ context.Context, event *AuditEvent) error {
	c.Logger.Info("authsome audit",
		log.String("action", event.Action),
		log.String("resource", event.Resource),
		log.String("resource_id", event.ResourceID),
		log.String("actor_id", event.ActorID),
		log.String("tenant", event.Tenant),
		log.String("outcome", event.Outcome),
		log.String("severity", event.Severity),
	)
	return nil
}

// NoopAuthorizer is a standalone Authorizer stub that always allows.
type NoopAuthorizer struct{}

// NewNoopAuthorizer creates an Authorizer that always returns allowed.
func NewNoopAuthorizer() *NoopAuthorizer { return &NoopAuthorizer{} }

// Check implements Authorizer.
func (a *NoopAuthorizer) Check(_ context.Context, _ *AuthzRequest) (*AuthzResult, error) {
	return &AuthzResult{Allowed: true, Reason: "standalone mode: always allowed"}, nil
}

// NoopKeyManager is a standalone KeyManager stub that returns errors.
type NoopKeyManager struct{}

// NewNoopKeyManager creates a KeyManager that does not manage keys.
func NewNoopKeyManager() *NoopKeyManager { return &NoopKeyManager{} }

func (k *NoopKeyManager) CreateKey(_ context.Context, _ *CreateKeyInput) (*KeyResult, error) {
	return nil, ErrKeyManagerNotAvailable
}

func (k *NoopKeyManager) ValidateKey(_ context.Context, _ string) (*ValidatedKey, error) {
	return nil, ErrKeyManagerNotAvailable
}

func (k *NoopKeyManager) RevokeKey(_ context.Context, _ string) error {
	return ErrKeyManagerNotAvailable
}

// NoopRelay is a standalone EventRelay stub that silently drops events.
type NoopRelay struct {
	Logger log.Logger
}

// NewNoopRelay creates an EventRelay that logs events at debug level and drops them.
func NewNoopRelay(logger log.Logger) *NoopRelay {
	return &NoopRelay{Logger: logger}
}

// Send implements EventRelay.
func (r *NoopRelay) Send(_ context.Context, event *WebhookEvent) error {
	r.Logger.Debug("authsome relay (noop)",
		log.String("type", event.Type),
		log.String("tenant_id", event.TenantID),
	)
	return nil
}

// RegisterEventTypes implements EventRelay.
func (r *NoopRelay) RegisterEventTypes(_ context.Context, _ []WebhookDefinition) error {
	return nil
}

// NoopVault is a standalone Vault stub that returns ErrVaultNotAvailable.
type NoopVault struct{}

// NewNoopVault creates a Vault that returns errors for all operations.
func NewNoopVault() *NoopVault { return &NoopVault{} }

func (v *NoopVault) GetSecret(_ context.Context, _ string) ([]byte, error) {
	return nil, ErrVaultNotAvailable
}

func (v *NoopVault) SetSecret(_ context.Context, _ string, _ []byte) error {
	return ErrVaultNotAvailable
}

func (v *NoopVault) IsFeatureEnabled(_ context.Context, _ string) bool {
	return false
}

func (v *NoopVault) GetConfig(_ context.Context, _ string) (string, error) {
	return "", ErrVaultNotAvailable
}

// NoopDispatcher is a standalone Dispatcher stub that drops all jobs.
type NoopDispatcher struct{}

// NewNoopDispatcher creates a Dispatcher that silently drops jobs.
func NewNoopDispatcher() *NoopDispatcher { return &NoopDispatcher{} }

func (d *NoopDispatcher) Enqueue(_ context.Context, _ string, _ []byte) error {
	return ErrDispatchNotAvailable
}

func (d *NoopDispatcher) Schedule(_ context.Context, _ string, _ []byte, _ time.Time) error {
	return ErrDispatchNotAvailable
}

// NoopLedger is a standalone Ledger stub that returns not-available errors.
type NoopLedger struct{}

// NewNoopLedger creates a Ledger that returns errors for all operations.
func NewNoopLedger() *NoopLedger { return &NoopLedger{} }

func (l *NoopLedger) RecordUsage(_ context.Context, _ string, _ int64) error {
	return ErrLedgerNotAvailable
}

func (l *NoopLedger) CheckEntitlement(_ context.Context, _ string) (bool, error) {
	return true, nil // Default: allow (fail-open)
}

// NoopMailer is a standalone Mailer stub that logs and drops emails.
type NoopMailer struct {
	Logger log.Logger
}

// NewNoopMailer creates a Mailer that logs emails at debug level and drops them.
func NewNoopMailer(logger log.Logger) *NoopMailer {
	return &NoopMailer{Logger: logger}
}

// SendEmail implements Mailer.
func (m *NoopMailer) SendEmail(_ context.Context, msg *EmailMessage) error {
	m.Logger.Debug("authsome mailer (noop)",
		log.String("to", strings.Join(msg.To, ",")),
		log.String("subject", msg.Subject),
	)
	return nil
}

// NoopSMSSender is a standalone SMS stub that returns ErrSMSNotAvailable.
type NoopSMSSender struct{}

// NewNoopSMSSender creates an SMSSender that returns errors for all operations.
func NewNoopSMSSender() *NoopSMSSender { return &NoopSMSSender{} }

// SendSMS implements SMSSender.
func (s *NoopSMSSender) SendSMS(_ context.Context, _ *SMSMessage) error {
	return ErrSMSNotAvailable
}

// NoopHerald is a standalone Herald stub that silently drops notifications.
type NoopHerald struct {
	Logger log.Logger
}

// NewNoopHerald creates a Herald that logs notifications at debug level and drops them.
func NewNoopHerald(logger log.Logger) *NoopHerald {
	return &NoopHerald{Logger: logger}
}

// Send implements Herald.
func (h *NoopHerald) Send(_ context.Context, req *HeraldSendRequest) error {
	h.Logger.Debug("authsome herald (noop)",
		log.String("template", req.Template),
		log.String("channel", req.Channel),
	)
	return nil
}

// Notify implements Herald.
func (h *NoopHerald) Notify(_ context.Context, req *HeraldNotifyRequest) error {
	h.Logger.Debug("authsome herald (noop)",
		log.String("template", req.Template),
	)
	return nil
}
