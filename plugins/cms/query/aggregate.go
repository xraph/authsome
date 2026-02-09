package query

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"

	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/authsome/plugins/cms/schema"
)

// SimpleAggregateConfig configures a simple aggregation query
// This is a simpler interface than AggregateQuery for common use cases.
type SimpleAggregateConfig struct {
	// Operator is the aggregation type (count, sum, avg, min, max)
	Operator AggregateOperator
	// Field is the field to aggregate (for sum, avg, min, max)
	Field string
	// GroupBy is the field to group results by
	GroupBy string
	// Filters are optional filters to apply
	Filters map[string]any
	// DateTrunc truncates dates for time-based grouping (day, week, month, year)
	DateTrunc string
}

// SimpleAggregateResult represents a simple aggregation result.
type SimpleAggregateResult struct {
	GroupValue any     `json:"groupValue,omitempty"`
	Count      int     `json:"count,omitempty"`
	Sum        float64 `json:"sum,omitempty"`
	Avg        float64 `json:"avg,omitempty"`
	Min        any     `json:"min,omitempty"`
	Max        any     `json:"max,omitempty"`
}

// Aggregator handles aggregation queries.
type Aggregator struct {
	db *bun.DB
}

// NewAggregator creates a new aggregator.
func NewAggregator(db *bun.DB) *Aggregator {
	return &Aggregator{db: db}
}

// SimpleAggregate performs a simple aggregation query on content entries.
func (a *Aggregator) SimpleAggregate(ctx context.Context, contentTypeID xid.ID, config *SimpleAggregateConfig) ([]*SimpleAggregateResult, error) {
	if config == nil {
		return nil, core.ErrInvalidQuery("aggregate config is required")
	}

	query := a.db.NewSelect().
		Model((*schema.ContentEntry)(nil)).
		Where("content_type_id = ?", contentTypeID).
		Where("deleted_at IS NULL")

	// Apply filters
	for field, value := range config.Filters {
		query = query.Where(fmt.Sprintf("data ->> '%s' = ?", field), value)
	}

	switch config.Operator {
	case AggCount:
		return a.aggregateCount(ctx, query, config)
	case AggSum:
		return a.aggregateSum(ctx, query, config)
	case AggAvg:
		return a.aggregateAvg(ctx, query, config)
	case AggMin:
		return a.aggregateMin(ctx, query, config)
	case AggMax:
		return a.aggregateMax(ctx, query, config)
	default:
		return nil, core.ErrInvalidQuery("unknown aggregation type: " + string(config.Operator))
	}
}

// aggregateCount counts entries, optionally grouped.
func (a *Aggregator) aggregateCount(ctx context.Context, query *bun.SelectQuery, config *SimpleAggregateConfig) ([]*SimpleAggregateResult, error) {
	if config.GroupBy != "" {
		return a.aggregateCountGrouped(ctx, query, config)
	}

	count, err := query.Count(ctx)
	if err != nil {
		return nil, core.ErrInternalError("failed to count entries", err)
	}

	return []*SimpleAggregateResult{{Count: count}}, nil
}

// aggregateCountGrouped counts entries grouped by a field.
func (a *Aggregator) aggregateCountGrouped(ctx context.Context, query *bun.SelectQuery, config *SimpleAggregateConfig) ([]*SimpleAggregateResult, error) {
	groupExpr := a.getGroupExpression(config)

	type countResult struct {
		GroupValue string `bun:"group_value"`
		Count      int    `bun:"count"`
	}

	var results []countResult

	err := query.
		ColumnExpr(groupExpr+" as group_value").
		ColumnExpr("COUNT(*) as count").
		Group("group_value").
		Order("count DESC").
		Scan(ctx, &results)
	if err != nil {
		return nil, core.ErrInternalError("failed to execute grouped count", err)
	}

	aggResults := make([]*SimpleAggregateResult, len(results))
	for i, r := range results {
		aggResults[i] = &SimpleAggregateResult{
			GroupValue: r.GroupValue,
			Count:      r.Count,
		}
	}

	return aggResults, nil
}

