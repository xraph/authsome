package sms

// Provider defines minimal SMS sending capabilities used by phone plugin
type Provider interface {
    SendSMS(to, message string) error
}