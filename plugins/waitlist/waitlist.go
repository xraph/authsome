package waitlist

import (
	"context"
	"errors"
	"time"

	"github.com/xraph/authsome/id"
)

// WaitlistStatus represents the approval state of a waitlist entry.
type WaitlistStatus string

const (
	// StatusPending means the entry is awaiting review.
	StatusPending WaitlistStatus = "pending"
	// StatusApproved means the entry has been approved for sign-up.
	StatusApproved WaitlistStatus = "approved"
	// StatusRejected means the entry has been rejected.
	StatusRejected WaitlistStatus = "rejected"
)

// WaitlistEntry is a single record on the waitlist.
type WaitlistEntry struct {
	ID        id.WaitlistID  `json:"id"`
	AppID     id.AppID       `json:"app_id"`
	Email     string         `json:"email"`
	Name      string         `json:"name"`
	Status    WaitlistStatus `json:"status"`
	UserID    *id.UserID     `json:"user_id,omitempty"` // set after sign-up
	IPAddress string         `json:"ip_address"`
	Note      string         `json:"note"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// WaitlistQuery holds filters for listing waitlist entries.
type WaitlistQuery struct {
	AppID  id.AppID       `json:"app_id"`
	Email  string         `json:"email,omitempty"`
	Status WaitlistStatus `json:"status,omitempty"`
	Cursor string         `json:"cursor,omitempty"`
	Limit  int            `json:"limit,omitempty"`
}

// WaitlistList wraps a paginated list of waitlist entries.
type WaitlistList struct {
	Entries    []*WaitlistEntry `json:"entries"`
	Total      int              `json:"total"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

// Store persists and queries waitlist entries.
type Store interface {
	// CreateEntry adds a new waitlist entry.
	CreateEntry(ctx context.Context, e *WaitlistEntry) error

	// GetEntry returns a waitlist entry by ID.
	GetEntry(ctx context.Context, entryID id.WaitlistID) (*WaitlistEntry, error)

	// GetEntryByEmail returns a waitlist entry by app and email.
	GetEntryByEmail(ctx context.Context, appID id.AppID, email string) (*WaitlistEntry, error)

	// UpdateEntryStatus changes the status and optionally sets a note.
	UpdateEntryStatus(ctx context.Context, entryID id.WaitlistID, status WaitlistStatus, note string) error

	// ListEntries returns waitlist entries matching the query.
	ListEntries(ctx context.Context, q *WaitlistQuery) (*WaitlistList, error)

	// CountByStatus returns counts of entries grouped by status for an app.
	CountByStatus(ctx context.Context, appID id.AppID) (pending int, approved int, rejected int, err error)

	// DeleteEntry permanently removes a waitlist entry.
	DeleteEntry(ctx context.Context, entryID id.WaitlistID) error
}

// Sentinel errors.
var (
	// ErrNotFound is returned when a waitlist entry is not found.
	ErrNotFound = errors.New("waitlist: entry not found")

	// ErrDuplicateEmail is returned when the email is already on the waitlist.
	ErrDuplicateEmail = errors.New("waitlist: email already on waitlist")
)
