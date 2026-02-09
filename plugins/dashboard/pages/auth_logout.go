package pages

import (
	"net/http"

	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// LogoutPage handles user logout.
func (p *PagesManager) LogoutPage(ctx *router.PageContext) (g.Node, error) {
	// Clear session cookie
	http.SetCookie(ctx.ResponseWriter, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	// Redirect to login
	ctx.SetHeader("Location", p.baseUIPath+"/auth/login")
	ctx.ResponseWriter.WriteHeader(http.StatusFound)

	return primitives.Container(
		primitives.Box(
			primitives.WithChildren(
				Div(
					Class("flex items-center justify-center min-h-screen"),
					P(g.Text("Logging out...")),
				),
			),
		),
	), nil
}
