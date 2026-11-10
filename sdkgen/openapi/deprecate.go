package openapi

import "strings"

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
