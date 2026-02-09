package email

// Provider defines minimal email sending capabilities used by plugins.
type Provider interface {
	SendMagicLink(to, url string) error
	SendOTP(to, otp string) error
	SendVerification(to, url string) error
}
