package retention

import (
	"context"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/settings"
)

// consentChecker is the slice of the consent plugin this one needs. Keeping it
// narrow and local means consent stays an optional dependency, resolved the
// way anomaly, geofence, impossibletravel and vpndetect all resolve geoip.
type consentChecker interface {
	HasConsent(ctx context.Context, userID id.UserID, appID id.AppID, purpose string) (bool, error)
}

// newConsentPolicy builds the consentPolicy closure against a live settings
// manager. Pulled out of OnInit so a test can drive the real closure through
// a real *settings.Manager without booting the rest of the plugin.
//
// mgr must not be nil; OnInit only calls this when engine.Settings() returned
// one. logger must not be nil either, for the same reason New already never
// leaves p.logger nil.
func newConsentPolicy(mgr *settings.Manager, logger log.Logger) func(ctx context.Context, appID id.AppID) (bool, string) {
	return func(ctx context.Context, appID id.AppID) (bool, string) {
		opts := settings.ResolveOpts{AppID: appID.String()}

		// settings.Get returns the registered code default (false, here)
		// alongside a non-nil error whenever the resolve itself failed, so
		// the value is not safe to read before the error is checked: doing
		// so would silently gate OFF during a settings-store outage, exactly
		// the failure this gate exists to avoid.
		require, err := settings.Get(ctx, mgr, SettingRequireConsent, opts)
		if err != nil {
			// An unreadable gate setting must not be read as "no gate".
			logger.Warn("retention: consent setting unreadable, gating on",
				log.String("app_id", appID.String()),
				log.String("error", err.Error()))
			return true, "marketing"
		}

		purpose, err := settings.Get(ctx, mgr, SettingConsentPurpose, opts)
		if err != nil || purpose == "" {
			purpose = "marketing"
		}
		return require, purpose
	}
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
