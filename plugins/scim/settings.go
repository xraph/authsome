package scim

import (
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/settings"
)

// Dynamic setting definitions for the SCIM plugin.
var (
	// SettingSCIMEnabled controls whether SCIM provisioning is enabled.
	SettingSCIMEnabled = settings.Define("scim.enabled", false,
		settings.WithDisplayName("Enable SCIM Provisioning"),
		settings.WithDescription("Enable SCIM 2.0 endpoints for automated user and group provisioning"),
		settings.WithCategory("SCIM"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithOrder(10),
	)

	// SettingAutoCreateUsers controls whether users are auto-created on SCIM push.
	SettingAutoCreateUsers = settings.Define("scim.auto_create_users", true,
		settings.WithDisplayName("Auto-Create Users"),
		settings.WithDescription("Automatically create new users when provisioned via SCIM"),
		settings.WithCategory("SCIM"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithVisibleWhen("scim.enabled", "true", "eq"),
		settings.WithOrder(20),
	)

	// SettingAutoSuspendUsers controls auto-suspension on SCIM deactivation.
	SettingAutoSuspendUsers = settings.Define("scim.auto_suspend_users", true,
		settings.WithDisplayName("Auto-Suspend Deprovisioned Users"),
		settings.WithDescription("Automatically suspend users when deactivated via SCIM"),
		settings.WithCategory("SCIM"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithVisibleWhen("scim.enabled", "true", "eq"),
		settings.WithOrder(30),
	)

	// SettingGroupSync controls whether SCIM Groups are synced to teams.
	SettingGroupSync = settings.Define("scim.group_sync", false,
		settings.WithDisplayName("Sync SCIM Groups to Teams"),
		settings.WithDescription("Map SCIM Group resources to organization teams"),
		settings.WithCategory("SCIM"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithVisibleWhen("scim.enabled", "true", "eq"),
		settings.WithOrder(40),
	)

	// SettingDefaultRole is the default org role for SCIM-provisioned users.
	SettingDefaultRole = settings.Define("scim.default_role", "member",
		settings.WithDisplayName("Default Member Role"),
		settings.WithDescription("Default organization role assigned to SCIM-provisioned users"),
		settings.WithCategory("SCIM"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldSelect),
		settings.WithOptions(
			formconfig.SelectOption{Label: "Member", Value: "member"},
			formconfig.SelectOption{Label: "Admin", Value: "admin"},
		),
		settings.WithVisibleWhen("scim.enabled", "true", "eq"),
		settings.WithOrder(50),
	)

	// SettingTokenExpiryDays controls default token expiry.
	SettingTokenExpiryDays = settings.Define("scim.token_expiry_days", 365,
		settings.WithDisplayName("Token Expiry (days)"),
		settings.WithDescription("Default expiry period for new SCIM bearer tokens (0 = no expiry)"),
		settings.WithCategory("SCIM"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Min: intPtr(0), Max: intPtr(3650)}),
		settings.WithHelpText("Set to 0 for tokens that never expire. Default: 365"),
		settings.WithVisibleWhen("scim.enabled", "true", "eq"),
		settings.WithOrder(60),
	)
)

func intPtr(v int) *int { return &v }
