package settings

import (
	"encoding/json"
	"testing"

	"github.com/xraph/authsome/formconfig"
)

func TestUIMetadata_DefineWithInputType(t *testing.T) {
	def := Define("ui.test", 42,
		WithInputType(formconfig.FieldNumber),
		WithHelpText("test help"),
		WithPlaceholder("enter number"),
		WithOrder(5),
	)

	if def.Def.UI == nil {
		t.Fatal("expected UI metadata to be set")
	}
	if def.Def.UI.InputType != formconfig.FieldNumber {
		t.Errorf("expected FieldNumber, got %s", def.Def.UI.InputType)
	}
	if def.Def.UI.HelpText != "test help" {
		t.Errorf("expected 'test help', got %s", def.Def.UI.HelpText)
	}
	if def.Def.UI.Placeholder != "enter number" {
		t.Errorf("expected 'enter number', got %s", def.Def.UI.Placeholder)
	}
	if def.Def.UI.Order != 5 {
		t.Errorf("expected order 5, got %d", def.Def.UI.Order)
	}
}

func TestUIMetadata_AutoInferBool(t *testing.T) {
	def := Define("ui.flag", false,
		WithHelpText("toggle this"),
	)

	if def.Def.UI == nil {
		t.Fatal("expected UI metadata to be set")
	}
	if def.Def.UI.InputType != formconfig.FieldSwitch {
		t.Errorf("expected auto-inferred FieldSwitch for bool, got %s", def.Def.UI.InputType)
	}
}

func TestUIMetadata_AutoInferInt(t *testing.T) {
	def := Define("ui.count", 10,
		WithHelpText("a number"),
	)

	if def.Def.UI == nil {
		t.Fatal("expected UI metadata to be set")
	}
	if def.Def.UI.InputType != formconfig.FieldNumber {
		t.Errorf("expected auto-inferred FieldNumber for int, got %s", def.Def.UI.InputType)
	}
}

func TestUIMetadata_AutoInferString(t *testing.T) {
	def := Define("ui.name", "hello",
		WithHelpText("a text field"),
	)

	if def.Def.UI == nil {
		t.Fatal("expected UI metadata to be set")
	}
	if def.Def.UI.InputType != formconfig.FieldText {
		t.Errorf("expected auto-inferred FieldText for string, got %s", def.Def.UI.InputType)
	}
}

func TestUIMetadata_AutoInferSlice(t *testing.T) {
	def := Define("ui.list", []string{},
		WithHelpText("a list"),
	)

	if def.Def.UI == nil {
		t.Fatal("expected UI metadata to be set")
	}
	if def.Def.UI.InputType != formconfig.FieldTextarea {
		t.Errorf("expected auto-inferred FieldTextarea for []string, got %s", def.Def.UI.InputType)
	}
}

func TestUIMetadata_ExplicitOverridesInference(t *testing.T) {
	// When user explicitly sets InputType, inference should NOT override it.
	def := Define("ui.flag", false,
		WithInputType(formconfig.FieldCheckbox),
		WithHelpText("explicit checkbox, not switch"),
	)

	if def.Def.UI == nil {
		t.Fatal("expected UI metadata to be set")
	}
	if def.Def.UI.InputType != formconfig.FieldCheckbox {
		t.Errorf("expected FieldCheckbox (explicit), got %s", def.Def.UI.InputType)
	}
}

func TestUIMetadata_NoUIWhenNoUIOptionsUsed(t *testing.T) {
	def := Define("plain.key", 42, WithScopes(ScopeGlobal))
	if def.Def.UI != nil {
		t.Error("expected nil UI when no UI options were used")
	}
}

func TestUIMetadata_WithOptions(t *testing.T) {
	def := Define("ui.choice", "fast",
		WithInputType(formconfig.FieldSelect),
		WithOptions(
			formconfig.SelectOption{Label: "Fast", Value: "fast"},
			formconfig.SelectOption{Label: "Slow", Value: "slow"},
		),
	)

	if def.Def.UI == nil {
		t.Fatal("expected UI metadata to be set")
	}
	if len(def.Def.UI.Options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(def.Def.UI.Options))
	}
	if def.Def.UI.Options[0].Value != "fast" {
		t.Errorf("expected first option value 'fast', got %s", def.Def.UI.Options[0].Value)
	}
}

