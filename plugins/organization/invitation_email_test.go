package organization_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/organization"
	orgplugin "github.com/xraph/authsome/plugins/organization"
)

// The organization plugin emails the invitation itself, because it is the only
// place the plaintext token exists. It hands the mail to whichever plugin
// registered as "notification".

type sentInvitation struct{ Email, Inviter, Org, URL string }

// fakeNotifier stands in for the notification plugin.
type fakeNotifier struct {
	mu   sync.Mutex
	sent []sentInvitation
	err  error
}

func (f *fakeNotifier) Name() string { return "notification" }

func (f *fakeNotifier) SendInvitation(_ context.Context, email, inviterName, orgName, acceptURL string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, sentInvitation{email, inviterName, orgName, acceptURL})
	return f.err
}

func (f *fakeNotifier) reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = nil
}

func (f *fakeNotifier) all() []sentInvitation {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]sentInvitation(nil), f.sent...)
}

const acceptTemplate = "https://app.example.com/accept?token={token}"

type emailFixture struct {
	p       *orgplugin.Plugin
	h       http.Handler
	inviter id.UserID
	org     *organization.Organization
}

func newEmailFixture(t *testing.T, cfg orgplugin.Config, extra ...authsome.Option) *emailFixture {
	t.Helper()
	p := orgplugin.New(cfg)
	eng := secutil.NewTestEngine(t, append([]authsome.Option{authsome.WithPlugin(p)}, extra...)...)
	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))
	appID, err := id.ParseAppID(orgTestAppID)
	require.NoError(t, err)
	u, _, err := eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID: appID, Email: "inviter@example.com", Password: "SecureP@ss123", FirstName: "Ines", LastName: "Marsh",
	})
	require.NoError(t, err)
	return &emailFixture{p: p, h: router.Handler(), inviter: u.ID, org: seedOrg(t, p, u.ID)}
}

func (f *emailFixture) create(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	body := postJSON(t, map[string]string{"email": "guest@example.com"})
	f.h.ServeHTTP(rec, orgReq("POST", "/v1/orgs/"+f.org.ID.String()+"/invitations", body, f.inviter))
	return rec
}

func TestCreateInvitation_sendsEmailWithAcceptLink(t *testing.T) {
	n := &fakeNotifier{}
	f := newEmailFixture(t, orgplugin.Config{InvitationURLTemplate: acceptTemplate}, authsome.WithPlugin(n))

	rec := f.create(t)
	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())

	sent := n.all()
	require.Len(t, sent, 1)
	assert.Equal(t, "guest@example.com", sent[0].Email)
	assert.Equal(t, "Ines Marsh", sent[0].Inviter)
	assert.Equal(t, "Acme", sent[0].Org)
	token := strings.TrimPrefix(sent[0].URL, "https://app.example.com/accept?token=")
	require.NotEmpty(t, token)
	assert.NotContains(t, token, "{token}")

	// The link works, and the response never carries the token.
	assert.Equal(t, http.StatusOK, getPreview(t, f.h, token).Code)
	assert.NotContains(t, rec.Body.String(), token)
}

func TestResendInvitation_sendsTheNewToken(t *testing.T) {
	n := &fakeNotifier{}
	f := newEmailFixture(t, orgplugin.Config{InvitationURLTemplate: acceptTemplate}, authsome.WithPlugin(n))
	inv := seedInvitation(t, f.p, f.org.ID, f.inviter, "tok-before", sentLongAgo)
	n.reset() // seeding went through CreateInvitation, which sends too

	rec := resendInvitation(t, f.h, f.org.ID, inv.ID, f.inviter)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	sent := n.all()
	require.Len(t, sent, 1)
	assert.Equal(t, "sofia@acme.com", sent[0].Email)
	assert.NotContains(t, sent[0].URL, "tok-before")
	token := strings.TrimPrefix(sent[0].URL, "https://app.example.com/accept?token=")
	assert.Equal(t, http.StatusOK, getPreview(t, f.h, token).Code)
	assert.Equal(t, http.StatusNotFound, getPreview(t, f.h, "tok-before").Code)
	assert.NotContains(t, rec.Body.String(), token)
}

func TestCreateInvitation_withoutSenderStillSucceeds(t *testing.T) {
	// A template but no notification plugin, then a plugin but no template.
	f := newEmailFixture(t, orgplugin.Config{InvitationURLTemplate: acceptTemplate})
	assert.Equal(t, http.StatusCreated, f.create(t).Code)

	n := &fakeNotifier{}
	g := newEmailFixture(t, orgplugin.Config{}, authsome.WithPlugin(n))
	assert.Equal(t, http.StatusCreated, g.create(t).Code)
	assert.Empty(t, n.all(), "no template, nothing sent")
}

func TestCreateInvitation_sendFailureStillReturnsTheInvitation(t *testing.T) {
	n := &fakeNotifier{err: errors.New("smtp down")}
	f := newEmailFixture(t, orgplugin.Config{InvitationURLTemplate: acceptTemplate}, authsome.WithPlugin(n))

	rec := f.create(t)
	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
	var inv organization.Invitation
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &inv))
	assert.Equal(t, organization.InvitationPending, inv.Status)
	assert.Len(t, n.all(), 1, "the send was attempted")
	assert.True(t, inv.ExpiresAt.After(time.Now()))

	// A failed resend still returns the invitation too.
	seeded := seedInvitation(t, f.p, f.org.ID, f.inviter, "tok-r", sentLongAgo)
	assert.Equal(t, http.StatusOK, resendInvitation(t, f.h, f.org.ID, seeded.ID, f.inviter).Code)
}
