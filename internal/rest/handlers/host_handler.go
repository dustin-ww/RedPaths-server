package handlers

import (
	"RedPaths-server/pkg/model"
	"RedPaths-server/pkg/service/active_directory"
	"net/http"

	"github.com/gin-gonic/gin"
)

type HostHandler struct {
	hostService *active_directory.HostService
}

func NewHostHandler(hostService *active_directory.HostService) *HostHandler {
	return &HostHandler{
		hostService: hostService,
	}
}

// CreateHost Function to create a standalone host without domain
func (h *HostHandler) CreateHost(c *gin.Context) {
	type CreateHostRequest struct {
		Ip string `json:"ipAddress" binding:"required"`
	}

	var request CreateHostRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request to create a new host",
			"details": err.Error(),
		})
		return
	}

	host := &model.Host{
		IP: request.Ip,
	}

	projectUid := c.Param("projectUID")

	projectUID, err := h.hostService.CreateWithUnknownDomain(
		c.Request.Context(),
		host,
		projectUid,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create a new host",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"uid":     projectUID,
		"message": "New host has been created",
	})
}
