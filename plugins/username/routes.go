package username

import "github.com/xraph/forge"

// Register registers username routes under the base auth group
func Register(router forge.Router, basePath string, h *Handler) {
	grp := router.Group(basePath)
	grp.POST("/username/signup", h.SignUp)
	grp.POST("/username/signin", h.SignIn)
}
