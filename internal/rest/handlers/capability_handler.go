package handlers

import (
	"RedPaths-server/pkg/service/engine"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CapabilityHandler struct {
	capabilityService *engine.CapabilityService
}

func NewCapabilityHandler(capabilityService *engine.CapabilityService) *CapabilityHandler {
	return &CapabilityHandler{
		capabilityService: capabilityService,
	}
}

// GetCatalogUsers retrieves all users for a project.
func (h *CapabilityHandler) GetCatalogCapabilities(c *gin.Context) {
	uid := c.Param("projectUID")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Project UID is required",
		})
		return
	}

	domains, err := h.capabilityService.GetCapabilitiesFromCatalog(c.Request.Context(), uid)

	if err != nil {
		errReturn := gin.H{
			"error":   "Failed to retrieve users",
			"details": err.Error(),
		}
		c.JSON(http.StatusInternalServerError, errReturn)
		return
	}

	c.JSON(http.StatusOK, domains)
}
