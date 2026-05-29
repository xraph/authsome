// Package contract is the consent plugin's forge-dashboard contract
// surface. The plugin's Store is the source of truth; this package
// wraps it as four intents: list, userConsents, grant, revoke.
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

// ConsentRecord is the wire shape consumers see. Field names mirror
// the resource.list columns in the manifest — renames are wire breaks.
type ConsentRecord struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	AppID     string `json:"appId,omitempty"`
	Purpose   string `json:"purpose"`
	Granted   bool   `json:"granted"`
	Version   string `json:"version,omitempty"`
	IPAddress string `json:"ipAddress,omitempty"`
	GrantedAt string `json:"grantedAt,omitempty"`
	RevokedAt string `json:"revokedAt,omitempty"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// ConsentItem is what list+detail handlers project Consent records into.
// Defined as its own type so future fields (PolicyTitle, etc.) can be
// added without forcing every store call site to materialise them.
type ConsentItem = ConsentRecord

// ConsentList is the wire shape for list responses.
type ConsentList struct {
	Items      []ConsentItem `json:"items"`
	NextCursor string        `json:"nextCursor,omitempty"`
}

// ListConsentsInput is the input for consent.list / consent.userConsents.
// All fields are optional; an empty input lists the principal-app's
// records.
type ListConsentsInput struct {
	UserID  string `json:"userId,omitempty"`
	Purpose string `json:"purpose,omitempty"`
	Cursor  string `json:"cursor,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

// GrantConsentInput is the wire shape for consent.grant.
type GrantConsentInput struct {
	UserID    string `json:"userId"`
	Purpose   string `json:"purpose"`
	Version   string `json:"version,omitempty"`
	IPAddress string `json:"ipAddress,omitempty"`
}

// RevokeConsentInput is the wire shape for consent.revoke.
type RevokeConsentInput struct {
	UserID  string `json:"userId"`
	Purpose string `json:"purpose"`
}

type ackResponse struct {
	OK bool   `json:"ok"`
	ID string `json:"id,omitempty"`
}

// ConsentStore is the surface this package needs from the consent
// plugin. Mirroring consent.Store's relevant subset as a local
// interface keeps this package decoupled from the parent plugin
// (cycle avoidance).
type ConsentStore interface {
	GrantConsent(ctx context.Context, c *ConsentRow) error
	RevokeConsent(ctx context.Context, userID id.UserID, appID id.AppID, purpose string) error
	ListConsents(ctx context.Context, q *ConsentQuery) ([]*ConsentRow, string, error)
}

