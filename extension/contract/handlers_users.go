// handlers_users.go: Phase C.1 of the authsome dashboard port.
//
// Implements the users.* intent surface the React shell binds in its
// /users page (list + detail drawer + row actions) and /users/:id +
// /users/create routes. Engine calls run against the platform app:
// admin operations are scoped to whichever app authsome considers
// "platform" at handler time (PlatformAppID, with a fallback to the
// engine config's AppID). The principal's subject becomes the admin
// actor recorded by the engine's hooks + audit emitter.
package contract

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/user"

	"github.com/xraph/forge/extensions/dashboard/contract"
)

// ────────────────────────────────────────────────────────────────────
// Wire shapes
// ────────────────────────────────────────────────────────────────────

// UserSummary is the table-row projection of a user.User. Lean by
// design: passwords, ban metadata, timestamps stay on the detail
// surface. The React shell's resource.list reads field names verbatim
// from the JSON, so renames here are wire-level breaking changes.
type UserSummary struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"emailVerified"`
	FirstName     string `json:"firstName,omitempty"`
	LastName      string `json:"lastName,omitempty"`
	Username      string `json:"username,omitempty"`
	Banned        bool   `json:"banned"`
	CreatedAt     string `json:"createdAt"`
}

// UserDetail extends UserSummary with the fields the detail drawer and
// the /users/:id route render. Splitting summary vs detail keeps the
// list payload small when there are thousands of users.
type UserDetail struct {
	UserSummary
	DisplayName       string `json:"displayName,omitempty"`
	Phone             string `json:"phone,omitempty"`
	PhoneVerified     bool   `json:"phoneVerified,omitempty"`
	Image             string `json:"image,omitempty"`
	BanReason         string `json:"banReason,omitempty"`
	BanExpiresAt      string `json:"banExpiresAt,omitempty"`
	PasswordChangedAt string `json:"passwordChangedAt,omitempty"`
	UpdatedAt         string `json:"updatedAt"`
	AppID             string `json:"appId,omitempty"`
	EnvID             string `json:"envId,omitempty"`
}

// ListUsersInput is the wire shape for the users.list query.
// The React shell sends nothing today; the fields exist so the
// upcoming search box / pagination controls don't need a schema change.
type ListUsersInput struct {
	Email  string `json:"email,omitempty"`
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// UsersListResponse is the users.list reply. The shell's resource.list
// extractor picks the first array-valued field, which is `users` —
// next_cursor and total ride alongside for upcoming paging UI.
type UsersListResponse struct {
	Users      []UserSummary `json:"users"`
	NextCursor string        `json:"nextCursor,omitempty"`
	Total      int           `json:"total,omitempty"`
}

// GetUserInput is the wire shape for users.detail.
type GetUserInput struct {
	ID string `json:"id"`
}

// CreateUserInput is the wire shape for users.create. Username is
// optional; FirstName/LastName are recommended but not required by the
// engine. Password is validated against the engine's password policy.
type CreateUserInput struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	Username  string `json:"username,omitempty"`
}

// UpdateUserInput is the wire shape for users.update. Only fields the
// caller actually wants to change need to be set — pointer-typed so we
// can distinguish "leave unchanged" from "set to empty/false".
type UpdateUserInput struct {
	ID            string  `json:"id"`
	FirstName     *string `json:"firstName,omitempty"`
	LastName      *string `json:"lastName,omitempty"`
	Username      *string `json:"username,omitempty"`
	EmailVerified *bool   `json:"emailVerified,omitempty"`
}

// BanUserInput is the wire shape for users.ban. ExpiresAt is RFC3339;
// empty means an indefinite ban.
type BanUserInput struct {
	ID        string `json:"id"`
	Reason    string `json:"reason,omitempty"`
	ExpiresAt string `json:"expiresAt,omitempty"`
}

// UnbanUserInput / DeleteUserInput share the same id-only shape.
type UnbanUserInput struct {
	ID string `json:"id"`
}
type DeleteUserInput struct {
	ID string `json:"id"`
}

// AckResponse is the canonical reply for mutating commands that don't
// project a payload of their own — the shell's react-query invalidator
// flips the list query stale and the table refetches automatically.
type AckResponse struct {
	OK bool   `json:"ok"`
	ID string `json:"id,omitempty"`
}

// ────────────────────────────────────────────────────────────────────
// Handlers
// ────────────────────────────────────────────────────────────────────

func usersListHandler(deps Deps) func(ctx context.Context, in ListUsersInput, p contract.Principal) (UsersListResponse, error) {
	return func(ctx context.Context, in ListUsersInput, _ contract.Principal) (UsersListResponse, error) {
		eng := deps.Engine
		if eng == nil {
			return UsersListResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		q := &user.Query{
			AppID:  defaultAppID(eng),
			Email:  strings.TrimSpace(in.Email),
			Cursor: in.Cursor,
			Limit:  in.Limit,
		}
		if q.Limit <= 0 {
			q.Limit = 50
		}
		list, err := eng.AdminListUsers(ctx, q)
		if err != nil {
			return UsersListResponse{}, mapEngineError(err)
		}
		out := UsersListResponse{
			Users:      make([]UserSummary, 0, len(list.Users)),
			NextCursor: list.NextCursor,
			Total:      list.Total,
		}
		for _, u := range list.Users {
			out.Users = append(out.Users, projectUserSummary(u))
		}
		return out, nil
	}
}

func usersDetailHandler(deps Deps) func(ctx context.Context, in GetUserInput, p contract.Principal) (UserDetail, error) {
	return func(ctx context.Context, in GetUserInput, _ contract.Principal) (UserDetail, error) {
		eng := deps.Engine
		if eng == nil {
			return UserDetail{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		uid, err := parseUserID(in.ID)
		if err != nil {
			return UserDetail{}, err
		}
		u, err := eng.AdminGetUser(ctx, uid)
		if err != nil {
			return UserDetail{}, mapEngineError(err)
		}
		return projectUserDetail(u), nil
	}
}

func usersCreateHandler(deps Deps) func(ctx context.Context, in CreateUserInput, p contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in CreateUserInput, p contract.Principal) (AckResponse, error) {
		eng := deps.Engine
		if eng == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		adminID, err := principalUserID(p)
		if err != nil {
			return AckResponse{}, err
		}
		email := strings.TrimSpace(in.Email)
		if email == "" {
			return AckResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "email is required"}
		}
		if in.Password == "" {
			return AckResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "password is required"}
		}
		appID := defaultAppID(eng)
		// envID stays empty — engine resolves the default env per app.
		u, err := eng.AdminCreateUser(ctx, adminID, appID, id.EnvironmentID{}, email, in.Password,
			strings.TrimSpace(in.FirstName), strings.TrimSpace(in.LastName), strings.TrimSpace(in.Username))
		if err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: u.ID.String()}, nil
	}
}

func usersUpdateHandler(deps Deps) func(ctx context.Context, in UpdateUserInput, p contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in UpdateUserInput, p contract.Principal) (AckResponse, error) {
		eng := deps.Engine
		if eng == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		adminID, err := principalUserID(p)
		if err != nil {
			return AckResponse{}, err
		}
		uid, err := parseUserID(in.ID)
		if err != nil {
			return AckResponse{}, err
		}
		updates := authsome.AdminUserUpdates{
			FirstName:     in.FirstName,
			LastName:      in.LastName,
			Username:      in.Username,
			EmailVerified: in.EmailVerified,
		}
		if err := eng.AdminUpdateUser(ctx, adminID, uid, updates); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: uid.String()}, nil
	}
}

func usersBanHandler(deps Deps) func(ctx context.Context, in BanUserInput, p contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in BanUserInput, p contract.Principal) (AckResponse, error) {
		eng := deps.Engine
		if eng == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		adminID, err := principalUserID(p)
		if err != nil {
			return AckResponse{}, err
		}
		uid, err := parseUserID(in.ID)
		if err != nil {
			return AckResponse{}, err
		}
		var expiresAt *time.Time
		if in.ExpiresAt != "" {
			t, parseErr := time.Parse(time.RFC3339, in.ExpiresAt)
			if parseErr != nil {
				return AckResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "expiresAt must be RFC3339"}
			}
			expiresAt = &t
		}
		if err := eng.AdminBanUser(ctx, adminID, uid, in.Reason, expiresAt); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: uid.String()}, nil
	}
}

func usersUnbanHandler(deps Deps) func(ctx context.Context, in UnbanUserInput, p contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in UnbanUserInput, p contract.Principal) (AckResponse, error) {
		eng := deps.Engine
		if eng == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		adminID, err := principalUserID(p)
		if err != nil {
			return AckResponse{}, err
		}
		uid, err := parseUserID(in.ID)
		if err != nil {
			return AckResponse{}, err
		}
		if err := eng.AdminUnbanUser(ctx, adminID, uid); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: uid.String()}, nil
	}
}

func usersDeleteHandler(deps Deps) func(ctx context.Context, in DeleteUserInput, p contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in DeleteUserInput, p contract.Principal) (AckResponse, error) {
		eng := deps.Engine
		if eng == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		adminID, err := principalUserID(p)
		if err != nil {
			return AckResponse{}, err
		}
		uid, err := parseUserID(in.ID)
		if err != nil {
			return AckResponse{}, err
		}
		if err := eng.AdminDeleteUser(ctx, adminID, uid); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: uid.String()}, nil
	}
}

// ────────────────────────────────────────────────────────────────────
// Helpers — shared with future Phase C slices once the same patterns
// (id parsing, principal extraction, engine-error mapping) recur.
// ────────────────────────────────────────────────────────────────────

func projectUserSummary(u *user.User) UserSummary {
	if u == nil {
		return UserSummary{}
	}
	return UserSummary{
		ID:            u.ID.String(),
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		Username:      u.Username,
		Banned:        u.Banned,
		CreatedAt:     u.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func projectUserDetail(u *user.User) UserDetail {
	if u == nil {
		return UserDetail{}
	}
	d := UserDetail{
		UserSummary:   projectUserSummary(u),
		DisplayName:   u.Name(),
		Phone:         u.Phone,
		PhoneVerified: u.PhoneVerified,
		Image:         u.Image,
		BanReason:     u.BanReason,
		UpdatedAt:     u.UpdatedAt.UTC().Format(time.RFC3339),
		AppID:         u.AppID.String(),
		EnvID:         u.EnvID.String(),
	}
	if u.BanExpires != nil {
		d.BanExpiresAt = u.BanExpires.UTC().Format(time.RFC3339)
	}
	if u.PasswordChangedAt != nil {
		d.PasswordChangedAt = u.PasswordChangedAt.UTC().Format(time.RFC3339)
	}
	return d
}

// parseUserID converts the wire id (a UUID string) into id.UserID with
// a uniform CodeBadRequest error for bad input — engine methods would
// otherwise surface this as a generic 500.
func parseUserID(s string) (id.UserID, error) {
	if strings.TrimSpace(s) == "" {
		return id.UserID{}, &contract.Error{Code: contract.CodeBadRequest, Message: "id is required"}
	}
	uid, err := id.ParseUserID(s)
	if err != nil {
		return id.UserID{}, &contract.Error{Code: contract.CodeBadRequest, Message: "invalid user id: " + err.Error()}
	}
	return uid, nil
}

// principalUserID extracts the authenticated admin's id.UserID from
// the contract principal. Admin operations require a real actor so
// the engine's hooks + audit trail can attribute the action; returns
// CodeUnauthenticated if the principal lacks a user (anonymous).
func principalUserID(p contract.Principal) (id.UserID, error) {
	if p.User == nil || p.User.Subject == "" {
		return id.UserID{}, &contract.Error{Code: contract.CodeUnauthenticated, Message: "admin operations require an authenticated user"}
	}
	uid, err := id.ParseUserID(p.User.Subject)
	if err != nil {
		return id.UserID{}, &contract.Error{Code: contract.CodeUnauthenticated, Message: "principal subject is not a valid user id"}
	}
	return uid, nil
}

// mapEngineError translates authsome engine errors into the contract's
// canonical error codes. Keeps every users.* handler's error tail
// uniform: anything not specifically mapped becomes CodeInternal with
// the engine's message preserved (the dispatcher's audit layer will
// already log the raw error).
//
// Mirrors the mapSignInError pattern in handlers.go but covers the
// admin-flow errors. Future Phase C slices reuse this helper directly.
func mapEngineError(err error) error {
	if err == nil {
		return nil
	}
	if ce, ok := err.(*contract.Error); ok {
		return ce
	}
	switch {
	case errors.Is(err, account.ErrEmailTaken):
		return &contract.Error{Code: contract.CodeBadRequest, Message: "Email is already in use"}
	case errors.Is(err, account.ErrUsernameTaken):
		return &contract.Error{Code: contract.CodeBadRequest, Message: "Username is already in use"}
	case errors.Is(err, account.ErrInvalidCredentials):
		return &contract.Error{Code: contract.CodeBadRequest, Message: "Invalid credentials"}
	case errors.Is(err, authsome.ErrNotStarted):
		return &contract.Error{Code: contract.CodeUnavailable, Message: "System is still initializing. Please try again in a moment."}
	}
	// "not found" engine wraps typically read "authsome: admin get user: ...".
	// Surface as CodeNotFound when the message looks like a lookup failure
	// so the React shell can render a 404 panel instead of a generic 500.
	msg := err.Error()
	if strings.Contains(msg, "not found") || strings.Contains(msg, "no rows") {
		return &contract.Error{Code: contract.CodeNotFound, Message: "Record not found"}
	}
	return &contract.Error{Code: contract.CodeInternal, Message: fmt.Sprintf("engine: %v", err)}
}
