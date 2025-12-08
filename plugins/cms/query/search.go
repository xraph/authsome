package query

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/xid"
	"github.com/uptrace/bun"

	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/authsome/plugins/cms/schema"
)

// SearchConfig configures full-text search behavior
type SearchConfig struct {
	// Query is the search query string
	Query string
	// Fields to search (if empty, searches all text fields)
	Fields []string
	// Language for text search (default: english)
	Language string
	// IncludeHighlights returns highlighted snippets
	IncludeHighlights bool
	// HighlightStartTag is the tag for highlight start (default: <mark>)
	HighlightStartTag string
	// HighlightEndTag is the tag for highlight end (default: </mark>)
	HighlightEndTag string
	// MinScore filters out low-relevance results
	MinScore float64
}

// DefaultSearchConfig returns the default search configuration
func DefaultSearchConfig() *SearchConfig {
	return &SearchConfig{
		Language:          "english",
		IncludeHighlights: false,
		HighlightStartTag: "<mark>",
		HighlightEndTag:   "</mark>",
		MinScore:          0,
	}
}

// SearchResult represents a search result with ranking
type SearchResult struct {
	Entry      *schema.ContentEntry
	Score      float64
	Highlights map[string]string
}

// Searcher handles full-text search operations
type Searcher struct {
	db *bun.DB
}

// NewSearcher creates a new searcher
func NewSearcher(db *bun.DB) *Searcher {
	return &Searcher{db: db}
}

// Search performs a full-text search on content entries
func (s *Searcher) Search(ctx context.Context, contentTypeID xid.ID, config *SearchConfig, page, pageSize int) ([]*SearchResult, int, error) {
	if config == nil || config.Query == "" {
		return nil, 0, nil
	}

	// Normalize search query
	searchQuery := normalizeSearchQuery(config.Query)
	if searchQuery == "" {
		return nil, 0, nil
	}

	language := config.Language
	if language == "" {
		language = "english"
	}

	// Build the search query using PostgreSQL's full-text search
	// We search in the JSONB data field by converting it to text
	query := s.db.NewSelect().
		Model((*schema.ContentEntry)(nil)).
		Where("content_type_id = ?", contentTypeID).
		Where("deleted_at IS NULL")

	// Create text search vector from specified fields or all data
	if len(config.Fields) > 0 {
		// Search specific fields
		fieldConditions := make([]string, 0, len(config.Fields))
		for _, field := range config.Fields {
			// Use JSONB ->> operator to get text value
			fieldConditions = append(fieldConditions, fmt.Sprintf(
				"to_tsvector('%s', COALESCE(data ->> '%s', '')) @@ plainto_tsquery('%s', ?)",
				language, field, language,
			))
		}
		query = query.Where("("+strings.Join(fieldConditions, " OR ")+")", searchQuery)
	} else {
		// Search all text in data JSONB
		query = query.Where(
			fmt.Sprintf("to_tsvector('%s', data::text) @@ plainto_tsquery('%s', ?)", language, language),
			searchQuery,
		)
	}

	// Count total results
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, core.ErrInternalError("failed to count search results", err)
	}

	// Add ranking and pagination
	// ts_rank returns a relevance score
	if len(config.Fields) > 0 {
		// Rank by first searchable field
		query = query.ColumnExpr(fmt.Sprintf(
			"ts_rank(to_tsvector('%s', COALESCE(data ->> '%s', '')), plainto_tsquery('%s', ?)) as rank",
			language, config.Fields[0], language,
		), searchQuery)
	} else {
		query = query.ColumnExpr(fmt.Sprintf(
			"ts_rank(to_tsvector('%s', data::text), plainto_tsquery('%s', ?)) as rank",
			language, language,
		), searchQuery)
	}

	query = query.Order("rank DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize)

	// Execute query
	var entries []*schema.ContentEntry
	err = query.Scan(ctx, &entries)
	if err != nil {
		return nil, 0, core.ErrInternalError("failed to execute search", err)
	}

	// Convert to search results
	results := make([]*SearchResult, len(entries))
	for i, entry := range entries {
		results[i] = &SearchResult{
			Entry:      entry,
			Score:      1.0, // Score is already used for ordering
			Highlights: make(map[string]string),
		}

		// Generate highlights if requested
		if config.IncludeHighlights {
			results[i].Highlights = s.generateHighlights(entry, config)
		}
	}

	return results, total, nil
}

