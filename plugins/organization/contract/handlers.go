// handlers.go: organization intent handlers owned by the
// organization plugin. Phase D.3 moved these from the auth
// contributor's handlers_organizations.go. The plugin handle arrives
// via Deps so we skip the engine.Plugin("organization") indirection
// the auth-contributor version used.
package contract

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"

	"github.com/xraph/forge/extensions/dashboard/contract"

	authcontract "github.com/xraph/authsome/extension/contract"
)

// ────────────────────────────────────────────────────────────────────
// Wire shapes
// ────────────────────────────────────────────────────────────────────

type OrgSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	CreatedAt string `json:"createdAt"`
}

type OrgDetail struct {
	OrgSummary
	AppID     string            `json:"appId,omitempty"`
	Logo      string            `json:"logo,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	UpdatedAt string            `json:"updatedAt"`
}

type MemberSummary struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	Role      string `json:"role"`
	CreatedAt string `json:"createdAt"`
}

type OrgListResponse struct {
	Organizations []OrgSummary `json:"organizations"`
}

type MembersListResponse struct {
	Members []MemberSummary `json:"members"`
}

type InvitationSummary struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	ExpiresAt string `json:"expiresAt"`
}

type InvitationsListResponse struct {
	Invitations []InvitationSummary `json:"invitations"`
}

type CreateInvitationResponse struct {
	InvitationSummary
	Token string `json:"token"`
}

type GetOrgInput struct {
	ID string `json:"id"`
}

type CreateOrgInput struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	Logo string `json:"logo,omitempty"`
}

type UpdateOrgInput struct {
	ID   string  `json:"id"`
	Name *string `json:"name,omitempty"`
	Logo *string `json:"logo,omitempty"`
}

type DeleteOrgInput struct {
	ID string `json:"id"`
}

type ListMembersInput struct {
	OrgID string `json:"orgId"`
}

type AddMemberInput struct {
	OrgID  string `json:"orgId"`
	UserID string `json:"userId"`
	Email  string `json:"email,omitempty"`
	Role   string `json:"role,omitempty"`
}

type ListInvitationsInput struct {
	OrgID string `json:"orgId"`
}

type CreateInvitationInput struct {
	OrgID string `json:"orgId"`
	Email string `json:"email"`
	Role  string `json:"role,omitempty"`
}

type RemoveMemberInput struct {
	ID string `json:"id"`
}

type ackResponse struct {
	OK bool   `json:"ok"`
	ID string `json:"id,omitempty"`
}

// ────────────────────────────────────────────────────────────────────
// Handlers
// ────────────────────────────────────────────────────────────────────

func orgsListHandler(deps Deps) func(ctx context.Context, _ struct{}, p contract.Principal) (OrgListResponse, error) {
	return func(ctx context.Context, _ struct{}, p contract.Principal) (OrgListResponse, error) {
		if deps.Engine == nil || deps.Plugin == nil {
			return OrgListResponse{}, unavailable()
		}
		list, err := deps.Plugin.AdminListOrganizations(ctx, authcontract.AppIDFromPrincipal(p, deps.Engine))
		if err != nil {
			return OrgListResponse{}, mapErr(err)
		}
		out := OrgListResponse{Organizations: make([]OrgSummary, 0, len(list))}
		for _, o := range list {
			out.Organizations = append(out.Organizations, projectOrgSummary(o))
		}
		return out, nil
	}
}

func orgsDetailHandler(deps Deps) func(ctx context.Context, in GetOrgInput, _ contract.Principal) (OrgDetail, error) {
	return func(ctx context.Context, in GetOrgInput, _ contract.Principal) (OrgDetail, error) {
		if deps.Engine == nil || deps.Plugin == nil {
			return OrgDetail{}, unavailable()
		}
		oid, err := parseOrgID(in.ID)
		if err != nil {
			return OrgDetail{}, err
		}
		o, err := deps.Plugin.GetOrganization(ctx, oid)
		if err != nil {
			return OrgDetail{}, mapErr(err)
		}
		return projectOrgDetail(o), nil
	}
}

func orgsCreateHandler(deps Deps) func(ctx context.Context, in CreateOrgInput, p contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in CreateOrgInput, p contract.Principal) (ackResponse, error) {
		if deps.Engine == nil || deps.Plugin == nil {
			return ackResponse{}, unavailable()
		}
		creatorID, err := principalUserID(p)
		if err != nil {
			return ackResponse{}, err
		}
		name := strings.TrimSpace(in.Name)
		slug := strings.TrimSpace(in.Slug)
		if name == "" || slug == "" {
			return ackResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "name and slug are required"}
		}
		o := &organization.Organization{
			AppID: authcontract.AppIDFromPrincipal(p, deps.Engine),
			Name:  name, Slug: slug, Logo: in.Logo,
			CreatedBy: creatorID,
		}
		if err := deps.Plugin.CreateOrganization(ctx, o); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: o.ID.String()}, nil
	}
}

func orgsUpdateHandler(deps Deps) func(ctx context.Context, in UpdateOrgInput, _ contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in UpdateOrgInput, _ contract.Principal) (ackResponse, error) {
		if deps.Engine == nil || deps.Plugin == nil {
			return ackResponse{}, unavailable()
		}
		oid, err := parseOrgID(in.ID)
		if err != nil {
			return ackResponse{}, err
		}
		current, err := deps.Plugin.GetOrganization(ctx, oid)
		if err != nil {
			return ackResponse{}, mapErr(err)
		}
		if in.Name != nil {
			current.Name = *in.Name
		}
		if in.Logo != nil {
			current.Logo = *in.Logo
		}
		if err := deps.Plugin.UpdateOrganization(ctx, current); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: oid.String()}, nil
	}
}

func orgsDeleteHandler(deps Deps) func(ctx context.Context, in DeleteOrgInput, _ contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in DeleteOrgInput, _ contract.Principal) (ackResponse, error) {
		if deps.Engine == nil || deps.Plugin == nil {
			return ackResponse{}, unavailable()
		}
		oid, err := parseOrgID(in.ID)
		if err != nil {
			return ackResponse{}, err
		}
		if err := deps.Plugin.DeleteOrganization(ctx, oid); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: oid.String()}, nil
	}
}

func orgsMembersListHandler(deps Deps) func(ctx context.Context, in ListMembersInput, _ contract.Principal) (MembersListResponse, error) {
	return func(ctx context.Context, in ListMembersInput, _ contract.Principal) (MembersListResponse, error) {
		if deps.Engine == nil || deps.Plugin == nil {
			return MembersListResponse{}, unavailable()
		}
		oid, err := parseOrgID(in.OrgID)
		if err != nil {
			return MembersListResponse{}, err
		}
		list, err := deps.Plugin.ListMembers(ctx, oid)
		if err != nil {
			return MembersListResponse{}, mapErr(err)
		}
		out := MembersListResponse{Members: make([]MemberSummary, 0, len(list))}
		for _, m := range list {
			out.Members = append(out.Members, MemberSummary{
				ID: m.ID.String(), UserID: m.UserID.String(),
				Role: string(m.Role), CreatedAt: m.CreatedAt.UTC().Format(time.RFC3339),
			})
		}
		return out, nil
	}
}

func orgsAddMemberHandler(deps Deps) func(context.Context, AddMemberInput, contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in AddMemberInput, p contract.Principal) (ackResponse, error) {
		if deps.Engine == nil || deps.Plugin == nil {
			return ackResponse{}, unavailable()
		}
		org, err := scopedOrg(ctx, deps, in.OrgID, p)
		if err != nil {
			return ackResponse{}, err
		}
		role, err := memberRole(in.Role)
		if err != nil {
			return ackResponse{}, err
		}
		var u *user.User
		if rawID := strings.TrimSpace(in.UserID); rawID != "" {
			uid, parseErr := id.ParseUserID(rawID)
			if parseErr != nil {
				return ackResponse{}, badReq("valid userId is required")
			}
			u, err = deps.Engine.GetUser(ctx, uid)
		} else if email := strings.ToLower(strings.TrimSpace(in.Email)); email != "" {
			address, parseErr := mail.ParseAddress(email)
			if parseErr != nil || address.Address != email {
				return ackResponse{}, badReq("valid email is required")
			}
			u, err = deps.Engine.GetUserByEmail(ctx, org.AppID, email)
		} else {
			return ackResponse{}, badReq("userId or email is required")
		}
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return ackResponse{}, &contract.Error{Code: contract.CodeNotFound, Message: "user not found in this app"}
			}
			return ackResponse{}, mapErr(err)
		}
		if u == nil || u.AppID != org.AppID {
			return ackResponse{}, &contract.Error{Code: contract.CodeNotFound, Message: "user not found in this app"}
		}
		uid := u.ID
		members, err := deps.Plugin.ListMembers(ctx, org.ID)
		if err != nil {
			return ackResponse{}, mapErr(err)
		}
		for _, member := range members {
			if member.UserID == uid {
				return ackResponse{}, &contract.Error{Code: contract.CodeConflict, Message: "user is already a member"}
			}
		}
		member := &organization.Member{ID: id.NewMemberID(), OrgID: org.ID, UserID: uid, Role: role}
		if err := deps.Plugin.AddMember(ctx, member); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: member.ID.String()}, nil
	}
}

func orgsInvitationsHandler(deps Deps) func(context.Context, ListInvitationsInput, contract.Principal) (InvitationsListResponse, error) {
	return func(ctx context.Context, in ListInvitationsInput, p contract.Principal) (InvitationsListResponse, error) {
		if deps.Engine == nil || deps.Plugin == nil {
			return InvitationsListResponse{}, unavailable()
		}
		org, err := scopedOrg(ctx, deps, in.OrgID, p)
		if err != nil {
			return InvitationsListResponse{}, err
		}
		list, err := deps.Plugin.ListInvitations(ctx, org.ID)
		if err != nil {
			return InvitationsListResponse{}, mapErr(err)
		}
		out := InvitationsListResponse{Invitations: make([]InvitationSummary, 0, len(list))}
		for _, inv := range list {
			out.Invitations = append(out.Invitations, projectInvitation(inv))
		}
		return out, nil
	}
}

func orgsCreateInvitationHandler(deps Deps) func(context.Context, CreateInvitationInput, contract.Principal) (CreateInvitationResponse, error) {
	return func(ctx context.Context, in CreateInvitationInput, p contract.Principal) (CreateInvitationResponse, error) {
		if deps.Engine == nil || deps.Plugin == nil {
			return CreateInvitationResponse{}, unavailable()
		}
		org, err := scopedOrg(ctx, deps, in.OrgID, p)
		if err != nil {
			return CreateInvitationResponse{}, err
		}
		inviterID, err := principalUserID(p)
		if err != nil {
			return CreateInvitationResponse{}, err
		}
		email := strings.ToLower(strings.TrimSpace(in.Email))
		address, err := mail.ParseAddress(email)
		if err != nil || address.Address != email {
			return CreateInvitationResponse{}, badReq("valid email is required")
		}
		role, err := memberRole(in.Role)
		if err != nil {
			return CreateInvitationResponse{}, err
		}
		list, err := deps.Plugin.ListInvitations(ctx, org.ID)
		if err != nil {
			return CreateInvitationResponse{}, mapErr(err)
		}
		for _, existing := range list {
			if strings.EqualFold(existing.Email, email) && existing.Status == organization.InvitationPending && (existing.ExpiresAt.IsZero() || time.Now().Before(existing.ExpiresAt)) {
				return CreateInvitationResponse{}, &contract.Error{Code: contract.CodeConflict, Message: "a pending invitation already exists for this email"}
			}
		}
		token, err := account.GenerateVerificationToken()
		if err != nil {
			return CreateInvitationResponse{}, mapErr(err)
		}
		inv := &organization.Invitation{ID: id.NewInvitationID(), OrgID: org.ID, Email: email, Role: role, InviterID: inviterID, Status: organization.InvitationPending, Token: token, ExpiresAt: time.Now().Add(72 * time.Hour)}
		if err := deps.Plugin.CreateInvitation(ctx, inv); err != nil {
			return CreateInvitationResponse{}, mapErr(err)
		}
		return CreateInvitationResponse{InvitationSummary: projectInvitation(inv), Token: token}, nil
	}
}

func orgsRemoveMemberHandler(deps Deps) func(ctx context.Context, in RemoveMemberInput, _ contract.Principal) (ackResponse, error) {
	return func(ctx context.Context, in RemoveMemberInput, _ contract.Principal) (ackResponse, error) {
		if deps.Engine == nil || deps.Plugin == nil {
			return ackResponse{}, unavailable()
		}
		mid, err := parseMemberID(in.ID)
		if err != nil {
			return ackResponse{}, err
		}
		if err := deps.Plugin.RemoveMember(ctx, mid); err != nil {
			return ackResponse{}, mapErr(err)
		}
		return ackResponse{OK: true, ID: mid.String()}, nil
	}
}

// ────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────

func projectOrgSummary(o *organization.Organization) OrgSummary {
	if o == nil {
		return OrgSummary{}
	}
	return OrgSummary{
		ID: o.ID.String(), Name: o.Name, Slug: o.Slug,
		CreatedAt: o.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func projectOrgDetail(o *organization.Organization) OrgDetail {
	if o == nil {
		return OrgDetail{}
	}
	return OrgDetail{
		OrgSummary: projectOrgSummary(o),
		AppID:      o.AppID.String(),
		Logo:       o.Logo,
		Metadata:   o.Metadata,
		UpdatedAt:  o.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func projectInvitation(inv *organization.Invitation) InvitationSummary {
	return InvitationSummary{
		ID: inv.ID.String(), Email: inv.Email, Role: string(inv.Role), Status: string(inv.Status),
		CreatedAt: inv.CreatedAt.UTC().Format(time.RFC3339), ExpiresAt: inv.ExpiresAt.UTC().Format(time.RFC3339),
	}
}

func scopedOrg(ctx context.Context, deps Deps, rawID string, p contract.Principal) (*organization.Organization, error) {
	oid, err := parseOrgID(rawID)
	if err != nil {
		return nil, err
	}
	org, err := deps.Plugin.GetOrganization(ctx, oid)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, &contract.Error{Code: contract.CodeNotFound, Message: "organization not found in this app"}
		}
		return nil, mapErr(err)
	}
	if org == nil || org.AppID != authcontract.AppIDFromPrincipal(p, deps.Engine) {
		return nil, &contract.Error{Code: contract.CodeNotFound, Message: "organization not found in this app"}
	}
	return org, nil
}

func memberRole(raw string) (organization.MemberRole, error) {
	if raw == "" {
		return organization.RoleMember, nil
	}
	role := organization.MemberRole(raw)
	switch role {
	case organization.RoleMember, organization.RoleAdmin, organization.RoleOwner:
		return role, nil
	default:
		return "", badReq("role must be member, admin, or owner")
	}
}

func parseOrgID(s string) (id.OrgID, error) {
	if strings.TrimSpace(s) == "" {
		return id.OrgID{}, badReq("id is required")
	}
	oid, err := id.ParseOrgID(s)
	if err != nil {
		return id.OrgID{}, badReq("invalid org id: " + err.Error())
	}
	return oid, nil
}

func parseMemberID(s string) (id.MemberID, error) {
	if strings.TrimSpace(s) == "" {
		return id.MemberID{}, badReq("id is required")
	}
	mid, err := id.ParseMemberID(s)
	if err != nil {
		return id.MemberID{}, badReq("invalid member id: " + err.Error())
	}
	return mid, nil
}

func principalUserID(p contract.Principal) (id.UserID, error) {
	if p.User == nil || p.User.Subject == "" {
		return id.UserID{}, badReq("authenticated principal required")
	}
	uid, err := id.ParseUserID(p.User.Subject)
	if err != nil {
		return id.UserID{}, badReq("invalid principal subject: " + err.Error())
	}
	return uid, nil
}

func badReq(msg string) error {
	return &contract.Error{Code: contract.CodeBadRequest, Message: msg}
}

func unavailable() error {
	return &contract.Error{Code: contract.CodeUnavailable, Message: "organization plugin not enabled"}
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	var ce *contract.Error
	if errors.As(err, &ce) {
		return ce
	}
	return &contract.Error{Code: contract.CodeInternal, Message: err.Error()}
}
