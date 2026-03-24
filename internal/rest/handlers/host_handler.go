package handlers

import (
	restcontext "RedPaths-server/internal/rest/context"
	"RedPaths-server/internal/rest/requests"
	"RedPaths-server/pkg/model"
	engine2 "RedPaths-server/pkg/model/engine"
	"RedPaths-server/pkg/service/active_directory"
	"RedPaths-server/pkg/service/engine"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type HostHandler struct {
	hostService       *active_directory.HostService
	capabilityService *engine.CapabilityService
}

func NewHostHandler(hostService *active_directory.HostService, capabilityService *engine.CapabilityService) *HostHandler {
	return &HostHandler{
		hostService:       hostService,
		capabilityService: capabilityService,
	}
}

// CreateHost Function to create a standalone host without domain
func (h *HostHandler) CreateHost(c *gin.Context) {
	/*type CreateHostRequest struct {
		Ip string `json:"ip_address" binding:"required"`
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

	//projectUID, err := h.hostService.(
	//	c.Request.Context(),
	//	host,
	//	projectUid,
	//	"UserInput",
	//)

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
	})*/
}

func (h *HostHandler) UpdateHost(c *gin.Context) {

	host := restcontext.Host(c)
	var fieldsToUpdate map[string]interface{}

	if err := c.BindJSON(&fieldsToUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_JSON"})
		return
	}
	updatedHost, err := h.hostService.UpdateHost(c.Request.Context(), host.UID, "UserInput", fieldsToUpdate)

	if err != nil {
		log.Printf("Sending 500 response while updating host because: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to update host",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Host updated successfully",
		"updated_domain": updatedHost,
	})
}

func (h *HostHandler) AddService(c *gin.Context) {
	var request requests.AddServiceRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request to add a new service",
			"details": err.Error(),
		})
		return
	}

	uid := c.Param("hostUID")

	service := &model.Service{
		Name: request.Name,
	}

	createdService, err := h.hostService.AddService(
		c.Request.Context(),
		request.AssertionContext,
		uid,
		service,
		"user",
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add service into host",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        "success",
		"message":       "New service has been added to host",
		"added_service": createdService,
	})
}

func (h *HostHandler) AddCapability(c *gin.Context) {
	var request requests.AddCapabilityRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request to add a new capability to host",
			"details": err.Error(),
		})
		return
	}

	hostUid := c.Param("hostUID")
	projectUid := c.Param("projectUID")

	capability := engine2.Capability{Name: request.Name,
		Scope:     engine2.ScopeType(request.Scope),
		RiskLevel: request.RiskLevel}

	createdCapability, err := h.capabilityService.CreateAndLinkCapability(
		c.Request.Context(),
		request.AssertionContext,
		&capability,
		hostUid,
		"Host",
		projectUid,
		"user",
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create and link capability to host",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":           "success",
		"message":          "New capability has been added to host",
		"added_capability": createdCapability,
	})
}

func (h *HostHandler) GetLinkedCapabilities(c *gin.Context) {

	dirNodeUID := c.Param("hostUID")

	capabilities, err := h.hostService.GetCapabilities(
		c.Request.Context(),
		dirNodeUID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get capabilities from host",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, capabilities)

}
