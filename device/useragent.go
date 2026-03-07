package device

import "strings"

// ParseUserAgent extracts browser, OS, and device type from a User-Agent string.
// Uses simple heuristic parsing to avoid external dependencies.
func ParseUserAgent(ua string) (browser, os, deviceType string) {
	ua = strings.TrimSpace(ua)
	if ua == "" {
		return "Unknown", "Unknown", "unknown"
	}

	browser = parseBrowser(ua)
	os = parseOS(ua)
	deviceType = parseDeviceType(ua)
	return
}

func parseBrowser(ua string) string {
	// Order matters: more specific checks first.
	switch {
	case strings.Contains(ua, "Edg/"):
		return "Edge"
	case strings.Contains(ua, "OPR/") || strings.Contains(ua, "Opera"):
		return "Opera"
	case strings.Contains(ua, "Brave"):
		return "Brave"
	case strings.Contains(ua, "Vivaldi"):
		return "Vivaldi"
	case strings.Contains(ua, "Chrome/") && !strings.Contains(ua, "Chromium"):
		return "Chrome"
	case strings.Contains(ua, "Firefox/"):
		return "Firefox"
	case strings.Contains(ua, "Safari/") && !strings.Contains(ua, "Chrome"):
		return "Safari"
	case strings.Contains(ua, "MSIE") || strings.Contains(ua, "Trident/"):
		return "Internet Explorer"
	default:
		return "Other"
	}
}

func parseOS(ua string) string {
	switch {
	case strings.Contains(ua, "iPhone") || strings.Contains(ua, "iPad"):
		return "iOS"
	case strings.Contains(ua, "Android"):
		return "Android"
	case strings.Contains(ua, "Windows"):
		return "Windows"
	case strings.Contains(ua, "Mac OS X") || strings.Contains(ua, "Macintosh"):
		return "macOS"
	case strings.Contains(ua, "Linux"):
		return "Linux"
	case strings.Contains(ua, "CrOS"):
		return "ChromeOS"
	default:
		return "Other"
	}
}

func parseDeviceType(ua string) string {
	switch {
	case strings.Contains(ua, "Mobile") || strings.Contains(ua, "iPhone") || strings.Contains(ua, "Android"):
		if strings.Contains(ua, "Tablet") || strings.Contains(ua, "iPad") {
			return "tablet"
		}
		return "mobile"
	case strings.Contains(ua, "iPad") || strings.Contains(ua, "Tablet"):
		return "tablet"
	default:
		return "desktop"
	}
}
