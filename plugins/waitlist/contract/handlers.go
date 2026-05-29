// Package contract is the waitlist plugin's forge-dashboard contract
// surface. Six intents wrap the plugin's Store: list, detail, approve,
// reject, delete, counts.
package contract

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"

	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/dispatcher"
	"github.com/xraph/forge/extensions/dashboard/contract/loader"

	authcontract "github.com/xraph/authsome/extension/contract"
)

//go:embed manifest.yaml
var manifestYAML []byte

// EntrySummary is the wire shape for waitlist.list rows.
type EntrySummary struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name,omitempty"`
	Status    string `json:"status"`
	UserID    string `json:"userId,omitempty"`
	IPAddress string `json:"ipAddress,omitempty"`
	Note      string `json:"note,omitempty"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// EntryList wraps a paginated list response.
type EntryList struct {
	Entries    []EntrySummary `json:"entries"`
	Total      int            `json:"total,omitempty"`
	NextCursor string         `json:"nextCursor,omitempty"`
}

type ListEntriesInput struct {
	Email  string `json:"email,omitempty"`
	Status string `json:"status,omitempty"`
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type GetEntryInput struct {
	ID string `json:"id"`
}

type ApproveInput struct {
	ID   string `json:"id"`
	Note string `json:"note,omitempty"`
}

type RejectInput struct {
	ID   string `json:"id"`
	Note string `json:"note,omitempty"`
}

type DeleteInput struct {
	ID string `json:"id"`
}

type CountsResponse struct {
	Pending  int `json:"pending"`
	Approved int `json:"approved"`
	Rejected int `json:"rejected"`
}

type ackResponse struct {
	OK bool   `json:"ok"`
	ID string `json:"id,omitempty"`
}

// EntryRow + EntryQuery + EntryList are the package-internal wire
// types the plugin's Store fronts. They mirror waitlist.WaitlistEntry
// / WaitlistQuery / WaitlistList so the contract subpackage stays
// decoupled from the parent (cycle avoidance).
type EntryRow struct {
	ID        id.WaitlistID
	AppID     id.AppID
	Email     string
	Name      string
	Status    string
	UserID    *id.UserID
	IPAddress string
	Note      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type EntryQuery struct {
	AppID  id.AppID
	Email  string
	Status string
	Cursor string
	Limit  int
}

type EntryListRows struct {
	Entries    []*EntryRow
	Total      int
	NextCursor string
}

// WaitlistStore is the surface this package needs from the waitlist
// plugin's Store.
type WaitlistStore interface {
	GetEntry(ctx context.Context, id id.WaitlistID) (*EntryRow, error)
	UpdateEntryStatus(ctx context.Context, id id.WaitlistID, status, note string) error
	ListEntries(ctx context.Context, q *EntryQuery) (*EntryListRows, error)
	CountByStatus(ctx context.Context, appID id.AppID) (pending, approved, rejected int, err error)
	DeleteEntry(ctx context.Context, id id.WaitlistID) error
}

type Deps struct {
	Engine *authsome.Engine
	Store  WaitlistStore
}

func Register(
	d *dispatcher.Dispatcher,
	reg contract.Registry,
	wreg contract.WardenRegistry,
	deps Deps,
) error {
	if deps.Engine == nil {
		return fmt.Errorf("waitlist/contract: Engine is required")
	}
	if deps.Store == nil {
		return fmt.Errorf("waitlist/contract: Store is required")
	}
	m, err := loader.Load(bytes.NewReader(manifestYAML), "waitlist/contract/manifest.yaml")
	if err != nil {
		return fmt.Errorf("waitlist/contract: load manifest: %w", err)
	}
	if err := loader.Validate(m, wreg); err != nil {
		return fmt.Errorf("waitlist/contract: validate manifest: %w", err)
	}
	if err := reg.Register(m); err != nil {
		return fmt.Errorf("waitlist/contract: register manifest: %w", err)
	}

	const c = "waitlist"
	if err := dispatcher.RegisterQuery(d, c, "waitlist.list", 1, listHandler(deps)); err != nil {
		return fmt.Errorf("waitlist/contract: register waitlist.list: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "waitlist.detail", 1, detailHandler(deps)); err != nil {
		return fmt.Errorf("waitlist/contract: register waitlist.detail: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "waitlist.approve", 1, approveHandler(deps)); err != nil {
		return fmt.Errorf("waitlist/contract: register waitlist.approve: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "waitlist.reject", 1, rejectHandler(deps)); err != nil {
		return fmt.Errorf("waitlist/contract: register waitlist.reject: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "waitlist.delete", 1, deleteHandler(deps)); err != nil {
		return fmt.Errorf("waitlist/contract: register waitlist.delete: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "waitlist.counts", 1, countsHandler(deps)); err != nil {
		return fmt.Errorf("waitlist/contract: register waitlist.counts: %w", err)
	}
	return nil
}

// ────────────────────────────────────────────────────────────────────
// Handlers
// ────────────────────────────────────────────────────────────────────

func listHandler(deps Deps) func(ctx context.Context, in ListEntriesInput, p contract.Principal) (EntryList, error) {
	return func(ctx context.Context, in ListEntriesInput, p contract.Principal) (EntryList, error) {
		if deps.Engine == nil || deps.Store == nil {
			return EntryList{}, unavailable()
		}
		q := &EntryQuery{
			AppID:  authcontract.AppIDFromPrincipal(p, deps.Engine),
			Email:  strings.TrimSpace(in.Email),
			Status: strings.TrimSpace(in.Status),
			Cursor: in.Cursor,
			Limit:  in.Limit,
		}
		if q.Limit <= 0 {
			q.Limit = 100
		}
		list, err := deps.Store.ListEntries(ctx, q)
		if err != nil {
			return EntryList{}, mapErr(err)
		}
		out := EntryList{
			Entries:    make([]EntrySummary, 0, len(list.Entries)),
			Total:      list.Total,
			NextCursor: list.NextCursor,
		}
		for _, e := range list.Entries {
			out.Entries = append(out.Entries, projectEntry(e))
		}
		return out, nil
	}
}

func detailHandler(deps Deps) func(ctx context.Context, in GetEntryInput, _ contract.Principal) (EntrySummary, error) {
	return func(ctx context.Context, in GetEntryInput, _ contract.Principal) (EntrySummary, error) {
		if deps.Engine == nil || deps.Store == nil {
			return EntrySummary{}, unavailable()
		}
		eid, err := parseWaitlistID(in.ID)
		if err != nil {
			return EntrySummary{}, err
		}
		row, err := deps.Store.GetEntry(ctx, eid)
		if err != nil {
			return EntrySummary{}, mapErr(err)
		}
		return projectEntry(row), nil
	}
}

func approveHandler(deps Deps) func(ctx context.Context, in ApproveInput, _ contract.Principal) (ackResponse, error) {
	return updateStatusHandler(deps, "approved", func(in ApproveInput) (string, string) {
		return in.ID, in.Note
	})
}

func rejectHandler(deps Deps) func(ctx context.Context, in RejectInput, _ contract.Principal) (ackResponse, error) {
	return updateStatusHandler(deps, "rejected", func(in RejectInput) (string, string) {
		return in.ID, in.Note
	})
}

func updateStatusHandler[T any](deps Deps, status string, unpack func(T) (string, string)) func(ctx context.Context, in T, _ contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in T, _ contract.Principal) (ackResponse, error) {
		if deps.Engine == nil || deps.Store == nil {
			return ackResponse{}, unavailable()
		}
		idStr, note := unpack(in)
		eid, err := parseWaitlistID(idStr)
		if err != nil {
			return ackResponse{}, err
		}
		if err := deps.Store.UpdateEntryStatus(ctx, eid, status, note); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: eid.String()}, nil
	}
}

func deleteHandler(deps Deps) func(ctx context.Context, in DeleteInput, _ contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in DeleteInput, _ contract.Principal) (ackResponse, error) {
		if deps.Engine == nil || deps.Store == nil {
			return ackResponse{}, unavailable()
		}
		eid, err := parseWaitlistID(in.ID)
		if err != nil {
			return ackResponse{}, err
		}
		if err := deps.Store.DeleteEntry(ctx, eid); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: eid.String()}, nil
	}
}

func countsHandler(deps Deps) func(ctx context.Context, _ struct{}, p contract.Principal) (CountsResponse, error) {
	return func(ctx context.Context, _ struct{}, p contract.Principal) (CountsResponse, error) {
		if deps.Engine == nil || deps.Store == nil {
			return CountsResponse{}, unavailable()
		}
		pending, approved, rejected, err := deps.Store.CountByStatus(ctx, authcontract.AppIDFromPrincipal(p, deps.Engine))
		if err != nil {
			return CountsResponse{}, mapErr(err)
		}
		return CountsResponse{Pending: pending, Approved: approved, Rejected: rejected}, nil
	}
}

// ────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────

func projectEntry(e *EntryRow) EntrySummary {
	if e == nil {
		return EntrySummary{}
	}
	out := EntrySummary{
		ID:        e.ID.String(),
		Email:     e.Email,
		Name:      e.Name,
		Status:    e.Status,
		IPAddress: e.IPAddress,
		Note:      e.Note,
		CreatedAt: e.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: e.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if e.UserID != nil {
		out.UserID = e.UserID.String()
	}
	return out
}

func parseWaitlistID(s string) (id.WaitlistID, error) {
	if strings.TrimSpace(s) == "" {
		return id.WaitlistID{}, badReq("id is required")
	}
	wid, err := id.ParseWaitlistID(s)
	if err != nil {
		return id.WaitlistID{}, badReq("invalid waitlist id: " + err.Error())
	}
	return wid, nil
}

func badReq(msg string) error {
	return &contract.Error{Code: contract.CodeBadRequest, Message: msg}
}

func unavailable() error {
	return &contract.Error{Code: contract.CodeUnavailable, Message: "waitlist plugin not enabled"}
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if ce, ok := err.(*contract.Error); ok {
		return ce
	}
	return &contract.Error{Code: contract.CodeInternal, Message: err.Error()}
}
