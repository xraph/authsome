package principal

// Chain is an actor chain, ordered nearest-caller-first.
//
// It follows RFC 8693. The session's subject is who the request is for, and
// the chain is who is doing the acting on their behalf. Element 0 is the
// immediate caller. A chain of length two is an ephemeral child acting
// through its registered parent. An empty chain means the subject is calling
// directly, which is the ordinary sign-in case.
type Chain []Ref

// Actor returns the immediate caller.
func (c Chain) Actor() (Ref, bool) {
	if len(c) == 0 {
		return Ref{}, false
	}
	return c[0], true
}

// Root returns the outermost hop, the principal furthest from the subject.
func (c Chain) Root() (Ref, bool) {
	if len(c) == 0 {
		return Ref{}, false
	}
	return c[len(c)-1], true
}

// Depth returns how many actors stand between the caller and the subject.
func (c Chain) Depth() int { return len(c) }

// Contains reports whether r appears anywhere in the chain.
func (c Chain) Contains(r Ref) bool {
	for _, got := range c {
		if got == r {
			return true
		}
	}
	return false
}

// Prepend returns a chain with r as the new immediate caller. The receiver is
// not modified, so a chain read off a session can be extended without the
// extension leaking back into the session.
func (c Chain) Prepend(r Ref) Chain {
	out := make(Chain, 0, len(c)+1)
	out = append(out, r)
	out = append(out, c...)
	return out
}
