package oauth2provider

import (
	"context"
	"errors"
	"testing"

	"github.com/xraph/authsome/id"
	oauth2contract "github.com/xraph/authsome/plugins/oauth2provider/contract"
)

func TestToContractErr(t *testing.T) {
	var v *oauth2contract.ValidationError
	if err := toContractErr(invalidClient("name required")); !errors.As(err, &v) || v.Msg != "name required" {
		t.Errorf("validation = %v", err)
	}
	for _, c := range []struct{ in, want error }{
		{ErrClientNotFound, oauth2contract.ErrClientNotFound},
		{ErrOperatorKey, oauth2contract.ErrOperatorKey},
		{errNoSigningKey, oauth2contract.ErrNoSigningKey},
	} {
		if got := toContractErr(c.in); !errors.Is(got, c.want) {
			t.Errorf("toContractErr(%v) = %v, want %v", c.in, got, c.want)
		}
	}
	if toContractErr(nil) != nil {
		t.Error("nil should stay nil")
	}
}

func TestContractService_BlankNameIsValidationError(t *testing.T) {
	s := contractService{p: &Plugin{}}
	_, err := s.CreateClientFor(context.Background(), id.NewAppID(), oauth2contract.CreateParams{})
	var v *oauth2contract.ValidationError
	if !errors.As(err, &v) {
		t.Fatalf("err = %v, want *ValidationError", err)
	}
}