// ConsentRow + ConsentQuery mirror the consent package's exported
// types. The parent plugin's contract.go adapts its own concrete
// types into these via a small shim so we don't import the parent
// package here.
type ConsentRow struct {
	ID        id.ConsentID
	UserID    id.UserID
	AppID     id.AppID
	Purpose   string
	Granted   bool
	Version   string
	IPAddress string
	GrantedAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ConsentQuery struct {
	UserID  id.UserID
	AppID   id.AppID
	Purpose string
	Cursor  string
	Limit   int
}

type Deps struct {
	Engine *authsome.Engine
	Store  ConsentStore
}

func Register(
	d *dispatcher.Dispatcher,
	reg contract.Registry,
	wreg contract.WardenRegistry,
	deps Deps,
) error {
	if deps.Engine == nil {
		return fmt.Errorf("consent/contract: Engine is required")
	}
	if deps.Store == nil {
		return fmt.Errorf("consent/contract: Store is required")
	}
	m, err := loader.Load(bytes.NewReader(manifestYAML), "consent/contract/manifest.yaml")
	if err != nil {
		return fmt.Errorf("consent/contract: load manifest: %w", err)
	}
	if err := loader.Validate(m, wreg); err != nil {
		return fmt.Errorf("consent/contract: validate manifest: %w", err)
	}
	if err := reg.Register(m); err != nil {
		return fmt.Errorf("consent/contract: register manifest: %w", err)
	}

	const c = "consent"
	if err := dispatcher.RegisterQuery(d, c, "consent.list", 1, listHandler(deps)); err != nil {
		return fmt.Errorf("consent/contract: register consent.list: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "consent.userConsents", 1, listHandler(deps)); err != nil {
		return fmt.Errorf("consent/contract: register consent.userConsents: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "consent.grant", 1, grantHandler(deps)); err != nil {
		return fmt.Errorf("consent/contract: register consent.grant: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "consent.revoke", 1, revokeHandler(deps)); err != nil {
		return fmt.Errorf("consent/contract: register consent.revoke: %w", err)
	}
	return nil
}

// ────────────────────────────────────────────────────────────────────
// Handlers
// ────────────────────────────────────────────────────────────────────

func listHandler(deps Deps) func(ctx context.Context, in ListConsentsInput, p contract.Principal) (ConsentList, error) {
	return func(ctx context.Context, in ListConsentsInput, p contract.Principal) (ConsentList, error) {
		if deps.Engine == nil || deps.Store == nil {
			return ConsentList{}, unavailable()
		}
		q := &ConsentQuery{
			AppID:   authcontract.AppIDFromPrincipal(p, deps.Engine),
			Purpose: strings.TrimSpace(in.Purpose),
			Cursor:  in.Cursor,
			Limit:   in.Limit,
		}
		if uidStr := strings.TrimSpace(in.UserID); uidStr != "" {
			uid, err := id.ParseUserID(uidStr)
			if err != nil {
				return ConsentList{}, badReq("invalid user id: " + err.Error())
			}
			q.UserID = uid
		}
		if q.Limit <= 0 {
			q.Limit = 100
		}
		rows, next, err := deps.Store.ListConsents(ctx, q)
		if err != nil {
			return ConsentList{}, mapErr(err)
		}
		out := ConsentList{Items: make([]ConsentItem, 0, len(rows)), NextCursor: next}
		for _, r := range rows {
			out.Items = append(out.Items, projectConsent(r))
		}
		return out, nil
	}
}

func grantHandler(deps Deps) func(ctx context.Context, in GrantConsentInput, p contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in GrantConsentInput, p contract.Principal) (ackResponse, error) {
		if deps.Engine == nil || deps.Store == nil {
			return ackResponse{}, unavailable()
		}
		uid, err := id.ParseUserID(strings.TrimSpace(in.UserID))
		if err != nil {
			return ackResponse{}, badReq("invalid user id: " + err.Error())
		}
		purpose := strings.TrimSpace(in.Purpose)
		if purpose == "" {
			return ackResponse{}, badReq("purpose is required")
		}
		row := &ConsentRow{
			ID:        id.NewConsentID(),
			UserID:    uid,
			AppID:     authcontract.AppIDFromPrincipal(p, deps.Engine),
			Purpose:   purpose,
			Granted:   true,
			Version:   in.Version,
			IPAddress: in.IPAddress,
			GrantedAt: time.Now().UTC(),
		}
		if err := deps.Store.GrantConsent(ctx, row); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: row.ID.String()}, nil
	}
}

func revokeHandler(deps Deps) func(ctx context.Context, in RevokeConsentInput, p contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in RevokeConsentInput, p contract.Principal) (ackResponse, error) {
		if deps.Engine == nil || deps.Store == nil {
			return ackResponse{}, unavailable()
		}
		uid, err := id.ParseUserID(strings.TrimSpace(in.UserID))
		if err != nil {
			return ackResponse{}, badReq("invalid user id: " + err.Error())
		}
		purpose := strings.TrimSpace(in.Purpose)
		if purpose == "" {
			return ackResponse{}, badReq("purpose is required")
		}
		appID := authcontract.AppIDFromPrincipal(p, deps.Engine)
		if err := deps.Store.RevokeConsent(ctx, uid, appID, purpose); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true}, nil
	}
}

// ────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────

func projectConsent(c *ConsentRow) ConsentItem {
	if c == nil {
		return ConsentItem{}
	}
	out := ConsentItem{
		ID:        c.ID.String(),
		UserID:    c.UserID.String(),
		AppID:     c.AppID.String(),
		Purpose:   c.Purpose,
		Granted:   c.Granted,
		Version:   c.Version,
		IPAddress: c.IPAddress,
		GrantedAt: c.GrantedAt.UTC().Format(time.RFC3339),
		CreatedAt: c.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: c.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if c.RevokedAt != nil {
		out.RevokedAt = c.RevokedAt.UTC().Format(time.RFC3339)
	}
	return out
}

func badReq(msg string) error {
	return &contract.Error{Code: contract.CodeBadRequest, Message: msg}
}

func unavailable() error {
	return &contract.Error{Code: contract.CodeUnavailable, Message: "consent plugin not enabled"}
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