// aggregateSum sums a numeric field.
func (a *Aggregator) aggregateSum(ctx context.Context, query *bun.SelectQuery, config *SimpleAggregateConfig) ([]*SimpleAggregateResult, error) {
	if config.Field == "" {
		return nil, core.ErrInvalidQuery("field is required for sum aggregation")
	}

	fieldExpr := fmt.Sprintf("COALESCE((data ->> '%s')::numeric, 0)", config.Field)

	if config.GroupBy != "" {
		groupExpr := a.getGroupExpression(config)

		type sumResult struct {
			GroupValue string  `bun:"group_value"`
			Sum        float64 `bun:"sum"`
		}

		var results []sumResult

		err := query.
			ColumnExpr(groupExpr+" as group_value").
			ColumnExpr(fmt.Sprintf("SUM(%s) as sum", fieldExpr)).
			Group("group_value").
			Order("sum DESC").
			Scan(ctx, &results)
		if err != nil {
			return nil, core.ErrInternalError("failed to execute grouped sum", err)
		}

		aggResults := make([]*SimpleAggregateResult, len(results))
		for i, r := range results {
			aggResults[i] = &SimpleAggregateResult{
				GroupValue: r.GroupValue,
				Sum:        r.Sum,
			}
		}

		return aggResults, nil
	}

	// Simple sum without grouping
	type sumResult struct {
		Sum float64 `bun:"sum"`
	}

	var result sumResult

	err := query.
		ColumnExpr(fmt.Sprintf("SUM(%s) as sum", fieldExpr)).
		Scan(ctx, &result)
	if err != nil {
		return nil, core.ErrInternalError("failed to execute sum", err)
	}

	return []*SimpleAggregateResult{{Sum: result.Sum}}, nil
}

// aggregateAvg averages a numeric field.
func (a *Aggregator) aggregateAvg(ctx context.Context, query *bun.SelectQuery, config *SimpleAggregateConfig) ([]*SimpleAggregateResult, error) {
	if config.Field == "" {
		return nil, core.ErrInvalidQuery("field is required for avg aggregation")
	}

	fieldExpr := fmt.Sprintf("COALESCE((data ->> '%s')::numeric, 0)", config.Field)

	if config.GroupBy != "" {
		groupExpr := a.getGroupExpression(config)

		type avgResult struct {
			GroupValue string  `bun:"group_value"`
			Avg        float64 `bun:"avg"`
			Count      int     `bun:"count"`
		}

		var results []avgResult

		err := query.
			ColumnExpr(groupExpr+" as group_value").
			ColumnExpr(fmt.Sprintf("AVG(%s) as avg", fieldExpr)).
			ColumnExpr("COUNT(*) as count").
			Group("group_value").
			Order("avg DESC").
			Scan(ctx, &results)
		if err != nil {
			return nil, core.ErrInternalError("failed to execute grouped avg", err)
		}

		aggResults := make([]*SimpleAggregateResult, len(results))
		for i, r := range results {
			aggResults[i] = &SimpleAggregateResult{
				GroupValue: r.GroupValue,
				Avg:        r.Avg,
				Count:      r.Count,
			}
		}

		return aggResults, nil
	}

	// Simple avg without grouping
	type avgResult struct {
		Avg   float64 `bun:"avg"`
		Count int     `bun:"count"`
	}

	var result avgResult

	err := query.
		ColumnExpr(fmt.Sprintf("AVG(%s) as avg", fieldExpr)).
		ColumnExpr("COUNT(*) as count").
		Scan(ctx, &result)
	if err != nil {
		return nil, core.ErrInternalError("failed to execute avg", err)
	}

	return []*SimpleAggregateResult{{Avg: result.Avg, Count: result.Count}}, nil
}

// aggregateMin finds the minimum value of a field.
func (a *Aggregator) aggregateMin(ctx context.Context, query *bun.SelectQuery, config *SimpleAggregateConfig) ([]*SimpleAggregateResult, error) {
	if config.Field == "" {
		return nil, core.ErrInvalidQuery("field is required for min aggregation")
	}

	fieldExpr := fmt.Sprintf("data ->> '%s'", config.Field)

	if config.GroupBy != "" {
		groupExpr := a.getGroupExpression(config)

		type minResult struct {
			GroupValue string `bun:"group_value"`
			Min        string `bun:"min"`
		}

		var results []minResult

		err := query.
			ColumnExpr(groupExpr+" as group_value").
			ColumnExpr(fmt.Sprintf("MIN(%s) as min", fieldExpr)).
			Group("group_value").
			Scan(ctx, &results)
		if err != nil {
			return nil, core.ErrInternalError("failed to execute grouped min", err)
		}

		aggResults := make([]*SimpleAggregateResult, len(results))
		for i, r := range results {
			aggResults[i] = &SimpleAggregateResult{
				GroupValue: r.GroupValue,
				Min:        r.Min,
			}
		}

		return aggResults, nil
	}

	// Simple min without grouping
	type minResult struct {
		Min string `bun:"min"`
	}

	var result minResult

	err := query.
		ColumnExpr(fmt.Sprintf("MIN(%s) as min", fieldExpr)).
		Scan(ctx, &result)
	if err != nil {
		return nil, core.ErrInternalError("failed to execute min", err)
	}

	return []*SimpleAggregateResult{{Min: result.Min}}, nil
}

