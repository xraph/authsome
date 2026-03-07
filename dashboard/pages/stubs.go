package pages

// This file provides stub implementations for functions referenced by
// generated _templ.go files. These are minimal stubs to make the build
// pass while the full implementations are developed.

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/a-h/templ"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// formatTimeAgo returns a human-readable relative time string.
func formatTimeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%dy ago", int(d.Hours()/(24*365)))
	}
}

// initials returns the first character of the user's name or email.
func initials(u *user.User) string {
	if u.FirstName != "" {
		return string([]rune(u.FirstName)[:1])
	}
	if u.Email != "" {
		return string([]rune(u.Email)[:1])
	}
	return "?"
}

// isSessionActive returns true if the session has not yet expired.
func isSessionActive(s *session.Session) bool {
	return s != nil && time.Now().Before(s.ExpiresAt)
}

// formatTimeRemaining returns a human-readable string for time remaining.
func formatTimeRemaining(t time.Time) string {
	d := time.Until(t)
	if d <= 0 {
		return "expired"
	}
	switch {
	case d < time.Minute:
		return "< 1m"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// sessionsTable renders a table of sessions. This is a stub that returns
// an empty component until the full implementation is built.
func sessionsTable(sessions []*session.Session, _ interface{}, _ interface{}, _ string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, _ io.Writer) error {
		return nil
	})
}
