package handlers

import (
	"RedPaths-server/pkg/service/active_directory"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ServiceHandler struct {
	serviceService *active_directory.ServiceService
}

func NewServiceHandler(serviceService *active_directory.ServiceService) *ServiceHandler {
	return &ServiceHandler{
		serviceService: serviceService,
	}
}

func (h *ServiceHandler) GetServices(c *gin.Context) {
	uid := c.Param("hostUID")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Host UID is required",
		})
		return
	}

	services, err := h.serviceService.GetHostServices(
		c.Request.Context(),
		uid,
	)

	if err != nil {
		errReturn := gin.H{
			"error":   "Failed to retrieve services for given host uid",
			"details": err.Error(),
		}
		log.Println(err)
		c.JSON(http.StatusInternalServerError, errReturn)
	}

	c.JSON(http.StatusOK, services)
}
