package sharedsignals

import (
	"context"
	"errors"
	"strings"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/sharedsignals/caep"
)

// Resolution is the outcome of turning a subject identifier into one of our
// users. Outcome is OutcomeApplied when UserID is set, OutcomeUnresolved when
// the identifier was acceptable but named nobody we know, and OutcomeRejected
// when the identifier was not one this stream may assert.
type Resolution struct {
	UserID id.UserID
	// SessionID is set only for a complex subject whose session member both
	// parsed as a valid session ID AND was verified to belong to UserID. A
	// session member that fails either check rejects the whole event rather
	// than being dropped, so callers may treat a non-nil SessionID here as
	// already scoped to UserID without re-checking ownership themselves.
	SessionID id.SessionID
	Outcome   string
}

func rejected() Resolution   { return Resolution{Outcome: OutcomeRejected} }
func unresolved() Resolution { return Resolution{Outcome: OutcomeUnresolved} }

func allowsFormat(s *InboundStream, format string) bool {
	for _, f := range s.AllowedSubjectFormats {
		if f == format {
			return true
		}
	}
	return false
}

func domainAllowed(s *InboundStream, email string) bool {
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return false
	}
	domain := strings.ToLower(email[at+1:])
	for _, d := range s.VerifiedDomains {
		if strings.ToLower(d) == domain {
			return true
		}
	}
	return false
}

// isPlausibleE164 is a cheap format check for a phone_number subject, per
// RFC 9493's use of E.164. It does not validate real dialing plans -- that is
// not our job -- only that the string has the shape of one: a leading '+', a
// non-zero country-code digit, and a sane total digit count. Anything else is
// refused before it ever reaches the store.
func isPlausibleE164(phone string) bool {
	if len(phone) < 8 || len(phone) > 16 || phone[0] != '+' {
		return false
	}
	digits := phone[1:]
	for i := 0; i < len(digits); i++ {
		d := digits[i]
		if d < '0' || d > '9' {
			return false
		}
		if i == 0 && d == '0' {
			return false // a country code never starts with 0
		}
	}
	return true
}

// resolveSubject maps a subject identifier onto an authsome user, scoped to
// the stream's app and environment. Nothing here trusts the identifier's own
// claim about who may assert it.
func (p *Plugin) resolveSubject(ctx context.Context, s *InboundStream,
	subj caep.SubjectID) (Resolution, error) {
	// A complex subject describes one principal through named members. The
	// user member is the identity; the session member narrows the action.
	if subj.IsComplex() {
		userMember, ok := subj.Member("user")
		if !ok {
			return rejected(), nil
		}
		res, err := p.resolveSimpleSubject(ctx, s, userMember)
		if err != nil || res.Outcome != OutcomeApplied {
			return res, err
		}
		if sessionMember, ok := subj.Member("session"); ok {
			// A session member narrows the blast radius from "every session
			// this user has" to "this one session". Trusting it unverified
			// would let an attacker who can name a session ID widen or
			// misdirect a revocation, so an unparsable ID or one that
			// belongs to someone else fails the whole event closed rather
			// than silently falling back to the broader, more destructive
			// action.
			sessionID, perr := id.ParseSessionID(sessionMember.ID)
			if perr != nil {
				return rejected(), nil
			}
			sess, serr := p.authStore.GetSession(ctx, sessionID)
			if serr != nil || sess == nil || sess.UserID != res.UserID {
				return rejected(), nil
			}
			res.SessionID = sessionID
		}
		return res, nil
	}

	if subj.Format == caep.FormatAliases {
		return p.resolveAliases(ctx, s, subj)
	}

	return p.resolveSimpleSubject(ctx, s, subj)
}

