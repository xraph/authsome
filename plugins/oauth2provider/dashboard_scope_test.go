package oauth2provider

import (
	"context"
	"errors"
	"testing"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/id"
)

// TestHandleDashboardDeleteClient_AppScope covers the dashboard's own delete
// action, which is a second way into the same store as the admin REST route.
//
// The page around it is already app-scoped: renderClientsPage resolves the app
// from the dashboard context and both create and list take it. Delete took only
// a client id off the form, so a caller who could reach any app's dashboard
// could post another app's client id and have it removed.
func TestHandleDashboardDeleteClient_AppScope(t *testing.T) {
	seed := func(t *testing.T, st Store, appID id.AppID) *OAuth2Client {
		t.Helper()
		c := &OAuth2Client{
			ID:       id.NewOAuth2ClientID(),
			AppID:    appID,
			ClientID: "seeded",
			Name:     "Seeded",
		}
		if err := st.CreateClient(context.Background(), c); err != nil {
			t.Fatalf("seed client: %v", err)
		}
		return c
	}

	t.Run("refuses to delete another app's client", func(t *testing.T) {
		st := NewMemoryStore()
		p := &Plugin{oauth2Store: st}
		caller, other := id.NewAppID(), id.NewAppID()
		victim := seed(t, st, other)

		err := p.handleDashboardDeleteClient(context.Background(), caller,
			contributor.Params{FormData: map[string]string{"client_id": victim.ID.String()}})

		if !errors.Is(err, ErrClientNotFound) {
			t.Fatalf("want ErrClientNotFound so the page does not disclose another app's client, got %v", err)
		}

		// The property that matters: the row survived.
		if _, getErr := st.GetClientByID(context.Background(), victim.ID); getErr != nil {
			t.Fatalf("another app's client must not be deleted, got %v", getErr)
		}
	})

	t.Run("deletes the caller's own client", func(t *testing.T) {
		st := NewMemoryStore()
		p := &Plugin{oauth2Store: st}
		caller := id.NewAppID()
		own := seed(t, st, caller)

		if err := p.handleDashboardDeleteClient(context.Background(), caller,
			contributor.Params{FormData: map[string]string{"client_id": own.ID.String()}}); err != nil {
			t.Fatalf("deleting the caller's own client: %v", err)
		}

		if _, getErr := st.GetClientByID(context.Background(), own.ID); !errors.Is(getErr, ErrClientNotFound) {
			t.Fatalf("the caller's own client should be gone, got %v", getErr)
		}
	})
}
