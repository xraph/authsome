package openapi

import "strings"

// ParseDeprecationRules reads the --deprecate-tags flag: tagPrefix=note
// rules separated by semicolons. A note can contain a semicolon written as
// \;. Rules without an "=" or with an empty prefix are skipped. Only the
// whole rule is trimmed, so "Billing =note" keeps the prefix "Billing ".
func ParseDeprecationRules(flag string) map[string]string {
	rules := make(map[string]string)
	var rule strings.Builder
	flush := func() {
		parts := strings.SplitN(strings.TrimSpace(rule.String()), "=", 2)
		if len(parts) == 2 && parts[0] != "" {
			rules[parts[0]] = strings.TrimSpace(parts[1])
		}
		rule.Reset()
	}
	for i := 0; i < len(flag); i++ {
		switch {
		case flag[i] == '\\' && i+1 < len(flag) && flag[i+1] == ';':
			rule.WriteByte(';')
			i++
		case flag[i] == ';':
			flush()
		default:
			rule.WriteByte(flag[i])
		}
	}
	flush()
	return rules
}

// DeprecationFor returns the note of the rule whose tag prefix starts one of
// tags, or "" when none does. Rules map a tag prefix (e.g. "Billing") to the
// note shown to callers. When several match, the longest prefix wins, so
// "Billing " (the operator groups such as "Billing Plans") can carry a
// different note from "Billing"; ties go to the alphabetically first prefix
// so the output is stable.
func DeprecationFor(tags []string, rules map[string]string) string {
	best := ""
	found := false
	for p := range rules {
		matches := false
		for _, tag := range tags {
			if strings.HasPrefix(tag, p) {
				matches = true
				break
			}
		}
		if !matches {
			continue
		}
		if !found || len(p) > len(best) || (len(p) == len(best) && p < best) {
			best, found = p, true
		}
	}
	if !found {
		return ""
	}
	return rules[best]
}
