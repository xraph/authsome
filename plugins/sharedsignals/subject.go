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
	UserID    id.UserID
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
			if sessionID, perr := id.ParseSessionID(sessionMember.ID); perr == nil {
				res.SessionID = sessionID
			}
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
		if !domainAllowed(s, subj.Email) {
			return rejected(), nil
		}
		u, err := p.authStore.GetUserByAnyEmail(ctx, s.AppID, s.EnvID, subj.Email)
		if err != nil {
			return unresolved(), nil //nolint:nilerr // a miss is not a failure
		}
		// An unverified address proves nothing about who owns it.
		record, err := p.authStore.GetUserEmailRecord(ctx, s.AppID, s.EnvID, subj.Email)
		if err != nil || record == nil || !record.Verified {
			return rejected(), nil //nolint:nilerr // refuse rather than guess
		}
		return Resolution{UserID: u.ID, Outcome: OutcomeApplied}, nil

	case caep.FormatPhoneNumber:
		u, err := p.authStore.GetUserByPhone(ctx, s.AppID, subj.PhoneNumber)
		if err != nil {
			return unresolved(), nil //nolint:nilerr // a miss is not a failure
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
