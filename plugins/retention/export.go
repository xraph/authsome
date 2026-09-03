package retention

import (
	"context"
	"fmt"

	"github.com/xraph/authsome/id"
)

// ExportUserData implements plugin.DataExportContributor. It returns every
// CRM ref held for the user, across all apps and providers, so a data
// subject can see exactly which external systems hold a copy of their
// record.
//
// A nil store (the plugin registered but never initialised) returns an
// empty payload and no error: a user asking for their data should not get a
// 500 because a plugin was misconfigured.
func (p *Plugin) ExportUserData(ctx context.Context, userID id.UserID) (category string, data any, err error) {
	if p.store == nil {
		return "retention", nil, nil
	}

	refs, err := p.store.ListRefsForUser(ctx, userID)
	if err != nil {
		return "", nil, fmt.Errorf("retention: export user data: %w", err)
	}
	if len(refs) == 0 {
		return "retention", nil, nil
	}
	return "retention", refs, nil
}
