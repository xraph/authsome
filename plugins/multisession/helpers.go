package multisession

import (
	"strings"
	"time"
)

// DeviceInfo contains parsed device information from user agent.
type DeviceInfo struct {
	DeviceType string // Desktop, Mobile, Tablet, Bot, Unknown
	OS         string // Windows, macOS, Linux, iOS, Android, ChromeOS, etc.
	OSVersion  string // OS version if available
	Browser    string // Chrome, Firefox, Safari, Edge, etc.
	BrowserVer string // Browser version if available
	IsMobile   bool
	IsTablet   bool
	IsDesktop  bool
	IsBot      bool
}

// ParseUserAgent parses a user agent string and extracts device information.
func ParseUserAgent(ua string) *DeviceInfo {
	info := &DeviceInfo{
		DeviceType: "Unknown",
		OS:         "Unknown",
		Browser:    "Unknown",
	}

	if ua == "" {
		return info
	}

	uaLower := strings.ToLower(ua)

	// Detect if it's a bot/crawler
	if containsAny(uaLower, []string{"bot", "crawler", "spider", "scrapy", "curl", "wget", "python", "go-http"}) {
		info.DeviceType = "Bot"
		info.IsBot = true
		info.Browser = detectBot(uaLower)

		return info
	}

	// Detect OS
	info.OS, info.OSVersion = detectOS(ua, uaLower)

	// Detect Browser
	info.Browser, info.BrowserVer = detectBrowser(ua, uaLower)

	// Detect Device Type
	info.DeviceType, info.IsMobile, info.IsTablet, info.IsDesktop = detectDeviceType(uaLower)

	return info
}

func detectOS(ua, uaLower string) (string, string) {
	switch {
	// iOS detection (must come before Mac detection)
	case strings.Contains(uaLower, "iphone"):
		return "iOS", extractVersion(ua, "iPhone OS ")
	case strings.Contains(uaLower, "ipad"):
		return "iPadOS", extractVersion(ua, "CPU OS ")
	// Android
	case strings.Contains(uaLower, "android"):
		return "Android", extractVersion(ua, "Android ")
	// Windows
	case strings.Contains(uaLower, "windows nt 10"):
		// Could be Windows 10 or 11
		if strings.Contains(ua, "Windows NT 10.0") {
			return "Windows", "10/11"
		}

		return "Windows", "10"
	case strings.Contains(uaLower, "windows nt 6.3"):
		return "Windows", "8.1"
	case strings.Contains(uaLower, "windows nt 6.2"):
		return "Windows", "8"
	case strings.Contains(uaLower, "windows nt 6.1"):
		return "Windows", "7"
	case strings.Contains(uaLower, "windows"):
		return "Windows", ""
	// macOS
	case strings.Contains(uaLower, "macintosh") || strings.Contains(uaLower, "mac os x"):
		return "macOS", extractMacVersion(ua)
	// Linux distributions
	case strings.Contains(uaLower, "ubuntu"):
		return "Ubuntu", ""
	case strings.Contains(uaLower, "fedora"):
		return "Fedora", ""
	case strings.Contains(uaLower, "debian"):
		return "Debian", ""
	case strings.Contains(uaLower, "linux"):
		return "Linux", ""
	// Chrome OS
	case strings.Contains(uaLower, "cros"):
		return "ChromeOS", ""
	// FreeBSD
	case strings.Contains(uaLower, "freebsd"):
		return "FreeBSD", ""
	default:
		return "Unknown", ""
	}
}

func detectBrowser(ua, uaLower string) (string, string) {
	// Order matters - more specific checks first

	// Edge (Chromium-based)
	if strings.Contains(uaLower, "edg/") {
		return "Edge", extractVersion(ua, "Edg/")
	}
	// Opera (OPR)
	if strings.Contains(uaLower, "opr/") {
		return "Opera", extractVersion(ua, "OPR/")
	}
	// Opera (older)
	if strings.Contains(uaLower, "opera") {
		return "Opera", extractVersion(ua, "Opera/")
	}
	// Brave
	if strings.Contains(uaLower, "brave") {
		return "Brave", ""
	}
	// Vivaldi
	if strings.Contains(uaLower, "vivaldi") {
		return "Vivaldi", extractVersion(ua, "Vivaldi/")
	}
	// Samsung Browser
	if strings.Contains(uaLower, "samsungbrowser") {
		return "Samsung Browser", extractVersion(ua, "SamsungBrowser/")
	}
	// UC Browser
	if strings.Contains(uaLower, "ucbrowser") {
		return "UC Browser", extractVersion(ua, "UCBrowser/")
	}
	// Chrome (must come after other Chromium browsers)
	if strings.Contains(uaLower, "chrome") && !strings.Contains(uaLower, "chromium") {
		return "Chrome", extractVersion(ua, "Chrome/")
	}
	// Chromium
	if strings.Contains(uaLower, "chromium") {
		return "Chromium", extractVersion(ua, "Chromium/")
	}
	// Firefox
	if strings.Contains(uaLower, "firefox") {
		return "Firefox", extractVersion(ua, "Firefox/")
	}
	// Safari (must come after Chrome check)
	if strings.Contains(uaLower, "safari") && !strings.Contains(uaLower, "chrome") {
		return "Safari", extractVersion(ua, "Version/")
	}
	// IE
	if strings.Contains(uaLower, "msie") || strings.Contains(uaLower, "trident") {
		return "Internet Explorer", ""
	}

	return "Unknown", ""
}

