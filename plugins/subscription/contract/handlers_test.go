package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/loader"
)

func TestManifest_Loads(t *testing.T) {
	m, err := loader.Load(bytes.NewReader(manifestYAML), "subscription/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if m.Contributor.Name != "subscription" {
		t.Errorf("contributor name = %q, want subscription", m.Contributor.Name)
	}
	if got := len(m.Intents); got != 26 {
		t.Errorf("intents = %d, want 26", got)
	}
}

func TestManifest_Validates(t *testing.T) {
	m, err := loader.Load(bytes.NewReader(manifestYAML), "subscription/contract/manifest.yaml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := loader.Validate(m, contract.NewWardenRegistry()); err != nil {
		t.Errorf("validate: %v", err)
	}
}

func TestPlansListHandler_UnavailableWhenServiceNil(t *testing.T) {
	h := plansListHandler(Deps{})
	_, err := h(context.Background(), struct{}{}, contract.Principal{})
	var ce *contract.Error
	if !errors.As(err, &ce) || ce.Code != contract.CodeUnavailable {
		t.Errorf("expected CodeUnavailable, got %v", err)
	}
}

func TestPricingTiersRequirePlanFeature(t *testing.T) {
	price, err := pricing(0, "usd", "monthly", []PriceTierSummary{{FeatureKey: "requests", Type: "graduated", UpTo: 1000, UnitAmount: 2}})
	if err != nil {
		t.Fatalf("pricing: %v", err)
	}
	if err := validateTierFeatures(price, nil); err == nil {
		t.Fatal("expected missing feature to be rejected")
	}
	features, err := planFeatures([]PlanFeature{{Key: "requests", Name: "Requests", Type: "metered", Limit: 1000, Period: "monthly"}})
	if err != nil {
		t.Fatalf("features: %v", err)
	}
	if err := validateTierFeatures(price, features); err != nil {
		t.Fatalf("valid tier rejected: %v", err)
	}
}

func TestInvoiceDetailWireIsFlat(t *testing.T) {
	encoded, err := json.Marshal(invoiceDetail{invoiceSummary: invoiceSummary{ID: "inv_1", Status: "pending", Total: 1200}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"id":"inv_1"`) || !strings.Contains(string(encoded), `"total":1200`) {
		t.Fatalf("invoice detail wire shape: %s", encoded)
	}
}
