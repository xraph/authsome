// contract.go: Wires the waitlist plugin into the forge-dashboard
// contract surface via plugin.ContractContributor. The subpackage
// declares its own EntryRow / EntryQuery / EntryListRows / WaitlistStore
// interface (cycle avoidance); this file adapts the plugin's
// concrete Store via a small shim.
package waitlist

import (
	"context"
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	wlcontract "github.com/xraph/authsome/plugins/waitlist/contract"

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
		return fmt.Errorf("waitlist: contract registration requires *authsome.Engine, got %T", engine)
	}
	if p.store == nil {
		return fmt.Errorf("waitlist: plugin store not initialised; SetWaitlistStore before RegisterContract")
	}
	return wlcontract.Register(d, reg, wreg, wlcontract.Deps{
		Engine: eng,
		Store:  &storeShim{inner: p.store},
	})
}

type storeShim struct {
	inner Store
}

func (s *storeShim) GetEntry(ctx context.Context, eid id.WaitlistID) (*wlcontract.EntryRow, error) {
	e, err := s.inner.GetEntry(ctx, eid)
	if err != nil {
		return nil, err
	}
	return entryToRow(e), nil
}

func (s *storeShim) UpdateEntryStatus(ctx context.Context, eid id.WaitlistID, status, note string) error {
	return s.inner.UpdateEntryStatus(ctx, eid, WaitlistStatus(status), note)
}

func (s *storeShim) ListEntries(ctx context.Context, q *wlcontract.EntryQuery) (*wlcontract.EntryListRows, error) {
	list, err := s.inner.ListEntries(ctx, &WaitlistQuery{
		AppID:  q.AppID,
		Email:  q.Email,
		Status: WaitlistStatus(q.Status),
		Cursor: q.Cursor,
		Limit:  q.Limit,
	})
	if err != nil {
		return nil, err
	}
	out := &wlcontract.EntryListRows{
		Entries:    make([]*wlcontract.EntryRow, 0, len(list.Entries)),
		Total:      list.Total,
		NextCursor: list.NextCursor,
	}
	for _, e := range list.Entries {
		out.Entries = append(out.Entries, entryToRow(e))
	}
	return out, nil
}

func (s *storeShim) CountByStatus(ctx context.Context, appID id.AppID) (int, int, int, error) {
	return s.inner.CountByStatus(ctx, appID)
}

func (s *storeShim) DeleteEntry(ctx context.Context, eid id.WaitlistID) error {
	return s.inner.DeleteEntry(ctx, eid)
}

func entryToRow(e *WaitlistEntry) *wlcontract.EntryRow {
	if e == nil {
		return nil
	}
	return &wlcontract.EntryRow{
		ID:        e.ID,
		AppID:     e.AppID,
		Email:     e.Email,
		Name:      e.Name,
		Status:    string(e.Status),
		UserID:    e.UserID,
		IPAddress: e.IPAddress,
		Note:      e.Note,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}
