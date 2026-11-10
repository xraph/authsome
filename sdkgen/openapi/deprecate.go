package openapi

import (
	"slices"
	"strings"
)

// DeprecationFor returns the note of the first rule whose tag prefix starts
// one of tags, or "" when none does. Rules map a tag prefix (e.g. "Billing")
// to the note shown to callers. Prefixes are tried in sorted order so the
// output is stable.
func DeprecationFor(tags []string, rules map[string]string) string {
	if len(rules) == 0 {
		return ""
	}
	prefixes := make([]string, 0, len(rules))
	for p := range rules {
		prefixes = append(prefixes, p)
	}
	slices.Sort(prefixes)
	for _, p := range prefixes {
		for _, tag := range tags {
			if strings.HasPrefix(tag, p) {
				return rules[p]
			}
		}
	}
	return ""
}