// resolveAliases requires every resolvable member to name the same user.
// Members that resolve to nobody are ignored; members that contradict each
// other kill the event.
//
// A member that is REJECTED on policy grounds (wrong issuer, disallowed
// format, unverified email, wrong environment, ...) is skipped exactly like
// one that resolved to nobody, rather than failing the whole aliases event.
// This is safe because every member independently enforces its own gates
// before it can contribute a candidate user at all -- a rejected member never
// produces a UserID, so it can never be the thing an attacker uses to
// impersonate or contradict. It only means a stream that lists ["iss_sub",
// "email"] but not, say, a foreign-issued iss_sub still lets the email member
// carry the event on its own.
func (p *Plugin) resolveAliases(ctx context.Context, s *InboundStream,
	subj caep.SubjectID) (Resolution, error) {
	if !allowsFormat(s, caep.FormatAliases) {
		return rejected(), nil
	}

	var found Resolution
	for _, alias := range subj.Identifiers {
		res, err := p.resolveSimpleSubject(ctx, s, alias)
		if err != nil {
			return Resolution{}, err
		}
		if res.Outcome != OutcomeApplied {
			continue
		}
		if found.Outcome != OutcomeApplied {
			found = res
			continue
		}
		if found.UserID != res.UserID {
			return rejected(), nil
		}
	}
	if found.Outcome != OutcomeApplied {
		return unresolved(), nil
	}
	return found, nil
}

func (p *Plugin) resolveSimpleSubject(ctx context.Context, s *InboundStream,
	subj caep.SubjectID) (Resolution, error) {
	if !allowsFormat(s, subj.Format) {
		return rejected(), nil
	}

	switch subj.Format {
	case caep.FormatIssSub:
		// An identity provider speaks only for subjects it issued.
		if subj.Issuer != s.Issuer {
			return rejected(), nil
		}
		return p.resolveViaLink(ctx, s, subj.Subject)

	case caep.FormatOpaque:
		return p.resolveViaLink(ctx, s, subj.ID)

	case caep.FormatEmail:
		// Trim once, up front, and use this exact string for both the
		// domain check and the store lookup below -- they must never
		// diverge, or the domain gate could pass on one spelling of the
		// address while the lookup resolves a different one.
		email := strings.TrimSpace(subj.Email)
		if !domainAllowed(s, email) {
			return rejected(), nil
		}
		u, err := p.authStore.GetUserByAnyEmail(ctx, s.AppID, s.EnvID, email)
		if err != nil {
			return unresolved(), nil //nolint:nilerr // a miss is not a failure
		}
		// Belt and braces: GetUserByAnyEmail is already scoped to s.EnvID,
		// but a stream must never act outside its own environment even if a
		// future backend forgets to honor that parameter, so the invariant
		// is checked again here where it is visible.
		if u.EnvID != s.EnvID {
			return rejected(), nil
		}
		// An unverified address proves nothing about who owns it.
		record, err := p.authStore.GetUserEmailRecord(ctx, s.AppID, s.EnvID, email)
		if err != nil || record == nil || !record.Verified {
			return rejected(), nil //nolint:nilerr // refuse rather than guess
		}
		return Resolution{UserID: u.ID, Outcome: OutcomeApplied}, nil

	case caep.FormatPhoneNumber:
		if !isPlausibleE164(subj.PhoneNumber) {
			return rejected(), nil
		}
		// GetUserByPhone is not environment-scoped in the core store
		// interface (it takes appID only), so a phone number could match a
		// user in a sibling environment of the same app. A staging stream
		// must not be able to revoke a production user's sessions just
		// because it learned their phone number, so the environment is
		// re-checked here after the lookup.
		u, err := p.authStore.GetUserByPhone(ctx, s.AppID, subj.PhoneNumber)
		if err != nil {
			return unresolved(), nil //nolint:nilerr // a miss is not a failure
		}
		if u.EnvID != s.EnvID {
			return rejected(), nil
		}
		if !u.PhoneVerified {
			return rejected(), nil
		}
		return Resolution{UserID: u.ID, Outcome: OutcomeApplied}, nil

	default:
		// account, uri, did and anything we do not recognise.
		return rejected(), nil
	}
}

func (p *Plugin) resolveViaLink(ctx context.Context, s *InboundStream,
	subject string) (Resolution, error) {
	if subject == "" {
		return rejected(), nil
	}
	link, err := p.store.GetSubjectLink(ctx, s.AppID, s.EnvID, s.Issuer, subject)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return unresolved(), nil
		}
		return Resolution{}, err
	}
	return Resolution{UserID: link.UserID, Outcome: OutcomeApplied}, nil
}
