package organization_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	orgplugin "github.com/xraph/authsome/plugins/organization"
	"github.com/xraph/authsome/store"
)

// Creating an organization with a slug another org in the app already holds
// answered 500, so a client couldn't tell "pick another slug" from a crash.
// It must answer 409.

func TestCreateOrg_TakenSlugIsConflict(t *testing.T) {
	_, h := newOrgHTTP(t)
	caller := id.NewUserID()
	body := postJSON(t, map[string]string{"name": "Acme", "slug": "taken"})

	first := httptest.NewRecorder()
	h.ServeHTTP(first, appReq(t, orgReq("POST", "/v1/orgs", body, caller)))
	require.Equal(t, http.StatusCreated, first.Code, "body=%s", first.Body.String())

	second := httptest.NewRecorder()
	h.ServeHTTP(second, appReq(t, orgReq("POST", "/v1/orgs", body, id.NewUserID())))
	assert.Equal(t, http.StatusConflict, second.Code, "body=%s", second.Body.String())
	assert.Contains(t, second.Body.String(), "slug")
}

// Callers of the Go API get a sentinel they can test for, and the memory
// store refuses the duplicate the way the SQL and mongo unique indexes do.
func TestCreateOrganization_TakenSlugIsErrSlugTaken(t *testing.T) {
	s := newTxTestSetup(t)
	appID, err := id.ParseAppID(orgTestAppID)
	require.NoError(t, err)

	dupe := &organization.Organization{
		ID:        id.NewOrgID(),
		AppID:     appID,
		Name:      "Dupe",
		Slug:      s.org.Slug,
		CreatedBy: id.NewUserID(),
		CreatedAt: time.Now(),
	}
	err = s.plugin.CreateOrganization(context.Background(), dupe)
	require.Error(t, err)
	assert.ErrorIs(t, err, orgplugin.ErrSlugTaken)

	err = s.eng.Store().CreateOrganization(context.Background(), dupe)
	assert.ErrorIs(t, err, store.ErrConflict, "the store itself must refuse a taken slug")
}
