package utils

import (
	"github.com/rs/xid"
	"github.com/xraph/forge"
)

// GetXIDParams retrieves and converts a string parameter from the context into an XID. Returns a nil XID on failure.
func GetXIDParams(ctx forge.Context, key string) xid.ID {
	p := ctx.Param(key)

	xpid, err := xid.FromString(p)
	if err != nil {
		return xid.NilID()
	}

	return xpid
}
