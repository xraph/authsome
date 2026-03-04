package middleware

import (
	"net"
	"net/http"

	"github.com/xraph/forge"
)

// IPAccessConfig configures IP-based access control.
type IPAccessConfig struct {
	// AllowList contains CIDR ranges that are explicitly allowed.
	// If non-empty, only requests from these ranges are permitted.
	AllowList []string `json:"allow_list"`

	// BlockList contains CIDR ranges that are always blocked.
	// Evaluated before AllowList — a blocked IP is rejected even if
	// it also appears in the allow list.
	BlockList []string `json:"block_list"`
}

// IPAccessMiddleware restricts access based on client IP address ranges.
// Block list is evaluated first; if the IP matches a blocked range, the
// request is rejected. If an allow list is configured, only IPs in the
// allow list are permitted.
func IPAccessMiddleware(cfg IPAccessConfig) forge.Middleware {
	blockNets := parseCIDRs(cfg.BlockList)
	allowNets := parseCIDRs(cfg.AllowList)

	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			// Skip if no rules configured.
			if len(blockNets) == 0 && len(allowNets) == 0 {
				return next(ctx)
			}

			ip := net.ParseIP(clientIPFromRequest(ctx.Request()))
			if ip == nil {
				return ctx.JSON(http.StatusForbidden, map[string]any{
					"error": "unable to determine client IP",
					"code":  http.StatusForbidden,
				})
			}

			// Block list takes precedence.
			for _, n := range blockNets {
				if n.Contains(ip) {
					return ctx.JSON(http.StatusForbidden, map[string]any{
						"error": "access denied",
						"code":  http.StatusForbidden,
					})
				}
			}

			// If allow list is configured, IP must match at least one range.
			if len(allowNets) > 0 {
				allowed := false
				for _, n := range allowNets {
					if n.Contains(ip) {
						allowed = true
						break
					}
				}
				if !allowed {
					return ctx.JSON(http.StatusForbidden, map[string]any{
						"error": "access denied",
						"code":  http.StatusForbidden,
					})
				}
			}

			return next(ctx)
		}
	}
}

func parseCIDRs(cidrs []string) []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			// Try as a single IP (e.g. "1.2.3.4" -> "1.2.3.4/32").
			ip := net.ParseIP(cidr)
			if ip == nil {
				continue
			}
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			n = &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)}
		}
		nets = append(nets, n)
	}
	return nets
}
