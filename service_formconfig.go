package authsome

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store"
)

// ──────────────────────────────────────────────────
// Form Config Management
// ──────────────────────────────────────────────────

// GetSignupFormConfig returns the active signup form config for an app.
func (e *Engine) GetSignupFormConfig(ctx context.Context, appID id.AppID) (*formconfig.FormConfig, error) {
	fc, err := e.store.GetFormConfig(ctx, appID, formconfig.FormTypeSignup)
	if err != nil {
		return nil, fmt.Errorf("authsome: get signup form config: %w", err)
	}
	return fc, nil
}

// SaveSignupFormConfig creates or updates a signup form config.
// On update, the version is automatically incremented.
func (e *Engine) SaveSignupFormConfig(ctx context.Context, fc *formconfig.FormConfig) error {
	fc.FormType = formconfig.FormTypeSignup

	existing, err := e.store.GetFormConfig(ctx, fc.AppID, formconfig.FormTypeSignup)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return fmt.Errorf("authsome: save signup form config: %w", err)
	}

	now := time.Now()
	if existing != nil {
		// Update existing config.
		fc.ID = existing.ID
		fc.Version = existing.Version + 1
		fc.CreatedAt = existing.CreatedAt
		fc.UpdatedAt = now

		if err := e.store.UpdateFormConfig(ctx, fc); err != nil {
			return fmt.Errorf("authsome: save signup form config: %w", err)
		}
	} else {
		// Create new config.
		if fc.ID.IsNil() {
			fc.ID = id.NewFormConfigID()
		}
		fc.Version = 1
		fc.CreatedAt = now
		fc.UpdatedAt = now

		if err := e.store.CreateFormConfig(ctx, fc); err != nil {
			return fmt.Errorf("authsome: save signup form config: %w", err)
		}
	}

	e.relayEvent(ctx, "formconfig.saved", fc.AppID.String(), map[string]string{
		"form_config_id": fc.ID.String(),
		"form_type":      fc.FormType,
	})

	return nil
}

// DeleteSignupFormConfig deletes the signup form config for an app.
func (e *Engine) DeleteSignupFormConfig(ctx context.Context, appID id.AppID) error {
	if err := e.store.DeleteFormConfig(ctx, appID, formconfig.FormTypeSignup); err != nil {
		return fmt.Errorf("authsome: delete signup form config: %w", err)
	}
	return nil
}

// ListFormConfigs returns all form configs for an app.
func (e *Engine) ListFormConfigs(ctx context.Context, appID id.AppID) ([]*formconfig.FormConfig, error) {
	return e.store.ListFormConfigs(ctx, appID)
}

// ──────────────────────────────────────────────────
// Branding Management
// ──────────────────────────────────────────────────

// GetOrgBranding returns the branding config for an organization.
func (e *Engine) GetOrgBranding(ctx context.Context, orgID id.OrgID) (*formconfig.BrandingConfig, error) {
	bc, err := e.store.GetBranding(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("authsome: get org branding: %w", err)
	}
	return bc, nil
}

// SaveOrgBranding creates or updates the branding config for an organization.
func (e *Engine) SaveOrgBranding(ctx context.Context, b *formconfig.BrandingConfig) error {
	if b.ID.IsNil() {
		b.ID = id.NewBrandingConfigID()
	}
	now := time.Now()
	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	b.UpdatedAt = now

	if err := e.store.SaveBranding(ctx, b); err != nil {
		return fmt.Errorf("authsome: save org branding: %w", err)
	}

	e.relayEvent(ctx, "branding.saved", b.OrgID.String(), map[string]string{
		"branding_id": b.ID.String(),
		"org_id":      b.OrgID.String(),
	})

	return nil
}

// DeleteOrgBranding deletes the branding config for an organization.
func (e *Engine) DeleteOrgBranding(ctx context.Context, orgID id.OrgID) error {
	if err := e.store.DeleteBranding(ctx, orgID); err != nil {
		return fmt.Errorf("authsome: delete org branding: %w", err)
	}
	return nil
}