func detectDeviceType(uaLower string) (deviceType string, isMobile, isTablet, isDesktop bool) {
	// Check for tablets first (they often have mobile in UA too)
	if containsAny(uaLower, []string{"ipad", "tablet", "kindle", "silk", "playbook"}) {
		return "Tablet", false, true, false
	}

	// Check for mobile devices
	if containsAny(uaLower, []string{"mobile", "iphone", "ipod", "android", "blackberry", "opera mini", "opera mobi", "iemobile", "windows phone"}) {
		// Android tablets often don't have "mobile" in their UA
		if strings.Contains(uaLower, "android") && !strings.Contains(uaLower, "mobile") {
			return "Tablet", false, true, false
		}

		return "Mobile", true, false, false
	}

	// Default to desktop
	return "Desktop", false, false, true
}

func detectBot(uaLower string) string {
	switch {
	case strings.Contains(uaLower, "googlebot"):
		return "Googlebot"
	case strings.Contains(uaLower, "bingbot"):
		return "Bingbot"
	case strings.Contains(uaLower, "yandex"):
		return "Yandex Bot"
	case strings.Contains(uaLower, "curl"):
		return "curl"
	case strings.Contains(uaLower, "wget"):
		return "wget"
	case strings.Contains(uaLower, "python"):
		return "Python"
	case strings.Contains(uaLower, "go-http"):
		return "Go HTTP Client"
	default:
		return "Bot"
	}
}

func extractVersion(ua, prefix string) string {
	idx := strings.Index(ua, prefix)
	if idx == -1 {
		return ""
	}

	start := idx + len(prefix)
	if start >= len(ua) {
		return ""
	}

	// Find the end of the version string (space, semicolon, or parenthesis)
	end := start
	for end < len(ua) {
		c := ua[end]
		if c == ' ' || c == ';' || c == ')' || c == '(' {
			break
		}

		end++
	}

	version := ua[start:end]
	// Return just the major.minor version for cleaner display
	parts := strings.Split(version, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}

	return version
}

func extractMacVersion(ua string) string {
	// Mac OS X 10_15_7 or Mac OS X 10.15.7
	idx := strings.Index(ua, "Mac OS X ")
	if idx == -1 {
		return ""
	}

	start := idx + len("Mac OS X ")
	if start >= len(ua) {
		return ""
	}

	end := start
	for end < len(ua) {
		c := ua[end]
		if c == ')' || c == ';' || c == ' ' {
			break
		}

		end++
	}

	version := ua[start:end]
	version = strings.ReplaceAll(version, "_", ".")

	parts := strings.Split(version, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}

	return version
}

func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}

	return false
}

// FormatDeviceInfo returns a human-readable device string.
func (d *DeviceInfo) FormatDeviceInfo() string {
	if d.Browser == "Unknown" && d.OS == "Unknown" {
		return "Unknown Device"
	}

	var parts []string

	if d.Browser != "Unknown" {
		if d.BrowserVer != "" {
			parts = append(parts, d.Browser+" "+d.BrowserVer)
		} else {
			parts = append(parts, d.Browser)
		}
	}

	if d.OS != "Unknown" {
		if d.OSVersion != "" {
			parts = append(parts, d.OS+" "+d.OSVersion)
		} else {
			parts = append(parts, d.OS)
		}
	}

	return strings.Join(parts, " on ")
}

// ShortDeviceInfo returns a compact device string.
func (d *DeviceInfo) ShortDeviceInfo() string {
	if d.Browser == "Unknown" && d.OS == "Unknown" {
		return "Unknown"
	}

	if d.Browser != "Unknown" {
		return d.Browser
	}

	return d.OS
}

// FormatRelativeTime formats a time as relative time (e.g., "2 hours ago").
func FormatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "Just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}

		return strings.Replace("X minutes ago", "X", itoa(mins), 1)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}

		return strings.Replace("X hours ago", "X", itoa(hours), 1)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "Yesterday"
		}

		return strings.Replace("X days ago", "X", itoa(days), 1)
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}

		return strings.Replace("X weeks ago", "X", itoa(weeks), 1)
	default:
		return t.Format("Jan 2, 2006")
	}
}

// FormatExpiresIn formats time until expiration.
func FormatExpiresIn(t time.Time) string {
	now := time.Now()
	if t.Before(now) {
		return "Expired"
	}

	diff := t.Sub(now)

	switch {
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins <= 1 {
			return "< 1 minute"
		}

		return strings.Replace("X minutes", "X", itoa(mins), 1)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour"
		}

		return strings.Replace("X hours", "X", itoa(hours), 1)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day"
		}

		return strings.Replace("X days", "X", itoa(days), 1)
	default:
		return t.Format("Jan 2")
	}
}

// Simple int to string without importing strconv in helpers.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}

	if i < 0 {
		return "-" + itoa(-i)
	}

	var digits []byte
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}

	return string(digits)
}

// IsSessionActive checks if a session is currently active.
func IsSessionActive(expiresAt time.Time) bool {
	return time.Now().Before(expiresAt)
}

// IsSessionExpiringSoon checks if session expires within given duration.
func IsSessionExpiringSoon(expiresAt time.Time, within time.Duration) bool {
	return time.Until(expiresAt) < within && time.Until(expiresAt) > 0
}