// aggregateMax finds the maximum value of a field.
func (a *Aggregator) aggregateMax(ctx context.Context, query *bun.SelectQuery, config *SimpleAggregateConfig) ([]*SimpleAggregateResult, error) {
	if config.Field == "" {
		return nil, core.ErrInvalidQuery("field is required for max aggregation")
	}

	fieldExpr := fmt.Sprintf("data ->> '%s'", config.Field)

	if config.GroupBy != "" {
		groupExpr := a.getGroupExpression(config)

		type maxResult struct {
			GroupValue string `bun:"group_value"`
			Max        string `bun:"max"`
		}

		var results []maxResult

		err := query.
			ColumnExpr(groupExpr+" as group_value").
			ColumnExpr(fmt.Sprintf("MAX(%s) as max", fieldExpr)).
			Group("group_value").
			Scan(ctx, &results)
		if err != nil {
			return nil, core.ErrInternalError("failed to execute grouped max", err)
		}

		aggResults := make([]*SimpleAggregateResult, len(results))
		for i, r := range results {
			aggResults[i] = &SimpleAggregateResult{
				GroupValue: r.GroupValue,
				Max:        r.Max,
			}
		}

		return aggResults, nil
	}

	// Simple max without grouping
	type maxResult struct {
		Max string `bun:"max"`
	}

	var result maxResult

	err := query.
		ColumnExpr(fmt.Sprintf("MAX(%s) as max", fieldExpr)).
		Scan(ctx, &result)
	if err != nil {
		return nil, core.ErrInternalError("failed to execute max", err)
	}

	return []*SimpleAggregateResult{{Max: result.Max}}, nil
}

// getGroupExpression returns the SQL expression for grouping.
func (a *Aggregator) getGroupExpression(config *SimpleAggregateConfig) string {
	// Check if grouping by a system field
	switch config.GroupBy {
	case "status":
		return "status"
	case "createdAt", "created_at":
		if config.DateTrunc != "" {
			return fmt.Sprintf("DATE_TRUNC('%s', created_at)", config.DateTrunc)
		}

		return "DATE(created_at)"
	case "updatedAt", "updated_at":
		if config.DateTrunc != "" {
			return fmt.Sprintf("DATE_TRUNC('%s', updated_at)", config.DateTrunc)
		}

		return "DATE(updated_at)"
	case "publishedAt", "published_at":
		if config.DateTrunc != "" {
			return fmt.Sprintf("DATE_TRUNC('%s', published_at)", config.DateTrunc)
		}

		return "DATE(published_at)"
	default:
		// Assume it's a JSONB field
		return fmt.Sprintf("data ->> '%s'", config.GroupBy)
	}
}

// =============================================================================
// Statistics Helpers
// =============================================================================

// GetEntryStats returns statistics for entries of a content type.
func (a *Aggregator) GetEntryStats(ctx context.Context, contentTypeID xid.ID) (*core.ContentTypeStatsDTO, error) {
	stats := &core.ContentTypeStatsDTO{
		ContentTypeID:   contentTypeID.String(),
		EntriesByStatus: make(map[string]int),
	}

	// Count by status
	type statusCount struct {
		Status string `bun:"status"`
		Count  int    `bun:"count"`
	}

	var statusCounts []statusCount

	err := a.db.NewSelect().
		Model((*schema.ContentEntry)(nil)).
		Column("status").
		ColumnExpr("COUNT(*) as count").
		Where("content_type_id = ?", contentTypeID).
		Where("deleted_at IS NULL").
		Group("status").
		Scan(ctx, &statusCounts)
	if err != nil {
		return nil, core.ErrInternalError("failed to get entry stats", err)
	}

	for _, sc := range statusCounts {
		stats.EntriesByStatus[sc.Status] = sc.Count

		stats.TotalEntries += sc.Count
		switch sc.Status {
		case "draft":
			stats.DraftEntries = sc.Count
		case "published":
			stats.PublishedEntries = sc.Count
		case "archived":
			stats.ArchivedEntries = sc.Count
		}
	}

	// Get last entry date
	var lastEntry time.Time

	err = a.db.NewSelect().
		Model((*schema.ContentEntry)(nil)).
		ColumnExpr("MAX(created_at)").
		Where("content_type_id = ?", contentTypeID).
		Where("deleted_at IS NULL").
		Scan(ctx, &lastEntry)

	if err == nil && !lastEntry.IsZero() {
		stats.LastEntryAt = &lastEntry
	}

	return stats, nil
}

