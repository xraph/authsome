package magiclink

import (
    "context"
    "fmt"
    "net/url"
    "strings"
    "time"

    "github.com/xraph/authsome/core/audit"
    "github.com/xraph/authsome/core/auth"
    "github.com/xraph/authsome/core/user"
    repo "github.com/xraph/authsome/repository"
    "github.com/xraph/authsome/internal/crypto"
)

// EmailProvider defines minimal interface to send magic links via email
type EmailProvider interface {
    SendMagicLink(to, url string) error
}

// Config for Magic Link service
type Config struct {
    ExpiresIn         time.Duration
    BaseURL           string
    DevExposeURL      bool
    AllowImplicitSignup bool
}

type Service struct {
    repo     *repo.MagicLinkRepository
    users    *user.Service
    auth     *auth.Service
    audit    *audit.Service
    provider EmailProvider
    config   Config
}

func NewService(r *repo.MagicLinkRepository, users *user.Service, authSvc *auth.Service, auditSvc *audit.Service, provider EmailProvider, cfg Config) *Service {
    if cfg.ExpiresIn == 0 { cfg.ExpiresIn = 10 * time.Minute }
    return &Service{repo: r, users: users, auth: authSvc, audit: auditSvc, provider: provider, config: cfg}
}

func (s *Service) Send(ctx context.Context, email, ip, ua string) (string, error) {
    e := strings.ToLower(strings.TrimSpace(email))
    if e == "" { return "", fmt.Errorf("missing email") }
    token, err := crypto.GenerateToken(32)
    if err != nil { return "", err }
    if err := s.repo.Create(ctx, e, token, time.Now().Add(s.config.ExpiresIn)); err != nil { return "", err }
    esc := url.QueryEscape(token)
    url := s.config.BaseURL + "/api/auth/magic-link/verify?token=" + esc
    if s.provider != nil { _ = s.provider.SendMagicLink(e, url) }
    if s.audit != nil {
        _ = s.audit.Log(ctx, nil, "magiclink_sent", "email:"+e, ip, ua, "")
    }
    if s.config.DevExposeURL || s.provider == nil { return url, nil }
    return "", nil
}

func (s *Service) Verify(ctx context.Context, token string, remember bool, ip, ua string) (*auth.AuthResponse, error) {
    t := strings.TrimSpace(token)
    if t == "" { return nil, fmt.Errorf("missing token") }
    rec, err := s.repo.FindByToken(ctx, t, time.Now())
    if err != nil || rec == nil {
        if s.audit != nil { _ = s.audit.Log(ctx, nil, "magiclink_verify_failed", "token:"+t, ip, ua, "") }
        return nil, fmt.Errorf("invalid or expired token")
    }
    _ = s.repo.Consume(ctx, rec, time.Now())
    // Find or create user
    u, err := s.users.FindByEmail(ctx, rec.Email)
    if err != nil || u == nil {
        if !s.config.AllowImplicitSignup { return nil, fmt.Errorf("user not found") }
        pwd, genErr := crypto.GenerateToken(16)
        if genErr != nil { return nil, genErr }
        name := rec.Email
        if at := strings.Index(rec.Email, "@"); at > 0 { name = rec.Email[:at] }
        u, err = s.users.Create(ctx, &user.CreateUserRequest{Email: rec.Email, Password: pwd, Name: name})
        if err != nil { return nil, err }
    }
    if s.audit != nil {
        uid := u.ID
        _ = s.audit.Log(ctx, &uid, "magiclink_verify_success", "email:"+rec.Email, ip, ua, "")
    }
    res, err := s.auth.CreateSessionForUser(ctx, u, remember, ip, ua)
    if err == nil && s.audit != nil {
        uid := u.ID
        _ = s.audit.Log(ctx, &uid, "magiclink_login", "user:"+uid.String(), ip, ua, "")
    }
    return res, err
}