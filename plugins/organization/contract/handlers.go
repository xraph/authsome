// handlers.go: organization intent handlers owned by the
// organization plugin. Phase D.3 moved these from the auth
// contributor's handlers_organizations.go. The plugin handle arrives
// via Deps so we skip the engine.Plugin("organization") indirection
// the auth-contributor version used.
package contract

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"

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
