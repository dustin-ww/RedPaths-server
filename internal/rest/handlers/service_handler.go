package handlers

import (
	restcontext "RedPaths-server/internal/rest/context"
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

func (h *ServiceHandler) UpdateService(c *gin.Context) {

	service := restcontext.Service(c)
	var fieldsToUpdate map[string]interface{}

	if err := c.BindJSON(&fieldsToUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_JSON"})
		return
	}
	updatedService, err := h.serviceService.UpdateService(c.Request.Context(), service.UID, "UserInput", fieldsToUpdate)

	if err != nil {
		log.Printf("Sending 500 response while updating service because: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to update service",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Service updated successfully",
		"updated_domain": updatedService,
	})
}
