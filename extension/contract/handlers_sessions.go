// handlers_sessions.go: Phase C.3 — Sessions dashboard.
//
// sessions.list takes a userID (typically supplied by the parent users
// row when the page is rendered as a drill-down). For the standalone
// /sessions index page, callers pass the userID explicitly or leave
// blank to receive an empty list — the engine doesn't expose a global
// session listing.
package contract

import (
	"context"
	"strings"
	"time"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"

	"github.com/xraph/forge/extensions/dashboard/contract"
)

type SessionSummary struct {
	ID             string `json:"id"`
	UserID         string `json:"userId"`
	IPAddress      string `json:"ipAddress,omitempty"`
	UserAgent      string `json:"userAgent,omitempty"`
	LastActivityAt string `json:"lastActivityAt,omitempty"`
	ExpiresAt      string `json:"expiresAt"`
	CreatedAt      string `json:"createdAt"`
}

type SessionDetail struct {
	SessionSummary
	AppID                 string `json:"appId,omitempty"`
	EnvID                 string `json:"envId,omitempty"`
	OrgID                 string `json:"orgId,omitempty"`
	DeviceID              string `json:"deviceId,omitempty"`
	ImpersonatedBy        string `json:"impersonatedBy,omitempty"`
	RefreshTokenExpiresAt string `json:"refreshTokenExpiresAt,omitempty"`
	PrincipalKind         string `json:"principalKind,omitempty"`
	UpdatedAt             string `json:"updatedAt,omitempty"`
}

type GetSessionInput struct{ ID string `json:"id"` }

type SessionsListInput struct {
	UserID string `json:"userId,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}
type SessionsListResponse struct{ Sessions []SessionSummary `json:"sessions"` }
type RevokeSessionInput struct{ ID string `json:"id"` }
type BulkRevokeSessionsInput struct{ UserID string `json:"userId"` }
type BulkRevokeResponse struct {
	OK    bool `json:"ok"`
	Count int  `json:"count"`
}

func sessionsListHandler(deps Deps) func(ctx context.Context, in SessionsListInput, _ contract.Principal) (SessionsListResponse, error) {
	return func(ctx context.Context, in SessionsListInput, _ contract.Principal) (SessionsListResponse, error) {
		if deps.Engine == nil {
			return SessionsListResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		var (
			list []*session.Session
			err  error
		)
		if uidStr := strings.TrimSpace(in.UserID); uidStr != "" {
			uid, pErr := parseUserID(uidStr)
			if pErr != nil {
				return SessionsListResponse{}, pErr
			}
			list, err = deps.Engine.ListSessions(ctx, uid)
		} else {
			// No userID supplied → return the recent global window.
			limit := in.Limit
			if limit <= 0 {
				limit = 100
			}
			list, err = deps.Engine.ListAllSessions(ctx, limit)
		}
		if err != nil {
			return SessionsListResponse{}, mapEngineError(err)
		}
		out := SessionsListResponse{Sessions: make([]SessionSummary, 0, len(list))}
		for _, s := range list {
			out.Sessions = append(out.Sessions, SessionSummary{
				ID: s.ID.String(), UserID: s.UserID.String(),
				IPAddress: s.IPAddress, UserAgent: s.UserAgent,
				LastActivityAt: s.LastActivityAt.UTC().Format(time.RFC3339),
				ExpiresAt:      s.ExpiresAt.UTC().Format(time.RFC3339),
				CreatedAt:      s.CreatedAt.UTC().Format(time.RFC3339),
			})
		}
		return out, nil
	}
}

func sessionsRevokeHandler(deps Deps) func(ctx context.Context, in RevokeSessionInput, _ contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in RevokeSessionInput, _ contract.Principal) (AckResponse, error) {
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		sid, err := parseSessionID(in.ID)
		if err != nil {
			return AckResponse{}, err
		}
		if err := deps.Engine.RevokeSession(ctx, sid); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: sid.String()}, nil
	}
}

func sessionsBulkRevokeHandler(deps Deps) func(ctx context.Context, in BulkRevokeSessionsInput, p contract.Principal) (BulkRevokeResponse, error) {
	return func(ctx context.Context, in BulkRevokeSessionsInput, p contract.Principal) (BulkRevokeResponse, error) {
		if deps.Engine == nil {
			return BulkRevokeResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		adminID, err := principalUserID(p)
		if err != nil {
			return BulkRevokeResponse{}, err
		}
		uid, err := parseUserID(in.UserID)
		if err != nil {
			return BulkRevokeResponse{}, err
		}
		n, err := deps.Engine.AdminBulkRevokeSessions(ctx, adminID, uid)
		if err != nil {
			return BulkRevokeResponse{}, mapEngineError(err)
		}
		return BulkRevokeResponse{OK: true, Count: n}, nil
	}
}

func sessionsDetailHandler(deps Deps) func(ctx context.Context, in GetSessionInput, _ contract.Principal) (SessionDetail, error) {
	return func(ctx context.Context, in GetSessionInput, _ contract.Principal) (SessionDetail, error) {
		if deps.Engine == nil {
			return SessionDetail{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		sid, err := parseSessionID(in.ID)
		if err != nil {
			return SessionDetail{}, err
		}
		s, err := deps.Engine.Store().GetSession(ctx, sid)
		if err != nil {
			return SessionDetail{}, mapEngineError(err)
		}
		d := SessionDetail{
			SessionSummary: SessionSummary{
				ID: s.ID.String(), UserID: s.UserID.String(),
				IPAddress: s.IPAddress, UserAgent: s.UserAgent,
				LastActivityAt: s.LastActivityAt.UTC().Format(time.RFC3339),
				ExpiresAt:      s.ExpiresAt.UTC().Format(time.RFC3339),
				CreatedAt:      s.CreatedAt.UTC().Format(time.RFC3339),
			},
			AppID:                 s.AppID.String(),
			EnvID:                 s.EnvID.String(),
			RefreshTokenExpiresAt: s.RefreshTokenExpiresAt.UTC().Format(time.RFC3339),
			PrincipalKind:         s.PrincipalKind,
			UpdatedAt:             s.UpdatedAt.UTC().Format(time.RFC3339),
		}
		if !s.OrgID.IsNil() {
			d.OrgID = s.OrgID.String()
		}
		if !s.DeviceID.IsNil() {
			d.DeviceID = s.DeviceID.String()
		}
		if !s.ImpersonatedBy.IsNil() {
			d.ImpersonatedBy = s.ImpersonatedBy.String()
		}
		return d, nil
	}
}

func parseSessionID(s string) (id.SessionID, error) {
	if strings.TrimSpace(s) == "" {
		return id.SessionID{}, &contract.Error{Code: contract.CodeBadRequest, Message: "id is required"}
	}
	sid, err := id.ParseSessionID(s)
	if err != nil {
		return id.SessionID{}, &contract.Error{Code: contract.CodeBadRequest, Message: "invalid session id: " + err.Error()}
	}
	return sid, nil
}