func TestUIMetadata_WithUIValidation(t *testing.T) {
	def := Define("ui.validated", 10,
		WithInputType(formconfig.FieldNumber),
		WithUIValidation(formconfig.Validation{Required: true, Min: intPtr(1), Max: intPtr(100)}),
	)

	if def.Def.UI == nil {
		t.Fatal("expected UI metadata to be set")
	}
	if def.Def.UI.Validation == nil {
		t.Fatal("expected validation to be set")
	}
	if !def.Def.UI.Validation.Required {
		t.Error("expected Required=true")
	}
	if *def.Def.UI.Validation.Min != 1 {
		t.Errorf("expected Min=1, got %d", *def.Def.UI.Validation.Min)
	}
	if *def.Def.UI.Validation.Max != 100 {
		t.Errorf("expected Max=100, got %d", *def.Def.UI.Validation.Max)
	}
}

func TestUIMetadata_WithReadOnly(t *testing.T) {
	def := Define("ui.readonly", "locked",
		WithReadOnly(),
	)

	if def.Def.UI == nil {
		t.Fatal("expected UI metadata to be set")
	}
	if !def.Def.UI.ReadOnly {
		t.Error("expected ReadOnly=true")
	}
}

func TestUIMetadata_WithVisibleWhen(t *testing.T) {
	def := Define("ui.conditional", "val",
		WithVisibleWhen("ui.flag", true),
	)

	if def.Def.UI == nil {
		t.Fatal("expected UI metadata to be set")
	}
	if def.Def.UI.Condition == nil {
		t.Fatal("expected condition to be set")
	}
	if def.Def.UI.Condition.Key != "ui.flag" {
		t.Errorf("expected condition key 'ui.flag', got %s", def.Def.UI.Condition.Key)
	}
	if def.Def.UI.Condition.Operator != "eq" {
		t.Errorf("expected operator 'eq', got %s", def.Def.UI.Condition.Operator)
	}

	var v bool
	if err := json.Unmarshal(def.Def.UI.Condition.Value, &v); err != nil {
		t.Fatalf("unmarshal condition value: %v", err)
	}
	if !v {
		t.Error("expected condition value true")
	}
}

func TestUIMetadata_WithVisibleWhen_CustomOperator(t *testing.T) {
	def := Define("ui.conditional2", "val",
		WithVisibleWhen("ui.mode", "advanced", "ne"),
	)

	if def.Def.UI.Condition.Operator != "ne" {
		t.Errorf("expected operator 'ne', got %s", def.Def.UI.Condition.Operator)
	}
}

func TestUIMetadata_WithSection(t *testing.T) {
	def := Define("ui.sectioned", 42,
		WithSection("Advanced"),
		WithHelpText("in advanced section"),
	)

	if def.Def.UI == nil {
		t.Fatal("expected UI metadata to be set")
	}
	if def.Def.UI.Section != "Advanced" {
		t.Errorf("expected section 'Advanced', got %s", def.Def.UI.Section)
	}
}

func TestUIMetadata_WithUI(t *testing.T) {
	ui := UIMetadata{
		InputType:   formconfig.FieldSelect,
		HelpText:    "full ui",
		Placeholder: "pick one",
		Order:       42,
	}
	def := Define("ui.full", "val", WithUI(ui))

	if def.Def.UI == nil {
		t.Fatal("expected UI metadata to be set")
	}
	if def.Def.UI.InputType != formconfig.FieldSelect {
		t.Errorf("expected FieldSelect, got %s", def.Def.UI.InputType)
	}
	if def.Def.UI.Order != 42 {
		t.Errorf("expected order 42, got %d", def.Def.UI.Order)
	}
}

func TestUIMetadata_JSONRoundTrip(t *testing.T) {
	def := Define("ui.json", 42,
		WithDisplayName("JSON Test"),
		WithDescription("test json roundtrip"),
		WithCategory("Testing"),
		WithInputType(formconfig.FieldNumber),
		WithHelpText("help text"),
		WithUIValidation(formconfig.Validation{Required: true}),
		WithOrder(10),
	)

	// Marshal Definition to JSON.
	data, err := json.Marshal(def.Def)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Unmarshal back.
	var d Definition
	if err := json.Unmarshal(data, &d); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if d.Key != "ui.json" {
		t.Errorf("expected key 'ui.json', got %s", d.Key)
	}
	if d.UI == nil {
		t.Fatal("expected UI metadata after roundtrip")
	}
	if d.UI.InputType != formconfig.FieldNumber {
		t.Errorf("expected FieldNumber after roundtrip, got %s", d.UI.InputType)
	}
	if d.UI.HelpText != "help text" {
		t.Errorf("expected 'help text' after roundtrip, got %s", d.UI.HelpText)
	}
	if d.UI.Validation == nil || !d.UI.Validation.Required {
		t.Error("expected validation to survive roundtrip")
	}
}

func intPtr(i int) *int { return &i }
