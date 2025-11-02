package stepup

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// Repository defines the interface for step-up data persistence
type Repository interface {
	// Verifications
	CreateVerification(ctx context.Context, verification *StepUpVerification) error
	GetVerification(ctx context.Context, id string) (*StepUpVerification, error)
	GetLatestVerification(ctx context.Context, userID, orgID string, level SecurityLevel) (*StepUpVerification, error)
	ListVerifications(ctx context.Context, userID, orgID string, limit, offset int) ([]*StepUpVerification, error)

	// Requirements
	CreateRequirement(ctx context.Context, requirement *StepUpRequirement) error
	GetRequirement(ctx context.Context, id string) (*StepUpRequirement, error)
	GetRequirementByToken(ctx context.Context, token string) (*StepUpRequirement, error)
	UpdateRequirement(ctx context.Context, requirement *StepUpRequirement) error
	ListPendingRequirements(ctx context.Context, userID, orgID string) ([]*StepUpRequirement, error)
	DeleteExpiredRequirements(ctx context.Context) error

	// Remembered devices
	CreateRememberedDevice(ctx context.Context, device *StepUpRememberedDevice) error
	GetRememberedDevice(ctx context.Context, userID, orgID, deviceID string) (*StepUpRememberedDevice, error)
	ListRememberedDevices(ctx context.Context, userID, orgID string) ([]*StepUpRememberedDevice, error)
	UpdateRememberedDevice(ctx context.Context, device *StepUpRememberedDevice) error
	DeleteRememberedDevice(ctx context.Context, id string) error
	DeleteExpiredRememberedDevices(ctx context.Context) error

	// Attempts
	CreateAttempt(ctx context.Context, attempt *StepUpAttempt) error
	ListAttempts(ctx context.Context, requirementID string) ([]*StepUpAttempt, error)
	CountFailedAttempts(ctx context.Context, userID, orgID string, since time.Time) (int, error)

	// Policies
	CreatePolicy(ctx context.Context, policy *StepUpPolicy) error
	GetPolicy(ctx context.Context, id string) (*StepUpPolicy, error)
	ListPolicies(ctx context.Context, orgID string) ([]*StepUpPolicy, error)
	UpdatePolicy(ctx context.Context, policy *StepUpPolicy) error
	DeletePolicy(ctx context.Context, id string) error

	// Audit logs
	CreateAuditLog(ctx context.Context, log *StepUpAuditLog) error
	ListAuditLogs(ctx context.Context, userID, orgID string, limit, offset int) ([]*StepUpAuditLog, error)
}

// BunRepository implements Repository using Bun ORM
type BunRepository struct {
	db *bun.DB
}

// NewBunRepository creates a new Bun-based repository
func NewBunRepository(db *bun.DB) *BunRepository {
	return &BunRepository{db: db}
}

// Verifications

func (r *BunRepository) CreateVerification(ctx context.Context, verification *StepUpVerification) error {
	_, err := r.db.NewInsert().Model(verification).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create verification: %w", err)
	}
	return nil
}

func (r *BunRepository) GetVerification(ctx context.Context, id string) (*StepUpVerification, error) {
	verification := &StepUpVerification{}
	err := r.db.NewSelect().
		Model(verification).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get verification: %w", err)
	}
	return verification, nil
}

func (r *BunRepository) GetLatestVerification(ctx context.Context, userID, orgID string, level SecurityLevel) (*StepUpVerification, error) {
	verification := &StepUpVerification{}
	err := r.db.NewSelect().
		Model(verification).
		Where("user_id = ? AND org_id = ? AND security_level = ?", userID, orgID, level).
		Where("expires_at > ?", time.Now()).
		Order("verified_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest verification: %w", err)
	}
	return verification, nil
}

func (r *BunRepository) ListVerifications(ctx context.Context, userID, orgID string, limit, offset int) ([]*StepUpVerification, error) {
	var verifications []*StepUpVerification
	err := r.db.NewSelect().
		Model(&verifications).
		Where("user_id = ? AND org_id = ?", userID, orgID).
		Order("verified_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list verifications: %w", err)
	}
	return verifications, nil
}

// Requirements

func (r *BunRepository) CreateRequirement(ctx context.Context, requirement *StepUpRequirement) error {
	_, err := r.db.NewInsert().Model(requirement).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create requirement: %w", err)
	}
	return nil
}

func (r *BunRepository) GetRequirement(ctx context.Context, id string) (*StepUpRequirement, error) {
	requirement := &StepUpRequirement{}
	err := r.db.NewSelect().
		Model(requirement).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get requirement: %w", err)
	}
	return requirement, nil
}

func (r *BunRepository) GetRequirementByToken(ctx context.Context, token string) (*StepUpRequirement, error) {
	requirement := &StepUpRequirement{}
	err := r.db.NewSelect().
		Model(requirement).
		Where("challenge_token = ?", token).
		Where("status = ?", "pending").
		Where("expires_at > ?", time.Now()).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get requirement by token: %w", err)
	}
	return requirement, nil
}

func (r *BunRepository) UpdateRequirement(ctx context.Context, requirement *StepUpRequirement) error {
	_, err := r.db.NewUpdate().
		Model(requirement).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update requirement: %w", err)
	}
	return nil
}