// SearchAll searches across all content types
func (s *Searcher) SearchAll(ctx context.Context, appID, envID xid.ID, config *SearchConfig, page, pageSize int) ([]*SearchResult, int, error) {
	if config == nil || config.Query == "" {
		return nil, 0, nil
	}

	searchQuery := normalizeSearchQuery(config.Query)
	if searchQuery == "" {
		return nil, 0, nil
	}

	language := config.Language
	if language == "" {
		language = "english"
	}

	query := s.db.NewSelect().
		Model((*schema.ContentEntry)(nil)).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("deleted_at IS NULL").
		Where(
			fmt.Sprintf("to_tsvector('%s', data::text) @@ plainto_tsquery('%s', ?)", language, language),
			searchQuery,
		)

	// Count total
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, core.ErrInternalError("failed to count search results", err)
	}

	// Add ranking and pagination
	query = query.ColumnExpr(fmt.Sprintf(
		"ts_rank(to_tsvector('%s', data::text), plainto_tsquery('%s', ?)) as rank",
		language, language,
	), searchQuery).
		Order("rank DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize)

	var entries []*schema.ContentEntry
	err = query.Scan(ctx, &entries)
	if err != nil {
		return nil, 0, core.ErrInternalError("failed to execute search", err)
	}

	results := make([]*SearchResult, len(entries))
	for i, entry := range entries {
		results[i] = &SearchResult{
			Entry:      entry,
			Score:      1.0,
			Highlights: make(map[string]string),
		}
		if config.IncludeHighlights {
			results[i].Highlights = s.generateHighlights(entry, config)
		}
	}

	return results, total, nil
}

// generateHighlights creates highlighted snippets for search results
func (s *Searcher) generateHighlights(entry *schema.ContentEntry, config *SearchConfig) map[string]string {
	highlights := make(map[string]string)
	if entry.Data == nil {
		return highlights
	}

	startTag := config.HighlightStartTag
	endTag := config.HighlightEndTag
	if startTag == "" {
		startTag = "<mark>"
	}
	if endTag == "" {
		endTag = "</mark>"
	}

	searchTerms := strings.Fields(strings.ToLower(config.Query))

	// Generate highlights for each field
	fieldsToCheck := config.Fields
	if len(fieldsToCheck) == 0 {
		// Check all string fields
		for field := range entry.Data {
			fieldsToCheck = append(fieldsToCheck, field)
		}
	}

	for _, field := range fieldsToCheck {
		if val, ok := entry.Data[field]; ok {
			if strVal, ok := val.(string); ok && strVal != "" {
				highlighted := highlightText(strVal, searchTerms, startTag, endTag)
				if highlighted != strVal {
					highlights[field] = highlighted
				}
			}
		}
	}

	return highlights
}

// highlightText highlights search terms in text
func highlightText(text string, searchTerms []string, startTag, endTag string) string {
	lowerText := strings.ToLower(text)
	result := text

	for _, term := range searchTerms {
		term = strings.ToLower(term)
		startIdx := 0
		for {
			idx := strings.Index(strings.ToLower(result[startIdx:]), term)
			if idx == -1 {
				break
			}
			actualIdx := startIdx + idx
			// Insert highlight tags
			result = result[:actualIdx] + startTag + result[actualIdx:actualIdx+len(term)] + endTag + result[actualIdx+len(term):]
			startIdx = actualIdx + len(startTag) + len(term) + len(endTag)
		}
	}

	// Truncate to snippet if too long
	maxLen := 200
	if len(result) > maxLen {
		// Find the first highlight
		highlightIdx := strings.Index(result, startTag)
		if highlightIdx > 50 {
			// Start before the highlight
			result = "..." + result[highlightIdx-30:]
		}
		if len(result) > maxLen {
			result = result[:maxLen] + "..."
		}
	}

	_ = lowerText // Silence unused warning
	return result
}

// normalizeSearchQuery cleans and normalizes a search query
func normalizeSearchQuery(query string) string {
	// Trim and collapse whitespace
	query = strings.TrimSpace(query)
	query = strings.Join(strings.Fields(query), " ")

	// Remove special characters that could break PostgreSQL text search
	// Keep alphanumeric, spaces, and common punctuation
	var result strings.Builder
	for _, r := range query {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ' || r == '-' || r == '\'' {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// SuggestSearch returns search suggestions based on existing content
func (s *Searcher) SuggestSearch(ctx context.Context, contentTypeID xid.ID, prefix string, limit int) ([]string, error) {
	if prefix == "" || limit <= 0 {
		return nil, nil
	}

	prefix = normalizeSearchQuery(prefix)
	if prefix == "" {
		return nil, nil
	}

	// Query distinct values that start with the prefix
	// This is a simplified suggestion - for production you might want trigram similarity
	var suggestions []string
	err := s.db.NewSelect().
		Model((*schema.ContentEntry)(nil)).
		ColumnExpr("DISTINCT data::text").
		Where("content_type_id = ?", contentTypeID).
		Where("deleted_at IS NULL").
		Where("data::text ILIKE ?", prefix+"%").
		Limit(limit).
		Scan(ctx, &suggestions)

	if err != nil {
		return nil, core.ErrInternalError("failed to get search suggestions", err)
	}

	return suggestions, nil
}
