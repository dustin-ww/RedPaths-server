package handlers

import (
	restcontext "RedPaths-server/internal/rest/context"
	"RedPaths-server/pkg/model"
	"RedPaths-server/pkg/service/active_directory"
	"github.com/gin-gonic/gin"

	"log"
	"net/http"
)

type DomainHandler struct {
	projectService *active_directory.ProjectService
	domainService  *active_directory.DomainService
}

func NewDomainHandler(projectService *active_directory.ProjectService, domainService *active_directory.DomainService) *DomainHandler {
	return &DomainHandler{
		projectService: projectService,
		domainService:  domainService,
	}
}

/*
func (h *DomainHandler) GetHosts(c *gin.Context) {
	uid := c.Param("domainUID")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Domain UID is required",
		})
		return
	}

	hosts, err := h.domainService.GetDomainHosts(
		c.Request.Context(),
		uid,
	)

	if err != nil {
		errReturn := gin.H{
			"error":   "Failed to retrieve hosts for given domain uid",
			"details": err.Error(),
		}
		log.Println(err)
		c.JSON(http.StatusInternalServerError, errReturn)
	}

	c.JSON(http.StatusOK, hosts)
}*/

func (h *DomainHandler) GetHosts(c *gin.Context) {
	domain := restcontext.Domain(c)

	hosts, err := h.domainService.GetDomainHosts(
		c.Request.Context(),
		domain.UID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, hosts)
}

func (h *DomainHandler) UpdateDomain(c *gin.Context) {

	domain := restcontext.Domain(c)
	var fieldsToUpdate map[string]interface{}

	if err := c.BindJSON(&fieldsToUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_JSON"})
		return
	}
	updatedDomain, err := h.domainService.UpdateDomain(c.Request.Context(), domain.UID, "UserInput", fieldsToUpdate)

	if err != nil {
		log.Printf("Sending 500 response while updating domain because: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to update domain",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Domain updated successfully",
		"updated_domain": updatedDomain,
	})
}

func (h *DomainHandler) AddHost(c *gin.Context) {
	domain := restcontext.Domain(c)
	type AddHostRequest struct {
		Ip string `json:"ipAddress" binding:"required"`
	}

	var request AddHostRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data to create a new host",
			"details": err.Error(),
		})
		return
	}

	host := &model.Host{
		IP: request.Ip,
	}

	_, err := h.domainService.AddHost(
		c.Request.Context(),
		domain.UID,
		host,
		"UserInput",
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add host",
			"details": err.Error(),
		})
		log.Printf("Sending client 500 error response for adding host to domain %s with message %s", domain.UID, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message":      "New user has been created",
		"created_host": nil,
	})
}
