package waitlist

import (
	"time"

	"github.com/xraph/grove"

	"github.com/xraph/authsome/id"
)

// ──────────────────────────────────────────────────
// Waitlist model (shared across SQL stores)
// ──────────────────────────────────────────────────

type waitlistModel struct {
	grove.BaseModel `grove:"table:authsome_waitlist_entries,alias:wl"`

	ID        string    `grove:"id,pk"`
	AppID     string    `grove:"app_id,notnull"`
	Email     string    `grove:"email,notnull"`
	Name      string    `grove:"name,notnull"`
	Status    string    `grove:"status,notnull"`
	UserID    string    `grove:"user_id"`
	IPAddress string    `grove:"ip_address,notnull"`
	Note      string    `grove:"note,notnull"`
	CreatedAt time.Time `grove:"created_at,notnull"`
	UpdatedAt time.Time `grove:"updated_at,notnull"`
}

// ──────────────────────────────────────────────────
// Converters
// ──────────────────────────────────────────────────

func toWaitlistEntry(m *waitlistModel) (*WaitlistEntry, error) {
	entryID, err := id.ParseWaitlistID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}

	e := &WaitlistEntry{
		ID:        entryID,
		AppID:     appID,
		Email:     m.Email,
		Name:      m.Name,
		Status:    WaitlistStatus(m.Status),
		IPAddress: m.IPAddress,
		Note:      m.Note,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}

	if m.UserID != "" {
		uid, err := id.ParseUserID(m.UserID)
		if err != nil {
			return nil, err
		}
		e.UserID = &uid
	}

	return e, nil
}

func fromWaitlistEntry(e *WaitlistEntry) *waitlistModel {
	m := &waitlistModel{
		ID:        e.ID.String(),
		AppID:     e.AppID.String(),
		Email:     e.Email,
		Name:      e.Name,
		Status:    string(e.Status),
		IPAddress: e.IPAddress,
		Note:      e.Note,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
	if e.UserID != nil {
		m.UserID = e.UserID.String()
	}
	return m
}
