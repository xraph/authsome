package anonymous

import (
    "context"
    "fmt"
    "time"
    "strings"

    "github.com/rs/xid"
    "github.com/xraph/authsome/core/session"
    coreuser "github.com/xraph/authsome/core/user"
    "github.com/xraph/authsome/internal/crypto"
    repo "github.com/xraph/authsome/repository"
)

// Service provides anonymous sign-in by creating a guest user and session
type Service struct {
    users  *repo.UserRepository
    sess   *session.Service
}

func NewService(users *repo.UserRepository, sess *session.Service) *Service { return &Service{users: users, sess: sess} }

// SignInGuest creates a guest user and returns a session token
func (s *Service) SignInGuest(ctx context.Context, ip, ua string) (*session.Session, error) {
    id := xid.New()
    email := fmt.Sprintf("guest-%s@guest.local", id.String())
    now := time.Now().UTC()
    // Ensure unique username to satisfy unique constraint
    u := &coreuser.User{ID: id, Email: email, Name: "Guest", CreatedAt: now, UpdatedAt: now, Username: "guest_" + id.String()}
    if err := s.users.Create(ctx, u); err != nil { return nil, err }
    sess, err := s.sess.Create(ctx, &session.CreateSessionRequest{UserID: id, Remember: false, IPAddress: ip, UserAgent: ua})
    if err != nil { return nil, err }
    return sess, nil
}

// LinkGuest upgrades an anonymous guest account to a real account
func (s *Service) LinkGuest(ctx context.Context, token, email, password, name string) (*coreuser.User, error) {
    if strings.TrimSpace(email) == "" || strings.TrimSpace(password) == "" {
        return nil, fmt.Errorf("missing fields")
    }
    // Resolve session and current user
    sess, err := s.sess.FindByToken(ctx, token)
    if err != nil || sess == nil { return nil, fmt.Errorf("invalid session") }
    u, err := s.users.FindByID(ctx, sess.UserID)
    if err != nil || u == nil { return nil, fmt.Errorf("user not found") }
    // Basic check to ensure it's a guest user
    if !strings.HasSuffix(strings.ToLower(u.Email), "@guest.local") {
        return nil, fmt.Errorf("account is not anonymous")
    }
    // Check email uniqueness
    if existing, err := s.users.FindByEmail(ctx, strings.ToLower(email)); err == nil && existing != nil {
        return nil, fmt.Errorf("email already exists")
    }
    // Hash password and update fields
    hash, err := crypto.HashPassword(password)
    if err != nil { return nil, err }
    u.Email = strings.ToLower(email)
    u.PasswordHash = hash
    if strings.TrimSpace(name) != "" { u.Name = name }
    u.UpdatedAt = time.Now().UTC()
    if err := s.users.Update(ctx, u); err != nil { return nil, err }
    return u, nil
}