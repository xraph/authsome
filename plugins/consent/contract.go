// contract.go: Wires the consent plugin into the forge-dashboard
// contract surface via plugin.ContractContributor. The
// plugins/consent/contract subpackage declares its own ConsentStore
// interface (cycle avoidance); this file adapts the plugin's
// concrete consent.Store into that interface via a small shim.
package consent

import (
	"context"
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	consentcontract "github.com/xraph/authsome/plugins/consent/contract"

	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/dispatcher"
)

var _ plugin.ContractContributor = (*Plugin)(nil)

func (p *Plugin) RegisterContract(
	d *dispatcher.Dispatcher,
	reg contract.Registry,
	wreg contract.WardenRegistry,
	engine plugin.Engine,
) error {
	eng, ok := engine.(*authsome.Engine)
	if !ok {
		return fmt.Errorf("consent: contract registration requires *authsome.Engine, got %T", engine)
	}
	if p.store == nil {
		return fmt.Errorf("consent: plugin store not initialised; SetConsentStore before RegisterContract")
	}
	return consentcontract.Register(d, reg, wreg, consentcontract.Deps{
		Engine: eng,
		Store:  &storeShim{inner: p.store},
	})
}

// storeShim adapts the plugin's concrete Store to the
// consentcontract.ConsentStore interface. The shim mirrors the small
// subset of methods the contract uses, and rewrites the wire-shape
// Consent type into ConsentRow on the way through.
type storeShim struct {
	inner Store
}

func (s *storeShim) GrantConsent(ctx context.Context, c *consentcontract.ConsentRow) error {
	return s.inner.GrantConsent(ctx, rowToConsent(c))
}

func (s *storeShim) RevokeConsent(ctx context.Context, userID id.UserID, appID id.AppID, purpose string) error {
	return s.inner.RevokeConsent(ctx, userID, appID, purpose)
}

func (s *storeShim) ListConsents(ctx context.Context, q *consentcontract.ConsentQuery) ([]*consentcontract.ConsentRow, string, error) {
	consents, next, err := s.inner.ListConsents(ctx, &Query{
		UserID:  q.UserID,
		AppID:   q.AppID,
		Purpose: q.Purpose,
		Cursor:  q.Cursor,
		Limit:   q.Limit,
	})
	if err != nil {
		return nil, "", err
	}
	out := make([]*consentcontract.ConsentRow, 0, len(consents))
	for _, c := range consents {
		out = append(out, consentToRow(c))
	}
	return out, next, nil
}

func consentToRow(c *Consent) *consentcontract.ConsentRow {
	if c == nil {
		return nil
	}
	return &consentcontract.ConsentRow{
		ID:        c.ID,
		UserID:    c.UserID,
		AppID:     c.AppID,
		Purpose:   c.Purpose,
		Granted:   c.Granted,
		Version:   c.Version,
		IPAddress: c.IPAddress,
		GrantedAt: c.GrantedAt,
		RevokedAt: c.RevokedAt,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func rowToConsent(r *consentcontract.ConsentRow) *Consent {
	if r == nil {
		return nil
	}
	return &Consent{
		ID:        r.ID,
		UserID:    r.UserID,
		AppID:     r.AppID,
		Purpose:   r.Purpose,
		Granted:   r.Granted,
		Version:   r.Version,
		IPAddress: r.IPAddress,
		GrantedAt: r.GrantedAt,
		RevokedAt: r.RevokedAt,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}
