package handlers

import (
	"RedPaths-server/pkg/service/active_directory"

	"github.com/gin-gonic/gin"
)

type GPOHandler struct {
	gpoService *active_directory.GPOService
}

func NewPGOHandler(gpoService *active_directory.GPOService) *GPOHandler {
	return &GPOHandler{
		gpoService: gpoService,
	}
}

func (h *GPOHandler) GetGPOSettings(c *gin.Context) {
	//domain := restcontext.Domain(c)
	panic("implement me")
}
