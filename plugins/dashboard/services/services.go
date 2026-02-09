package services

import (
	"fmt"
	"net/http"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/environment"
	"github.com/xraph/authsome/core/forms"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/core/webhook"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
)

type Services struct {
	basePath   string
	sessionSvc session.ServiceInterface
	userSvc    user.ServiceInterface
	authSvc    auth.ServiceInterface
	appSvc     app.Service
	orgSvc     organization.CompositeOrganizationService
	rbacSvc    rbac.ServiceInterface
	apikeySvc  *apikey.Service
	formsSvc   *forms.Service
	auditSvc   *audit.Service
	webhookSvc *webhook.Service
	envSvc     environment.EnvironmentService
}

func NewServices(
	basePath string,
	sessionSvc session.ServiceInterface,
	userSvc user.ServiceInterface,
	authSvc auth.ServiceInterface,
	appSvc app.Service,
	orgSvc organization.CompositeOrganizationService,
	rbacSvc rbac.ServiceInterface,
	apikeySvc *apikey.Service,
	formsSvc *forms.Service,
	auditSvc *audit.Service,
	webhookSvc *webhook.Service,
	envSvc environment.EnvironmentService,
) *Services {
	return &Services{
		basePath:   basePath,
		sessionSvc: sessionSvc,
		userSvc:    userSvc,
		authSvc:    authSvc,
		appSvc:     appSvc,
		orgSvc:     orgSvc,
		rbacSvc:    rbacSvc,
		apikeySvc:  apikeySvc,
		formsSvc:   formsSvc,
		auditSvc:   auditSvc,
		webhookSvc: webhookSvc,
		envSvc:     envSvc,
	}
}

func (s *Services) SessionService() session.ServiceInterface {
	return s.sessionSvc
}

func (s *Services) UserService() user.ServiceInterface {
	return s.userSvc
}

func (s *Services) AuthService() auth.ServiceInterface {
	return s.authSvc
}

func (s *Services) AppService() app.Service {
	return s.appSvc
}

func (s *Services) OrganizationService() organization.CompositeOrganizationService {
	return s.orgSvc
}

func (s *Services) RBACService() rbac.ServiceInterface {
	return s.rbacSvc
}

func (s *Services) APIKeyService() *apikey.Service {
	return s.apikeySvc
}

func (s *Services) FormsService() *forms.Service {
	return s.formsSvc
}

func (s *Services) AuditService() *audit.Service {
	return s.auditSvc
}

func (s *Services) WebhookService() *webhook.Service {
	return s.webhookSvc
}

func (s *Services) EnvironmentService() environment.EnvironmentService {
	return s.envSvc
}

func (s *Services) AuthMiddleware(next router.PageHandler) router.PageHandler {
	return func(ctx *router.PageContext) (g.Node, error) {
		user, sess, err := s.CheckExistingPageSession(ctx)
		if err != nil {
			// Invalid session - redirect to login
			ctx.SetHeader("Location", s.basePath+"/auth/login")
			ctx.ResponseWriter.WriteHeader(http.StatusFound)

			return nil, nil
		}

		// Store user in context for handler access
		ctx.Set(UserKey, user)
		ctx.Set(SessionKey, sess)
		ctx.Set(AuthenticatedKey, true)

		// Also enrich Go context with user ID for service layer access
		goCtx := ctx.Request.Context()
		if user != nil && !user.ID.IsNil() {
			goCtx = contexts.SetUserID(goCtx, user.ID)
		}

		if sess != nil && !sess.AppID.IsNil() {
			goCtx = contexts.SetAppID(goCtx, sess.AppID)
		}

		ctx.Request = ctx.Request.WithContext(goCtx)

		// Continue to next handler
		return next(ctx)
	}
}

// AppContextMiddleware injects app context into ForgeUI page context
// AND enriches the Go context with app ID and environment ID for service layer access.
func (s *Services) AppContextMiddleware(next router.PageHandler) router.PageHandler {
	return func(ctx *router.PageContext) (g.Node, error) {
		goCtx := ctx.Request.Context()

		// Extract appId from URL params
		appIDStr := ctx.Param("appId")
		if appIDStr != "" {
			// Parse and fetch the app
			appID, err := xid.FromString(appIDStr)
			if err == nil {
				// Set app ID in Go context for service layer access
				goCtx = contexts.SetAppID(goCtx, appID)

				if s.AppService() != nil {
					currentApp, err := s.AppService().FindAppByID(goCtx, appID)
					if err == nil && currentApp != nil {
						// Store app in page context for layouts
						ctx.Set("currentApp", currentApp)
						ctx.Set("appId", appIDStr)
					}
				}

				// Try to get environment ID from cookie
				envSet := false

				if envCookie, err := ctx.Request.Cookie("authsome_environment"); err == nil && envCookie != nil && envCookie.Value != "" {
					if envID, err := xid.FromString(envCookie.Value); err == nil && !envID.IsNil() {
						goCtx = contexts.SetEnvironmentID(goCtx, envID)
						envSet = true
					}
				}

				// Fall back to default environment for the app
				if !envSet && s.envSvc != nil {
					defaultEnv, err := s.envSvc.GetDefaultEnvironment(goCtx, appID)
					if err == nil && defaultEnv != nil {
						goCtx = contexts.SetEnvironmentID(goCtx, defaultEnv.ID)
					}
				}

				// Update request with enriched context
				ctx.Request = ctx.Request.WithContext(goCtx)
			}
		}

		return next(ctx)
	}
}

func (s *Services) AuthlessMiddleware(next router.PageHandler) router.PageHandler {
	return func(ctx *router.PageContext) (g.Node, error) {
		user, sess, err := s.CheckExistingPageSession(ctx)
		fmt.Println("-------------------------------- 1 --------------------------------", s.basePath)
		fmt.Println("user", user)
		fmt.Println("sess", sess)
		fmt.Println("err", err)
		fmt.Println("-------------------------------- 1 --------------------------------", s.basePath)

		if err == nil && user != nil && sess != nil {
			fmt.Println("-------------------------------- 2 --------------------------------", s.basePath)
			fmt.Println("user", user)
			fmt.Println("sess", sess)
			fmt.Println("err", err)
			fmt.Println("-------------------------------- 2 --------------------------------", s.basePath)
			// Invalid session - redirect to login
			ctx.SetHeader("Location", s.basePath)
			ctx.ResponseWriter.WriteHeader(http.StatusFound)

			return nil, nil
		}

		// Continue to next handler
		return next(ctx)
	}
}
