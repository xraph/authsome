package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// AlertRepository defines the interface for alert operations
type AlertRepository interface {
	// Alert config operations
	CreateAlertConfig(ctx context.Context, config *core.AlertConfig) error
	GetAlertConfig(ctx context.Context, id xid.ID) (*core.AlertConfig, error)
	GetAlertConfigByType(ctx context.Context, orgID xid.ID, alertType core.AlertType) (*core.AlertConfig, error)
	ListAlertConfigs(ctx context.Context, orgID xid.ID) ([]*core.AlertConfig, error)
	UpdateAlertConfig(ctx context.Context, config *core.AlertConfig) error
	DeleteAlertConfig(ctx context.Context, id xid.ID) error

	// Alert operations
	CreateAlert(ctx context.Context, alert *core.Alert) error
	GetAlert(ctx context.Context, id xid.ID) (*core.Alert, error)
	ListAlerts(ctx context.Context, orgID xid.ID, status *core.AlertStatus, page, pageSize int) ([]*core.Alert, int, error)
	ListPendingAlerts(ctx context.Context) ([]*core.Alert, error)
	UpdateAlert(ctx context.Context, alert *core.Alert) error
	DeleteAlert(ctx context.Context, id xid.ID) error
	CountAlertsByConfigToday(ctx context.Context, configID xid.ID) (int, error)
	GetLastAlertByConfig(ctx context.Context, configID xid.ID) (*core.Alert, error)

	// Alert template operations
	CreateAlertTemplate(ctx context.Context, template *core.AlertTemplate) error
	GetAlertTemplate(ctx context.Context, appID xid.ID, alertType core.AlertType, channel core.AlertChannel) (*core.AlertTemplate, error)
	ListAlertTemplates(ctx context.Context, appID xid.ID) ([]*core.AlertTemplate, error)
	UpdateAlertTemplate(ctx context.Context, template *core.AlertTemplate) error
	DeleteAlertTemplate(ctx context.Context, id xid.ID) error
}

// alertRepository implements AlertRepository using Bun
type alertRepository struct {
	db *bun.DB
}

// NewAlertRepository creates a new alert repository
func NewAlertRepository(db *bun.DB) AlertRepository {
	return &alertRepository{db: db}
}

// CreateAlertConfig creates a new alert config
func (r *alertRepository) CreateAlertConfig(ctx context.Context, config *core.AlertConfig) error {
	model := alertConfigToSchema(config)
	_, err := r.db.NewInsert().Model(model).Exec(ctx)
	return err
}

// GetAlertConfig returns an alert config by ID
func (r *alertRepository) GetAlertConfig(ctx context.Context, id xid.ID) (*core.AlertConfig, error) {
	var config schema.SubscriptionAlertConfig
	err := r.db.NewSelect().
		Model(&config).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return schemaToAlertConfig(&config), nil
}

// GetAlertConfigByType returns an alert config by type
func (r *alertRepository) GetAlertConfigByType(ctx context.Context, orgID xid.ID, alertType core.AlertType) (*core.AlertConfig, error) {
	var config schema.SubscriptionAlertConfig
	err := r.db.NewSelect().
		Model(&config).
		Where("organization_id = ?", orgID).
		Where("alert_type = ?", string(alertType)).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return schemaToAlertConfig(&config), nil
}

// ListAlertConfigs returns all alert configs for an organization
func (r *alertRepository) ListAlertConfigs(ctx context.Context, orgID xid.ID) ([]*core.AlertConfig, error) {
	var configs []schema.SubscriptionAlertConfig
	err := r.db.NewSelect().
		Model(&configs).
		Where("organization_id = ?", orgID).
		Order("alert_type ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*core.AlertConfig, len(configs))
	for i, c := range configs {
		result[i] = schemaToAlertConfig(&c)
	}
	return result, nil
}

// UpdateAlertConfig updates an alert config
func (r *alertRepository) UpdateAlertConfig(ctx context.Context, config *core.AlertConfig) error {
	model := alertConfigToSchema(config)
	model.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(model).
		WherePK().
		Exec(ctx)
	return err
}

// DeleteAlertConfig deletes an alert config
func (r *alertRepository) DeleteAlertConfig(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.SubscriptionAlertConfig)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// CreateAlert creates a new alert
func (r *alertRepository) CreateAlert(ctx context.Context, alert *core.Alert) error {
	model := alertToSchema(alert)
	_, err := r.db.NewInsert().Model(model).Exec(ctx)
	return err
}

// GetAlert returns an alert by ID
func (r *alertRepository) GetAlert(ctx context.Context, id xid.ID) (*core.Alert, error) {
	var alert schema.SubscriptionAlert
	err := r.db.NewSelect().
		Model(&alert).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return schemaToAlert(&alert), nil
}

// ListAlerts returns alerts for an organization
func (r *alertRepository) ListAlerts(ctx context.Context, orgID xid.ID, status *core.AlertStatus, page, pageSize int) ([]*core.Alert, int, error) {
	var alerts []schema.SubscriptionAlert
	query := r.db.NewSelect().
		Model(&alerts).
		Where("organization_id = ?", orgID)

	if status != nil {
		query = query.Where("status = ?", string(*status))
	}

	count, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	err = query.
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*core.Alert, len(alerts))
	for i, a := range alerts {
		result[i] = schemaToAlert(&a)
	}
	return result, count, nil
}

