package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/schema"
)

// NotificationQueueRepository defines the interface for notification queue operations
type NotificationQueueRepository interface {
	// Enqueue adds an item to the retry queue
	Enqueue(ctx context.Context, item *schema.NotificationQueue) error
	// Dequeue retrieves items ready for retry
	Dequeue(ctx context.Context, limit int) ([]*schema.NotificationQueue, error)
	// Update updates an item's retry state
	Update(ctx context.Context, item *schema.NotificationQueue) error
	// Delete removes an item from the queue
	Delete(ctx context.Context, id xid.ID) error
	// MarkFailed marks an item as permanently failed
	MarkFailed(ctx context.Context, id xid.ID, lastError string) error
	// MarkSucceeded marks an item as succeeded
	MarkSucceeded(ctx context.Context, id xid.ID) error
	// GetStats returns queue statistics
	GetStats(ctx context.Context) (*schema.NotificationQueueStats, error)
	// GetByID retrieves a queue item by ID
	GetByID(ctx context.Context, id xid.ID) (*schema.NotificationQueue, error)
	// CleanupOld removes old completed/failed items
	CleanupOld(ctx context.Context, olderThan time.Time) error
}

// notificationQueueRepository implements NotificationQueueRepository
type notificationQueueRepository struct {
	db *bun.DB
}

// NewNotificationQueueRepository creates a new notification queue repository
func NewNotificationQueueRepository(db *bun.DB) NotificationQueueRepository {
	return &notificationQueueRepository{db: db}
}

// Enqueue adds an item to the retry queue
func (r *notificationQueueRepository) Enqueue(ctx context.Context, item *schema.NotificationQueue) error {
	if item.ID.IsNil() {
		item.ID = xid.New()
	}
	if item.Status == "" {
		item.Status = schema.NotificationQueueStatusPending
	}
	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()

	_, err := r.db.NewInsert().Model(item).Exec(ctx)
	return err
}

// Dequeue retrieves items ready for retry
func (r *notificationQueueRepository) Dequeue(ctx context.Context, limit int) ([]*schema.NotificationQueue, error) {
	var items []*schema.NotificationQueue

	// Get items that are:
	// 1. Pending status
	// 2. Next retry time is now or in the past (or NULL for first attempt)
	// 3. Not exceeded max attempts
	now := time.Now()

	err := r.db.NewSelect().
		Model(&items).
		Where("status = ?", schema.NotificationQueueStatusPending).
		Where("(next_retry_at IS NULL OR next_retry_at <= ?)", now).
		Where("attempts < max_attempts").
		OrderExpr("CASE priority WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'normal' THEN 3 ELSE 4 END").
		Order("created_at ASC").
		Limit(limit).
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	// Mark items as processing
	for _, item := range items {
		item.Status = schema.NotificationQueueStatusProcessing
		item.UpdatedAt = time.Now()
		_, _ = r.db.NewUpdate().
			Model(item).
			Column("status", "updated_at").
			Where("id = ?", item.ID).
			Exec(ctx)
	}

	return items, nil
}

// Update updates an item's retry state
func (r *notificationQueueRepository) Update(ctx context.Context, item *schema.NotificationQueue) error {
	item.UpdatedAt = time.Now()

	_, err := r.db.NewUpdate().
		Model(item).
		Column("attempts", "last_error", "status", "next_retry_at", "updated_at").
		Where("id = ?", item.ID).
		Exec(ctx)
	return err
}

