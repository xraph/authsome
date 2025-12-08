// Package migrations provides migration utilities for the subscription plugin.
package migrations

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/subscription/schema"
	"github.com/xraph/forge"
)

// FeaturesMigration handles migration of existing PlanFeature data to the new Feature system
type FeaturesMigration struct {
	db     *bun.DB
	logger forge.Logger
}

// NewFeaturesMigration creates a new features migration utility
func NewFeaturesMigration(db *bun.DB, logger forge.Logger) *FeaturesMigration {
	return &FeaturesMigration{
		db:     db,
		logger: logger,
	}
}

// MigrateResult contains the results of a migration operation
type MigrateResult struct {
	FeaturesCreated int      `json:"featuresCreated"`
	LinksCreated    int      `json:"linksCreated"`
	Errors          []string `json:"errors,omitempty"`
}

// MigrateExistingFeatures migrates existing PlanFeature entries to the new Feature system
// This is a non-destructive operation that creates new Feature entities and links them to plans
func (m *FeaturesMigration) MigrateExistingFeatures(ctx context.Context, appID xid.ID) (*MigrateResult, error) {
	result := &MigrateResult{
		Errors: make([]string, 0),
	}

	m.logger.Info("starting feature migration", forge.F("appId", appID.String()))

	// Start transaction
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Step 1: Get all unique feature keys from existing PlanFeature entries
	type featureKeyInfo struct {
		Key         string `bun:"key"`
		Name        string `bun:"name"`
		Description string `bun:"description"`
		Type        string `bun:"type"`
	}

	var uniqueFeatures []featureKeyInfo
	err = tx.NewSelect().
		Model((*schema.SubscriptionPlanFeature)(nil)).
		ColumnExpr("DISTINCT spf.key, spf.name, spf.description, spf.type").
		Join("JOIN subscription_plans sp ON sp.id = spf.plan_id").
		Where("sp.app_id = ?", appID).
		Scan(ctx, &uniqueFeatures)
	if err != nil {
		return nil, fmt.Errorf("failed to get unique features: %w", err)
	}

	m.logger.Info("found unique features", forge.F("count", len(uniqueFeatures)))

	// Step 2: Create Feature entities for each unique key
	featureKeyToID := make(map[string]xid.ID)

	for _, uf := range uniqueFeatures {
		// Check if feature already exists
		var existing schema.Feature
		err := tx.NewSelect().
			Model(&existing).
			Where("app_id = ?", appID).
			Where("key = ?", uf.Key).
			Scan(ctx)
		if err == nil {
			// Feature already exists, use its ID
			featureKeyToID[uf.Key] = existing.ID
			m.logger.Debug("feature already exists", forge.F("key", uf.Key))
			continue
		}

		// Create new feature
		now := time.Now()
		feature := &schema.Feature{
			ID:          xid.New(),
			AppID:       appID,
			Key:         uf.Key,
			Name:        uf.Name,
			Description: uf.Description,
			Type:        uf.Type,
			ResetPeriod: determineResetPeriod(uf.Type),
			IsPublic:    true,
			Metadata:    make(map[string]interface{}),
		}
		feature.CreatedAt = now
		feature.UpdatedAt = now

		_, err = tx.NewInsert().Model(feature).Exec(ctx)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to create feature %s: %v", uf.Key, err))
			continue
		}

		featureKeyToID[uf.Key] = feature.ID
		result.FeaturesCreated++
		m.logger.Debug("created feature", forge.F("key", uf.Key), forge.F("id", feature.ID.String()))
	}

	// Step 3: Get all PlanFeature entries and create PlanFeatureLink entries
	var planFeatures []schema.SubscriptionPlanFeature
	err = tx.NewSelect().
		Model(&planFeatures).
		Join("JOIN subscription_plans sp ON sp.id = spf.plan_id").
		Where("sp.app_id = ?", appID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan features: %w", err)
	}

	for _, pf := range planFeatures {
		featureID, ok := featureKeyToID[pf.Key]
		if !ok {
			result.Errors = append(result.Errors, fmt.Sprintf("feature ID not found for key %s", pf.Key))
			continue
		}

		// Check if link already exists
		var existing schema.PlanFeatureLink
		err := tx.NewSelect().
			Model(&existing).
			Where("plan_id = ?", pf.PlanID).
			Where("feature_id = ?", featureID).
			Scan(ctx)
		if err == nil {
			// Link already exists, skip
			continue
		}

		// Create link
		now := time.Now()
		link := &schema.PlanFeatureLink{
			ID:               xid.New(),
			PlanID:           pf.PlanID,
			FeatureID:        featureID,
			Value:            pf.Value,
			IsBlocked:        false,
			IsHighlighted:    false,
			OverrideSettings: make(map[string]interface{}),
			CreatedAt:        now,
			UpdatedAt:        now,
		}

		_, err = tx.NewInsert().Model(link).Exec(ctx)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to create link for plan %s feature %s: %v", pf.PlanID.String(), pf.Key, err))
			continue
		}

		result.LinksCreated++
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	m.logger.Info("feature migration completed",
		forge.F("featuresCreated", result.FeaturesCreated),
		forge.F("linksCreated", result.LinksCreated),
		forge.F("errors", len(result.Errors)))

	return result, nil
}

