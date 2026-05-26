// Package contract wires the organization plugin's intent surface
// into the forge-dashboard contract registry. The `/organizations`
// page itself remains declared on the auth contributor — only the
// intent handlers + their declarations move here. Cross-contributor
// invocations work because the dashboard's contract dispatcher routes
// by intent name globally.
package contract

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"

	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/dispatcher"
	"github.com/xraph/forge/extensions/dashboard/contract/loader"
)

//go:embed manifest.yaml
var manifestYAML []byte

// OrgService is the surface of the organization plugin the contract
// handlers need. Declared here as an interface so this package doesn't
// import plugins/organization directly (which would create a cycle:
// plugins/organization → plugins/organization/contract →
// plugins/organization). The parent plugin's contract.go satisfies
// this interface by passing its own *Plugin into Deps.
type OrgService interface {
	AdminListOrganizations(ctx context.Context, appID id.AppID) ([]*organization.Organization, error)
	GetOrganization(ctx context.Context, orgID id.OrgID) (*organization.Organization, error)
	CreateOrganization(ctx context.Context, o *organization.Organization) error
	UpdateOrganization(ctx context.Context, o *organization.Organization) error
	DeleteOrganization(ctx context.Context, orgID id.OrgID) error
	ListMembers(ctx context.Context, orgID id.OrgID) ([]*organization.Member, error)
	RemoveMember(ctx context.Context, memberID id.MemberID) error
}

// Deps carries the typed plugin handle alongside the engine so the
// handlers can call CreateOrganization / GetOrganization / etc.
// directly without engine.Plugin("organization") indirection. Plugin
// is typed as the local OrgService interface (cycle avoidance).
type Deps struct {
	Engine *authsome.Engine
	Plugin OrgService
}

func Register(
	d *dispatcher.Dispatcher,
	reg contract.Registry,
	wreg contract.WardenRegistry,
	deps Deps,
) error {
	if deps.Engine == nil {
		return fmt.Errorf("organization/contract: Engine is required")
	}
	if deps.Plugin == nil {
		return fmt.Errorf("organization/contract: Plugin is required")
	}
	m, err := loader.Load(bytes.NewReader(manifestYAML), "organization/contract/manifest.yaml")
	if err != nil {
		return fmt.Errorf("organization/contract: load manifest: %w", err)
	}
	if err := loader.Validate(m, wreg); err != nil {
		return fmt.Errorf("organization/contract: validate manifest: %w", err)
	}
	if err := reg.Register(m); err != nil {
		return fmt.Errorf("organization/contract: register manifest: %w", err)
	}

	const c = "organization"
	if err := dispatcher.RegisterQuery(d, c, "orgs.list", 1, orgsListHandler(deps)); err != nil {
		return fmt.Errorf("organization/contract: register orgs.list: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "orgs.detail", 1, orgsDetailHandler(deps)); err != nil {
		return fmt.Errorf("organization/contract: register orgs.detail: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "orgs.create", 1, orgsCreateHandler(deps)); err != nil {
		return fmt.Errorf("organization/contract: register orgs.create: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "orgs.update", 1, orgsUpdateHandler(deps)); err != nil {
		return fmt.Errorf("organization/contract: register orgs.update: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "orgs.delete", 1, orgsDeleteHandler(deps)); err != nil {
		return fmt.Errorf("organization/contract: register orgs.delete: %w", err)
	}
	if err := dispatcher.RegisterQuery(d, c, "orgs.members", 1, orgsMembersListHandler(deps)); err != nil {
		return fmt.Errorf("organization/contract: register orgs.members: %w", err)
	}
	if err := dispatcher.RegisterCommand(d, c, "orgs.removeMember", 1, orgsRemoveMemberHandler(deps)); err != nil {
		return fmt.Errorf("organization/contract: register orgs.removeMember: %w", err)
	}
	return nil
}
