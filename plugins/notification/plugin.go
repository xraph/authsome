package notification

import (
	"context"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/user"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin           = (*Plugin)(nil)
	_ plugin.OnInit           = (*Plugin)(nil)
	_ plugin.AfterSignUp      = (*Plugin)(nil)
	_ plugin.AfterUserCreate  = (*Plugin)(nil)
	_ plugin.SettingsProvider = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Dynamic setting definitions
// ──────────────────────────────────────────────────

var (
	// SettingAppName controls the application name used in notification templates.
	SettingAppName = settings.Define("notification.app_name", "AuthSome",
		settings.WithDisplayName("Application Name"),
		settings.WithDescription("Application name used in notification templates"),
		settings.WithCategory("Notifications"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithPlaceholder("My App"),
		settings.WithHelpText("Used in notification templates for branding"),
		settings.WithOrder(10),
	)

	// SettingBaseURL controls the application root URL for notification links.
	SettingBaseURL = settings.Define("notification.base_url", "",
		settings.WithDisplayName("Base URL"),
		settings.WithDescription("Application root URL for building links in notifications"),
		settings.WithCategory("Notifications"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithPlaceholder("https://example.com"),
		settings.WithHelpText("Used to build action links in notifications"),
		settings.WithOrder(20),
	)

	// SettingDefaultLocale controls the default locale for notifications.
	SettingDefaultLocale = settings.Define("notification.default_locale", "en",
		settings.WithDisplayName("Default Locale"),
		settings.WithDescription("Default locale for notification templates"),
		settings.WithCategory("Notifications"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithPlaceholder("en"),
		settings.WithHelpText("Language/locale for notification content. Default: en"),
		settings.WithOrder(30),
	)

	// SettingAsync controls whether notifications are sent asynchronously.
	SettingAsync = settings.Define("notification.async", false,
		settings.WithDisplayName("Async Delivery"),
		settings.WithDescription("Send notifications asynchronously via dispatch queue"),
		settings.WithCategory("Notifications"),
		settings.WithScopes(settings.ScopeGlobal),
		settings.WithHelpText("When enabled, notifications are queued for async delivery"),
		settings.WithOrder(40),
	)
)

// DeclareSettings implements plugin.SettingsProvider.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "notification", SettingAppName); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "notification", SettingBaseURL); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "notification", SettingDefaultLocale); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "notification", SettingAsync)
}

// userLookup provides user email resolution for recipient lookup.
type userLookup interface {
	GetUser(ctx context.Context, userID id.UserID) (*user.User, error)
}

// Plugin is the Herald-backed notification plugin. It replaces the legacy
// email plugin with a unified multi-channel notification system.
type Plugin struct {
	config    Config
	herald    bridge.Herald
	templates bridge.HeraldTemplateManager
	hooks     *hook.Bus
	logger    log.Logger
	mappings  map[string]*Mapping
	users     userLookup
}

// New creates a new notification plugin with optional configuration.
func New(cfg ...Config) *Plugin {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	if c.AppName == "" {
		c.AppName = "AuthSome"
	}
	if c.DefaultLocale == "" {
		c.DefaultLocale = "en"
	}

	// Merge user-provided mappings with defaults.
	mappings := DefaultMappings()
	if c.Mappings != nil {
		for action, m := range c.Mappings {
			if m == nil {
				// nil entry disables the mapping.
				delete(mappings, action)
			} else {
				mappings[action] = m
			}
		}
	}

	return &Plugin{
		config:   c,
		mappings: mappings,
	}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "notification" }

// OnInit extracts the Herald bridge and hook bus from the engine during
// initialization using duck-typing (same pattern as MFA plugin).
func (p *Plugin) OnInit(_ context.Context, engine any) error {
	// Discover Herald bridge (required).
	type heraldGetter interface {
		Herald() bridge.Herald
	}
	if hg, ok := engine.(heraldGetter); ok {
		p.herald = hg.Herald()
	}
	if p.herald == nil {
		return bridge.ErrHeraldNotAvailable
	}

	// Discover template manager (optional, available when real Herald is configured).
	if tm, ok := p.herald.(bridge.HeraldTemplateManager); ok {
		p.templates = tm
	}

	// Discover hook bus.
	type hooksGetter interface {
		Hooks() *hook.Bus
	}
	if hg, ok := engine.(hooksGetter); ok {
		p.hooks = hg.Hooks()
	}

	// Discover logger.
	type loggerGetter interface {
		Logger() log.Logger
	}
	if lg, ok := engine.(loggerGetter); ok {
		p.logger = lg.Logger()
	}
	if p.logger == nil {
		p.logger = log.NewNoopLogger()
	}

	// Discover user store for recipient resolution fallback.
	// We can't use Store() because its return type (store.Store) can't
	// be imported here without circular deps. Instead, check if the
	// engine itself satisfies userLookup (has GetUser).
	if ul, ok := engine.(userLookup); ok {
		p.users = ul
	}

	// Discover plugin-contributed notification mappings.
	type registryGetter interface {
		Plugins() interface{ Plugins() []plugin.Plugin }
	}
	if rg, ok := engine.(registryGetter); ok {
		for _, pl := range rg.Plugins().Plugins() {
			if contributor, ok := pl.(plugin.NotificationMappingContributor); ok {
				for action, m := range contributor.NotificationMappings() {
					// Plugin-contributed mappings don't override user-provided config mappings.
					if _, exists := p.mappings[action]; !exists {
						p.mappings[action] = &Mapping{
							Template: m.Template,
							Channels: m.Channels,
							Enabled:  m.Enabled,
						}
					}
				}
			}
		}
	}

	// Seed default Herald templates so notifications don't fail with
	// "template not found" on a fresh database.
	if p.templates != nil {
		appID := ""
		type platformAppIDGetter interface {
			PlatformAppID() id.AppID
		}
		if pg, ok := engine.(platformAppIDGetter); ok {
			if pid := pg.PlatformAppID(); !pid.IsNil() {
				appID = pid.String()
			}
		}
		if err := p.templates.SeedDefaultTemplates(context.Background(), appID); err != nil {
			p.logger.Warn("notification plugin: failed to seed default templates",
				log.String("error", err.Error()),
			)
		}
	}

	// Register a global hook handler that auto-sends notifications for
	// all mapped actions that aren't handled by direct plugin hooks.
	if p.hooks != nil && p.herald != nil {
		p.hooks.On("herald-notification", func(ctx context.Context, event *hook.Event) error {
			return p.handleHookEvent(ctx, event)
		})
	}

	return nil
}

// OnAfterSignUp sends a welcome notification to the newly registered user.
func (p *Plugin) OnAfterSignUp(ctx context.Context, u *user.User, _ *session.Session) error {
	if p.herald == nil {
		return nil
	}

	m, ok := p.mappings[hook.ActionSignUp]
	if !ok || !m.Enabled {
		return nil
	}

	name := u.Name()
	if name == "" {
		name = u.Email
	}

	if err := p.herald.Notify(ctx, &bridge.HeraldNotifyRequest{
		Template: m.Template,
		Channels: m.Channels,
		To:       []string{u.Email},
		UserID:   u.ID.String(),
		Locale:   p.config.DefaultLocale,
		Async:    p.config.Async,
		Data: map[string]any{
			"user_name": name,
			"app_name":  p.config.AppName,
			"login_url": p.config.BaseURL + "/login",
		},
	}); err != nil {
		p.logger.Warn("notification plugin: failed to send welcome notification",
			log.String("email", u.Email),
			log.String("error", err.Error()),
		)
	}

	return nil
}

// OnAfterUserCreate sends a verification notification to the newly created user.
func (p *Plugin) OnAfterUserCreate(ctx context.Context, u *user.User) error {
	if p.herald == nil {
		return nil
	}

	// Only send if user has not yet verified their email.
	if u.EmailVerified {
		return nil
	}

	m, ok := p.mappings[hook.ActionUserCreate]
	if !ok || !m.Enabled {
		return nil
	}

	name := u.Name()
	if name == "" {
		name = u.Email
	}

	if err := p.herald.Notify(ctx, &bridge.HeraldNotifyRequest{
		Template: m.Template,
		Channels: m.Channels,
		To:       []string{u.Email},
		UserID:   u.ID.String(),
		Locale:   p.config.DefaultLocale,
		Async:    p.config.Async,
		Data: map[string]any{
			"user_name":  name,
			"app_name":   p.config.AppName,
			"verify_url": p.config.BaseURL + "/verify-email",
		},
	}); err != nil {
		p.logger.Warn("notification plugin: failed to send verification notification",
			log.String("email", u.Email),
			log.String("error", err.Error()),
		)
	}

	return nil
}

// SendPasswordReset sends a password reset notification. This is typically
// called by the engine's password reset flow.
func (p *Plugin) SendPasswordReset(ctx context.Context, email, name, resetURL string) error {
	if p.herald == nil {
		return bridge.ErrHeraldNotAvailable
	}

	m, ok := p.mappings[hook.ActionPasswordReset]
	if !ok || !m.Enabled {
		return nil
	}

	if name == "" {
		name = email
	}

	return p.herald.Notify(ctx, &bridge.HeraldNotifyRequest{
		Template: m.Template,
		Channels: m.Channels,
		To:       []string{email},
		Locale:   p.config.DefaultLocale,
		Async:    p.config.Async,
		Data: map[string]any{
			"user_name":  name,
			"app_name":   p.config.AppName,
			"reset_url":  resetURL,
			"expires_in": "1 hour",
		},
	})
}

// SendInvitation sends an organization invitation notification.
func (p *Plugin) SendInvitation(ctx context.Context, email, inviterName, orgName, acceptURL string) error {
	if p.herald == nil {
		return bridge.ErrHeraldNotAvailable
	}

	m, ok := p.mappings[hook.ActionInvitationAccept]
	if !ok || !m.Enabled {
		return nil
	}

	return p.herald.Notify(ctx, &bridge.HeraldNotifyRequest{
		Template: m.Template,
		Channels: m.Channels,
		To:       []string{email},
		Locale:   p.config.DefaultLocale,
		Async:    p.config.Async,
		Data: map[string]any{
			"inviter_name": inviterName,
			"org_name":     orgName,
			"app_name":     p.config.AppName,
			"accept_url":   acceptURL,
		},
	})
}

// handleHookEvent is the global hook handler that catches actions not handled
// by direct plugin hooks (e.g. password_change, account_locked).
func (p *Plugin) handleHookEvent(ctx context.Context, event *hook.Event) error {
	// Skip actions that are handled by direct plugin hooks to avoid
	// sending duplicate notifications.
	switch event.Action {
	case hook.ActionSignUp, hook.ActionUserCreate:
		return nil
	}

	m, ok := p.mappings[event.Action]
	if !ok || !m.Enabled {
		return nil
	}

	// Build recipient list from event metadata.
	var to []string
	if email, ok := event.Metadata["email"]; ok && email != "" {
		to = []string{email}
	}

	// Fallback: resolve email from ActorID via user store lookup.
	if len(to) == 0 && event.ActorID != "" && p.users != nil {
		if uid, err := id.ParseUserID(event.ActorID); err == nil {
			if u, err := p.users.GetUser(ctx, uid); err == nil && u.Email != "" {
				to = []string{u.Email}
			}
		}
	}

	if len(to) == 0 {
		// No recipient — skip silently.
		return nil
	}

	// Build template data from event metadata.
	data := make(map[string]any, len(event.Metadata)+2)
	for k, v := range event.Metadata {
		data[k] = v
	}
	data["app_name"] = p.config.AppName

	return p.herald.Notify(ctx, &bridge.HeraldNotifyRequest{
		AppID:    event.Tenant,
		Template: m.Template,
		Channels: m.Channels,
		To:       to,
		UserID:   event.ActorID,
		Locale:   p.config.DefaultLocale,
		Async:    p.config.Async,
		Data:     data,
	})
}