// ValidateMigration checks if all existing PlanFeature entries have corresponding Feature entities and links
func (m *FeaturesMigration) ValidateMigration(ctx context.Context, appID xid.ID) (bool, []string, error) {
	var issues []string

	// Get all plan features
	var planFeatures []schema.SubscriptionPlanFeature
	err := m.db.NewSelect().
		Model(&planFeatures).
		Join("JOIN subscription_plans sp ON sp.id = spf.plan_id").
		Where("sp.app_id = ?", appID).
		Scan(ctx)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get plan features: %w", err)
	}

	for _, pf := range planFeatures {
		// Check if feature exists
		var feature schema.Feature
		err := m.db.NewSelect().
			Model(&feature).
			Where("app_id = ?", appID).
			Where("key = ?", pf.Key).
			Scan(ctx)
		if err != nil {
			issues = append(issues, fmt.Sprintf("feature not found for key %s", pf.Key))
			continue
		}

		// Check if link exists
		var link schema.PlanFeatureLink
		err = m.db.NewSelect().
			Model(&link).
			Where("plan_id = ?", pf.PlanID).
			Where("feature_id = ?", feature.ID).
			Scan(ctx)
		if err != nil {
			issues = append(issues, fmt.Sprintf("link not found for plan %s feature %s", pf.PlanID.String(), pf.Key))
		}
	}

	return len(issues) == 0, issues, nil
}

// GetMigrationStatus returns the current migration status for an app
func (m *FeaturesMigration) GetMigrationStatus(ctx context.Context, appID xid.ID) (*MigrationStatus, error) {
	status := &MigrationStatus{}

	// Count old plan features
	count, err := m.db.NewSelect().
		Model((*schema.SubscriptionPlanFeature)(nil)).
		Join("JOIN subscription_plans sp ON sp.id = spf.plan_id").
		Where("sp.app_id = ?", appID).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count plan features: %w", err)
	}
	status.OldPlanFeaturesCount = count

	// Count new features
	count, err = m.db.NewSelect().
		Model((*schema.Feature)(nil)).
		Where("app_id = ?", appID).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count features: %w", err)
	}
	status.NewFeaturesCount = count

	// Count links
	count, err = m.db.NewSelect().
		Model((*schema.PlanFeatureLink)(nil)).
		Join("JOIN subscription_features sf ON sf.id = spfl.feature_id").
		Where("sf.app_id = ?", appID).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count links: %w", err)
	}
	status.LinksCount = count

	// Validate
	valid, issues, _ := m.ValidateMigration(ctx, appID)
	status.IsValid = valid
	status.Issues = issues

	return status, nil
}

