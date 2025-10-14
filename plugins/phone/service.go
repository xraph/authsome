package phone

import (
    "context"
    "errors"
    "fmt"
    "math/rand"
    "strings"
    "time"

    "github.com/xraph/authsome/core/audit"
    "github.com/xraph/authsome/core/auth"
    "github.com/xraph/authsome/core/user"
    repo "github.com/xraph/authsome/repository"
    "github.com/xraph/authsome/internal/crypto"
)

// SMSProvider defines minimal interface to send SMS codes
type SMSProvider interface {
    SendSMS(to, message string) error
}

// Config for Phone service
type Config struct {
    CodeLength   int
    ExpiresIn    time.Duration
    MaxAttempts  int
    DevExposeCode bool
    AllowImplicitSignup bool
}

type Service struct {
    repo     *repo.PhoneRepository
    users    *user.Service
    auth     *auth.Service
    audit    *audit.Service
    provider SMSProvider
    config   Config
}

func NewService(r *repo.PhoneRepository, users *user.Service, authSvc *auth.Service, auditSvc *audit.Service, provider SMSProvider, cfg Config) *Service {
    if cfg.CodeLength == 0 { cfg.CodeLength = 6 }
    if cfg.ExpiresIn == 0 { cfg.ExpiresIn = 5 * time.Minute }
    if cfg.MaxAttempts == 0 { cfg.MaxAttempts = 5 }
    return &Service{repo: r, users: users, auth: authSvc, audit: auditSvc, provider: provider, config: cfg}
}

func (s *Service) SendCode(ctx context.Context, phone, ip, ua string) (string, error) {
    p := strings.TrimSpace(phone)
    if p == "" { return "", fmt.Errorf("missing phone") }
    // Generate numeric code
    rand.Seed(time.Now().UnixNano())
    max := int64(1)
    for i := 0; i < s.config.CodeLength; i++ { max *= 10 }
    code := int64(rand.Intn(int(max)))
    otp := fmt.Sprintf("%0*d", s.config.CodeLength, code)
    if err := s.repo.Create(ctx, p, otp, time.Now().Add(s.config.ExpiresIn)); err != nil { return "", err }
    if s.provider != nil { _ = s.provider.SendSMS(p, fmt.Sprintf("Your code: %s", otp)) }
    if s.audit != nil { _ = s.audit.Log(ctx, nil, "phone_code_sent", "phone:"+p, ip, ua, "") }
    if s.config.DevExposeCode { return otp, nil }
    return "", nil
}

func (s *Service) Verify(ctx context.Context, phone, code, email string, remember bool, ip, ua string) (*auth.AuthResponse, error) {
    p := strings.TrimSpace(phone)
    c := strings.TrimSpace(code)
    if p == "" || c == "" { return nil, fmt.Errorf("missing fields") }
    rec, err := s.repo.FindByPhone(ctx, p, time.Now())
    if err != nil { return nil, err }
    if rec == nil { return nil, errors.New("code not found or expired") }
    if rec.Attempts >= s.config.MaxAttempts { return nil, errors.New("too many attempts") }
    if rec.Code != c {
        _ = s.repo.IncrementAttempts(ctx, rec)
        if s.audit != nil { _ = s.audit.Log(ctx, nil, "phone_verify_failed", "phone:"+p, ip, ua, "") }
        return nil, nil
    }
    _ = s.repo.Consume(ctx, rec, time.Now())
    // Associate verification with provided email for session creation
    e := strings.ToLower(strings.TrimSpace(email))
    if e == "" { return nil, fmt.Errorf("missing fields") }
    u, err := s.users.FindByEmail(ctx, e)
    if err != nil || u == nil {
        if !s.config.AllowImplicitSignup { return nil, fmt.Errorf("user not found") }
        pwd, genErr := crypto.GenerateToken(16)
        if genErr != nil { return nil, genErr }
        name := e
        u, err = s.users.Create(ctx, &user.CreateUserRequest{Email: e, Password: pwd, Name: name})
        if err != nil { return nil, err }
    }
    if s.audit != nil {
        uid := u.ID
        _ = s.audit.Log(ctx, &uid, "phone_verify_success", "phone:"+p+" email:"+e, ip, ua, "")
    }
    res, err := s.auth.CreateSessionForUser(ctx, u, remember, ip, ua)
    if err == nil && s.audit != nil {
        uid := u.ID
        _ = s.audit.Log(ctx, &uid, "phone_login", "user:"+uid.String(), ip, ua, "")
    }
    return res, err
}