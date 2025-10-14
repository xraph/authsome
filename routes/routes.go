package routes

import (
    "github.com/xraph/authsome/handlers"
    "github.com/xraph/authsome/plugins/jwt"
    "github.com/xraph/forge"
)

// Register registers auth routes using forge.App
func Register(app *forge.App, basePath string, h *handlers.AuthHandler) {
    auth := app.Group(basePath)
    auth.POST("/signup", h.SignUp)
    auth.POST("/signin", h.SignIn)
    auth.POST("/signout", h.SignOut)
    auth.GET("/session", h.GetSession)
    auth.GET("/devices", h.ListDevices)
    auth.POST("/devices/revoke", h.RevokeDevice)
    auth.POST("/user/update", h.UpdateUser)
}

// RegisterAudit registers audit routes under a base path
func RegisterAudit(app *forge.App, basePath string, h *handlers.AuditHandler) {
    grp := app.Group(basePath)
    grp.GET("/audit/events", h.ListEvents)
}

// RegisterOrganization registers organization routes under a base path
func RegisterOrganization(app *forge.App, basePath string, h *handlers.OrganizationHandler) {
    org := app.Group(basePath)
    // Organizations
    org.POST("/orgs", h.CreateOrganization)
    org.GET("/orgs", h.GetOrganizations)
    org.POST("/orgs/update", h.UpdateOrganization)
    org.POST("/orgs/delete", h.DeleteOrganization)

    // Members
    org.POST("/orgs/members", h.CreateMember)
    org.GET("/orgs/members", h.GetMembers)
    org.POST("/orgs/members/update", h.UpdateMember)
    org.POST("/orgs/members/delete", h.DeleteMember)

    // Teams
    org.POST("/orgs/teams", h.CreateTeam)
    org.GET("/orgs/teams", h.GetTeams)
    org.POST("/orgs/teams/update", h.UpdateTeam)
    org.POST("/orgs/teams/delete", h.DeleteTeam)

    // Team members
    org.POST("/orgs/team_members", h.AddTeamMember)
    org.GET("/orgs/team_members", h.GetTeamMembers)
    org.POST("/orgs/team_members/remove", h.RemoveTeamMember)

    // Invitations
    org.POST("/orgs/invitations", h.CreateInvitation)

    // Policies
    org.POST("/orgs/policies", h.CreatePolicy)
    org.GET("/orgs/policies", h.GetPolicies)
    org.POST("/orgs/policies/delete", h.DeletePolicy)
    org.POST("/orgs/policies/update", h.UpdatePolicy)

    // Organization by ID endpoints (registered after specific paths to avoid conflicts)
    org.GET("/orgs/{id}", h.GetOrganizationByID)
    org.POST("/orgs/{id}/update", h.UpdateOrganizationByID)
    org.POST("/orgs/{id}/delete", h.DeleteOrganizationByID)

    // Roles
    org.POST("/orgs/roles", h.CreateRole)
    org.GET("/orgs/roles", h.GetRoles)

    // User role assignments
    org.POST("/orgs/user_roles/assign", h.AssignUserRole)
    org.POST("/orgs/user_roles/remove", h.RemoveUserRole)
    org.GET("/orgs/user_roles", h.GetUserRoles)
}

// RegisterAPIKey registers API key routes under a base path
func RegisterAPIKey(app *forge.App, basePath string, h *handlers.APIKeyHandler) {
    grp := app.Group(basePath)
    RegisterAPIKeyRoutes(grp, h)
}

// RegisterJWT registers JWT routes under a base path
func RegisterJWT(app *forge.App, basePath string, h *jwt.Handler) {
    grp := app.Group(basePath)
    RegisterJWTRoutes(grp, h)
}

// RegisterWebhook registers webhook routes under a base path
func RegisterWebhook(app *forge.App, basePath string, h *handlers.WebhookHandler) {
    grp := app.Group(basePath)
    RegisterWebhookRoutes(grp, h)
}

// RegisterNotification registers notification routes under a base path
func RegisterNotification(app *forge.App, basePath string, h *handlers.NotificationHandler) {
    grp := app.Group(basePath)
    RegisterNotificationRoutes(grp, h)
}
