package services

import (
	"fmt"
	"net/http"

	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
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

func (s *Services) AuthMiddleware(next router.PageHandler) router.PageHandler {
	return func(ctx *router.PageContext) (g.Node, error) {
		user, sess, err := s.CheckExistingPageSession(ctx)
		if err != nil {
			// Invalid session - redirect to login
			ctx.SetHeader("Location", s.basePath+"/login")
			ctx.ResponseWriter.WriteHeader(http.StatusFound)
			return nil, nil
		}

		// Store user in context for handler access
		ctx.Set(UserKey, user)
		ctx.Set(SessionKey, sess)
		ctx.Set(AuthenticatedKey, true)

		// Continue to next handler
		return next(ctx)
	}
}

func (s *Services) AuthlessMiddleware(next router.PageHandler) router.PageHandler {
	return func(ctx *router.PageContext) (g.Node, error) {
		user, sess, err := s.CheckExistingPageSession(ctx)
		if err == nil && user != nil && sess != nil {
			fmt.Println("--------------------------------", s.basePath)
			fmt.Println("user", user)
			fmt.Println("sess", sess)
			fmt.Println("err", err)
			fmt.Println("--------------------------------", s.basePath)
			// Invalid session - redirect to login
			ctx.SetHeader("Location", s.basePath)
			ctx.ResponseWriter.WriteHeader(http.StatusFound)
			return nil, nil
		}

		// Continue to next handler
		return next(ctx)
	}
}
