package subscription

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dashauth "github.com/xraph/forge/extensions/dashboard/auth"
	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/dispatcher"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
)

// The dashboard contract asks the plugin's own manage/billing check, so it
// fails closed in the same places the HTTP routes do.
func TestRegisterContract_UsesBillingManageCheck(t *testing.T) {
	operator, member := id.NewUserID(), id.NewUserID()
	f := newBillingFixture(t, "organization", id.NewOrgID().String())
	f.engine.managers[operator.String()] = true

	disp := dispatcher.New(dispatcher.NoopMetricsEmitter{})
	require.NoError(t, f.plugin.RegisterContract(disp, contract.NewRegistry(), contract.NewWardenRegistry(), &authsome.Engine{}))

	listInvoices := func(userID id.UserID) error {
		p := contract.Principal{User: &dashauth.UserInfo{Subject: userID.String()}, Claims: map[string]any{"app_id": f.appID}}
		_, _, err := disp.Dispatch(context.Background(), contract.Request{
			Kind: contract.KindQuery, Contributor: "subscription", Intent: "invoices.list", IntentVersion: 1,
		}, p)
		return err
	}
	denied := func(err error) bool {
		var ce *contract.Error
		return errors.As(err, &ce) && ce.Code == contract.CodePermissionDenied
	}

	assert.NoError(t, listInvoices(operator))
	assert.True(t, denied(listInvoices(member)), "a plain member is refused")

	f.plugin.engine = bareEngine{}
	assert.True(t, denied(listInvoices(operator)), "an engine without RBAC refuses everyone")
}
