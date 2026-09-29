package user

import "time"

// IsBanned reports whether the user is barred from authenticating at the
// given instant. A ban with an expiry in the past no longer applies.
func (u *User) IsBanned(now time.Time) bool {
	if u == nil || !u.Banned {
		return false
	}
	return u.BanExpires == nil || u.BanExpires.After(now)
}
