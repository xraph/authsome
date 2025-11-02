package routes

import (
    "github.com/xraph/authsome/handlers"
    "github.com/xraph/authsome/plugins/jwt"
    "github.com/xraph/forge"
)

// Register registers auth routes using forge.Router
func Register(router forge.Router, basePath string, h *handlers.AuthHandler) {
    auth := router.Group(basePath)
    auth.POST("/signup", h.SignUp)
    auth.POST("/signin", h.SignIn)
    auth.POST("/signout", h.SignOut)
    auth.GET("/session", h.GetSession)
    auth.GET("/devices", h.ListDevices)
    auth.POST("/devices/revoke", h.RevokeDevice)
    auth.POST("/user/update", h.UpdateUser)
}

// RegisterAudit registers audit routes under a base path
func RegisterAudit(router forge.Router, basePath string, h *handlers.AuditHandler) {
    grp := router.Group(basePath)
    grp.GET("/audit/events", h.ListEvents)
}

// RegisterOrganization registers organization routes under a base path
func RegisterOrganization(router forge.Router, basePath string, h *handlers.OrganizationHandler) {
    org := router.Group(basePath)
    // Organizations
    org.POST("/", h.CreateOrganization)
    org.GET("/", h.GetOrganizations)
    org.POST("/update", h.UpdateOrganization)
    org.POST("/delete", h.DeleteOrganization)

    // Members
    org.POST("/members", h.CreateMember)
    org.GET("/members", h.GetMembers)
    org.POST("/members/update", h.UpdateMember)
    org.POST("/members/delete", h.DeleteMember)

    // Teams
    org.POST("/teams", h.CreateTeam)
    org.GET("/teams", h.GetTeams)
    org.POST("/teams/update", h.UpdateTeam)
    org.POST("/teams/delete", h.DeleteTeam)

    // Team members
    org.POST("/team_members", h.AddTeamMember)
    org.GET("/team_members", h.GetTeamMembers)
    org.POST("/team_members/remove", h.RemoveTeamMember)

    // Invitations
    org.POST("/invitations", h.CreateInvitation)

    // Policies
    org.POST("/policies", h.CreatePolicy)
    org.GET("/policies", h.GetPolicies)
    org.POST("/policies/delete", h.DeletePolicy)
    org.POST("/policies/update", h.UpdatePolicy)

    // Organization by ID endpoints (registered after specific paths to avoid conflicts)
    org.GET("/{id}", h.GetOrganizationByID)
    org.POST("/{id}/update", h.UpdateOrganizationByID)
    org.POST("/{id}/delete", h.DeleteOrganizationByID)

    // Roles
    org.POST("/roles", h.CreateRole)
    org.GET("/roles", h.GetRoles)

    // User role assignments
    org.POST("/user_roles/assign", h.AssignUserRole)
    org.POST("/user_roles/remove", h.RemoveUserRole)
    org.GET("/user_roles", h.GetUserRoles)
}

// RegisterAPIKey registers API key routes under a base path
func RegisterAPIKey(router forge.Router, basePath string, h *handlers.APIKeyHandler) {
    grp := router.Group(basePath)
    RegisterAPIKeyRoutes(grp, h)
}

// RegisterJWT registers JWT routes under a base path
func RegisterJWT(router forge.Router, basePath string, h *jwt.Handler) {
    grp := router.Group(basePath)
    RegisterJWTRoutes(grp, h)
}

// RegisterWebhook registers webhook routes under a base path
func RegisterWebhook(router forge.Router, basePath string, h *handlers.WebhookHandler) {
    grp := router.Group(basePath)
    RegisterWebhookRoutes(grp, h)
}

// RegisterNotification registers notification routes under a base path
func RegisterNotification(router forge.Router, basePath string, h *handlers.NotificationHandler) {
    grp := router.Group(basePath)
    RegisterNotificationRoutes(grp, h)
}
