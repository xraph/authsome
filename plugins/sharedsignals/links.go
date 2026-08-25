package sharedsignals

import (
	"context"
	"errors"

	"github.com/xraph/authsome/id"
)

// SubjectLinker is implemented by this plugin so other plugins can record the
// upstream identity they just authenticated without importing the concrete
// type. Callers reach it through engine.Plugin("sharedsignals") and a type
// assertion, the same way risk contributors are wired.
type SubjectLinker interface {
	LinkSubject(ctx context.Context, appID id.AppID, envID id.EnvironmentID,
		issuer, subject string, userID id.UserID, source string) error
}

var _ SubjectLinker = (*Plugin)(nil)

// LinkSubject records that (issuer, subject) is this user, so a later CAEP
// event naming that pair resolves. Calling it repeatedly is safe: the store
// upserts on the tuple.
func (p *Plugin) LinkSubject(ctx context.Context, appID id.AppID, envID id.EnvironmentID,
	issuer, subject string, userID id.UserID, source string) error {
	if issuer == "" {
		return errors.New("sharedsignals: link subject: issuer is required")
	}
	if subject == "" {
		return errors.New("sharedsignals: link subject: subject is required")
	}
	if userID.IsNil() {
		return errors.New("sharedsignals: link subject: user is required")
	}
	if source == "" {
		source = SourceManual
	}
	if p.store == nil {
		return errors.New("sharedsignals: link subject: no store configured")
	}

	return p.store.UpsertSubjectLink(ctx, &SubjectLink{
		ID:      id.NewSSFLinkID(),
		AppID:   appID,
		EnvID:   envID,
		Issuer:  issuer,
		Subject: subject,
		UserID:  userID,
		Source:  source,
	})
}