// Delete removes an item from the queue
func (r *notificationQueueRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.NotificationQueue)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// MarkFailed marks an item as permanently failed
func (r *notificationQueueRepository) MarkFailed(ctx context.Context, id xid.ID, lastError string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.NotificationQueue)(nil)).
		Set("status = ?", schema.NotificationQueueStatusFailed).
		Set("last_error = ?", lastError).
		Set("processed_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// MarkSucceeded marks an item as succeeded
func (r *notificationQueueRepository) MarkSucceeded(ctx context.Context, id xid.ID) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.NotificationQueue)(nil)).
		Set("status = ?", schema.NotificationQueueStatusSucceeded).
		Set("processed_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// GetStats returns queue statistics
func (r *notificationQueueRepository) GetStats(ctx context.Context) (*schema.NotificationQueueStats, error) {
	stats := &schema.NotificationQueueStats{}

	// Count pending
	pending, err := r.db.NewSelect().
		Model((*schema.NotificationQueue)(nil)).
		Where("status = ?", schema.NotificationQueueStatusPending).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	stats.PendingCount = int64(pending)

	// Count processing
	processing, err := r.db.NewSelect().
		Model((*schema.NotificationQueue)(nil)).
		Where("status = ?", schema.NotificationQueueStatusProcessing).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	stats.ProcessingCount = int64(processing)

	// Count succeeded
	succeeded, err := r.db.NewSelect().
		Model((*schema.NotificationQueue)(nil)).
		Where("status = ?", schema.NotificationQueueStatusSucceeded).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	stats.SucceededCount = int64(succeeded)

	// Count failed
	failed, err := r.db.NewSelect().
		Model((*schema.NotificationQueue)(nil)).
		Where("status = ?", schema.NotificationQueueStatusFailed).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	stats.FailedCount = int64(failed)

	stats.TotalCount = stats.PendingCount + stats.ProcessingCount + stats.SucceededCount + stats.FailedCount

	return stats, nil
}

// GetByID retrieves a queue item by ID
func (r *notificationQueueRepository) GetByID(ctx context.Context, id xid.ID) (*schema.NotificationQueue, error) {
	item := &schema.NotificationQueue{}
	err := r.db.NewSelect().
		Model(item).
		Where("id = ?", id).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return item, nil
}

// CleanupOld removes old completed/failed items
func (r *notificationQueueRepository) CleanupOld(ctx context.Context, olderThan time.Time) error {
	_, err := r.db.NewDelete().
		Model((*schema.NotificationQueue)(nil)).
		Where("status IN (?, ?)", schema.NotificationQueueStatusSucceeded, schema.NotificationQueueStatusFailed).
		Where("processed_at < ?", olderThan).
		Exec(ctx)
	return err
}

// DatabaseRetryStorage implements notification.RetryStorage using the database repository
type DatabaseRetryStorage struct {
	repo NotificationQueueRepository
}

// NewDatabaseRetryStorage creates a new database-backed retry storage
func NewDatabaseRetryStorage(repo NotificationQueueRepository) notification.RetryStorage {
	return &DatabaseRetryStorage{repo: repo}
}

// Enqueue adds an item to the retry queue
func (s *DatabaseRetryStorage) Enqueue(ctx context.Context, item *notification.RetryItem) error {
	queueItem := &schema.NotificationQueue{
		ID:          item.ID,
		AppID:       item.AppID,
		Type:        string(item.Type),
		Priority:    string(item.Priority),
		Recipient:   item.Recipient,
		Subject:     item.Subject,
		Body:        item.Body,
		TemplateKey: item.TemplateKey,
		Attempts:    item.Attempts,
		MaxAttempts: 3, // Default max attempts
		LastError:   item.LastError,
		Status:      schema.NotificationQueueStatusPending,
		NextRetryAt: &item.NextRetry,
		CreatedAt:   item.CreatedAt,
	}
	return s.repo.Enqueue(ctx, queueItem)
}

// Dequeue retrieves items ready for retry
func (s *DatabaseRetryStorage) Dequeue(ctx context.Context, limit int) ([]*notification.RetryItem, error) {
	queueItems, err := s.repo.Dequeue(ctx, limit)
	if err != nil {
		return nil, err
	}

	items := make([]*notification.RetryItem, len(queueItems))
	for i, qi := range queueItems {
		nextRetry := time.Now()
		if qi.NextRetryAt != nil {
			nextRetry = *qi.NextRetryAt
		}
		items[i] = &notification.RetryItem{
			ID:          qi.ID,
			AppID:       qi.AppID,
			Type:        notification.NotificationType(qi.Type),
			Priority:    notification.NotificationPriority(qi.Priority),
			Recipient:   qi.Recipient,
			Subject:     qi.Subject,
			Body:        qi.Body,
			TemplateKey: qi.TemplateKey,
			Attempts:    qi.Attempts,
			LastError:   qi.LastError,
			NextRetry:   nextRetry,
			CreatedAt:   qi.CreatedAt,
		}
	}
	return items, nil
}

// Update updates an item's retry state
func (s *DatabaseRetryStorage) Update(ctx context.Context, item *notification.RetryItem) error {
	queueItem := &schema.NotificationQueue{
		ID:          item.ID,
		Attempts:    item.Attempts,
		LastError:   item.LastError,
		Status:      schema.NotificationQueueStatusPending,
		NextRetryAt: &item.NextRetry,
	}
	return s.repo.Update(ctx, queueItem)
}

// Delete removes an item from the queue
func (s *DatabaseRetryStorage) Delete(ctx context.Context, id xid.ID) error {
	return s.repo.MarkSucceeded(ctx, id)
}

// MarkFailed marks an item as permanently failed
func (s *DatabaseRetryStorage) MarkFailed(ctx context.Context, item *notification.RetryItem) error {
	return s.repo.MarkFailed(ctx, item.ID, item.LastError)
}

// GetStats returns queue statistics
func (s *DatabaseRetryStorage) GetStats(ctx context.Context) (*notification.RetryStats, error) {
	stats, err := s.repo.GetStats(ctx)
	if err != nil {
		return nil, err
	}
	return &notification.RetryStats{
		PendingCount:   stats.PendingCount,
		FailedCount:    stats.FailedCount,
		ProcessedCount: stats.SucceededCount,
	}, nil
}

