// Package caep implements the OpenID CAEP event types and the RFC 9493
// subject identifiers they carry. It has no authsome dependencies so it can
// be tested against spec fixtures on its own.
package caep

import (
	"encoding/json"
	"fmt"
)

// Subject identifier formats defined by RFC 9493.
const (
	FormatAccount     = "account"
	FormatEmail       = "email"
	FormatIssSub      = "iss_sub"
	FormatOpaque      = "opaque"
	FormatPhoneNumber = "phone_number"
	FormatDID         = "did"
	FormatURI         = "uri"
	FormatAliases     = "aliases"
)

// ComplexSubjectMembers are the member names the SSF spec defines for a
// Complex Subject. Any other member name is carried but not interpreted.
var ComplexSubjectMembers = []string{
	"user", "device", "session", "application", "tenant", "org_unit", "group",
}

// SubjectID is an RFC 9493 subject identifier. A simple subject sets Format
// and the members that format requires. A complex subject sets Members
// instead, with Format empty.
type SubjectID struct {
	Format      string
	Email       string
	PhoneNumber string
	Issuer      string
	Subject     string
	ID          string
	URI         string
	URL         string
	Identifiers []SubjectID
	Members     map[string]SubjectID
}

// IsComplex reports whether this is a Complex Subject, meaning it carries
// named members rather than a format of its own.
func (s SubjectID) IsComplex() bool { return s.Format == "" && len(s.Members) > 0 }

// Member returns the named member of a Complex Subject.
func (s SubjectID) Member(name string) (SubjectID, bool) {
	m, ok := s.Members[name]
	return m, ok
}

type subjectWire struct {
	Format      string            `json:"format"`
	Email       string            `json:"email"`
	PhoneNumber string            `json:"phone_number"`
	Issuer      string            `json:"iss"`
	Subject     string            `json:"sub"`
	ID          string            `json:"id"`
	URI         string            `json:"uri"`
	URL         string            `json:"url"`
	Identifiers []json.RawMessage `json:"identifiers"`
}

// ParseSubjectID decodes a subject identifier, simple or complex.
func ParseSubjectID(raw json.RawMessage) (SubjectID, error) {
	return parseSubjectID(raw, 0)
}

func parseSubjectID(raw json.RawMessage, depth int) (SubjectID, error) {
	if depth > 1 {
		return SubjectID{}, fmt.Errorf("caep: subject nested too deeply")
	}

	var w subjectWire
	if err := json.Unmarshal(raw, &w); err != nil {
		return SubjectID{}, fmt.Errorf("caep: decode subject: %w", err)
	}

	s := SubjectID{
		Format:      w.Format,
		Email:       w.Email,
		PhoneNumber: w.PhoneNumber,
		Issuer:      w.Issuer,
		Subject:     w.Subject,
		ID:          w.ID,
		URI:         w.URI,
		URL:         w.URL,
	}

	if w.Format == FormatAliases {
		for _, item := range w.Identifiers {
			alias, err := parseSubjectID(item, depth+1)
			if err != nil {
				return SubjectID{}, err
			}
			if alias.Format == FormatAliases {
				return SubjectID{}, fmt.Errorf("caep: aliases may not nest aliases")
			}
			s.Identifiers = append(s.Identifiers, alias)
		}
		return s, nil
	}

	if w.Format != "" {
		return s, nil
	}

	// No format member: treat it as a Complex Subject and decode the members
	// we recognise. Unknown member names are ignored here; the receiver
	// enforces critical_subject_members separately.
	var members map[string]json.RawMessage
	if err := json.Unmarshal(raw, &members); err != nil {
		return SubjectID{}, fmt.Errorf("caep: decode complex subject: %w", err)
	}
	for _, name := range ComplexSubjectMembers {
		item, ok := members[name]
		if !ok {
			continue
		}
		member, err := parseSubjectID(item, depth+1)
		if err != nil {
			return SubjectID{}, err
		}
		if s.Members == nil {
			s.Members = make(map[string]SubjectID, len(members))
		}
		s.Members[name] = member
	}
	if len(s.Members) == 0 {
		return SubjectID{}, fmt.Errorf("caep: subject has no format and no known members")
	}
	return s, nil
}
