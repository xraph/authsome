package email

import (
	"bytes"
	"fmt"
	"html/template"
)

// ──────────────────────────────────────────────────
// Template helpers
// ──────────────────────────────────────────────────

func renderHTML(tmplStr string, data any) string {
	t := template.Must(template.New("").Parse(tmplStr))
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return ""
	}
	return buf.String()
}

// ──────────────────────────────────────────────────
// Welcome Email
// ──────────────────────────────────────────────────

const welcomeHTMLTmpl = `<!DOCTYPE html>
<html>
<body style="font-family: sans-serif; color: #333;">
<h2>Welcome to {{.AppName}}!</h2>
<p>Hi {{.Name}},</p>
<p>Thanks for signing up. Your account is ready to use.</p>
<p>If you have any questions, feel free to reach out to our support team.</p>
<p>— The {{.AppName}} Team</p>
</body>
</html>`

// WelcomeEmail returns a welcome email for a newly registered user.
func WelcomeEmail(name, appName string) (subject, html, text string) {
	subject = fmt.Sprintf("Welcome to %s", appName)
	html = renderHTML(welcomeHTMLTmpl, map[string]string{
		"Name":    name,
		"AppName": appName,
	})
	text = fmt.Sprintf("Welcome to %s!\n\nHi %s,\n\nThanks for signing up. Your account is ready to use.\n\n— The %s Team", appName, name, appName)
	return
}

// ──────────────────────────────────────────────────
// Verification Email
// ──────────────────────────────────────────────────

const verificationHTMLTmpl = `<!DOCTYPE html>
<html>
<body style="font-family: sans-serif; color: #333;">
<h2>Verify your email</h2>
<p>Hi {{.Name}},</p>
<p>Please verify your email address to complete your {{.AppName}} setup.</p>
<p><a href="{{.VerifyURL}}" style="display:inline-block;padding:10px 20px;background:#4F46E5;color:#fff;text-decoration:none;border-radius:4px;">Verify Email</a></p>
<p>Or copy this link into your browser:<br>{{.VerifyURL}}</p>
<p>— The {{.AppName}} Team</p>
</body>
</html>`

// VerificationEmail returns an email verification message.
func VerificationEmail(name, appName, verifyURL string) (subject, html, text string) {
	subject = fmt.Sprintf("Verify your %s email", appName)
	html = renderHTML(verificationHTMLTmpl, map[string]string{
		"Name":      name,
		"AppName":   appName,
		"VerifyURL": verifyURL,
	})
	text = fmt.Sprintf("Verify your email\n\nHi %s,\n\nPlease verify your email address by visiting:\n%s\n\n— The %s Team", name, verifyURL, appName)
	return
}

// ──────────────────────────────────────────────────
// Password Reset Email
// ──────────────────────────────────────────────────

//nolint:gosec // G101: not credentials, HTML template variable name
const passwordResetHTMLTmpl = `<!DOCTYPE html>
<html>
<body style="font-family: sans-serif; color: #333;">
<h2>Reset your password</h2>
<p>Hi {{.Name}},</p>
<p>We received a request to reset your {{.AppName}} password.</p>
<p><a href="{{.ResetURL}}" style="display:inline-block;padding:10px 20px;background:#4F46E5;color:#fff;text-decoration:none;border-radius:4px;">Reset Password</a></p>
<p>Or copy this link into your browser:<br>{{.ResetURL}}</p>
<p>If you did not request this, you can safely ignore this email.</p>
<p>— The {{.AppName}} Team</p>
</body>
</html>`

// PasswordResetEmail returns a password reset email.
func PasswordResetEmail(name, appName, resetURL string) (subject, html, text string) {
	subject = fmt.Sprintf("Reset your %s password", appName)
	html = renderHTML(passwordResetHTMLTmpl, map[string]string{
		"Name":     name,
		"AppName":  appName,
		"ResetURL": resetURL,
	})
	text = fmt.Sprintf("Reset your password\n\nHi %s,\n\nReset your %s password by visiting:\n%s\n\nIf you did not request this, ignore this email.\n\n— The %s Team", name, appName, resetURL, appName)
	return
}

// ──────────────────────────────────────────────────
// Invitation Email
// ──────────────────────────────────────────────────

const invitationHTMLTmpl = `<!DOCTYPE html>
<html>
<body style="font-family: sans-serif; color: #333;">
<h2>You are invited!</h2>
<p>Hi,</p>
<p>{{.InviterName}} has invited you to join <strong>{{.OrgName}}</strong> on {{.AppName}}.</p>
<p><a href="{{.AcceptURL}}" style="display:inline-block;padding:10px 20px;background:#4F46E5;color:#fff;text-decoration:none;border-radius:4px;">Accept Invitation</a></p>
<p>Or copy this link into your browser:<br>{{.AcceptURL}}</p>
<p>— The {{.AppName}} Team</p>
</body>
</html>`

// InvitationEmail returns an organization invitation email.
func InvitationEmail(inviterName, orgName, appName, acceptURL string) (subject, html, text string) {
	subject = fmt.Sprintf("You're invited to join %s on %s", orgName, appName)
	html = renderHTML(invitationHTMLTmpl, map[string]string{
		"InviterName": inviterName,
		"OrgName":     orgName,
		"AppName":     appName,
		"AcceptURL":   acceptURL,
	})
	text = fmt.Sprintf("You're invited!\n\n%s has invited you to join %s on %s.\n\nAccept: %s\n\n— The %s Team", inviterName, orgName, appName, acceptURL, appName)
	return
}
