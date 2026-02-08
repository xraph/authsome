package pages

import (
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/icons"
	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ProfilePage shows the current user's profile and allows editing
func (p *PagesManager) ProfilePage(ctx *router.PageContext) (g.Node, error) {
	// Get current user from context
	var currentUser *user.User
	if userRaw, ok := ctx.Get("user"); ok {
		currentUser, _ = userRaw.(*user.User)
	}

	// Get user ID for bridge calls
	userID := ""
	userName := "User"
	userEmail := ""
	if currentUser != nil {
		userID = currentUser.ID.String()
		if currentUser.Name != "" {
			userName = currentUser.Name
		} else {
			userName = currentUser.Email
		}
		userEmail = currentUser.Email
	}

	return primitives.Container(
		Div(
			Class("space-y-6"),
			g.Attr("x-data", `{
				profile: {
					id: '`+userID+`',
					email: '`+userEmail+`',
					name: '`+userName+`',
					image: '',
					emailVerified: false
				},
				loading: true,
				error: null,
				saving: false,
				saveError: '',
				saveSuccess: '',
				form: {
					email: '',
					name: '',
					image: ''
				},
				password: {
					currentPassword: '',
					newPassword: '',
					confirmPassword: '',
					updating: false,
					error: '',
					success: ''
				},
				sessions: {
					list: [],
					loading: false,
					error: '',
					revoking: null
				},
				async loadProfile() {
					this.loading = true;
					this.error = null;
					try {
						const result = await $go('getCurrentUserProfile', {});
						this.profile = result;
						this.form.email = result.email || '';
						this.form.name = result.name || '';
						this.form.image = result.image || '';
					} catch (err) {
						console.error('Failed to load profile:', err);
						this.error = err.message || 'Failed to load profile';
					} finally {
						this.loading = false;
					}
				},
				async loadSessions() {
					this.sessions.loading = true;
					this.sessions.error = '';
					try {
						const result = await $go('getCurrentUserSessions', {});
						this.sessions.list = result.sessions || [];
					} catch (err) {
						console.error('Failed to load sessions:', err);
						this.sessions.error = err.message || 'Failed to load sessions';
					} finally {
						this.sessions.loading = false;
					}
				},
				async saveProfile() {
					this.saving = true;
					this.saveError = '';
					this.saveSuccess = '';
					try {
						await $go('updateCurrentUserProfile', {
							name: this.form.name || null,
							image: this.form.image || null
						});
						this.saveSuccess = 'Profile updated successfully';
						this.profile.name = this.form.name;
						this.profile.image = this.form.image;
						setTimeout(() => this.saveSuccess = '', 3000);
					} catch (err) {
						this.saveError = err.message || 'Failed to update profile';
					} finally {
						this.saving = false;
					}
				},
				async changePassword() {
					if (!this.password.currentPassword) {
						this.password.error = 'Please enter your current password';
						return;
					}
					if (!this.password.newPassword) {
						this.password.error = 'Please enter a new password';
						return;
					}
					if (this.password.newPassword !== this.password.confirmPassword) {
						this.password.error = 'Passwords do not match';
						return;
					}
					if (this.password.newPassword.length < 8) {
						this.password.error = 'Password must be at least 8 characters';
						return;
					}
					
					this.password.updating = true;
					this.password.error = '';
					this.password.success = '';
					try {
						await $go('changeCurrentUserPassword', {
							currentPassword: this.password.currentPassword,
							newPassword: this.password.newPassword
						});
						this.password.success = 'Password changed successfully';
						this.password.currentPassword = '';
						this.password.newPassword = '';
						this.password.confirmPassword = '';
						setTimeout(() => this.password.success = '', 3000);
					} catch (err) {
						this.password.error = err.message || 'Failed to change password';
					} finally {
						this.password.updating = false;
					}
				},
				async revokeSession(sessionId) {
					if (!confirm('Are you sure you want to revoke this session?')) {
						return;
					}
					this.sessions.revoking = sessionId;
					try {
						await $go('revokeCurrentUserSession', { sessionId });
						await this.loadSessions();
					} catch (err) {
						alert(err.message || 'Failed to revoke session');
					} finally {
						this.sessions.revoking = null;
					}
				},
				get passwordStrength() {
					const pwd = this.password.newPassword;
					if (!pwd) return { score: 0, label: '', color: '' };
					let score = 0;
					if (pwd.length >= 8) score++;
					if (pwd.length >= 12) score++;
					if (/[a-z]/.test(pwd) && /[A-Z]/.test(pwd)) score++;
					if (/[0-9]/.test(pwd)) score++;
					if (/[^a-zA-Z0-9]/.test(pwd)) score++;
					
					if (score <= 2) return { score, label: 'Weak', color: 'bg-red-500' };
					if (score <= 3) return { score, label: 'Fair', color: 'bg-yellow-500' };
					if (score <= 4) return { score, label: 'Good', color: 'bg-blue-500' };
					return { score, label: 'Strong', color: 'bg-green-500' };
				},
				formatDate(dateStr) {
					if (!dateStr) return 'Unknown';
					return new Date(dateStr).toLocaleString();
				},
				isCurrentSession(sessionId) {
					// Check if this session matches the current session cookie
					return false; // Will be determined by comparing with current session
				}
			}`),
			g.Attr("x-init", "loadProfile(); loadSessions()"),

			// Header
			Div(
				Class("flex items-center justify-between"),
				Div(
					H1(Class("text-3xl font-bold tracking-tight text-slate-900 dark:text-white"), g.Text("My Profile")),
					P(Class("text-sm text-slate-600 dark:text-gray-400 mt-1"), g.Text("Manage your account settings and preferences")),
				),
			),

			// Loading state
			Div(
				g.Attr("x-show", "loading"),
				Class("flex items-center justify-center py-16"),
				Div(
					Class("text-center"),
					Div(Class("animate-spin rounded-full h-12 w-12 border-b-2 border-violet-600 mx-auto mb-4")),
					P(Class("text-sm text-slate-600 dark:text-gray-400"), g.Text("Loading profile...")),
				),
			),

			// Error state
			Div(
				g.Attr("x-show", "!loading && error"),
				card.Card(
					card.Content(
						Div(
							Class("text-center py-12"),
							Div(
								Class("rounded-full bg-red-100 dark:bg-red-900/20 p-6 w-20 h-20 mx-auto mb-4 flex items-center justify-center"),
								icons.AlertCircle(icons.WithSize(32), icons.WithClass("text-red-600 dark:text-red-400")),
							),
							H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-2"), g.Text("Failed to Load Profile")),
							P(Class("text-sm text-slate-600 dark:text-gray-400"), g.Attr("x-text", "error")),
						),
					),
				),
			),

			// Profile content
			Div(
				g.Attr("x-show", "!loading && !error"),
				Class("space-y-6"),

				// Success message
				Div(
					g.Attr("x-show", "saveSuccess"),
					Class("p-4 rounded-lg bg-green-50 dark:bg-green-900/20 text-green-600 dark:text-green-400 flex items-center gap-2"),
					icons.CheckCircle(icons.WithSize(20)),
					Span(g.Attr("x-text", "saveSuccess")),
				),

				// Error message
				Div(
					g.Attr("x-show", "saveError"),
					Class("p-4 rounded-lg bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 flex items-center gap-2"),
					icons.AlertCircle(icons.WithSize(20)),
					Span(g.Attr("x-text", "saveError")),
				),

				// Profile Header Card
				card.Card(
					card.Content(
						Div(
							Class("flex items-center gap-6 py-2"),
							// Avatar
							Div(
								Class("relative"),
								Div(
									Class("flex-shrink-0 w-24 h-24 rounded-full bg-gradient-to-br from-violet-500 to-purple-600 flex items-center justify-center text-white font-bold text-3xl"),
									g.Attr("x-text", "profile.name ? profile.name.charAt(0).toUpperCase() : (profile.email ? profile.email.charAt(0).toUpperCase() : '?')"),
								),
								// Verified badge
								Div(
									g.Attr("x-show", "profile.emailVerified"),
									Class("absolute -bottom-1 -right-1 w-8 h-8 rounded-full bg-emerald-500 flex items-center justify-center border-4 border-white dark:border-gray-900"),
									icons.Check(icons.WithSize(16), icons.WithClass("text-white")),
								),
							),
							// User info
							Div(
								Class("flex-1"),
								H2(
									Class("text-2xl font-bold text-slate-900 dark:text-white"),
									g.Attr("x-text", "profile.name || profile.email"),
								),
								P(
									Class("text-sm text-slate-600 dark:text-gray-400 mt-1"),
									g.Attr("x-text", "profile.email"),
								),
								// Status badges
								Div(
									Class("flex items-center gap-2 mt-3"),
									Span(
										g.Attr("x-show", "profile.emailVerified"),
										Class("inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-emerald-100 dark:bg-emerald-500/20 text-emerald-700 dark:text-emerald-400"),
										Span(Class("w-1.5 h-1.5 rounded-full bg-emerald-500")),
										Span(g.Text("Email Verified")),
									),
									Span(
										g.Attr("x-show", "!profile.emailVerified"),
										Class("inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-amber-100 dark:bg-amber-500/20 text-amber-700 dark:text-amber-400"),
										Span(Class("w-1.5 h-1.5 rounded-full bg-amber-500")),
										Span(g.Text("Email Not Verified")),
									),
								),
							),
						),
					),
				),

				// Personal Information Card
				card.Card(
					card.Header(
						card.Title("Personal Information"),
						card.Description("Update your profile information"),
					),
					card.Content(
						Div(
							Class("space-y-4"),

							// Email (read-only)
							Div(
								Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"), g.Text("Email")),
								Input(
									Type("email"),
									g.Attr("x-model", "profile.email"),
									Disabled(),
									Class("w-full rounded-md border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 px-3 py-2 text-sm text-gray-500 dark:text-gray-400 cursor-not-allowed"),
								),
								P(Class("text-xs text-gray-500 dark:text-gray-400 mt-1"), g.Text("Email cannot be changed from this page")),
							),

							// Name
							Div(
								Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"), g.Text("Display Name")),
								Input(
									Type("text"),
									g.Attr("x-model", "form.name"),
									Class("w-full rounded-md border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"),
									Placeholder("Enter your name"),
								),
							),

							// Image URL
							Div(
								Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"), g.Text("Profile Image URL")),
								Input(
									Type("url"),
									g.Attr("x-model", "form.image"),
									Class("w-full rounded-md border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"),
									Placeholder("https://example.com/avatar.jpg"),
								),
							),
						),
					),
					card.Footer(
						Div(
							Class("flex justify-end"),
							button.Button(
								Div(
									Class("flex items-center gap-2"),
									g.El("span",
										g.Attr("x-show", "!saving"),
										icons.Save(icons.WithSize(16)),
									),
									g.El("span",
										g.Attr("x-show", "saving"),
										Class("animate-spin"),
										icons.Loader(icons.WithSize(16)),
									),
									Span(g.Attr("x-text", "saving ? 'Saving...' : 'Save Changes'")),
								),
								button.WithAttrs(
									g.Attr("@click", "saveProfile()"),
									g.Attr(":disabled", "saving"),
								),
							),
						),
					),
				),

				// Password Update Card
				card.Card(
					card.Header(
						card.Title("Change Password"),
						card.Description("Update your account password"),
					),
					card.Content(
						Div(
							Class("space-y-4"),

							// Success message
							Div(
								g.Attr("x-show", "password.success"),
								Class("p-3 rounded-lg bg-green-50 dark:bg-green-900/20 text-green-600 dark:text-green-400 text-sm flex items-center gap-2"),
								icons.CheckCircle(icons.WithSize(16)),
								Span(g.Attr("x-text", "password.success")),
							),

							// Error message
							Div(
								g.Attr("x-show", "password.error"),
								Class("p-3 rounded-lg bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 text-sm flex items-center gap-2"),
								icons.AlertCircle(icons.WithSize(16)),
								Span(g.Attr("x-text", "password.error")),
							),

							// Current Password
							Div(
								Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"), g.Text("Current Password")),
								Input(
									Type("password"),
									g.Attr("x-model", "password.currentPassword"),
									Class("w-full rounded-md border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"),
									Placeholder("Enter current password"),
								),
							),

							// New Password
							Div(
								Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"), g.Text("New Password")),
								Input(
									Type("password"),
									g.Attr("x-model", "password.newPassword"),
									Class("w-full rounded-md border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"),
									Placeholder("Enter new password"),
								),
								// Password strength indicator
								Div(
									g.Attr("x-show", "password.newPassword"),
									Class("mt-2"),
									Div(
										Class("flex items-center gap-2 mb-1"),
										Div(
											Class("flex-1 h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden"),
											Div(
												g.Attr(":class", "passwordStrength.color"),
												g.Attr(":style", "`width: ${(passwordStrength.score / 5) * 100}%`"),
												Class("h-full transition-all duration-300"),
											),
										),
										Span(
											g.Attr("x-text", "passwordStrength.label"),
											Class("text-xs font-medium text-gray-600 dark:text-gray-400"),
										),
									),
									P(Class("text-xs text-gray-500 dark:text-gray-400"), g.Text("Use at least 8 characters with mixed case, numbers, and symbols")),
								),
							),

							// Confirm Password
							Div(
								Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"), g.Text("Confirm New Password")),
								Input(
									Type("password"),
									g.Attr("x-model", "password.confirmPassword"),
									Class("w-full rounded-md border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"),
									Placeholder("Confirm new password"),
								),
							),
						),
					),
					card.Footer(
						Div(
							Class("flex justify-end"),
							button.Button(
								Div(
									Class("flex items-center gap-2"),
									g.El("span",
										g.Attr("x-show", "!password.updating"),
										icons.Key(icons.WithSize(16)),
									),
									g.El("span",
										g.Attr("x-show", "password.updating"),
										Class("animate-spin"),
										icons.Loader(icons.WithSize(16)),
									),
									Span(g.Attr("x-text", "password.updating ? 'Changing...' : 'Change Password'")),
								),
								button.WithAttrs(
									g.Attr("@click", "changePassword()"),
									g.Attr(":disabled", "password.updating"),
								),
							),
						),
					),
				),

				// Active Sessions Card
				card.Card(
					card.Header(
						card.Title("Active Sessions"),
						card.Description("Manage your active sessions across devices"),
					),
					card.Content(
						Div(
							Class("space-y-4"),

							// Loading sessions
							Div(
								g.Attr("x-show", "sessions.loading"),
								Class("text-center py-8"),
								Div(Class("animate-spin rounded-full h-8 w-8 border-b-2 border-violet-600 mx-auto mb-2")),
								P(Class("text-sm text-slate-600 dark:text-gray-400"), g.Text("Loading sessions...")),
							),

							// Sessions error
							Div(
								g.Attr("x-show", "!sessions.loading && sessions.error"),
								Class("p-3 rounded-lg bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 text-sm"),
								Span(g.Attr("x-text", "sessions.error")),
							),

							// Sessions list
							Div(
								g.Attr("x-show", "!sessions.loading && !sessions.error"),
								Class("space-y-3"),

								// Empty state
								Div(
									g.Attr("x-show", "sessions.list.length === 0"),
									Class("text-center py-8 text-gray-500 dark:text-gray-400"),
									icons.Monitor(icons.WithSize(48), icons.WithClass("mx-auto mb-3 opacity-50")),
									P(g.Text("No active sessions found")),
								),

								// Session items
								g.El("template", g.Attr("x-for", "session in sessions.list"), g.Attr(":key", "session.id"),
									Div(
										Class("flex items-center justify-between p-4 rounded-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors"),
										Div(
											Class("flex items-center gap-4"),
											// Device icon
											Div(
												Class("flex-shrink-0 w-10 h-10 rounded-full bg-gray-100 dark:bg-gray-800 flex items-center justify-center"),
												icons.Monitor(icons.WithSize(20), icons.WithClass("text-gray-600 dark:text-gray-400")),
											),
											// Session info
											Div(
												Div(
													Class("flex items-center gap-2"),
													Span(
														Class("text-sm font-medium text-slate-900 dark:text-white"),
														g.Attr("x-text", "session.userAgent || 'Unknown Device'"),
													),
													Span(
														g.Attr("x-show", "session.isCurrent"),
														Class("px-2 py-0.5 text-xs font-medium rounded-full bg-green-100 dark:bg-green-500/20 text-green-700 dark:text-green-400"),
														g.Text("Current"),
													),
												),
												Div(
													Class("text-xs text-gray-500 dark:text-gray-400 mt-1"),
													Span(g.Attr("x-text", "session.ipAddress || 'Unknown IP'")),
													Span(g.Text(" • ")),
													Span(g.Attr("x-text", "'Last active: ' + formatDate(session.lastActiveAt || session.createdAt)")),
												),
											),
										),
										// Revoke button
										Div(
											g.Attr("x-show", "!session.isCurrent"),
											button.Button(
												Div(
													Class("flex items-center gap-1.5"),
													g.El("span",
														g.Attr("x-show", "sessions.revoking !== session.id"),
														icons.X(icons.WithSize(14)),
													),
													g.El("span",
														g.Attr("x-show", "sessions.revoking === session.id"),
														Class("animate-spin"),
														icons.Loader(icons.WithSize(14)),
													),
													Span(g.Text("Revoke")),
												),
												button.WithVariant("outline"),
												button.WithSize("sm"),
												button.WithAttrs(
													g.Attr("@click", "revokeSession(session.id)"),
													g.Attr(":disabled", "sessions.revoking === session.id"),
												),
											),
										),
									),
								),
							),
						),
					),
				),
			),
		),
	), nil
}
