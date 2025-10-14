package audit

import (
    "context"
    "time"
    "github.com/rs/xid"
)

// Repository defines persistence for audit events
type Repository interface {
    Create(ctx context.Context, e *Event) error
    List(ctx context.Context, limit, offset int) ([]*Event, error)
    Search(ctx context.Context, params ListParams) ([]*Event, error)
    Count(ctx context.Context) (int, error)
    SearchCount(ctx context.Context, params ListParams) (int, error)
}

// Service handles audit logging
type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

// Log creates an audit event with timestamps
func (s *Service) Log(ctx context.Context, userID *xid.ID, action, resource, ip, ua, metadata string) error {
    e := &Event{
        ID:        xid.New(),
        UserID:    userID,
        Action:    action,
        Resource:  resource,
        IPAddress: ip,
        UserAgent: ua,
        Metadata:  metadata,
        CreatedAt: time.Now().UTC(),
        UpdatedAt: time.Now().UTC(),
    }
    return s.repo.Create(ctx, e)
}

// List returns recent audit events ordered by CreatedAt desc
func (s *Service) List(ctx context.Context, limit, offset int) ([]*Event, error) {
    if limit <= 0 { limit = 50 }
    if offset < 0 { offset = 0 }
    return s.repo.List(ctx, limit, offset)
}

// ListWithTotal returns events and total count for pagination
func (s *Service) ListWithTotal(ctx context.Context, limit, offset int) ([]*Event, int, error) {
    if limit <= 0 { limit = 50 }
    if offset < 0 { offset = 0 }
    events, err := s.repo.List(ctx, limit, offset)
    if err != nil { return nil, 0, err }
    total, err := s.repo.Count(ctx)
    if err != nil { return nil, 0, err }
    return events, total, nil
}

// ListParams defines filters for searching audit events
type ListParams struct {
    UserID *xid.ID
    Action string
    Since  *time.Time
    Until  *time.Time
    Limit  int
    Offset int
}

// Search returns events matching provided filters
func (s *Service) Search(ctx context.Context, params ListParams) ([]*Event, error) {
    if params.Limit <= 0 { params.Limit = 50 }
    if params.Offset < 0 { params.Offset = 0 }
    return s.repo.Search(ctx, params)
}

// SearchWithTotal returns filtered events and total count for pagination
func (s *Service) SearchWithTotal(ctx context.Context, params ListParams) ([]*Event, int, error) {
    if params.Limit <= 0 { params.Limit = 50 }
    if params.Offset < 0 { params.Offset = 0 }
    events, err := s.repo.Search(ctx, params)
    if err != nil { return nil, 0, err }
    total, err := s.repo.SearchCount(ctx, params)
    if err != nil { return nil, 0, err }
    return events, total, nil
}