package cms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/contexts"
	env "github.com/xraph/authsome/core/environment"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/authsome/plugins/cms/pages"
	"github.com/xraph/forgeui/bridge"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// DashboardExtension implements ui.DashboardExtension for the CMS plugin.
type DashboardExtension struct {
	plugin     *Plugin
	baseUIPath string
}

// NewDashboardExtension creates a new dashboard extension.
func NewDashboardExtension(plugin *Plugin) *DashboardExtension {
	return &DashboardExtension{
		plugin:     plugin,
		baseUIPath: "/api/identity/ui", // Default base path
	}
}

// SetRegistry sets the extension registry reference (deprecated but kept for compatibility).
func (e *DashboardExtension) SetRegistry(registry any) {
	// No longer needed - layout handled by ForgeUI
}

// ExtensionID returns the unique identifier for this extension.
func (e *DashboardExtension) ExtensionID() string {
	return "cms"
}

// NavigationItems returns navigation items for the dashboard.
func (e *DashboardExtension) NavigationItems() []ui.NavigationItem {
	return []ui.NavigationItem{
		{
			ID:       "cms",
			Label:    "Content",
			Icon:     lucide.Database(Class("size-4")),
			Position: ui.NavPositionMain,
			Order:    35, // After Users (20), before Secrets (60)
			URLBuilder: func(basePath string, currentApp *app.App) string {
				if currentApp == nil {
					return basePath + "/cms"
				}

				return basePath + "/app/" + currentApp.ID.String() + "/cms"
			},
			ActiveChecker: func(activePage string) bool {
				return activePage == "cms" ||
					activePage == "cms-types" ||
					activePage == "cms-type-detail" ||
					activePage == "cms-type-create" ||
					activePage == "cms-entries" ||
					activePage == "cms-entry-detail" ||
					activePage == "cms-entry-create" ||
					activePage == "cms-entry-edit" ||
					activePage == "cms-components"
			},
			RequiresPlugin: "cms",
		},
	}
}

