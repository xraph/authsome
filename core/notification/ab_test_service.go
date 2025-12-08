package notification

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"math/big"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// ABTestService handles A/B testing operations
type ABTestService struct {
	repo Repository
}

// NewABTestService creates a new A/B test service
func NewABTestService(repo Repository) *ABTestService {
	return &ABTestService{repo: repo}
}

// CreateVariant creates a new A/B test variant for a template
func (s *ABTestService) CreateVariant(ctx context.Context, baseTemplateID xid.ID, variantName string, weight int, subject, body string) (*schema.NotificationTemplate, error) {
	// Get base template
	baseTemplate, err := s.repo.FindTemplateByID(ctx, baseTemplateID)
	if err != nil {
		return nil, fmt.Errorf("failed to find base template: %w", err)
	}
	if baseTemplate == nil {
		return nil, TemplateNotFound()
	}

	// Generate unique AB test group if base template doesn't have one
	abTestGroup := baseTemplate.ABTestGroup
	if abTestGroup == "" {
		abTestGroup = xid.New().String()
		// Update base template with AB test group
		baseTemplate.ABTestGroup = abTestGroup
		baseTemplate.ABTestEnabled = true
	}

	// Create variant template
	variant := &schema.NotificationTemplate{
		ID:             xid.New(),
		AppID:          baseTemplate.AppID,
		OrganizationID: baseTemplate.OrganizationID,
		TemplateKey:    baseTemplate.TemplateKey,
		Name:           variantName,
		Type:           baseTemplate.Type,
		Language:       baseTemplate.Language,
		Subject:        subject,
		Body:           body,
		Variables:      baseTemplate.Variables,
		Metadata:       baseTemplate.Metadata,
		Active:         true,
		ParentID:       &baseTemplateID,
		ABTestGroup:    abTestGroup,
		ABTestEnabled:  true,
		ABTestWeight:   weight,
	}

	if err := s.repo.CreateTemplate(ctx, variant); err != nil {
		return nil, fmt.Errorf("failed to create variant: %w", err)
	}

	return variant, nil
}

