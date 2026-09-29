// Package mask shortens personal identifiers for log lines. A log is not
// the place for an address: a user id finds the account when support needs
// it, and a leaked log then names nobody.
package mask

import "strings"

// Email keeps the first character and the domain: "a***@example.com".
// Anything that is not an address is masked whole.
func Email(s string) string {
	at := strings.LastIndexByte(s, '@')
	if at <= 0 || at == len(s)-1 {
		if s == "" {
			return ""
		}
		return "***"
	}
	return s[:1] + "***@" + s[at+1:]
}

// Emails masks each address.
func Emails(addrs []string) []string {
	out := make([]string, len(addrs))
	for i, a := range addrs {
		out[i] = Email(a)
	}
	return out
}

// Phone keeps the last four digits: "***1234".
func Phone(s string) string {
	if len(s) <= 4 {
		if s == "" {
			return ""
		}
		return "****"
	}
	return "***" + s[len(s)-4:]
}

// Identifier masks whatever a sign-in identifier is: an address as an
// address, a phone number as a phone number, anything else whole.
func Identifier(s string) string {
	switch {
	case strings.ContainsRune(s, '@'):
		return Email(s)
	case s != "" && (s[0] == '+' || (s[0] >= '0' && s[0] <= '9')):
		return Phone(s)
	case s == "":
		return ""
	default:
		return "***"
	}
}
