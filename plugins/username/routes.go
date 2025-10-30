package username

import "github.com/xraph/forge"

// Register registers username routes under the base auth group
func Register(app *forge.App, basePath string, h *Handler) {
	grp := app.Group(basePath)
	grp.POST("/username/signup", h.SignUp)
	grp.POST("/username/signin", h.SignIn)
}