// GetCMSStats returns overall CMS statistics.
func (a *Aggregator) GetCMSStats(ctx context.Context, appID, envID xid.ID) (*core.CMSStatsDTO, error) {
	stats := &core.CMSStatsDTO{
		EntriesByStatus: make(map[string]int),
		EntriesByType:   make(map[string]int),
	}

	// Count content types
	contentTypeCount, err := a.db.NewSelect().
		Model((*schema.ContentType)(nil)).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("deleted_at IS NULL").
		Count(ctx)
	if err != nil {
		return nil, core.ErrInternalError("failed to count content types", err)
	}

	stats.TotalContentTypes = contentTypeCount

	// Count entries by status
	type statusCount struct {
		Status string `bun:"status"`
		Count  int    `bun:"count"`
	}

	var statusCounts []statusCount

	err = a.db.NewSelect().
		Model((*schema.ContentEntry)(nil)).
		Column("status").
		ColumnExpr("COUNT(*) as count").
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("deleted_at IS NULL").
		Group("status").
		Scan(ctx, &statusCounts)
	if err != nil {
		return nil, core.ErrInternalError("failed to count entries by status", err)
	}

	for _, sc := range statusCounts {
		stats.EntriesByStatus[sc.Status] = sc.Count
		stats.TotalEntries += sc.Count
	}

	// Count entries by type
	type typeCount struct {
		TypeID string `bun:"content_type_id"`
		Count  int    `bun:"count"`
	}

	var typeCounts []typeCount

	err = a.db.NewSelect().
		Model((*schema.ContentEntry)(nil)).
		Column("content_type_id").
		ColumnExpr("COUNT(*) as count").
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("deleted_at IS NULL").
		Group("content_type_id").
		Scan(ctx, &typeCounts)
	if err != nil {
		return nil, core.ErrInternalError("failed to count entries by type", err)
	}

	for _, tc := range typeCounts {
		stats.EntriesByType[tc.TypeID] = tc.Count
	}

	// Count recently updated (last 7 days)
	weekAgo := time.Now().AddDate(0, 0, -7)

	recentCount, err := a.db.NewSelect().
		Model((*schema.ContentEntry)(nil)).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("deleted_at IS NULL").
		Where("updated_at > ?", weekAgo).
		Count(ctx)
	if err == nil {
		stats.RecentlyUpdated = recentCount
	}

	// Count scheduled entries
	scheduledCount, err := a.db.NewSelect().
		Model((*schema.ContentEntry)(nil)).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("deleted_at IS NULL").
		Where("status = ?", "scheduled").
		Count(ctx)
	if err == nil {
		stats.ScheduledEntries = scheduledCount
	}

	// Count revisions
	revisionCount, err := a.db.NewSelect().
		Model((*schema.ContentRevision)(nil)).
		Join("JOIN cms_content_entries ce ON ce.id = content_revision.entry_id").
		Where("ce.app_id = ?", appID).
		Where("ce.environment_id = ?", envID).
		Count(ctx)
	if err == nil {
		stats.TotalRevisions = revisionCount
	}

	return stats, nil
}

// GetTimeSeriesStats returns entry counts over time.
func (a *Aggregator) GetTimeSeriesStats(ctx context.Context, contentTypeID xid.ID, dateTrunc string, limit int) ([]map[string]any, error) {
	if dateTrunc == "" {
		dateTrunc = "day"
	}

	if limit <= 0 {
		limit = 30
	}

	type timeResult struct {
		Period time.Time `bun:"period"`
		Count  int       `bun:"count"`
	}

	var results []timeResult

	err := a.db.NewSelect().
		Model((*schema.ContentEntry)(nil)).
		ColumnExpr(fmt.Sprintf("DATE_TRUNC('%s', created_at) as period", dateTrunc)).
		ColumnExpr("COUNT(*) as count").
		Where("content_type_id = ?", contentTypeID).
		Where("deleted_at IS NULL").
		Group("period").
		Order("period DESC").
		Limit(limit).
		Scan(ctx, &results)
	if err != nil {
		return nil, core.ErrInternalError("failed to get time series stats", err)
	}

	// Convert to generic map for flexibility
	data := make([]map[string]any, len(results))
	for i, r := range results {
		data[i] = map[string]any{
			"period": r.Period,
			"count":  r.Count,
		}
	}

	return data, nil
}