// MigrationStatus represents the current migration status
type MigrationStatus struct {
	OldPlanFeaturesCount int      `json:"oldPlanFeaturesCount"`
	NewFeaturesCount     int      `json:"newFeaturesCount"`
	LinksCount           int      `json:"linksCount"`
	IsValid              bool     `json:"isValid"`
	Issues               []string `json:"issues,omitempty"`
}

// SyncFeatureFromLegacy syncs a single feature from legacy PlanFeature to the new system
func (m *FeaturesMigration) SyncFeatureFromLegacy(ctx context.Context, appID xid.ID, featureKey string) error {
	// Get all plan features with this key
	var planFeatures []schema.SubscriptionPlanFeature
	err := m.db.NewSelect().
		Model(&planFeatures).
		Join("JOIN subscription_plans sp ON sp.id = spf.plan_id").
		Where("sp.app_id = ?", appID).
		Where("spf.key = ?", featureKey).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to get plan features: %w", err)
	}

	if len(planFeatures) == 0 {
		return nil // No features to sync
	}

	// Use first one as template
	template := planFeatures[0]

	// Get or create feature
	var feature schema.Feature
	err = m.db.NewSelect().
		Model(&feature).
		Where("app_id = ?", appID).
		Where("key = ?", featureKey).
		Scan(ctx)
	if err != nil {
		// Create new feature
		now := time.Now()
		feature = schema.Feature{
			ID:          xid.New(),
			AppID:       appID,
			Key:         template.Key,
			Name:        template.Name,
			Description: template.Description,
			Type:        template.Type,
			ResetPeriod: determineResetPeriod(template.Type),
			IsPublic:    true,
			Metadata:    make(map[string]interface{}),
		}
		feature.CreatedAt = now
		feature.UpdatedAt = now

		_, err = m.db.NewInsert().Model(&feature).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create feature: %w", err)
		}
	}

	// Create links for each plan
	for _, pf := range planFeatures {
		// Check if link exists
		var existing schema.PlanFeatureLink
		err := m.db.NewSelect().
			Model(&existing).
			Where("plan_id = ?", pf.PlanID).
			Where("feature_id = ?", feature.ID).
			Scan(ctx)
		if err == nil {
			continue // Link already exists
		}

		// Create link
		now := time.Now()
		link := &schema.PlanFeatureLink{
			ID:               xid.New(),
			PlanID:           pf.PlanID,
			FeatureID:        feature.ID,
			Value:            pf.Value,
			IsBlocked:        false,
			IsHighlighted:    false,
			OverrideSettings: make(map[string]interface{}),
			CreatedAt:        now,
			UpdatedAt:        now,
		}

		_, err = m.db.NewInsert().Model(link).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create link: %w", err)
		}
	}

	return nil
}

// ExportFeatures exports features and links to JSON format
func (m *FeaturesMigration) ExportFeatures(ctx context.Context, appID xid.ID) ([]byte, error) {
	type exportData struct {
		Features []schema.Feature         `json:"features"`
		Links    []schema.PlanFeatureLink `json:"links"`
	}

	var data exportData

	// Get features
	err := m.db.NewSelect().
		Model(&data.Features).
		Relation("Tiers").
		Where("sf.app_id = ?", appID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get features: %w", err)
	}

	// Get links
	err = m.db.NewSelect().
		Model(&data.Links).
		Join("JOIN subscription_features sf ON sf.id = spfl.feature_id").
		Where("sf.app_id = ?", appID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get links: %w", err)
	}

	return json.MarshalIndent(data, "", "  ")
}

// Helper functions

func determineResetPeriod(featureType string) string {
	switch featureType {
	case "limit", "metered":
		return "billing_period"
	default:
		return "none"
	}
}
