package retention

import (
	"context"

	"github.com/xraph/authsome/id"
)

// consentChecker is the slice of the consent plugin this one needs. Keeping it
// narrow and local means consent stays an optional dependency, resolved the
// way anomaly, geofence, impossibletravel and vpndetect all resolve geoip.
type consentChecker interface {
	HasConsent(ctx context.Context, userID id.UserID, appID id.AppID, purpose string) (bool, error)
}

// allowSend is the worker's AllowSend. It runs at delivery, not at enqueue, so
// a user who revokes between login and send has that revocation honoured.
//
// Every uncertain branch answers "do not send". Turning the gate on is an
// explicit request for consent to be established before data leaves, and
// treating an error or a missing consent plugin as permission would quietly
// defeat that.
func (p *Plugin) allowSend(ctx context.Context, j *Job) (bool, string) {
	require, purpose := false, "marketing"
	if p.consentPolicy != nil {
		require, purpose = p.consentPolicy(ctx, j.AppID)
	}
	if !require {
		return true, ""
	}
	if p.consent == nil {
		return false, "consent required but the consent plugin is unavailable"
	}
	granted, err := p.consent.HasConsent(ctx, j.UserID, j.AppID, purpose)
	if err != nil {
		return false, "consent lookup failed: " + err.Error()
	}
	if !granted {
		return false, "no active consent for purpose " + purpose
	}
	return true, ""
}