// ListPendingAlerts returns all pending alerts
func (r *alertRepository) ListPendingAlerts(ctx context.Context) ([]*core.Alert, error) {
	var alerts []schema.SubscriptionAlert
	err := r.db.NewSelect().
		Model(&alerts).
		Where("status = ?", string(core.AlertStatusPending)).
		Order("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*core.Alert, len(alerts))
	for i, a := range alerts {
		result[i] = schemaToAlert(&a)
	}
	return result, nil
}

// UpdateAlert updates an alert
func (r *alertRepository) UpdateAlert(ctx context.Context, alert *core.Alert) error {
	model := alertToSchema(alert)
	model.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(model).
		WherePK().
		Exec(ctx)
	return err
}

// DeleteAlert deletes an alert
func (r *alertRepository) DeleteAlert(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.SubscriptionAlert)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// CountAlertsByConfigToday counts alerts sent today for a config
func (r *alertRepository) CountAlertsByConfigToday(ctx context.Context, configID xid.ID) (int, error) {
	startOfDay := time.Now().Truncate(24 * time.Hour)
	count, err := r.db.NewSelect().
		Model((*schema.SubscriptionAlert)(nil)).
		Where("config_id = ?", configID).
		Where("created_at >= ?", startOfDay).
		Count(ctx)
	return count, err
}

// GetLastAlertByConfig returns the last alert for a config
func (r *alertRepository) GetLastAlertByConfig(ctx context.Context, configID xid.ID) (*core.Alert, error) {
	var alert schema.SubscriptionAlert
	err := r.db.NewSelect().
		Model(&alert).
		Where("config_id = ?", configID).
		Order("created_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return schemaToAlert(&alert), nil
}

// CreateAlertTemplate creates a new alert template
func (r *alertRepository) CreateAlertTemplate(ctx context.Context, template *core.AlertTemplate) error {
	model := alertTemplateToSchema(template)
	_, err := r.db.NewInsert().Model(model).Exec(ctx)
	return err
}

// GetAlertTemplate returns an alert template
func (r *alertRepository) GetAlertTemplate(ctx context.Context, appID xid.ID, alertType core.AlertType, channel core.AlertChannel) (*core.AlertTemplate, error) {
	var template schema.SubscriptionAlertTemplate
	err := r.db.NewSelect().
		Model(&template).
		Where("app_id = ?", appID).
		Where("alert_type = ?", string(alertType)).
		Where("channel = ?", string(channel)).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return schemaToAlertTemplate(&template), nil
}

// ListAlertTemplates returns all alert templates for an app
func (r *alertRepository) ListAlertTemplates(ctx context.Context, appID xid.ID) ([]*core.AlertTemplate, error) {
	var templates []schema.SubscriptionAlertTemplate
	err := r.db.NewSelect().
		Model(&templates).
		Where("app_id = ?", appID).
		Order("alert_type ASC", "channel ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*core.AlertTemplate, len(templates))
	for i, t := range templates {
		result[i] = schemaToAlertTemplate(&t)
	}
	return result, nil
}

// UpdateAlertTemplate updates an alert template
func (r *alertRepository) UpdateAlertTemplate(ctx context.Context, template *core.AlertTemplate) error {
	model := alertTemplateToSchema(template)
	model.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(model).
		WherePK().
		Exec(ctx)
	return err
}

// DeleteAlertTemplate deletes an alert template
func (r *alertRepository) DeleteAlertTemplate(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.SubscriptionAlertTemplate)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// Helper functions

func schemaToAlertConfig(s *schema.SubscriptionAlertConfig) *core.AlertConfig {
	channels := make([]core.AlertChannel, len(s.Channels))
	for i, c := range s.Channels {
		channels[i] = core.AlertChannel(c)
	}

	return &core.AlertConfig{
		ID:               s.ID,
		AppID:            s.AppID,
		OrganizationID:   s.OrganizationID,
		AlertType:        core.AlertType(s.AlertType),
		IsEnabled:        s.IsEnabled,
		ThresholdPercent: s.ThresholdPercent,
		MetricKey:        s.MetricKey,
		DaysBeforeEnd:    s.DaysBeforeEnd,
		Channels:         channels,
		Recipients:       s.Recipients,
		WebhookURL:       s.WebhookURL,
		SlackChannel:     s.SlackChannel,
		MinInterval:      s.MinInterval,
		MaxAlertsPerDay:  s.MaxAlertsPerDay,
		SnoozedUntil:     s.SnoozedUntil,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
}

func alertConfigToSchema(c *core.AlertConfig) *schema.SubscriptionAlertConfig {
	channels := make([]string, len(c.Channels))
	for i, ch := range c.Channels {
		channels[i] = string(ch)
	}

	return &schema.SubscriptionAlertConfig{
		ID:               c.ID,
		AppID:            c.AppID,
		OrganizationID:   c.OrganizationID,
		AlertType:        string(c.AlertType),
		IsEnabled:        c.IsEnabled,
		ThresholdPercent: c.ThresholdPercent,
		MetricKey:        c.MetricKey,
		DaysBeforeEnd:    c.DaysBeforeEnd,
		Channels:         channels,
		Recipients:       c.Recipients,
		WebhookURL:       c.WebhookURL,
		SlackChannel:     c.SlackChannel,
		MinInterval:      c.MinInterval,
		MaxAlertsPerDay:  c.MaxAlertsPerDay,
		SnoozedUntil:     c.SnoozedUntil,
		CreatedAt:        c.CreatedAt,
		UpdatedAt:        c.UpdatedAt,
	}
}

func schemaToAlert(s *schema.SubscriptionAlert) *core.Alert {
	channels := make([]core.AlertChannel, len(s.Channels))
	for i, c := range s.Channels {
		channels[i] = core.AlertChannel(c)
	}

	var deliveryStatus map[string]string
	if s.DeliveryStatus != "" {
		_ = json.Unmarshal([]byte(s.DeliveryStatus), &deliveryStatus)
	}

	var metadata map[string]interface{}
	if s.Metadata != "" {
		_ = json.Unmarshal([]byte(s.Metadata), &metadata)
	}

	return &core.Alert{
		ID:             s.ID,
		AppID:          s.AppID,
		OrganizationID: s.OrganizationID,
		ConfigID:       s.ConfigID,
		Type:           core.AlertType(s.Type),
		Severity:       core.AlertSeverity(s.Severity),
		Status:         core.AlertStatus(s.Status),
		Title:          s.Title,
		Message:        s.Message,
		MetricKey:      s.MetricKey,
		CurrentValue:   s.CurrentValue,
		ThresholdValue: s.ThresholdValue,
		LimitValue:     s.LimitValue,
		SubscriptionID: s.SubscriptionID,
		InvoiceID:      s.InvoiceID,
		Channels:       channels,
		SentAt:         s.SentAt,
		DeliveryStatus: deliveryStatus,
		AcknowledgedAt: s.AcknowledgedAt,
		AcknowledgedBy: s.AcknowledgedBy,
		ResolvedAt:     s.ResolvedAt,
		Resolution:     s.Resolution,
		Metadata:       metadata,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
}

func alertToSchema(a *core.Alert) *schema.SubscriptionAlert {
	channels := make([]string, len(a.Channels))
	for i, ch := range a.Channels {
		channels[i] = string(ch)
	}

	deliveryStatus := ""
	if a.DeliveryStatus != nil {
		deliveryStatusBytes, _ := json.Marshal(a.DeliveryStatus)
		deliveryStatus = string(deliveryStatusBytes)
	}

	metadata := ""
	if a.Metadata != nil {
		metadataBytes, _ := json.Marshal(a.Metadata)
		metadata = string(metadataBytes)
	}

	return &schema.SubscriptionAlert{
		ID:             a.ID,
		AppID:          a.AppID,
		OrganizationID: a.OrganizationID,
		ConfigID:       a.ConfigID,
		Type:           string(a.Type),
		Severity:       string(a.Severity),
		Status:         string(a.Status),
		Title:          a.Title,
		Message:        a.Message,
		MetricKey:      a.MetricKey,
		CurrentValue:   a.CurrentValue,
		ThresholdValue: a.ThresholdValue,
		LimitValue:     a.LimitValue,
		SubscriptionID: a.SubscriptionID,
		InvoiceID:      a.InvoiceID,
		Channels:       channels,
		SentAt:         a.SentAt,
		DeliveryStatus: deliveryStatus,
		AcknowledgedAt: a.AcknowledgedAt,
		AcknowledgedBy: a.AcknowledgedBy,
		ResolvedAt:     a.ResolvedAt,
		Resolution:     a.Resolution,
		Metadata:       metadata,
		CreatedAt:      a.CreatedAt,
		UpdatedAt:      a.UpdatedAt,
	}
}

func schemaToAlertTemplate(s *schema.SubscriptionAlertTemplate) *core.AlertTemplate {
	return &core.AlertTemplate{
		ID:            s.ID,
		AppID:         s.AppID,
		AlertType:     core.AlertType(s.AlertType),
		Channel:       core.AlertChannel(s.Channel),
		Subject:       s.Subject,
		TitleTemplate: s.TitleTemplate,
		BodyTemplate:  s.BodyTemplate,
		IsDefault:     s.IsDefault,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}

func alertTemplateToSchema(t *core.AlertTemplate) *schema.SubscriptionAlertTemplate {
	return &schema.SubscriptionAlertTemplate{
		ID:            t.ID,
		AppID:         t.AppID,
		AlertType:     string(t.AlertType),
		Channel:       string(t.Channel),
		Subject:       t.Subject,
		TitleTemplate: t.TitleTemplate,
		BodyTemplate:  t.BodyTemplate,
		IsDefault:     t.IsDefault,
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
	}
}
