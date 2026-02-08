package bridge

import (
	"time"

	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forgeui/bridge"
)

// =============================================================================
// Input/Output Types
// =============================================================================

// GetStatsInput is the input for getting overall statistics
type GetStatsInput struct {
	AppID  string `json:"appId"`
	Period string `json:"period,omitempty"` // today, week, month, year, all
}

// GetStatsOutput is the output for getting overall statistics
type GetStatsOutput struct {
	Data OverallStatsDTO `json:"data"`
}

// OverallStatsDTO represents overall OAuth/OIDC statistics
type OverallStatsDTO struct {
	ClientCount          int64           `json:"clientCount"`
	ActiveTokens         int64           `json:"activeTokens"`
	TotalTokensIssued    int64           `json:"totalTokensIssued"`
	TotalUsers           int64           `json:"totalUsers"`
	ActiveDeviceCodes    int64           `json:"activeDeviceCodes"`
	TokensByType         TokensByTypeDTO `json:"tokensByType"`
	TokensIssuedOverTime []TimeSeriesDTO `json:"tokensIssuedOverTime"`
	TopClients           []TopClientDTO  `json:"topClients"`
}

// TokensByTypeDTO represents token counts by type
type TokensByTypeDTO struct {
	AccessTokens  int64 `json:"accessTokens"`
	RefreshTokens int64 `json:"refreshTokens"`
	IDTokens      int64 `json:"idTokens"`
}

// TimeSeriesDTO represents a time series data point
type TimeSeriesDTO struct {
	Timestamp time.Time `json:"timestamp"`
	Count     int64     `json:"count"`
}

// TopClientDTO represents a client with token count
type TopClientDTO struct {
	ClientID   string `json:"clientId"`
	ClientName string `json:"clientName"`
	TokenCount int64  `json:"tokenCount"`
}

// =============================================================================
// Bridge Functions
// =============================================================================

// GetStats retrieves overall OAuth/OIDC statistics
func (bm *BridgeManager) GetStats(ctx bridge.Context, input GetStatsInput) (*GetStatsOutput, error) {
	goCtx, _, appID, err := bm.buildContextWithAppID(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Get environment ID
	envID, _ := contexts.GetEnvironmentID(goCtx)

	// Determine time range based on period
	var since time.Time
	now := time.Now()
	switch input.Period {
	case "today":
		since = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "week":
		since = now.AddDate(0, 0, -7)
	case "month":
		since = now.AddDate(0, -1, 0)
	case "year":
		since = now.AddDate(-1, 0, 0)
	case "all", "":
		since = time.Time{} // Beginning of time
	default:
		return nil, errs.BadRequest("invalid period: must be 'today', 'week', 'month', 'year', or 'all'")
	}

	// Get client count
	clientCount, err := bm.clientRepo.CountByAppAndEnv(goCtx, appID, envID)
	if err != nil {
		clientCount = 0
	}

	// Get active tokens count
	activeTokens, err := bm.tokenRepo.CountActiveByApp(goCtx, appID)
	if err != nil {
		activeTokens = 0
	}

	// Get total tokens issued (in period)
	var totalTokensIssued int64
	if since.IsZero() {
		totalTokensIssued, _ = bm.tokenRepo.CountByApp(goCtx, appID)
	} else {
		totalTokensIssued, _ = bm.tokenRepo.CountByAppSince(goCtx, appID, since)
	}

	// Get unique users count
	totalUsers, err := bm.tokenRepo.CountUniqueUsersByApp(goCtx, appID)
	if err != nil {
		totalUsers = 0
	}

	// Get active device codes count
	var activeDeviceCodes int64
	if bm.service.GetDeviceFlowService() != nil {
		activeDeviceCodes, _ = bm.deviceCodeRepo.CountByAppEnvAndStatus(goCtx, appID, envID, "pending")
	}

	// Get tokens by type
	tokensByType := TokensByTypeDTO{
		AccessTokens:  0,
		RefreshTokens: 0,
		IDTokens:      0,
	}
	accessCount, _ := bm.tokenRepo.CountByAppAndType(goCtx, appID, "access_token")
	refreshCount, _ := bm.tokenRepo.CountByAppAndType(goCtx, appID, "refresh_token")
	idCount, _ := bm.tokenRepo.CountByAppAndType(goCtx, appID, "id_token")
	tokensByType.AccessTokens = accessCount
	tokensByType.RefreshTokens = refreshCount
	tokensByType.IDTokens = idCount

	// Get tokens issued over time (last 30 days, daily)
	tokensOverTime := make([]TimeSeriesDTO, 0)
	if input.Period == "month" || input.Period == "all" || input.Period == "" {
		for i := 29; i >= 0; i-- {
			day := now.AddDate(0, 0, -i)
			dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
			dayEnd := dayStart.AddDate(0, 0, 1)

			count, _ := bm.tokenRepo.CountByAppBetween(goCtx, appID, dayStart, dayEnd)
			tokensOverTime = append(tokensOverTime, TimeSeriesDTO{
				Timestamp: dayStart,
				Count:     count,
			})
		}
	}

	// Get top clients by token count
	topClients := make([]TopClientDTO, 0)
	clients, _ := bm.clientRepo.ListByAppAndEnv(goCtx, appID, envID, 1, 5) // Top 5
	for _, client := range clients {
		tokenCount, _ := bm.tokenRepo.CountByClientID(goCtx, client.ClientID)
		topClients = append(topClients, TopClientDTO{
			ClientID:   client.ClientID,
			ClientName: client.Name,
			TokenCount: tokenCount,
		})
	}

	stats := OverallStatsDTO{
		ClientCount:          clientCount,
		ActiveTokens:         activeTokens,
		TotalTokensIssued:    totalTokensIssued,
		TotalUsers:           totalUsers,
		ActiveDeviceCodes:    activeDeviceCodes,
		TokensByType:         tokensByType,
		TokensIssuedOverTime: tokensOverTime,
		TopClients:           topClients,
	}

	return &GetStatsOutput{
		Data: stats,
	}, nil
}