func (r *BunRepository) ListPendingRequirements(ctx context.Context, userID, orgID string) ([]*StepUpRequirement, error) {
	var requirements []*StepUpRequirement
	err := r.db.NewSelect().
		Model(&requirements).
		Where("user_id = ? AND org_id = ?", userID, orgID).
		Where("status = ?", "pending").
		Where("expires_at > ?", time.Now()).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending requirements: %w", err)
	}
	return requirements, nil
}

func (r *BunRepository) DeleteExpiredRequirements(ctx context.Context) error {
	_, err := r.db.NewDelete().
		Model((*StepUpRequirement)(nil)).
		Where("expires_at < ?", time.Now()).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete expired requirements: %w", err)
	}
	return nil
}

// Remembered devices

func (r *BunRepository) CreateRememberedDevice(ctx context.Context, device *StepUpRememberedDevice) error {
	_, err := r.db.NewInsert().Model(device).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create remembered device: %w", err)
	}
	return nil
}

func (r *BunRepository) GetRememberedDevice(ctx context.Context, userID, orgID, deviceID string) (*StepUpRememberedDevice, error) {
	device := &StepUpRememberedDevice{}
	err := r.db.NewSelect().
		Model(device).
		Where("user_id = ? AND org_id = ? AND device_id = ?", userID, orgID, deviceID).
		Where("expires_at > ?", time.Now()).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get remembered device: %w", err)
	}
	return device, nil
}

func (r *BunRepository) ListRememberedDevices(ctx context.Context, userID, orgID string) ([]*StepUpRememberedDevice, error) {
	var devices []*StepUpRememberedDevice
	err := r.db.NewSelect().
		Model(&devices).
		Where("user_id = ? AND org_id = ?", userID, orgID).
		Where("expires_at > ?", time.Now()).
		Order("remembered_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list remembered devices: %w", err)
	}
	return devices, nil
}

func (r *BunRepository) UpdateRememberedDevice(ctx context.Context, device *StepUpRememberedDevice) error {
	_, err := r.db.NewUpdate().
		Model(device).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update remembered device: %w", err)
	}
	return nil
}

func (r *BunRepository) DeleteRememberedDevice(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().
		Model((*StepUpRememberedDevice)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete remembered device: %w", err)
	}
	return nil
}

func (r *BunRepository) DeleteExpiredRememberedDevices(ctx context.Context) error {
	_, err := r.db.NewDelete().
		Model((*StepUpRememberedDevice)(nil)).
		Where("expires_at < ?", time.Now()).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete expired remembered devices: %w", err)
	}
	return nil
}

// Attempts

func (r *BunRepository) CreateAttempt(ctx context.Context, attempt *StepUpAttempt) error {
	_, err := r.db.NewInsert().Model(attempt).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create attempt: %w", err)
	}
	return nil
}

func (r *BunRepository) ListAttempts(ctx context.Context, requirementID string) ([]*StepUpAttempt, error) {
	var attempts []*StepUpAttempt
	err := r.db.NewSelect().
		Model(&attempts).
		Where("requirement_id = ?", requirementID).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list attempts: %w", err)
	}
	return attempts, nil
}

func (r *BunRepository) CountFailedAttempts(ctx context.Context, userID, orgID string, since time.Time) (int, error) {
	count, err := r.db.NewSelect().
		Model((*StepUpAttempt)(nil)).
		Where("user_id = ? AND org_id = ?", userID, orgID).
		Where("success = false").
		Where("created_at > ?", since).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count failed attempts: %w", err)
	}
	return count, nil
}

// Policies

func (r *BunRepository) CreatePolicy(ctx context.Context, policy *StepUpPolicy) error {
	_, err := r.db.NewInsert().Model(policy).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create policy: %w", err)
	}
	return nil
}

func (r *BunRepository) GetPolicy(ctx context.Context, id string) (*StepUpPolicy, error) {
	policy := &StepUpPolicy{}
	err := r.db.NewSelect().
		Model(policy).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}
	return policy, nil
}

func (r *BunRepository) ListPolicies(ctx context.Context, orgID string) ([]*StepUpPolicy, error) {
	var policies []*StepUpPolicy
	err := r.db.NewSelect().
		Model(&policies).
		Where("org_id = ?", orgID).
		Where("enabled = true").
		Order("priority DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}
	return policies, nil
}

func (r *BunRepository) UpdatePolicy(ctx context.Context, policy *StepUpPolicy) error {
	policy.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(policy).
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}
	return nil
}

func (r *BunRepository) DeletePolicy(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().
		Model((*StepUpPolicy)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}
	return nil
}

// Audit logs

func (r *BunRepository) CreateAuditLog(ctx context.Context, log *StepUpAuditLog) error {
	_, err := r.db.NewInsert().Model(log).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}
	return nil
}

func (r *BunRepository) ListAuditLogs(ctx context.Context, userID, orgID string, limit, offset int) ([]*StepUpAuditLog, error) {
	var logs []*StepUpAuditLog
	query := r.db.NewSelect().Model(&logs)

	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if orgID != "" {
		query = query.Where("org_id = ?", orgID)
	}

	err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}
	return logs, nil
}

