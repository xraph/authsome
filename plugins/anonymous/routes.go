package anonymous

import "github.com/xraph/forge"

// Register registers anonymous routes under basePath
func Register(router forge.Router, basePath string, h *Handler) {
	grp := router.Group(basePath)
	grp.POST("/anonymous/signin", h.SignIn)
	grp.POST("/anonymous/link", h.Link)
}