// SelectVariant selects a variant based on weighted distribution
func (s *ABTestService) SelectVariant(ctx context.Context, appID xid.ID, orgID *xid.ID, templateKey, notifType, language string) (*schema.NotificationTemplate, error) {
	// First, try to find templates with A/B testing enabled
	filter := &ListTemplatesFilter{
		AppID:    appID,
		Type:     (*NotificationType)(&notifType),
		Language: &language,
	}

	// Get all active templates for this key
	response, err := s.repo.ListTemplates(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	// Filter for templates matching key and with AB testing enabled
	var variants []*schema.NotificationTemplate
	for _, tmpl := range response.Data {
		if tmpl.TemplateKey == templateKey && tmpl.ABTestEnabled && tmpl.Active {
			// Check org scope
			if orgID != nil && tmpl.OrganizationID != nil && *tmpl.OrganizationID == *orgID {
				variants = append(variants, tmpl)
			} else if orgID == nil && tmpl.OrganizationID == nil {
				variants = append(variants, tmpl)
			}
		}
	}

	// If no variants found or AB testing not enabled, use standard resolution
	if len(variants) == 0 {
		return s.repo.FindTemplateByKeyOrgScoped(ctx, appID, orgID, templateKey, notifType, language)
	}

	// Calculate total weight
	totalWeight := 0
	for _, v := range variants {
		totalWeight += v.ABTestWeight
	}

	if totalWeight == 0 {
		// No weights set, return first variant
		return variants[0], nil
	}

	// Generate random number [0, totalWeight)
	randomBig, err := rand.Int(rand.Reader, big.NewInt(int64(totalWeight)))
	if err != nil {
		// Fall back to first variant on error
		return variants[0], nil
	}
	random := int(randomBig.Int64())

	// Select variant based on weight
	cumulative := 0
	for _, v := range variants {
		cumulative += v.ABTestWeight
		if random < cumulative {
			return v, nil
		}
	}

	// Fallback (should not reach here)
	return variants[0], nil
}

// GetABTestResults gets the performance results for all variants in an AB test
func (s *ABTestService) GetABTestResults(ctx context.Context, abTestGroup string) (*ABTestResults, error) {
	// This would query analytics data for all variants in the group
	// For now, return a placeholder
	results := &ABTestResults{
		ABTestGroup: abTestGroup,
		Variants:    []VariantPerformance{},
	}

	return results, nil
}

// DeclareWinner declares a winning variant and deactivates others
func (s *ABTestService) DeclareWinner(ctx context.Context, winnerID xid.ID, abTestGroup string) error {
	// Get all variants in the AB test group
	filter := &ListTemplatesFilter{}
	response, err := s.repo.ListTemplates(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}

	// Deactivate all variants in the group except the winner
	for _, tmpl := range response.Data {
		if tmpl.ABTestGroup == abTestGroup {
			if tmpl.ID == winnerID {
				// Winner: keep enabled with full weight
				if err := s.repo.UpdateProvider(ctx, tmpl.ID, tmpl.Metadata, true, true); err != nil {
					return fmt.Errorf("failed to update winner: %w", err)
				}
			} else {
				// Loser: disable AB testing
				active := false
				updateReq := &UpdateTemplateRequest{
					Active: &active,
				}
				if err := s.repo.UpdateTemplate(ctx, tmpl.ID, updateReq); err != nil {
					return fmt.Errorf("failed to deactivate variant: %w", err)
				}
			}
		}
	}

	return nil
}

// CalculateStatisticalSignificance calculates if one variant is statistically better than another
func (s *ABTestService) CalculateStatisticalSignificance(variant1, variant2 VariantPerformance) *StatisticalSignificance {
	// Simple z-test for proportion difference
	p1 := variant1.ConversionRate
	p2 := variant2.ConversionRate
	n1 := float64(variant1.TotalSent)
	n2 := float64(variant2.TotalSent)

	// Pooled proportion
	pPool := (p1*n1 + p2*n2) / (n1 + n2)

	// Standard error
	se := math.Sqrt(pPool * (1 - pPool) * (1/n1 + 1/n2))

	// Z-score
	zScore := 0.0
	if se > 0 {
		zScore = (p1 - p2) / se
	}

	// P-value (two-tailed test)
	// Simplified: for |z| > 1.96, p < 0.05 (95% confidence)
	pValue := 1.0
	if math.Abs(zScore) > 1.96 {
		pValue = 0.05
	} else if math.Abs(zScore) > 1.645 {
		pValue = 0.10
	}

	return &StatisticalSignificance{
		ZScore:            zScore,
		PValue:            pValue,
		IsSignificant:     pValue < 0.05,
		ConfidenceLevel:   0.95,
		DifferencePercent: (p1 - p2) * 100,
	}
}

// ABTestResults represents the results of an A/B test
type ABTestResults struct {
	ABTestGroup  string                   `json:"abTestGroup"`
	Variants     []VariantPerformance     `json:"variants"`
	Winner       *xid.ID                  `json:"winner,omitempty"`
	Significance *StatisticalSignificance `json:"significance,omitempty"`
}

// VariantPerformance represents performance metrics for a variant
type VariantPerformance struct {
	TemplateID     xid.ID  `json:"templateId"`
	TemplateName   string  `json:"templateName"`
	TotalSent      int64   `json:"totalSent"`
	TotalOpened    int64   `json:"totalOpened"`
	TotalClicked   int64   `json:"totalClicked"`
	TotalConverted int64   `json:"totalConverted"`
	OpenRate       float64 `json:"openRate"`
	ClickRate      float64 `json:"clickRate"`
	ConversionRate float64 `json:"conversionRate"`
}

// StatisticalSignificance represents statistical test results
type StatisticalSignificance struct {
	ZScore            float64 `json:"zScore"`
	PValue            float64 `json:"pValue"`
	IsSignificant     bool    `json:"isSignificant"`
	ConfidenceLevel   float64 `json:"confidenceLevel"`
	DifferencePercent float64 `json:"differencePercent"`
}
