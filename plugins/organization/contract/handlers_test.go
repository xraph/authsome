package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/user"
	dashauth "github.com/xraph/forge/extensions/dashboard/auth"
	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/loader"
	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

type invitationService struct {
	OrgService
	org         *organization.Organization
	invitations []*organization.Invitation
	members     []*organization.Member
}

func (s *invitationService) GetOrganization(_ context.Context, _ id.OrgID) (*organization.Organization, error) {
	return s.org, nil
}

func (s *invitationService) ListInvitations(_ context.Context, _ id.OrgID) ([]*organization.Invitation, error) {
	return s.invitations, nil
}

func (s *invitationService) CreateInvitation(_ context.Context, inv *organization.Invitation) error {
	s.invitations = append(s.invitations, inv)
	return nil
}

func (s *invitationService) ListMembers(_ context.Context, _ id.OrgID) ([]*organization.Member, error) {
	return s.members, nil
}

func (s *invitationService) AddMember(_ context.Context, member *organization.Member) error {
	s.members = append(s.members, member)
	return nil
}

func TestManifest_Loads(t *testing.T) {
	m, err := loader.Load(bytes.NewReader(manifestYAML), "organization/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if m.Contributor.Name != "organization" {
		t.Errorf("contributor name = %q, want organization", m.Contributor.Name)
	}
	if got := len(m.Intents); got != 10 {
		t.Errorf("intents = %d, want 10", got)
	}
}

func TestManifest_Validates(t *testing.T) {
	m, err := loader.Load(bytes.NewReader(manifestYAML), "organization/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := loader.Validate(m, contract.NewWardenRegistry()); err != nil {
		t.Errorf("validate: %v", err)
	}
}

func TestOrgsListHandler_UnavailableWhenEngineNil(t *testing.T) {
	h := orgsListHandler(Deps{})
	_, err := h(context.Background(), struct{}{}, contract.Principal{})
	var ce *contract.Error
	if !errors.As(err, &ce) || ce.Code != contract.CodeUnavailable {
		t.Errorf("expected CodeUnavailable, got %v", err)
	}
}

func TestInvitationHandlers_OneTimeTokenAndAppScope(t *testing.T) {
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	if err != nil {
		t.Fatal(err)
	}
	mem := memory.New()
	engine, err := authsome.NewEngine(authsome.WithStore(mem), authsome.WithWarden(w))
	if err != nil {
		t.Fatal(err)
	}
	appID := id.NewAppID()
	org := &organization.Organization{ID: id.NewOrgID(), AppID: appID}
	service := &invitationService{org: org}
	deps := Deps{Engine: engine, Plugin: service}
	p := contract.Principal{User: &dashauth.UserInfo{Subject: id.NewUserID().String()}, Claims: map[string]any{"app_id": appID.String()}}
	existing := &user.User{ID: id.NewUserID(), AppID: appID, Email: "member@example.com"}
	if err := mem.CreateUserWithPrimaryEmail(context.Background(), existing, user.NewPrimaryEmail(existing, "admin")); err != nil {
		t.Fatal(err)
	}
	added, err := orgsAddMemberHandler(deps)(context.Background(), AddMemberInput{OrgID: org.ID.String(), Email: " Member@Example.com "}, p)
	if err != nil || !added.OK || len(service.members) != 1 || service.members[0].UserID != existing.ID {
		t.Fatalf("add existing user by email: response=%+v members=%+v err=%v", added, service.members, err)
	}

	created, err := orgsCreateInvitationHandler(deps)(context.Background(), CreateInvitationInput{OrgID: org.ID.String(), Email: " Person@Example.com ", Role: "admin"}, p)
	if err != nil {
		t.Fatal(err)
	}
	if created.Token == "" || created.Email != "person@example.com" || created.Role != "admin" {
		t.Fatalf("unexpected invitation: %+v", created)
	}
	if created.ExpiresAt == "" {
		t.Fatal("invitation should expire")
	}
	listed, err := orgsInvitationsHandler(deps)(context.Background(), ListInvitationsInput{OrgID: org.ID.String()}, p)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(listed)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Invitations) != 1 || string(encoded) == "" || containsToken(encoded, created.Token) {
		t.Fatalf("list leaked token or lost invitation: %s", encoded)
	}
	_, err = orgsCreateInvitationHandler(deps)(context.Background(), CreateInvitationInput{OrgID: org.ID.String(), Email: "person@example.com"}, p)
	var ce *contract.Error
	if !errors.As(err, &ce) || ce.Code != contract.CodeConflict {
		t.Fatalf("duplicate invitation: %v", err)
	}

	otherApp := id.NewAppID()
	p.Claims["app_id"] = otherApp.String()
	_, err = orgsInvitationsHandler(deps)(context.Background(), ListInvitationsInput{OrgID: org.ID.String()}, p)
	if !errors.As(err, &ce) || ce.Code != contract.CodeNotFound {
		t.Fatalf("cross-app list: %v", err)
	}
}

func containsToken(data []byte, token string) bool {
	return bytes.Contains(data, []byte(token))
}
