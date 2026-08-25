package dpop

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSupportedAlgs_MatchesAllowedAlgs pins the correspondence between the
// list advertised in server metadata (SupportedAlgs, consumed by
// oauth2provider's discovery document) and the algorithms Parse actually
// accepts (allowedAlgs). SupportedAlgs is a separate, hand-maintained
// literal rather than a derivation of allowedAlgs, so nothing stops the two
// from drifting apart if one is edited without the other.
//
// If they ever disagreed, a client would either be told an algorithm is
// supported when Parse would reject it (broken interoperability), or never
// learn about one Parse does accept (needless restriction). This test is in
// package dpop, not dpop_test, because allowedAlgs is unexported: pinning
// the correspondence requires seeing both sides.
func TestSupportedAlgs_MatchesAllowedAlgs(t *testing.T) {
	want := make([]string, 0, len(allowedAlgs))
	for alg := range allowedAlgs {
		want = append(want, alg)
	}
	sort.Strings(want)

	got := append([]string(nil), SupportedAlgs()...)
	sort.Strings(got)

	assert.Equal(t, want, got)
}
