package pages

import (
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/icons"
	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// UserEditPage allows editing user information.
func (p *PagesManager) UserEditPage(ctx *router.PageContext) (g.Node, error) {
	appID := ctx.Param("appId")
	userID := ctx.Param("userId")

	return primitives.Container(
		Div(
			Class("space-y-2"),
			g.Attr("x-data", `{
				user: null,
				loading: true,
				error: null,
				saving: false,
				saveError: '',
				saveSuccess: '',
				form: {
					email: '',
					name: '',
					image: '',
					emailVerified: false
				},
				password: {
					newPassword: '',
					confirmPassword: '',
					updating: false,
					error: '',
					success: ''
				},
				roles: {
					available: [],
					assigned: [],
					selected: [],
					loading: false,
					saving: false,
					error: '',
					success: ''
				},
				async loadUser() {
					this.loading = true;
					this.error = null;
					try {
						const result = await $go('getUserDetail', {
							userId: '`+userID+`',
							appId: '`+appID+`'
						});
						this.user = result;
						this.form.email = result.email || '';
						this.form.name = result.name || '';
						this.form.image = result.image || '';
						this.form.emailVerified = result.emailVerified || false;
					} catch (err) {
						console.error('Failed to load user:', err);
						this.error = err.message || 'Failed to load user';
					} finally {
						this.loading = false;
					}
				},
				async loadRoles() {
					this.roles.loading = true;
					try {
						const [availableResult, assignedResult] = await Promise.all([
							$go('listRoles', { appId: '`+appID+`' }),
							$go('getUserRoles', { userId: '`+userID+`', appId: '`+appID+`' })
						]);
						this.roles.available = availableResult.roles || [];
						this.roles.assigned = assignedResult.roles || [];
						this.roles.selected = this.roles.assigned.map(r => r.id);
					} catch (err) {
						console.error('Failed to load roles:', err);
						this.roles.error = err.message || 'Failed to load roles';
					} finally {
						this.roles.loading = false;
					}
				},
				async save() {
					this.saving = true;
					this.saveError = '';
					this.saveSuccess = '';
					try {
						await $go('updateUser', {
							userId: '`+userID+`',
							appId: '`+appID+`',
							name: this.form.name || null,
							email: this.form.email || null,
							emailVerified: this.form.emailVerified,
							image: this.form.image || null
						});
						this.saveSuccess = 'User updated successfully';
						setTimeout(() => this.saveSuccess = '', 3000);
					} catch (err) {
						this.saveError = err.message || 'Failed to update user';
					} finally {
						this.saving = false;
					}
				},
				async updatePassword() {
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
						await $go('updateUserPassword', {
							userId: '`+userID+`',
							appId: '`+appID+`',
							newPassword: this.password.newPassword
						});
						this.password.success = 'Password updated successfully';
						this.password.newPassword = '';
						this.password.confirmPassword = '';
						setTimeout(() => this.password.success = '', 3000);
					} catch (err) {
						this.password.error = err.message || 'Failed to update password';
					} finally {
						this.password.updating = false;
					}
				},
				toggleRole(roleId) {
					const index = this.roles.selected.indexOf(roleId);
					if (index > -1) {
						this.roles.selected.splice(index, 1);
					} else {
						this.roles.selected.push(roleId);
					}
				},
				isRoleSelected(roleId) {
					return this.roles.selected.includes(roleId);
				},
				async saveRoles() {
					this.roles.saving = true;
					this.roles.error = '';
					this.roles.success = '';
					try {
						await $go('updateUserRoles', {
							userId: '`+userID+`',
							appId: '`+appID+`',
							roleIds: this.roles.selected
						});
						this.roles.success = 'Roles updated successfully';
						setTimeout(() => this.roles.success = '', 3000);
					} catch (err) {
						this.roles.error = err.message || 'Failed to update roles';
					} finally {
						this.roles.saving = false;
					}
				},
				async deleteUser() {
					if (!confirm('Are you sure you want to delete this user? This action cannot be undone.')) {
						return;
					}
					try {
						await $go('deleteUser', { userId: '`+userID+`' });
						alert('User deleted successfully');
						window.location.href = '/api/identity/ui/app/`+appID+`/users';
					} catch (err) {
						alert(err.message || 'Failed to delete user');
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
				}
			}`),
			g.Attr("x-init", "loadUser(); loadRoles()"),

			// Header
			Div(
				Class("flex items-center justify-between"),
				Div(
					button.Button(
						Div(
							Class("flex items-center gap-2"),
							icons.ArrowLeft(icons.WithSize(16)),
							Span(g.Text("Back to Users")),
						),
						button.WithVariant("ghost"),
						button.WithSize("sm"),
						button.WithAttrs(g.Attr("@click", "window.location.href = '/api/identity/ui/app/"+appID+"/users'")),
					),
				),
				H1(Class("text-3xl font-bold text-slate-900 dark:text-white"), g.Text("Edit User")),
			),

			// Loading state
			Div(
				g.Attr("x-show", "loading"),
				Class("flex items-center justify-center py-16"),
				Div(
					Class("text-center"),
					Div(Class("animate-spin rounded-full h-12 w-12 border-b-2 border-violet-600 mx-auto mb-4")),
					P(Class("text-sm text-slate-600 dark:text-gray-400"), g.Text("Loading user...")),
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
							H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-2"), g.Text("User Not Found")),
							P(Class("text-sm text-slate-600 dark:text-gray-400"), g.Attr("x-text", "error")),
						),
					),
				),
			),

			// Edit form content
			Div(
				g.Attr("x-show", "!loading && !error && user"),
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

				// Personal Information Card
				card.Card(
					card.Header(
						card.Title("Personal Information"),
						card.Description("Update user's basic information"),
					),
					card.Content(
						Div(
							Class("space-y-4"),

							// Email
							Div(
								Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"), g.Text("Email")),
								Input(
									Type("email"),
									g.Attr("x-model", "form.email"),
									Class("w-full rounded-md border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"),
									Placeholder("user@example.com"),
								),
							),

							// Name
							Div(
								Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"), g.Text("Name")),
								Input(
									Type("text"),
									g.Attr("x-model", "form.name"),
									Class("w-full rounded-md border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"),
									Placeholder("John Doe"),
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

							// Email Verified Checkbox
							Div(
								Class("flex items-center gap-2"),
								Input(
									Type("checkbox"),
									g.Attr("x-model", "form.emailVerified"),
									ID("emailVerified"),
									Class("rounded border-gray-300 text-blue-600 focus:ring-blue-500"),
								),
								Label(
									For("emailVerified"),
									Class("text-sm font-medium text-gray-700 dark:text-gray-300 cursor-pointer"),
									g.Text("Email Verified"),
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
									g.Attr("@click", "save()"),
									g.Attr(":disabled", "saving"),
								),
							),
						),
					),
				),

				// Password Update Card
				card.Card(
					card.Header(
						card.Title("Password"),
						card.Description("Update user's password"),
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
								Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"), g.Text("Confirm Password")),
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
									Span(g.Attr("x-text", "password.updating ? 'Updating...' : 'Update Password'")),
								),
								button.WithAttrs(
									g.Attr("@click", "updatePassword()"),
									g.Attr(":disabled", "password.updating"),
								),
							),
						),
					),
				),

				// Roles & Permissions Card
				card.Card(
					card.Header(
						card.Title("Roles & Permissions"),
						card.Description("Manage user's roles"),
					),
					card.Content(
						Div(
							Class("space-y-4"),

							// Success message
							Div(
								g.Attr("x-show", "roles.success"),
								Class("p-3 rounded-lg bg-green-50 dark:bg-green-900/20 text-green-600 dark:text-green-400 text-sm flex items-center gap-2"),
								icons.CheckCircle(icons.WithSize(16)),
								Span(g.Attr("x-text", "roles.success")),
							),

							// Error message
							Div(
								g.Attr("x-show", "roles.error"),
								Class("p-3 rounded-lg bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 text-sm flex items-center gap-2"),
								icons.AlertCircle(icons.WithSize(16)),
								Span(g.Attr("x-text", "roles.error")),
							),

							// Loading roles
							Div(
								g.Attr("x-show", "roles.loading"),
								Class("text-center py-8"),
								Div(Class("animate-spin rounded-full h-8 w-8 border-b-2 border-violet-600 mx-auto mb-2")),
								P(Class("text-sm text-slate-600 dark:text-gray-400"), g.Text("Loading roles...")),
							),

							// Roles list
							Div(
								g.Attr("x-show", "!roles.loading"),
								Class("space-y-2"),
								g.El("template", g.Attr("x-for", "role in roles.available"),
									Div(
										Class("flex items-start gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800/50"),
										Input(
											Type("checkbox"),
											g.Attr(":checked", "isRoleSelected(role.id)"),
											g.Attr("@change", "toggleRole(role.id)"),
											Class("mt-1 rounded border-gray-300 text-blue-600 focus:ring-blue-500"),
										),
										Div(
											Class("flex-1"),
											Label(
												Class("text-sm font-medium text-gray-900 dark:text-white cursor-pointer"),
												g.Attr("x-text", "role.displayName || role.name"),
											),
											P(
												Class("text-xs text-gray-500 dark:text-gray-400 mt-0.5"),
												g.Attr("x-text", "role.description || ''"),
											),
										),
									),
								),
							),

							// No roles available
							Div(
								g.Attr("x-show", "!roles.loading && roles.available.length === 0"),
								Class("text-center py-8 text-gray-500 dark:text-gray-400"),
								P(g.Text("No roles available")),
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
										g.Attr("x-show", "!roles.saving"),
										icons.Save(icons.WithSize(16)),
									),
									g.El("span",
										g.Attr("x-show", "roles.saving"),
										Class("animate-spin"),
										icons.Loader(icons.WithSize(16)),
									),
									Span(g.Attr("x-text", "roles.saving ? 'Saving...' : 'Save Roles'")),
								),
								button.WithAttrs(
									g.Attr("@click", "saveRoles()"),
									g.Attr(":disabled", "roles.saving"),
								),
							),
						),
					),
				),

				// Danger Zone Card
				card.Card(
					card.Header(
						card.Title("Danger Zone"),
						card.Description("Irreversible actions"),
					),
					card.Content(
						Div(
							Class("flex items-center justify-between p-4 border border-red-200 dark:border-red-900 rounded-lg bg-red-50 dark:bg-red-900/10"),
							Div(
								H4(Class("text-sm font-semibold text-red-900 dark:text-red-400"), g.Text("Delete User")),
								P(Class("text-xs text-red-700 dark:text-red-500 mt-1"), g.Text("Once deleted, this user cannot be recovered")),
							),
							button.Button(
								Div(
									Class("flex items-center gap-2"),
									icons.Trash(icons.WithSize(16)),
									Span(g.Text("Delete User")),
								),
								button.WithVariant("destructive"),
								button.WithAttrs(g.Attr("@click", "deleteUser()")),
							),
						),
					),
				),
			),
		),
	), nil
}