// Routes returns dashboard routes.
func (e *DashboardExtension) Routes() []ui.Route {
	return []ui.Route{
		// CMS Settings Page (in Settings section)
		{
			Method:       "GET",
			Path:         "/settings/cms",
			Handler:      e.ServeCMSSettings,
			Name:         "cms.dashboard.settings",
			Summary:      "CMS Settings",
			Description:  "Configure CMS settings and content types",
			Tags:         []string{"Dashboard", "Settings", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// CMS Overview
		{
			Method:       "GET",
			Path:         "/cms",
			Handler:      e.ServeCMSOverview,
			Name:         "cms.dashboard.overview",
			Summary:      "CMS Overview",
			Description:  "View CMS overview with content type list",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Content Types list
		{
			Method:       "GET",
			Path:         "/cms/types",
			Handler:      e.ServeContentTypesList,
			Name:         "cms.dashboard.types.list",
			Summary:      "Content Types",
			Description:  "List all content types",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Create Content Type page
		{
			Method:       "GET",
			Path:         "/cms/types/create",
			Handler:      e.ServeCreateContentType,
			Name:         "cms.dashboard.types.create",
			Summary:      "Create Content Type",
			Description:  "Create a new content type",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Create Content Type action
		{
			Method:       "POST",
			Path:         "/cms/types/create",
			Handler:      e.HandleCreateContentType,
			Name:         "cms.dashboard.types.create.submit",
			Summary:      "Submit Create Content Type",
			Description:  "Process content type creation form",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Content Type detail
		{
			Method:       "GET",
			Path:         "/cms/types/:typeName",
			Handler:      e.ServeContentTypeDetail,
			Name:         "cms.dashboard.types.detail",
			Summary:      "Content Type Detail",
			Description:  "View content type details and fields",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Delete Content Type action
		{
			Method:       "POST",
			Path:         "/cms/types/:typeName/delete",
			Handler:      e.HandleDeleteContentType,
			Name:         "cms.dashboard.types.delete",
			Summary:      "Delete Content Type",
			Description:  "Delete a content type and all its fields",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Add Field action
		{
			Method:       "POST",
			Path:         "/cms/types/:typeName/fields",
			Handler:      e.HandleAddField,
			Name:         "cms.dashboard.fields.create",
			Summary:      "Add Field",
			Description:  "Add a new field to a content type",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Update Field action
		{
			Method:       "POST",
			Path:         "/cms/types/:typeName/fields/:fieldName/update",
			Handler:      e.HandleUpdateField,
			Name:         "cms.dashboard.fields.update",
			Summary:      "Update Field",
			Description:  "Update a field in a content type",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Delete Field action
		{
			Method:       "POST",
			Path:         "/cms/types/:typeName/fields/:fieldName/delete",
			Handler:      e.HandleDeleteField,
			Name:         "cms.dashboard.fields.delete",
			Summary:      "Delete Field",
			Description:  "Delete a field from a content type",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Update Display Settings
		{
			Method:       "POST",
			Path:         "/cms/types/:typeName/settings/display",
			Handler:      e.HandleUpdateDisplaySettings,
			Name:         "cms.dashboard.settings.display",
			Summary:      "Update Display Settings",
			Description:  "Update content type display settings (title, description, preview fields)",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Update Feature Settings
		{
			Method:       "POST",
			Path:         "/cms/types/:typeName/settings/features",
			Handler:      e.HandleUpdateFeatureSettings,
			Name:         "cms.dashboard.settings.features",
			Summary:      "Update Feature Settings",
			Description:  "Update content type feature settings (revisions, drafts, etc.)",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Content Entries list
		{
			Method:       "GET",
			Path:         "/cms/types/:typeName/entries",
			Handler:      e.ServeEntriesList,
			Name:         "cms.dashboard.entries.list",
			Summary:      "Content Entries",
			Description:  "List entries for a content type",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Create Entry page
		{
			Method:       "GET",
			Path:         "/cms/types/:typeName/entries/create",
			Handler:      e.ServeCreateEntry,
			Name:         "cms.dashboard.entries.create",
			Summary:      "Create Entry",
			Description:  "Create a new content entry",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Create Entry action
		{
			Method:       "POST",
			Path:         "/cms/types/:typeName/entries/create",
			Handler:      e.HandleCreateEntry,
			Name:         "cms.dashboard.entries.create.submit",
			Summary:      "Submit Create Entry",
			Description:  "Process entry creation form",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Entry detail
		{
			Method:       "GET",
			Path:         "/cms/types/:typeName/entries/:entryId",
			Handler:      e.ServeEntryDetail,
			Name:         "cms.dashboard.entries.detail",
			Summary:      "Entry Detail",
			Description:  "View entry details",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Edit Entry page
		{
			Method:       "GET",
			Path:         "/cms/types/:typeName/entries/:entryId/edit",
			Handler:      e.ServeEditEntry,
			Name:         "cms.dashboard.entries.edit",
			Summary:      "Edit Entry",
			Description:  "Edit a content entry",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Update Entry action
		{
			Method:       "POST",
			Path:         "/cms/types/:typeName/entries/:entryId/update",
			Handler:      e.HandleUpdateEntry,
			Name:         "cms.dashboard.entries.update",
			Summary:      "Update Entry",
			Description:  "Process entry update form",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Delete Entry action
		{
			Method:       "POST",
			Path:         "/cms/types/:typeName/entries/:entryId/delete",
			Handler:      e.HandleDeleteEntry,
			Name:         "cms.dashboard.entries.delete",
			Summary:      "Delete Entry",
			Description:  "Delete a content entry",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Component Schemas list
		{
			Method:       "GET",
			Path:         "/cms/components",
			Handler:      e.ServeComponentSchemasList,
			Name:         "cms.dashboard.components.list",
			Summary:      "Component Schemas",
			Description:  "List all component schemas",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Create Component Schema page
		{
			Method:       "GET",
			Path:         "/cms/components/create",
			Handler:      e.ServeCreateComponentSchema,
			Name:         "cms.dashboard.components.create",
			Summary:      "Create Component Schema",
			Description:  "Create a new component schema",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Create Component Schema action
		{
			Method:       "POST",
			Path:         "/cms/components/create",
			Handler:      e.HandleCreateComponentSchema,
			Name:         "cms.dashboard.components.create.submit",
			Summary:      "Submit Create Component Schema",
			Description:  "Process component schema creation form",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Component Schema detail/edit
		{
			Method:       "GET",
			Path:         "/cms/components/:componentSlug",
			Handler:      e.ServeComponentSchemaDetail,
			Name:         "cms.dashboard.components.detail",
			Summary:      "Component Schema Detail",
			Description:  "View/edit component schema",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Update Component Schema action
		{
			Method:       "POST",
			Path:         "/cms/components/:componentSlug",
			Handler:      e.HandleUpdateComponentSchema,
			Name:         "cms.dashboard.components.update",
			Summary:      "Update Component Schema",
			Description:  "Process component schema update form",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Delete Component Schema action
		{
			Method:       "POST",
			Path:         "/cms/components/:componentSlug/delete",
			Handler:      e.HandleDeleteComponentSchema,
			Name:         "cms.dashboard.components.delete",
			Summary:      "Delete Component Schema",
			Description:  "Delete a component schema",
			Tags:         []string{"Dashboard", "CMS"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
	}
}

// SettingsSections returns settings sections (deprecated).
func (e *DashboardExtension) SettingsSections() []ui.SettingsSection {
	return nil
}

// SettingsPages returns settings pages.
func (e *DashboardExtension) SettingsPages() []ui.SettingsPage {
	return []ui.SettingsPage{
		{
			ID:            "cms-settings",
			Label:         "Content Management",
			Description:   "Configure CMS settings and content types",
			Icon:          lucide.Database(Class("h-5 w-5")),
			Category:      "integrations",
			Order:         30,
			Path:          "cms",
			RequirePlugin: "cms",
			RequireAdmin:  true,
		},
	}
}

// DashboardWidgets returns dashboard widgets.
func (e *DashboardExtension) DashboardWidgets() []ui.DashboardWidget {
	return []ui.DashboardWidget{
		{
			ID:    "cms-stats",
			Title: "Content",
			Icon:  lucide.Database(Class("size-5")),
			Order: 35,
			Size:  1,
			Renderer: func(basePath string, currentApp *app.App) g.Node {
				return e.renderCMSWidget(currentApp)
			},
		},
	}
}

// BridgeFunctions returns bridge functions for CMS.
func (e *DashboardExtension) BridgeFunctions() []ui.BridgeFunction {
	return []ui.BridgeFunction{
		// Content Type Operations
		{
			Name:        "getContentTypes",
			Handler:     e.bridgeGetContentTypes,
			Description: "Get list of content types for an app",
		},
		{
			Name:        "getContentType",
			Handler:     e.bridgeGetContentType,
			Description: "Get a specific content type by name",
		},
		{
			Name:        "createContentType",
			Handler:     e.bridgeCreateContentType,
			Description: "Create a new content type",
		},
		{
			Name:        "updateContentType",
			Handler:     e.bridgeUpdateContentType,
			Description: "Update a content type",
		},
		{
			Name:        "deleteContentType",
			Handler:     e.bridgeDeleteContentType,
			Description: "Delete a content type",
		},
		{
			Name:        "getContentTypeStats",
			Handler:     e.bridgeGetContentTypeStats,
			Description: "Get statistics for content types",
		},

		// Field Operations
		{
			Name:        "listFields",
			Handler:     e.bridgeListFields,
			Description: "List all fields for a content type",
		},
		{
			Name:        "addField",
			Handler:     e.bridgeAddField,
			Description: "Add a new field to a content type",
		},
		{
			Name:        "updateField",
			Handler:     e.bridgeUpdateField,
			Description: "Update a field in a content type",
		},
		{
			Name:        "deleteField",
			Handler:     e.bridgeDeleteField,
			Description: "Delete a field from a content type",
		},
		{
			Name:        "reorderFields",
			Handler:     e.bridgeReorderFields,
			Description: "Reorder fields in a content type",
		},
		{
			Name:        "getFieldTypes",
			Handler:     e.bridgeGetFieldTypes,
			Description: "Get all available field types",
		},

		// Entry Operations
		{
			Name:        "getEntries",
			Handler:     e.bridgeGetEntries,
			Description: "Get content entries for a type",
		},
		{
			Name:        "getEntry",
			Handler:     e.bridgeGetEntry,
			Description: "Get a specific entry by ID",
		},
		{
			Name:        "createEntry",
			Handler:     e.bridgeCreateEntry,
			Description: "Create a new content entry",
		},
		{
			Name:        "updateEntry",
			Handler:     e.bridgeUpdateEntry,
			Description: "Update a content entry",
		},
		{
			Name:        "deleteEntry",
			Handler:     e.bridgeDeleteEntry,
			Description: "Delete a content entry",
		},
		{
			Name:        "publishEntry",
			Handler:     e.bridgePublishEntry,
			Description: "Publish a content entry",
		},
		{
			Name:        "unpublishEntry",
			Handler:     e.bridgeUnpublishEntry,
			Description: "Unpublish a content entry",
		},
		{
			Name:        "archiveEntry",
			Handler:     e.bridgeArchiveEntry,
			Description: "Archive a content entry",
		},
		{
			Name:        "getEntryStats",
			Handler:     e.bridgeGetEntryStats,
			Description: "Get statistics for entries of a content type",
		},

		// Bulk Operations
		{
			Name:        "bulkPublish",
			Handler:     e.bridgeBulkPublish,
			Description: "Publish multiple entries",
		},
		{
			Name:        "bulkUnpublish",
			Handler:     e.bridgeBulkUnpublish,
			Description: "Unpublish multiple entries",
		},
		{
			Name:        "bulkDelete",
			Handler:     e.bridgeBulkDelete,
			Description: "Delete multiple entries",
		},

		// Component Schema Operations
		{
			Name:        "getComponentSchemas",
			Handler:     e.bridgeGetComponentSchemas,
			Description: "Get list of component schemas",
		},
		{
			Name:        "getComponentSchema",
			Handler:     e.bridgeGetComponentSchema,
			Description: "Get a specific component schema by name",
		},
		{
			Name:        "createComponentSchema",
			Handler:     e.bridgeCreateComponentSchema,
			Description: "Create a new component schema",
		},
		{
			Name:        "updateComponentSchema",
			Handler:     e.bridgeUpdateComponentSchema,
			Description: "Update a component schema",
		},
		{
			Name:        "deleteComponentSchema",
			Handler:     e.bridgeDeleteComponentSchema,
			Description: "Delete a component schema",
		},

		// Revision Operations
		{
			Name:        "listRevisions",
			Handler:     e.bridgeListRevisions,
			Description: "List revisions for an entry",
		},
		{
			Name:        "getRevision",
			Handler:     e.bridgeGetRevision,
			Description: "Get a specific revision",
		},
		{
			Name:        "restoreRevision",
			Handler:     e.bridgeRestoreRevision,
			Description: "Restore an entry to a specific revision",
		},
	}
}

// =============================================================================
// Bridge Function Implementations
// =============================================================================

// buildContextFromBridge builds a Go context with app ID, env ID, and user ID from bridge context.
func (e *DashboardExtension) buildContextFromBridge(bridgeCtx bridge.Context, appID string) (context.Context, error) {
	// Parse and set app ID
	id, err := xid.FromString(appID)
	if err != nil {
		return nil, errs.BadRequest("invalid appId")
	}

	goCtx := contexts.SetAppID(context.Background(), id)

	// Extract environment ID from request or cookie
	envIDSet := false

	if req := bridgeCtx.Request(); req != nil {
		// Try to get envId from request parameter
		if envIDStr := req.FormValue("envId"); envIDStr != "" {
			if envID, err := xid.FromString(envIDStr); err == nil {
				goCtx = contexts.SetEnvironmentID(goCtx, envID)
				envIDSet = true
			}
		} else if envCookie, err := req.Cookie("authsome_env"); err == nil && envCookie.Value != "" {
			// Fallback to cookie
			if envID, err := xid.FromString(envCookie.Value); err == nil {
				goCtx = contexts.SetEnvironmentID(goCtx, envID)
				envIDSet = true
			}
		}
	}

	// If no environment specified, get the default environment for this app
	if !envIDSet {
		serviceRegistry := e.plugin.authInst.GetServiceRegistry()
		if serviceRegistry != nil && serviceRegistry.HasEnvironmentService() {
			envSvc := serviceRegistry.EnvironmentService()
			if envSvc != nil {
				// List environments for this app and get the first one (default)
				filter := &env.ListEnvironmentsFilter{
					AppID: id,
				}
				if envResp, err := envSvc.ListEnvironments(goCtx, filter); err == nil && envResp != nil && len(envResp.Data) > 0 {
					// Look for production or default environment first
					for _, environment := range envResp.Data {
						if environment.Name == "production" || environment.Name == "default" {
							goCtx = contexts.SetEnvironmentID(goCtx, environment.ID)
							envIDSet = true

							break
						}
					}
					// If not found, use the first environment
					if !envIDSet && len(envResp.Data) > 0 {
						goCtx = contexts.SetEnvironmentID(goCtx, envResp.Data[0].ID)
					}
				}
			}
		}
	}

	// Extract user ID from bridge context (authenticated user)
	if user := bridgeCtx.User(); user != nil {
		userData := user.Data()
		if userIDStr, ok := userData["id"].(string); ok {
			if userID, err := xid.FromString(userIDStr); err == nil {
				goCtx = contexts.SetUserID(goCtx, userID)
			}
		}
	}

	return goCtx, nil
}

// =============================================================================
// Content Type Bridge Functions
// =============================================================================

type BridgeContentTypeInput struct {
	AppID string `json:"appId"          validate:"required"`
	Name  string `json:"name,omitempty"`
}

type BridgeCreateContentTypeInput struct {
	AppID       string `json:"appId"                 validate:"required"`
	Title       string `json:"title"                 validate:"required"`
	Name        string `json:"name"                  validate:"required"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

type BridgeUpdateContentTypeInput struct {
	AppID       string                       `json:"appId"                 validate:"required"`
	Name        string                       `json:"name"                  validate:"required"`
	Title       string                       `json:"title,omitempty"`
	Description string                       `json:"description,omitempty"`
	Icon        string                       `json:"icon,omitempty"`
	Settings    *core.ContentTypeSettingsDTO `json:"settings,omitempty"`
}

func (e *DashboardExtension) bridgeGetContentTypes(ctx bridge.Context, input BridgeContentTypeInput) (*core.ListContentTypesResponse, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	return e.plugin.contentTypeSvc.List(goCtx, &core.ListContentTypesQuery{})
}

func (e *DashboardExtension) bridgeGetContentType(ctx bridge.Context, input BridgeContentTypeInput) (*core.ContentTypeDTO, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	return e.plugin.contentTypeSvc.GetByName(goCtx, input.Name)
}

func (e *DashboardExtension) bridgeCreateContentType(ctx bridge.Context, input BridgeCreateContentTypeInput) (*core.ContentTypeDTO, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	return e.plugin.contentTypeSvc.Create(goCtx, &core.CreateContentTypeRequest{
		Title:       input.Title,
		Name:        input.Name,
		Description: input.Description,
		Icon:        input.Icon,
	})
}

func (e *DashboardExtension) bridgeUpdateContentType(ctx bridge.Context, input BridgeUpdateContentTypeInput) (*core.ContentTypeDTO, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Get content type to get its ID
	contentType, err := e.plugin.contentTypeSvc.GetByName(goCtx, input.Name)
	if err != nil {
		return nil, err
	}

	contentTypeID, err := xid.FromString(contentType.ID)
	if err != nil {
		return nil, errs.BadRequest("invalid content type ID")
	}

	return e.plugin.contentTypeSvc.Update(goCtx, contentTypeID, &core.UpdateContentTypeRequest{
		Title:       input.Title,
		Description: input.Description,
		Icon:        input.Icon,
		Settings:    input.Settings,
	})
}

func (e *DashboardExtension) bridgeDeleteContentType(ctx bridge.Context, input BridgeContentTypeInput) (map[string]bool, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Get content type to get its ID
	contentType, err := e.plugin.contentTypeSvc.GetByName(goCtx, input.Name)
	if err != nil {
		return nil, err
	}

	contentTypeID, err := xid.FromString(contentType.ID)
	if err != nil {
		return nil, errs.BadRequest("invalid content type ID")
	}

	err = e.plugin.contentTypeSvc.Delete(goCtx, contentTypeID)
	if err != nil {
		return nil, err
	}

	return map[string]bool{"success": true}, nil
}

func (e *DashboardExtension) bridgeGetContentTypeStats(ctx bridge.Context, input BridgeContentTypeInput) (*core.CMSStatsDTO, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	return e.plugin.contentTypeSvc.GetStats(goCtx)
}

// =============================================================================
// Field Bridge Functions
// =============================================================================

type BridgeFieldInput struct {
	AppID     string `json:"appId"               validate:"required"`
	TypeName  string `json:"typeName"            validate:"required"`
	FieldName string `json:"fieldName,omitempty"`
}

type BridgeCreateFieldInput struct {
	AppID       string                `json:"appId"                 validate:"required"`
	TypeName    string                `json:"typeName"              validate:"required"`
	Title       string                `json:"title"                 validate:"required"`
	Name        string                `json:"name"                  validate:"required"`
	Type        string                `json:"type"                  validate:"required"`
	Description string                `json:"description,omitempty"`
	Required    bool                  `json:"required,omitempty"`
	Unique      bool                  `json:"unique,omitempty"`
	Indexed     bool                  `json:"indexed,omitempty"`
	Localized   bool                  `json:"localized,omitempty"`
	Options     *core.FieldOptionsDTO `json:"options,omitempty"`
}

type BridgeUpdateFieldInput struct {
	AppID       string                `json:"appId"                 validate:"required"`
	TypeName    string                `json:"typeName"              validate:"required"`
	FieldName   string                `json:"fieldName"             validate:"required"`
	Title       string                `json:"title,omitempty"`
	Description string                `json:"description,omitempty"`
	Required    *bool                 `json:"required,omitempty"`
	Unique      *bool                 `json:"unique,omitempty"`
	Indexed     *bool                 `json:"indexed,omitempty"`
	Localized   *bool                 `json:"localized,omitempty"`
	Options     *core.FieldOptionsDTO `json:"options,omitempty"`
}

type BridgeReorderFieldsInput struct {
	AppID      string   `json:"appId"      validate:"required"`
	TypeName   string   `json:"typeName"   validate:"required"`
	FieldOrder []string `json:"fieldOrder" validate:"required"`
}

func (e *DashboardExtension) bridgeListFields(ctx bridge.Context, input BridgeFieldInput) (map[string]any, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(goCtx, input.TypeName)
	if err != nil {
		return nil, err
	}

	contentTypeID, err := xid.FromString(contentType.ID)
	if err != nil {
		return nil, errs.BadRequest("invalid content type ID")
	}

	fields, err := e.plugin.fieldSvc.List(goCtx, contentTypeID)
	if err != nil {
		return nil, err
	}

	return map[string]any{"fields": fields}, nil
}

func (e *DashboardExtension) bridgeAddField(ctx bridge.Context, input BridgeCreateFieldInput) (*core.ContentFieldDTO, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(goCtx, input.TypeName)
	if err != nil {
		return nil, err
	}

	contentTypeID, err := xid.FromString(contentType.ID)
	if err != nil {
		return nil, errs.BadRequest("invalid content type ID")
	}

	return e.plugin.fieldSvc.Create(goCtx, contentTypeID, &core.CreateFieldRequest{
		Title:       input.Title,
		Name:        input.Name,
		Type:        input.Type,
		Description: input.Description,
		Required:    input.Required,
		Unique:      input.Unique,
		Indexed:     input.Indexed,
		Localized:   input.Localized,
		Options:     input.Options,
	})
}

func (e *DashboardExtension) bridgeUpdateField(ctx bridge.Context, input BridgeUpdateFieldInput) (*core.ContentFieldDTO, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(goCtx, input.TypeName)
	if err != nil {
		return nil, err
	}

	contentTypeID, err := xid.FromString(contentType.ID)
	if err != nil {
		return nil, errs.BadRequest("invalid content type ID")
	}

	return e.plugin.fieldSvc.UpdateByName(goCtx, contentTypeID, input.FieldName, &core.UpdateFieldRequest{
		Title:       input.Title,
		Description: input.Description,
		Required:    input.Required,
		Unique:      input.Unique,
		Indexed:     input.Indexed,
		Localized:   input.Localized,
		Options:     input.Options,
	})
}

func (e *DashboardExtension) bridgeDeleteField(ctx bridge.Context, input BridgeFieldInput) (map[string]bool, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(goCtx, input.TypeName)
	if err != nil {
		return nil, err
	}

	contentTypeID, err := xid.FromString(contentType.ID)
	if err != nil {
		return nil, errs.BadRequest("invalid content type ID")
	}

	err = e.plugin.fieldSvc.DeleteByName(goCtx, contentTypeID, input.FieldName)
	if err != nil {
		return nil, err
	}

	return map[string]bool{"success": true}, nil
}

func (e *DashboardExtension) bridgeReorderFields(ctx bridge.Context, input BridgeReorderFieldsInput) (map[string]any, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(goCtx, input.TypeName)
	if err != nil {
		return nil, err
	}

	contentTypeID, err := xid.FromString(contentType.ID)
	if err != nil {
		return nil, errs.BadRequest("invalid content type ID")
	}

	// Get all fields to map names to IDs
	fields, err := e.plugin.fieldSvc.List(goCtx, contentTypeID)
	if err != nil {
		return nil, err
	}

	// Create a map of field names to IDs
	fieldNameToID := make(map[string]string)
	for _, field := range fields {
		fieldNameToID[field.Name] = field.ID
	}

	// Convert field order strings to FieldOrderItem
	fieldOrders := make([]core.FieldOrderItem, len(input.FieldOrder))
	for i, fieldName := range input.FieldOrder {
		fieldID, ok := fieldNameToID[fieldName]
		if !ok {
			return nil, errs.BadRequest("field not found: " + fieldName)
		}

		fieldOrders[i] = core.FieldOrderItem{
			FieldID: fieldID,
			Order:   i,
		}
	}

	err = e.plugin.fieldSvc.Reorder(goCtx, contentTypeID, &core.ReorderFieldsRequest{
		FieldOrders: fieldOrders,
	})
	if err != nil {
		return nil, err
	}

	return map[string]any{"message": "fields reordered", "success": true}, nil
}

func (e *DashboardExtension) bridgeGetFieldTypes(ctx bridge.Context, input map[string]any) (map[string]any, error) {
	return map[string]any{
		"fieldTypes": core.GetAllFieldTypes(),
	}, nil
}

// =============================================================================
// Entry Bridge Functions
// =============================================================================

type BridgeEntryInput struct {
	AppID    string `json:"appId"             validate:"required"`
	TypeName string `json:"typeName"          validate:"required"`
	EntryID  string `json:"entryId,omitempty"`
}

type BridgeEntriesQueryInput struct {
	AppID     string         `json:"appId"     validate:"required"`
	TypeName  string         `json:"typeName"  validate:"required"`
	Page      int            `json:"page"`
	PageSize  int            `json:"pageSize"`
	Search    string         `json:"search"`
	Status    string         `json:"status"`
	SortBy    string         `json:"sortBy"`
	SortOrder string         `json:"sortOrder"`
	Filters   map[string]any `json:"filters"`
	Select    []string       `json:"select"`
	Populate  []string       `json:"populate"`
}

type BridgeCreateEntryInput struct {
	AppID    string         `json:"appId"            validate:"required"`
	TypeName string         `json:"typeName"         validate:"required"`
	Data     map[string]any `json:"data"             validate:"required"`
	Status   string         `json:"status,omitempty"`
}

type BridgeUpdateEntryInput struct {
	AppID    string         `json:"appId"            validate:"required"`
	TypeName string         `json:"typeName"         validate:"required"`
	EntryID  string         `json:"entryId"          validate:"required"`
	Data     map[string]any `json:"data"             validate:"required"`
	Status   string         `json:"status,omitempty"`
}

type BridgeBulkOperationInput struct {
	AppID    string   `json:"appId"    validate:"required"`
	TypeName string   `json:"typeName" validate:"required"`
	IDs      []string `json:"ids"      validate:"required"`
}

func (e *DashboardExtension) bridgeGetEntries(ctx bridge.Context, input BridgeEntriesQueryInput) (*core.ListEntriesResponse, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Set defaults
	if input.Page == 0 {
		input.Page = 1
	}

	if input.PageSize == 0 {
		input.PageSize = 20
	}

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(goCtx, input.TypeName)
	if err != nil {
		return nil, err
	}

	contentTypeID, err := xid.FromString(contentType.ID)
	if err != nil {
		return nil, errs.BadRequest("invalid content type ID")
	}

	return e.plugin.entrySvc.List(goCtx, contentTypeID, &core.ListEntriesQuery{
		Page:      input.Page,
		PageSize:  input.PageSize,
		Search:    input.Search,
		Status:    input.Status,
		SortBy:    input.SortBy,
		SortOrder: input.SortOrder,
		Filters:   input.Filters,
		Select:    input.Select,
		Populate:  input.Populate,
	})
}

func (e *DashboardExtension) bridgeGetEntry(ctx bridge.Context, input BridgeEntryInput) (*core.ContentEntryDTO, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	entryID, err := xid.FromString(input.EntryID)
	if err != nil {
		return nil, errs.BadRequest("invalid entryId")
	}

	return e.plugin.entrySvc.GetByID(goCtx, entryID)
}

func (e *DashboardExtension) bridgeCreateEntry(ctx bridge.Context, input BridgeCreateEntryInput) (*core.ContentEntryDTO, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	if input.Status == "" {
		input.Status = "draft"
	}

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(goCtx, input.TypeName)
	if err != nil {
		return nil, err
	}

	contentTypeID, err := xid.FromString(contentType.ID)
	if err != nil {
		return nil, errs.BadRequest("invalid content type ID")
	}

	return e.plugin.entrySvc.Create(goCtx, contentTypeID, &core.CreateEntryRequest{
		Data:   input.Data,
		Status: input.Status,
	})
}

func (e *DashboardExtension) bridgeUpdateEntry(ctx bridge.Context, input BridgeUpdateEntryInput) (*core.ContentEntryDTO, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	entryID, err := xid.FromString(input.EntryID)
	if err != nil {
		return nil, errs.BadRequest("invalid entryId")
	}

	return e.plugin.entrySvc.Update(goCtx, entryID, &core.UpdateEntryRequest{
		Data:   input.Data,
		Status: input.Status,
	})
}

func (e *DashboardExtension) bridgeDeleteEntry(ctx bridge.Context, input BridgeEntryInput) (map[string]bool, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	entryID, err := xid.FromString(input.EntryID)
	if err != nil {
		return nil, errs.BadRequest("invalid entryId")
	}

	err = e.plugin.entrySvc.Delete(goCtx, entryID)
	if err != nil {
		return nil, err
	}

	return map[string]bool{"success": true}, nil
}

func (e *DashboardExtension) bridgePublishEntry(ctx bridge.Context, input BridgeEntryInput) (*core.ContentEntryDTO, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	entryID, err := xid.FromString(input.EntryID)
	if err != nil {
		return nil, errs.BadRequest("invalid entryId")
	}

	return e.plugin.entrySvc.Publish(goCtx, entryID, &core.PublishEntryRequest{})
}

func (e *DashboardExtension) bridgeUnpublishEntry(ctx bridge.Context, input BridgeEntryInput) (*core.ContentEntryDTO, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	entryID, err := xid.FromString(input.EntryID)
	if err != nil {
		return nil, errs.BadRequest("invalid entryId")
	}

	return e.plugin.entrySvc.Unpublish(goCtx, entryID)
}

func (e *DashboardExtension) bridgeArchiveEntry(ctx bridge.Context, input BridgeEntryInput) (*core.ContentEntryDTO, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	entryID, err := xid.FromString(input.EntryID)
	if err != nil {
		return nil, errs.BadRequest("invalid entryId")
	}

	return e.plugin.entrySvc.Archive(goCtx, entryID)
}

func (e *DashboardExtension) bridgeGetEntryStats(ctx bridge.Context, input BridgeEntryInput) (*core.ContentTypeStatsDTO, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(goCtx, input.TypeName)
	if err != nil {
		return nil, err
	}

	contentTypeID, err := xid.FromString(contentType.ID)
	if err != nil {
		return nil, errs.BadRequest("invalid content type ID")
	}

	return e.plugin.entrySvc.GetStats(goCtx, contentTypeID)
}

func (e *DashboardExtension) bridgeBulkPublish(ctx bridge.Context, input BridgeBulkOperationInput) (map[string]any, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	ids := make([]xid.ID, len(input.IDs))
	for i, idStr := range input.IDs {
		id, err := xid.FromString(idStr)
		if err != nil {
			return nil, errs.BadRequest("invalid entry ID: " + idStr)
		}

		ids[i] = id
	}

	err = e.plugin.entrySvc.BulkPublish(goCtx, ids)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"message": "entries published",
		"count":   len(ids),
		"success": true,
	}, nil
}

func (e *DashboardExtension) bridgeBulkUnpublish(ctx bridge.Context, input BridgeBulkOperationInput) (map[string]any, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	ids := make([]xid.ID, len(input.IDs))
	for i, idStr := range input.IDs {
		id, err := xid.FromString(idStr)
		if err != nil {
			return nil, errs.BadRequest("invalid entry ID: " + idStr)
		}

		ids[i] = id
	}

	err = e.plugin.entrySvc.BulkUnpublish(goCtx, ids)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"message": "entries unpublished",
		"count":   len(ids),
		"success": true,
	}, nil
}

func (e *DashboardExtension) bridgeBulkDelete(ctx bridge.Context, input BridgeBulkOperationInput) (map[string]any, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	ids := make([]xid.ID, len(input.IDs))
	for i, idStr := range input.IDs {
		id, err := xid.FromString(idStr)
		if err != nil {
			return nil, errs.BadRequest("invalid entry ID: " + idStr)
		}

		ids[i] = id
	}

	err = e.plugin.entrySvc.BulkDelete(goCtx, ids)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"message": "entries deleted",
		"count":   len(ids),
		"success": true,
	}, nil
}

// =============================================================================
// Component Schema Bridge Functions
// =============================================================================

type BridgeComponentSchemaInput struct {
	AppID string `json:"appId"          validate:"required"`
	Name  string `json:"name,omitempty"`
}

type BridgeCreateComponentSchemaInput struct {
	AppID       string                   `json:"appId"                 validate:"required"`
	Title       string                   `json:"title"                 validate:"required"`
	Name        string                   `json:"name"                  validate:"required"`
	Description string                   `json:"description,omitempty"`
	Icon        string                   `json:"icon,omitempty"`
	Fields      []core.NestedFieldDefDTO `json:"fields,omitempty"`
}

type BridgeUpdateComponentSchemaInput struct {
	AppID       string                   `json:"appId"                 validate:"required"`
	Name        string                   `json:"name"                  validate:"required"`
	Title       string                   `json:"title,omitempty"`
	Description string                   `json:"description,omitempty"`
	Icon        string                   `json:"icon,omitempty"`
	Fields      []core.NestedFieldDefDTO `json:"fields,omitempty"`
}

func (e *DashboardExtension) bridgeGetComponentSchemas(ctx bridge.Context, input BridgeComponentSchemaInput) (*core.ListComponentSchemasResponse, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	return e.plugin.componentSchemaSvc.List(goCtx, &core.ListComponentSchemasQuery{})
}

func (e *DashboardExtension) bridgeGetComponentSchema(ctx bridge.Context, input BridgeComponentSchemaInput) (*core.ComponentSchemaDTO, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	return e.plugin.componentSchemaSvc.GetByName(goCtx, input.Name)
}

func (e *DashboardExtension) bridgeCreateComponentSchema(ctx bridge.Context, input BridgeCreateComponentSchemaInput) (*core.ComponentSchemaDTO, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	return e.plugin.componentSchemaSvc.Create(goCtx, &core.CreateComponentSchemaRequest{
		Title:       input.Title,
		Name:        input.Name,
		Description: input.Description,
		Icon:        input.Icon,
		Fields:      input.Fields,
	})
}

func (e *DashboardExtension) bridgeUpdateComponentSchema(ctx bridge.Context, input BridgeUpdateComponentSchemaInput) (*core.ComponentSchemaDTO, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Get component schema
	component, err := e.plugin.componentSchemaSvc.GetByName(goCtx, input.Name)
	if err != nil {
		return nil, err
	}

	componentID, err := xid.FromString(component.ID)
	if err != nil {
		return nil, errs.BadRequest("invalid component schema ID")
	}

	return e.plugin.componentSchemaSvc.Update(goCtx, componentID, &core.UpdateComponentSchemaRequest{
		Title:       input.Title,
		Description: input.Description,
		Icon:        input.Icon,
		Fields:      input.Fields,
	})
}

func (e *DashboardExtension) bridgeDeleteComponentSchema(ctx bridge.Context, input BridgeComponentSchemaInput) (map[string]bool, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	// Get component schema
	component, err := e.plugin.componentSchemaSvc.GetByName(goCtx, input.Name)
	if err != nil {
		return nil, err
	}

	componentID, err := xid.FromString(component.ID)
	if err != nil {
		return nil, errs.BadRequest("invalid component schema ID")
	}

	err = e.plugin.componentSchemaSvc.Delete(goCtx, componentID)
	if err != nil {
		return nil, err
	}

	return map[string]bool{"success": true}, nil
}

// =============================================================================
// Revision Bridge Functions
// =============================================================================

type BridgeRevisionInput struct {
	AppID    string `json:"appId"              validate:"required"`
	EntryID  string `json:"entryId"            validate:"required"`
	Version  int    `json:"version,omitempty"`
	Page     int    `json:"page,omitempty"`
	PageSize int    `json:"pageSize,omitempty"`
}

func (e *DashboardExtension) bridgeListRevisions(ctx bridge.Context, input BridgeRevisionInput) (*core.PaginatedResponse[*core.RevisionDTO], error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	entryID, err := xid.FromString(input.EntryID)
	if err != nil {
		return nil, errs.BadRequest("invalid entryId")
	}

	if input.Page == 0 {
		input.Page = 1
	}

	if input.PageSize == 0 {
		input.PageSize = 20
	}

	if e.plugin.revisionSvc == nil {
		return nil, errs.BadRequest("revision service not available")
	}

	return e.plugin.revisionSvc.List(goCtx, entryID, &core.ListRevisionsQuery{
		Page:     input.Page,
		PageSize: input.PageSize,
	})
}

func (e *DashboardExtension) bridgeGetRevision(ctx bridge.Context, input BridgeRevisionInput) (*core.RevisionDTO, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	entryID, err := xid.FromString(input.EntryID)
	if err != nil {
		return nil, errs.BadRequest("invalid entryId")
	}

	if e.plugin.revisionSvc == nil {
		return nil, errs.BadRequest("revision service not available")
	}

	return e.plugin.revisionSvc.GetByVersion(goCtx, entryID, input.Version)
}

func (e *DashboardExtension) bridgeRestoreRevision(ctx bridge.Context, input BridgeRevisionInput) (*core.ContentEntryDTO, error) {
	goCtx, err := e.buildContextFromBridge(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	entryID, err := xid.FromString(input.EntryID)
	if err != nil {
		return nil, errs.BadRequest("invalid entryId")
	}

	if e.plugin.revisionSvc == nil {
		return nil, errs.BadRequest("revision service not available")
	}

	// Get the revision data
	revision, err := e.plugin.revisionSvc.GetByVersion(goCtx, entryID, input.Version)
	if err != nil {
		return nil, err
	}

	// Update the entry with the revision data
	return e.plugin.entrySvc.Update(goCtx, entryID, &core.UpdateEntryRequest{
		Data: revision.Data,
	})
}

// =============================================================================
// Helper Methods
// =============================================================================

// extractAppFromURL extracts the app from the URL parameter.
func (e *DashboardExtension) extractAppFromURL(ctx *router.PageContext) (*app.App, error) {
	appIDStr := ctx.Param("appId")
	if appIDStr == "" {
		return nil, errs.RequiredField("appId")
	}

	appID, err := xid.FromString(appIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid app ID format: %w", err)
	}

	return &app.App{ID: appID}, nil
}

// getBasePath returns the dashboard base path.
func (e *DashboardExtension) getBasePath() string {
	return e.baseUIPath
}

// injectContext injects app and environment IDs into context.
func (e *DashboardExtension) injectContext(ctx *router.PageContext) context.Context {
	reqCtx := ctx.Request.Context()

	// Get app ID from URL
	var appID xid.ID

	if appIDStr := ctx.Param("appId"); appIDStr != "" {
		if id, err := xid.FromString(appIDStr); err == nil {
			appID = id
			reqCtx = contexts.SetAppID(reqCtx, appID)
		}
	}

	// Get environment ID from header or context
	if envIDStr := ctx.Request.Header.Get("X-Environment-Id"); envIDStr != "" {
		if envID, err := xid.FromString(envIDStr); err == nil {
			reqCtx = contexts.SetEnvironmentID(reqCtx, envID)
		}
	}

	// Try to get from existing context
	if envID, ok := contexts.GetEnvironmentID(ctx.Request.Context()); ok {
		reqCtx = contexts.SetEnvironmentID(reqCtx, envID)
	}

	// If no environment ID yet, try to get default environment for the app
	if _, ok := contexts.GetEnvironmentID(reqCtx); !ok && !appID.IsNil() {
		if envSvc := e.plugin.authInst.GetServiceRegistry().EnvironmentService(); envSvc != nil {
			if defaultEnv, err := envSvc.GetDefaultEnvironment(reqCtx, appID); err == nil && defaultEnv != nil {
				reqCtx = contexts.SetEnvironmentID(reqCtx, defaultEnv.ID)
			}
		}
	}

	return reqCtx
}

// =============================================================================
// Widget Renderer
// =============================================================================

func (e *DashboardExtension) renderCMSWidget(currentApp *app.App) g.Node {
	ctx := context.Background()
	if currentApp != nil {
		ctx = contexts.SetAppID(ctx, currentApp.ID)
	}

	// Get stats
	stats, err := e.plugin.contentTypeSvc.GetStats(ctx)
	if err != nil {
		stats = &core.CMSStatsDTO{
			TotalContentTypes: 0,
			TotalEntries:      0,
		}
	}

	return Div(
		Class("text-center"),
		Div(
			Class("grid grid-cols-2 gap-4"),
			Div(
				Div(
					Class("text-2xl font-bold text-slate-900 dark:text-white"),
					g.Text(strconv.Itoa(stats.TotalContentTypes)),
				),
				Div(
					Class("text-xs text-slate-500 dark:text-gray-400"),
					g.Text("Content Types"),
				),
			),
			Div(
				Div(
					Class("text-2xl font-bold text-slate-900 dark:text-white"),
					g.Text(strconv.Itoa(stats.TotalEntries)),
				),
				Div(
					Class("text-xs text-slate-500 dark:text-gray-400"),
					g.Text("Total Entries"),
				),
			),
		),
	)
}

// =============================================================================
// CMS Settings Handler
// =============================================================================

func (e *DashboardExtension) ServeCMSSettings(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	// Get content types for stats
	result, err := e.plugin.contentTypeSvc.List(context.Background(), &core.ListContentTypesQuery{
		PageSize: 100,
	})
	if err != nil {
		result = &core.ListContentTypesResponse{ContentTypes: []*core.ContentTypeSummaryDTO{}}
	}

	// Get stats
	stats, _ := e.plugin.contentTypeSvc.GetStats(context.Background())

	basePath := e.getBasePath()
	content := e.renderCMSSettingsContent(currentApp, basePath, result.ContentTypes, stats)

	return content, nil
}

// renderCMSSettingsContent renders the CMS settings page content.
func (e *DashboardExtension) renderCMSSettingsContent(currentApp *app.App, basePath string, contentTypes []*core.ContentTypeSummaryDTO, stats *core.CMSStatsDTO) g.Node {
	appBase := basePath + "/app/" + currentApp.ID.String()

	// Build stats display
	var totalTypes, totalEntries int
	if stats != nil {
		totalTypes = stats.TotalContentTypes
		totalEntries = stats.TotalEntries
	}

	return Div(
		Class("space-y-2"),

		// Header
		Div(
			H2(Class("text-lg font-semibold text-slate-900 dark:text-white"),
				g.Text("Content Management")),
			P(Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
				g.Text("Configure CMS settings and manage your content types")),
		),

		// Stats overview
		Div(
			Class("grid gap-4 md:grid-cols-2"),

			// Content Types card
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("flex items-center gap-3 mb-2"),
					Div(
						Class("flex h-10 w-10 items-center justify-center rounded-lg bg-violet-100 dark:bg-violet-900/20"),
						lucide.Database(Class("h-5 w-5 text-violet-600 dark:text-violet-400")),
					),
					Div(
						H3(Class("text-2xl font-bold text-slate-900 dark:text-white"),
							g.Text(strconv.Itoa(totalTypes))),
						P(Class("text-sm text-slate-600 dark:text-gray-400"),
							g.Text("Content Types")),
					),
				),
			),

			// Entries card
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("flex items-center gap-3 mb-2"),
					Div(
						Class("flex h-10 w-10 items-center justify-center rounded-lg bg-blue-100 dark:bg-blue-900/20"),
						lucide.FileText(Class("h-5 w-5 text-blue-600 dark:text-blue-400")),
					),
					Div(
						H3(Class("text-2xl font-bold text-slate-900 dark:text-white"),
							g.Text(strconv.Itoa(totalEntries))),
						P(Class("text-sm text-slate-600 dark:text-gray-400"),
							g.Text("Total Entries")),
					),
				),
			),
		),

		// Quick actions
		Div(
			Class("grid gap-4 md:grid-cols-3"),

			// Manage Content Types
			A(
				Href(appBase+"/cms/types"),
				Class("block p-6 rounded-lg border border-slate-200 bg-white shadow-sm hover:shadow-md transition-shadow dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("flex items-center gap-3 mb-2"),
					Div(
						Class("flex h-10 w-10 items-center justify-center rounded-lg bg-violet-100 dark:bg-violet-900/20"),
						lucide.Layers(Class("h-5 w-5 text-violet-600 dark:text-violet-400")),
					),
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
						g.Text("Content Types")),
				),
				P(Class("text-sm text-slate-600 dark:text-gray-400"),
					g.Text("Define and manage your content schemas")),
			),

			// Create Content Type
			A(
				Href(appBase+"/cms/types/create"),
				Class("block p-6 rounded-lg border border-slate-200 bg-white shadow-sm hover:shadow-md transition-shadow dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("flex items-center gap-3 mb-2"),
					Div(
						Class("flex h-10 w-10 items-center justify-center rounded-lg bg-green-100 dark:bg-green-900/20"),
						lucide.Plus(Class("h-5 w-5 text-green-600 dark:text-green-400")),
					),
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
						g.Text("New Content Type")),
				),
				P(Class("text-sm text-slate-600 dark:text-gray-400"),
					g.Text("Create a new content type schema")),
			),

			// CMS Overview
			A(
				Href(appBase+"/cms"),
				Class("block p-6 rounded-lg border border-slate-200 bg-white shadow-sm hover:shadow-md transition-shadow dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("flex items-center gap-3 mb-2"),
					Div(
						Class("flex h-10 w-10 items-center justify-center rounded-lg bg-slate-100 dark:bg-slate-900/20"),
						lucide.LayoutDashboard(Class("h-5 w-5 text-slate-600 dark:text-slate-400")),
					),
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
						g.Text("CMS Overview")),
				),
				P(Class("text-sm text-slate-600 dark:text-gray-400"),
					g.Text("View the full CMS dashboard")),
			),
		),

		// Recent content types
		g.If(len(contentTypes) > 0,
			Div(
				Class("rounded-lg border border-slate-200 bg-white dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("px-6 py-4 border-b border-slate-200 dark:border-gray-800"),
					H3(Class("text-base font-semibold text-slate-900 dark:text-white"),
						g.Text("Recent Content Types")),
				),
				Div(
					Class("divide-y divide-slate-200 dark:divide-gray-800"),
					g.Group(g.Map(contentTypes, func(ct *core.ContentTypeSummaryDTO) g.Node {
						return A(
							Href(appBase+"/cms/types/"+ct.Name),
							Class("flex items-center justify-between px-6 py-4 hover:bg-slate-50 dark:hover:bg-gray-800/50 transition-colors"),
							Div(
								Class("flex items-center gap-3"),
								Div(
									Class("flex h-8 w-8 items-center justify-center rounded bg-slate-100 dark:bg-gray-800"),
									lucide.FileCode(Class("h-4 w-4 text-slate-600 dark:text-gray-400")),
								),
								Div(
									H4(Class("text-sm font-medium text-slate-900 dark:text-white"),
										g.Text(ct.Name)),
									P(Class("text-xs text-slate-500 dark:text-gray-400"),
										g.Textf("%d entries", ct.EntryCount)),
								),
							),
							lucide.ChevronRight(Class("h-4 w-4 text-slate-400")),
						)
					})),
				),
			),
		),
	)
}

// =============================================================================
// CMS Overview Handler
// =============================================================================

func (e *DashboardExtension) ServeCMSOverview(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.getBasePath()

	// Use the dynamic version that fetches data via bridge functions
	content := pages.CMSOverviewDynamic(currentApp, basePath)

	return content, nil
}

// =============================================================================
// Content Types Handlers
// =============================================================================

func (e *DashboardExtension) ServeContentTypesList(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	reqCtx := e.injectContext(ctx)

	searchQuery := ctx.Query("search")

	page, _ := strconv.Atoi(ctx.Query("page"))
	if page < 1 {
		page = 1
	}

	pageSize := 20

	// Get content types
	result, err := e.plugin.contentTypeSvc.List(reqCtx, &core.ListContentTypesQuery{
		Search:   searchQuery,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		result = &core.ListContentTypesResponse{ContentTypes: []*core.ContentTypeSummaryDTO{}}
	}

	basePath := e.getBasePath()
	content := pages.ContentTypesListPage(currentApp, basePath, result.ContentTypes, page, pageSize, result.TotalItems, searchQuery)

	return content, nil
}

func (e *DashboardExtension) ServeCreateContentType(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.getBasePath()
	errMsg := ctx.Query("error")

	content := pages.CreateContentTypePage(currentApp, basePath, errMsg)

	return content, nil
}

func (e *DashboardExtension) HandleCreateContentType(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	reqCtx := e.injectContext(ctx)
	basePath := e.getBasePath()
	appBase := basePath + "/app/" + currentApp.ID.String()

	// Parse form
	req := &core.CreateContentTypeRequest{
		Title:       ctx.Request.FormValue("name"),
		Name:        ctx.Request.FormValue("slug"),
		Description: ctx.Request.FormValue("description"),
		Icon:        ctx.Request.FormValue("icon"),
	}

	// Create content type
	result, err := e.plugin.contentTypeSvc.Create(reqCtx, req)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types/create?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)

		return nil, nil
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types/"+result.Name, http.StatusSeeOther)

	return nil, nil
}

func (e *DashboardExtension) ServeContentTypeDetail(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	reqCtx := e.injectContext(ctx)
	basePath := e.getBasePath()
	typeName := ctx.Param("typeName")

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(reqCtx, typeName)
	if err != nil {
		return nil, errs.NotFound("Content type not found")
	}

	// Get stats
	contentTypeID, _ := xid.FromString(contentType.ID)
	stats, _ := e.plugin.entrySvc.GetStats(reqCtx, contentTypeID)

	// Get environment ID from context (set by injectContext)
	var envIDStr string
	if envID, ok := contexts.GetEnvironmentID(reqCtx); ok {
		envIDStr = envID.String()
	}

	// Get all content types for relation field dropdown
	allContentTypes := []*core.ContentTypeSummaryDTO{}

	ctResult, _ := e.plugin.contentTypeSvc.List(reqCtx, &core.ListContentTypesQuery{PageSize: 100})
	if ctResult != nil {
		allContentTypes = ctResult.ContentTypes
	}

	// Get all component schemas for nested field dropdowns
	allComponentSchemas := []*core.ComponentSchemaSummaryDTO{}

	csResult, _ := e.plugin.componentSchemaSvc.List(reqCtx, &core.ListComponentSchemasQuery{PageSize: 100})
	if csResult != nil {
		allComponentSchemas = csResult.Components
	}

	content := pages.ContentTypeDetailPage(currentApp, basePath, contentType, stats, envIDStr, allContentTypes, allComponentSchemas)

	return content, nil
}

// HandleAddField handles adding a new field to a content type.
func (e *DashboardExtension) HandleAddField(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	reqCtx := e.injectContext(ctx)
	basePath := e.getBasePath()
	typeName := ctx.Param("typeName")
	appBase := basePath + "/app/" + currentApp.ID.String()

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(reqCtx, typeName)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types?error=Content+type+not+found", http.StatusSeeOther)

		return nil, nil
	}

	contentTypeID, _ := xid.FromString(contentType.ID)

	// Parse form values
	req := &core.CreateFieldRequest{
		Title:       ctx.Request.FormValue("name"),
		Name:        ctx.Request.FormValue("slug"),
		Type:        ctx.Request.FormValue("type"),
		Description: ctx.Request.FormValue("description"),
		Required:    ctx.Request.FormValue("required") == "true",
		Unique:      ctx.Request.FormValue("unique") == "true",
		Indexed:     ctx.Request.FormValue("indexed") == "true",
		Localized:   ctx.Request.FormValue("localized") == "true",
		Options:     e.parseFieldOptionsFromRequest(ctx),
	}

	// Create the field
	_, err = e.plugin.fieldSvc.Create(reqCtx, contentTypeID, req)
	if err != nil {
		// Redirect back with error
		http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types/"+typeName+"?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)

		return nil, nil
	}

	// Redirect back to content type detail
	http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types/"+typeName, http.StatusSeeOther)

	return nil, nil
}

// HandleUpdateField handles updating a field in a content type.
func (e *DashboardExtension) HandleUpdateField(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	reqCtx := e.injectContext(ctx)
	basePath := e.getBasePath()
	typeName := ctx.Param("typeName")
	fieldName := ctx.Param("fieldName")
	appBase := basePath + "/app/" + currentApp.ID.String()

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(reqCtx, typeName)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types?error=Content+type+not+found", http.StatusSeeOther)

		return nil, nil
	}

	contentTypeID, _ := xid.FromString(contentType.ID)

	// Parse form values for update
	req := &core.UpdateFieldRequest{
		Title:       ctx.Request.FormValue("name"),
		Description: ctx.Request.FormValue("description"),
		Options:     e.parseFieldOptionsFromRequest(ctx),
	}

	// Parse boolean fields
	if ctx.Request.FormValue("required") != "" {
		v := ctx.Request.FormValue("required") == "true"
		req.Required = &v
	}

	if ctx.Request.FormValue("unique") != "" {
		v := ctx.Request.FormValue("unique") == "true"
		req.Unique = &v
	}

	if ctx.Request.FormValue("indexed") != "" {
		v := ctx.Request.FormValue("indexed") == "true"
		req.Indexed = &v
	}

	if ctx.Request.FormValue("localized") != "" {
		v := ctx.Request.FormValue("localized") == "true"
		req.Localized = &v
	}

	// Update the field
	_, err = e.plugin.fieldSvc.UpdateByName(reqCtx, contentTypeID, fieldName, req)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types/"+typeName+"?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)

		return nil, nil
	}

	// Redirect back to content type detail
	http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types/"+typeName, http.StatusSeeOther)

	return nil, nil
}

// HandleDeleteField handles deleting a field from a content type.
func (e *DashboardExtension) HandleDeleteField(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	reqCtx := e.injectContext(ctx)
	basePath := e.getBasePath()
	typeName := ctx.Param("typeName")
	fieldName := ctx.Param("fieldName")
	appBase := basePath + "/app/" + currentApp.ID.String()

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(reqCtx, typeName)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types?error=Content+type+not+found", http.StatusSeeOther)

		return nil, nil
	}

	contentTypeID, _ := xid.FromString(contentType.ID)

	// Delete the field
	err = e.plugin.fieldSvc.DeleteByName(reqCtx, contentTypeID, fieldName)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types/"+typeName+"?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)

		return nil, nil
	}

	// Redirect back to content type detail
	http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types/"+typeName, http.StatusSeeOther)

	return nil, nil
}

// HandleUpdateDisplaySettings handles updating content type display settings.
func (e *DashboardExtension) HandleUpdateDisplaySettings(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	reqCtx := e.injectContext(ctx)
	basePath := e.getBasePath()
	typeName := ctx.Param("typeName")
	appBase := basePath + "/app/" + currentApp.ID.String()

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(reqCtx, typeName)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types?error=Content+type+not+found", http.StatusSeeOther)

		return nil, nil
	}

	contentTypeID, _ := xid.FromString(contentType.ID)

	// Parse form values
	titleField := ctx.Request.FormValue("titleField")
	descriptionField := ctx.Request.FormValue("descriptionField")
	previewField := ctx.Request.FormValue("previewField")

	// Get current settings
	currentSettings := contentType.Settings

	// Update display fields
	req := &core.UpdateContentTypeRequest{
		Settings: &core.ContentTypeSettingsDTO{
			TitleField:         titleField,
			DescriptionField:   descriptionField,
			PreviewField:       previewField,
			EnableRevisions:    currentSettings.EnableRevisions,
			EnableDrafts:       currentSettings.EnableDrafts,
			EnableSoftDelete:   currentSettings.EnableSoftDelete,
			EnableSearch:       currentSettings.EnableSearch,
			EnableScheduling:   currentSettings.EnableScheduling,
			DefaultPermissions: currentSettings.DefaultPermissions,
			MaxEntries:         currentSettings.MaxEntries,
		},
	}

	// Update content type
	_, err = e.plugin.contentTypeSvc.Update(reqCtx, contentTypeID, req)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types/"+typeName+"?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)

		return nil, nil
	}

	// Redirect back to content type detail
	http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types/"+typeName+"?success=Display+settings+saved", http.StatusSeeOther)

	return nil, nil
}

// HandleUpdateFeatureSettings handles updating content type feature settings.
func (e *DashboardExtension) HandleUpdateFeatureSettings(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	reqCtx := e.injectContext(ctx)
	basePath := e.getBasePath()
	typeName := ctx.Param("typeName")
	appBase := basePath + "/app/" + currentApp.ID.String()

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(reqCtx, typeName)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types?error=Content+type+not+found", http.StatusSeeOther)

		return nil, nil
	}

	contentTypeID, _ := xid.FromString(contentType.ID)

	// Parse form values (checkboxes are only sent if checked)
	enableRevisions := ctx.Request.FormValue("enableRevisions") == "on"
	enableDrafts := ctx.Request.FormValue("enableDrafts") == "on"
	enableSoftDelete := ctx.Request.FormValue("enableSoftDelete") == "on"
	enableSearch := ctx.Request.FormValue("enableSearch") == "on"
	enableScheduling := ctx.Request.FormValue("enableScheduling") == "on"

	// Get current settings
	currentSettings := contentType.Settings

	// Update feature settings
	req := &core.UpdateContentTypeRequest{
		Settings: &core.ContentTypeSettingsDTO{
			TitleField:         currentSettings.TitleField,
			DescriptionField:   currentSettings.DescriptionField,
			PreviewField:       currentSettings.PreviewField,
			EnableRevisions:    enableRevisions,
			EnableDrafts:       enableDrafts,
			EnableSoftDelete:   enableSoftDelete,
			EnableSearch:       enableSearch,
			EnableScheduling:   enableScheduling,
			DefaultPermissions: currentSettings.DefaultPermissions,
			MaxEntries:         currentSettings.MaxEntries,
		},
	}

	// Update content type
	_, err = e.plugin.contentTypeSvc.Update(reqCtx, contentTypeID, req)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types/"+typeName+"?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)

		return nil, nil
	}

	// Redirect back to content type detail
	http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types/"+typeName+"?success=Feature+settings+saved", http.StatusSeeOther)

	return nil, nil
}

// HandleDeleteContentType handles deleting a content type.
func (e *DashboardExtension) HandleDeleteContentType(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	reqCtx := e.injectContext(ctx)
	basePath := e.getBasePath()
	typeName := ctx.Param("typeName")
	appBase := basePath + "/app/" + currentApp.ID.String()

	// Get content type to get its ID
	contentType, err := e.plugin.contentTypeSvc.GetByName(reqCtx, typeName)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types?error=Content+type+not+found", http.StatusSeeOther)

		return nil, nil
	}

	contentTypeID, _ := xid.FromString(contentType.ID)

	// Check if there are entries - if so, don't allow delete
	entries, _ := e.plugin.entrySvc.List(reqCtx, contentTypeID, &core.ListEntriesQuery{PageSize: 1})
	if entries != nil && entries.TotalItems > 0 {
		http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types/"+typeName+"?error=Cannot+delete+content+type+with+existing+entries.+Delete+all+entries+first.", http.StatusSeeOther)

		return nil, nil
	}

	// Delete the content type (this also deletes all fields)
	err = e.plugin.contentTypeSvc.Delete(reqCtx, contentTypeID)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types/"+typeName+"?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)

		return nil, nil
	}

	// Redirect back to content types list
	http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types?success=Content+type+deleted+successfully", http.StatusSeeOther)

	return nil, nil
}

// =============================================================================
// Content Entries Handlers
// =============================================================================

func (e *DashboardExtension) ServeEntriesList(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	reqCtx := e.injectContext(ctx)
	basePath := e.getBasePath()
	typeName := ctx.Param("typeName")
	searchQuery := ctx.Query("search")
	statusFilter := ctx.Query("status")

	page, _ := strconv.Atoi(ctx.Query("page"))
	if page < 1 {
		page = 1
	}

	pageSize := 20

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(reqCtx, typeName)
	if err != nil {
		return nil, errs.NotFound("Content type not found")
	}

	// Get entries
	contentTypeID, _ := xid.FromString(contentType.ID)

	result, err := e.plugin.entrySvc.List(reqCtx, contentTypeID, &core.ListEntriesQuery{
		Search:   searchQuery,
		Status:   statusFilter,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		result = &core.ListEntriesResponse{Entries: []*core.ContentEntryDTO{}}
	}

	// Get stats
	stats, _ := e.plugin.entrySvc.GetStats(reqCtx, contentTypeID)

	content := pages.EntriesListPage(currentApp, basePath, contentType, result.Entries, stats, page, pageSize, result.TotalItems, searchQuery, statusFilter)

	return content, nil
}

func (e *DashboardExtension) ServeCreateEntry(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	reqCtx := e.injectContext(ctx)
	basePath := e.getBasePath()
	typeName := ctx.Param("typeName")
	errMsg := ctx.Query("error")

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(reqCtx, typeName)
	if err != nil {
		return nil, errs.NotFound("Content type not found")
	}

	// Resolve component references for object/array fields
	e.resolveComponentReferences(ctx.Request.Context(), contentType)

	content := pages.CreateEntryPage(currentApp, basePath, contentType, errMsg)

	return content, nil
}

func (e *DashboardExtension) HandleCreateEntry(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	reqCtx := e.injectContext(ctx)
	basePath := e.getBasePath()
	typeName := ctx.Param("typeName")
	appBase := basePath + "/app/" + currentApp.ID.String()

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(reqCtx, typeName)
	if err != nil {
		return nil, errs.NotFound("Content type not found")
	}

	contentTypeID, _ := xid.FromString(contentType.ID)

	// Parse form data into map
	data := make(map[string]any)

	for _, field := range contentType.Fields {
		value := ctx.Request.FormValue("data[" + field.Name + "]")

		// Handle boolean fields specially (checkboxes send "true" string or nothing)
		if field.Type == "boolean" {
			data[field.Name] = value == "true" || value == "on" || value == "1"

			continue
		}

		if value != "" {
			// Parse JSON for object, array, oneOf, and json field types
			switch field.Type {
			case "object", "array", "oneOf", "json":
				var parsedValue any
				if err := json.Unmarshal([]byte(value), &parsedValue); err == nil {
					data[field.Name] = parsedValue
				} else {
					// If parsing fails, store as-is (validation will catch it)
					data[field.Name] = value
				}
			case "number", "integer", "float":
				// Parse number fields to preserve numeric types
				if field.Type == "integer" {
					if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
						data[field.Name] = intVal
					} else {
						data[field.Name] = value
					}
				} else {
					if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
						data[field.Name] = floatVal
					} else {
						data[field.Name] = value
					}
				}
			default:
				data[field.Name] = value
			}
		}
	}

	// Create entry
	req := &core.CreateEntryRequest{
		Data:   data,
		Status: "draft",
	}

	result, err := e.plugin.entrySvc.Create(reqCtx, contentTypeID, req)
	if err != nil {
		// Format validation errors with field details if available
		errorMsg := err.Error()

		// Try to extract validation details from the error
		cmsErr := &errs.AuthsomeError{}
		if errors.As(err, &cmsErr) {
			if cmsErr.Code == core.ErrCodeEntryValidationFailed {
				if details, ok := cmsErr.Details.(map[string]string); ok && len(details) > 0 {
					errorMsg = "Validation failed:\n"
					var errorMsgSb2377 strings.Builder
					for field, msg := range details {
						errorMsgSb2377.WriteString(fmt.Sprintf("• %s: %s\n", field, msg))
					}
					errorMsg += errorMsgSb2377.String()
				}
			}
		}

		http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types/"+typeName+"/entries/create?error="+url.QueryEscape(errorMsg), http.StatusSeeOther)

		return nil, nil
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types/"+typeName+"/entries/"+result.ID, http.StatusSeeOther)

	return nil, nil
}

func (e *DashboardExtension) ServeEntryDetail(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	reqCtx := e.injectContext(ctx)
	basePath := e.getBasePath()
	typeName := ctx.Param("typeName")
	entryIDStr := ctx.Param("entryId")

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(reqCtx, typeName)
	if err != nil {
		return nil, errs.NotFound("Content type not found")
	}

	// Get entry
	entryID, err := xid.FromString(entryIDStr)
	if err != nil {
		return nil, errs.BadRequest("Invalid entry ID")
	}

	entry, err := e.plugin.entrySvc.GetByID(reqCtx, entryID)
	if err != nil {
		return nil, errs.NotFound("Entry not found")
	}

	// Get revisions
	var revisionDTOs []*core.ContentRevisionDTO

	if e.plugin.revisionSvc != nil {
		revisions, _ := e.plugin.revisionSvc.List(reqCtx, entryID, &core.ListRevisionsQuery{PageSize: 5})
		if revisions != nil && revisions.Items != nil {
			revisionDTOs = make([]*core.ContentRevisionDTO, len(revisions.Items))
			for i, rev := range revisions.Items {
				revisionDTOs[i] = &core.ContentRevisionDTO{
					ID:           rev.ID,
					EntryID:      rev.EntryID,
					Version:      rev.Version,
					Data:         rev.Data,
					ChangeReason: rev.Reason,
					ChangedBy:    rev.ChangedBy,
					CreatedAt:    rev.CreatedAt,
				}
			}
		}
	}

	content := pages.EntryDetailPage(currentApp, basePath, contentType, entry, revisionDTOs)

	return content, nil
}

func (e *DashboardExtension) ServeEditEntry(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	reqCtx := e.injectContext(ctx)
	basePath := e.getBasePath()
	typeName := ctx.Param("typeName")
	entryIDStr := ctx.Param("entryId")
	errMsg := ctx.Query("error")

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(reqCtx, typeName)
	if err != nil {
		return nil, errs.NotFound("Content type not found")
	}

	// Resolve component references for object/array fields
	e.resolveComponentReferences(ctx.Request.Context(), contentType)

	// Get entry
	entryID, err := xid.FromString(entryIDStr)
	if err != nil {
		return nil, errs.BadRequest("Invalid entry ID")
	}

	entry, err := e.plugin.entrySvc.GetByID(reqCtx, entryID)
	if err != nil {
		return nil, errs.NotFound("Entry not found")
	}

	content := pages.EditEntryPage(currentApp, basePath, contentType, entry, errMsg)

	return content, nil
}

func (e *DashboardExtension) HandleUpdateEntry(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	reqCtx := e.injectContext(ctx)
	basePath := e.getBasePath()
	typeName := ctx.Param("typeName")
	entryIDStr := ctx.Param("entryId")
	appBase := basePath + "/app/" + currentApp.ID.String()

	// Get content type
	contentType, err := e.plugin.contentTypeSvc.GetByName(reqCtx, typeName)
	if err != nil {
		return nil, errs.NotFound("Content type not found")
	}

	// Get entry ID
	entryID, err := xid.FromString(entryIDStr)
	if err != nil {
		return nil, errs.BadRequest("Invalid entry ID")
	}

	// Parse form data into map
	data := make(map[string]any)

	for _, field := range contentType.Fields {
		value := ctx.Request.FormValue("data[" + field.Name + "]")

		// Handle boolean fields specially (checkboxes send "true" string or nothing)
		if field.Type == "boolean" {
			data[field.Name] = value == "true" || value == "on" || value == "1"

			continue
		}

		if value != "" {
			// Parse JSON for object, array, oneOf, and json field types
			switch field.Type {
			case "object", "array", "oneOf", "json":
				var parsedValue any
				if err := json.Unmarshal([]byte(value), &parsedValue); err == nil {
					data[field.Name] = parsedValue
				} else {
					// If parsing fails, store as-is (validation will catch it)
					data[field.Name] = value
				}
			case "number", "integer", "float":
				// Parse number fields to preserve numeric types
				if field.Type == "integer" {
					if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
						data[field.Name] = intVal
					} else {
						data[field.Name] = value
					}
				} else {
					if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
						data[field.Name] = floatVal
					} else {
						data[field.Name] = value
					}
				}
			default:
				data[field.Name] = value
			}
		}
	}

	// Update entry
	status := ctx.Request.FormValue("status")
	req := &core.UpdateEntryRequest{
		Data:   data,
		Status: status,
	}

	_, err = e.plugin.entrySvc.Update(reqCtx, entryID, req)
	if err != nil {
		// Format validation errors with field details if available
		errorMsg := err.Error()

		// Try to extract validation details from the error
		cmsErr := &errs.AuthsomeError{}
		if errors.As(err, &cmsErr) {
			if cmsErr.Code == core.ErrCodeEntryValidationFailed {
				if details, ok := cmsErr.Details.(map[string]string); ok && len(details) > 0 {
					errorMsg = "Validation failed:\n"
					var errorMsgSb2567 strings.Builder
					for field, msg := range details {
						errorMsgSb2567.WriteString(fmt.Sprintf("• %s: %s\n", field, msg))
					}
					errorMsg += errorMsgSb2567.String()
				}
			}
		}

		http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types/"+typeName+"/entries/"+entryIDStr+"/edit?error="+url.QueryEscape(errorMsg), http.StatusSeeOther)

		return nil, nil
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types/"+typeName+"/entries/"+entryIDStr, http.StatusSeeOther)

	return nil, nil
}

// HandleDeleteEntry handles deleting a content entry.
func (e *DashboardExtension) HandleDeleteEntry(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	reqCtx := e.injectContext(ctx)
	basePath := e.getBasePath()
	typeName := ctx.Param("typeName")
	entryIDStr := ctx.Param("entryId")
	appBase := basePath + "/app/" + currentApp.ID.String()

	// Get entry ID
	entryID, err := xid.FromString(entryIDStr)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types/"+typeName+"/entries?error=Invalid+entry+ID", http.StatusSeeOther)

		return nil, nil
	}

	// Delete the entry
	err = e.plugin.entrySvc.Delete(reqCtx, entryID)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types/"+typeName+"/entries?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)

		return nil, nil
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/types/"+typeName+"/entries?success=Entry+deleted+successfully", http.StatusSeeOther)

	return nil, nil
}

// =============================================================================
// Component Schema Handlers
// =============================================================================

func (e *DashboardExtension) ServeComponentSchemasList(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	reqCtx := e.injectContext(ctx)
	basePath := e.getBasePath()

	searchQuery := ctx.Query("search")

	page, _ := strconv.Atoi(ctx.Query("page"))
	if page < 1 {
		page = 1
	}

	pageSize := 20

	// Get component schemas
	result, err := e.plugin.componentSchemaSvc.List(reqCtx, &core.ListComponentSchemasQuery{
		Search:   searchQuery,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		result = &core.ListComponentSchemasResponse{Components: []*core.ComponentSchemaSummaryDTO{}}
	}

	content := pages.ComponentSchemasPage(currentApp, basePath, result.Components, page, pageSize, result.TotalItems, searchQuery)

	return content, nil
}

func (e *DashboardExtension) ServeCreateComponentSchema(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.getBasePath()
	errMsg := ctx.Query("error")

	content := pages.CreateComponentSchemaPage(currentApp, basePath, errMsg)

	return content, nil
}

func (e *DashboardExtension) HandleCreateComponentSchema(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	reqCtx := e.injectContext(ctx)
	basePath := e.getBasePath()
	appBase := basePath + "/app/" + currentApp.ID.String()

	// Parse nested fields from JSON
	var fields []core.NestedFieldDefDTO

	fieldsJSON := ctx.Request.FormValue("fields")
	if fieldsJSON != "" {
		if err := json.Unmarshal([]byte(fieldsJSON), &fields); err != nil {
			http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/components/create?error=Invalid+fields+format", http.StatusSeeOther)

			return nil, nil
		}
	}

	// Create request
	req := &core.CreateComponentSchemaRequest{
		Title:       ctx.Request.FormValue("name"),
		Name:        ctx.Request.FormValue("slug"),
		Description: ctx.Request.FormValue("description"),
		Icon:        ctx.Request.FormValue("icon"),
		Fields:      fields,
	}

	// Create component schema
	result, err := e.plugin.componentSchemaSvc.Create(reqCtx, req)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/components/create?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)

		return nil, nil
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/components/"+result.Name, http.StatusSeeOther)

	return nil, nil
}

func (e *DashboardExtension) ServeComponentSchemaDetail(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	reqCtx := e.injectContext(ctx)
	basePath := e.getBasePath()
	componentSlug := ctx.Param("componentSlug")
	errMsg := ctx.Query("error")

	// Get component schema
	component, err := e.plugin.componentSchemaSvc.GetByName(reqCtx, componentSlug)
	if err != nil {
		return nil, errs.NotFound("Component schema not found")
	}

	content := pages.EditComponentSchemaPage(currentApp, basePath, component, errMsg)

	return content, nil
}

func (e *DashboardExtension) HandleUpdateComponentSchema(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	reqCtx := e.injectContext(ctx)
	basePath := e.getBasePath()
	componentSlug := ctx.Param("componentSlug")
	appBase := basePath + "/app/" + currentApp.ID.String()

	// Get existing component to get its ID
	component, err := e.plugin.componentSchemaSvc.GetByName(reqCtx, componentSlug)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/components?error=Component+not+found", http.StatusSeeOther)

		return nil, nil
	}

	componentID, _ := xid.FromString(component.ID)

	// Parse nested fields from JSON
	var fields []core.NestedFieldDefDTO

	fieldsJSON := ctx.Request.FormValue("fields")
	if fieldsJSON != "" {
		if err := json.Unmarshal([]byte(fieldsJSON), &fields); err != nil {
			http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/components/"+componentSlug+"?error=Invalid+fields+format", http.StatusSeeOther)

			return nil, nil
		}
	}

	// Create update request
	req := &core.UpdateComponentSchemaRequest{
		Title:       ctx.Request.FormValue("name"),
		Description: ctx.Request.FormValue("description"),
		Icon:        ctx.Request.FormValue("icon"),
		Fields:      fields,
	}

	// Update component schema
	_, err = e.plugin.componentSchemaSvc.Update(reqCtx, componentID, req)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/components/"+componentSlug+"?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)

		return nil, nil
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/components/"+componentSlug+"?success=Component+updated", http.StatusSeeOther)

	return nil, nil
}

func (e *DashboardExtension) HandleDeleteComponentSchema(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	reqCtx := e.injectContext(ctx)
	basePath := e.getBasePath()
	componentSlug := ctx.Param("componentSlug")
	appBase := basePath + "/app/" + currentApp.ID.String()

	// Get existing component to get its ID
	component, err := e.plugin.componentSchemaSvc.GetByName(reqCtx, componentSlug)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/components?error=Component+not+found", http.StatusSeeOther)

		return nil, nil
	}

	componentID, _ := xid.FromString(component.ID)

	// Delete component schema
	err = e.plugin.componentSchemaSvc.Delete(reqCtx, componentID)
	if err != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/components/"+componentSlug+"?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)

		return nil, nil
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, appBase+"/cms/components?success=Component+deleted", http.StatusSeeOther)

	return nil, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// resolveComponentReferences resolves ComponentRef to NestedFields for object/array fields.
func (e *DashboardExtension) resolveComponentReferences(ctx context.Context, contentType *core.ContentTypeDTO) {
	if contentType == nil || len(contentType.Fields) == 0 {
		return
	}

	// Fields is []*ContentFieldDTO, so contentType.Fields[i] is already a pointer
	for i := range contentType.Fields {
		e.resolveFieldComponentRef(ctx, contentType.Fields[i])
	}
}

// resolveFieldComponentRef resolves a single field's ComponentRef recursively.
func (e *DashboardExtension) resolveFieldComponentRef(ctx context.Context, field *core.ContentFieldDTO) {
	if field == nil {
		return
	}

	// Handle object and array types
	if field.Type == "object" || field.Type == "array" {
		// If ComponentRef is set and NestedFields is empty, resolve the reference
		if field.Options.ComponentRef != "" && len(field.Options.NestedFields) == 0 {
			componentSchema, err := e.plugin.componentSchemaSvc.GetByName(ctx, field.Options.ComponentRef)
			if err == nil && componentSchema != nil && len(componentSchema.Fields) > 0 {
				field.Options.NestedFields = componentSchema.Fields
			}
		}

		// Recursively resolve nested fields that might have their own ComponentRef
		for i := range field.Options.NestedFields {
			nestedField := &field.Options.NestedFields[i]
			if nestedField.Options != nil && nestedField.Options.ComponentRef != "" {
				e.resolveNestedFieldComponentRef(ctx, nestedField)
			}
		}

		return
	}

	// Handle oneOf type - resolve ComponentRef for each schema option
	if field.Type == "oneOf" && len(field.Options.Schemas) > 0 {
		for key, schemaOpt := range field.Options.Schemas {
			modified := false

			// If this schema option has a ComponentRef, resolve it to NestedFields
			if schemaOpt.ComponentRef != "" && len(schemaOpt.NestedFields) == 0 {
				componentSchema, err := e.plugin.componentSchemaSvc.GetByName(ctx, schemaOpt.ComponentRef)
				if err == nil && componentSchema != nil && len(componentSchema.Fields) > 0 {
					schemaOpt.NestedFields = componentSchema.Fields
					modified = true
				}
			}

			// Recursively resolve nested fields within the schema option
			for i := range schemaOpt.NestedFields {
				nestedField := &schemaOpt.NestedFields[i]
				if nestedField.Options != nil && nestedField.Options.ComponentRef != "" {
					e.resolveNestedFieldComponentRef(ctx, nestedField)

					modified = true
				}
			}

			// Write back to map if any changes were made
			if modified {
				field.Options.Schemas[key] = schemaOpt
			}
		}
	}
}

// resolveNestedFieldComponentRef resolves ComponentRef for nested fields recursively.
func (e *DashboardExtension) resolveNestedFieldComponentRef(ctx context.Context, field *core.NestedFieldDefDTO) {
	if field == nil || field.Options == nil {
		return
	}

	// Handle object and array types
	if field.Type == "object" || field.Type == "array" {
		// If ComponentRef is set and NestedFields is empty, resolve the reference
		if field.Options.ComponentRef != "" && len(field.Options.NestedFields) == 0 {
			componentSchema, err := e.plugin.componentSchemaSvc.GetByName(ctx, field.Options.ComponentRef)
			if err == nil && componentSchema != nil && len(componentSchema.Fields) > 0 {
				field.Options.NestedFields = componentSchema.Fields
			}
		}

		// Recursively resolve nested fields that might have their own ComponentRef
		for i := range field.Options.NestedFields {
			nestedField := &field.Options.NestedFields[i]
			if nestedField.Options != nil && nestedField.Options.ComponentRef != "" {
				e.resolveNestedFieldComponentRef(ctx, nestedField)
			}
		}

		return
	}

	// Handle oneOf type in nested fields
	if field.Type == "oneOf" && len(field.Options.Schemas) > 0 {
		for key, schemaOpt := range field.Options.Schemas {
			modified := false

			if schemaOpt.ComponentRef != "" && len(schemaOpt.NestedFields) == 0 {
				componentSchema, err := e.plugin.componentSchemaSvc.GetByName(ctx, schemaOpt.ComponentRef)
				if err == nil && componentSchema != nil && len(componentSchema.Fields) > 0 {
					schemaOpt.NestedFields = componentSchema.Fields
					modified = true
				}
			}

			// Recursively resolve nested fields within the schema option
			for i := range schemaOpt.NestedFields {
				nestedField := &schemaOpt.NestedFields[i]
				if nestedField.Options != nil && nestedField.Options.ComponentRef != "" {
					e.resolveNestedFieldComponentRef(ctx, nestedField)

					modified = true
				}
			}

			// Write back to map if any changes were made
			if modified {
				field.Options.Schemas[key] = schemaOpt
			}
		}
	}
}
