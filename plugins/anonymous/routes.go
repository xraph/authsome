package anonymous

import "github.com/xraph/forge"

// Register registers anonymous routes under basePath
func Register(app *forge.App, basePath string, h *Handler) {
	grp := app.Group(basePath)
	grp.POST("/anonymous/signin", h.SignIn)
	grp.POST("/anonymous/link", h.Link)
}
