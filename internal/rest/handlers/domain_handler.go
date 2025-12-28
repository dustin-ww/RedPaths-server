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
	domain := c.MustGet("domain").(*model.Domain)

	hosts, err := h.domainService.GetDomainHosts(
		c.Request.Context(),
		domain.UID,
	)

	project := restcontext.Project(c)
	domain := restcontext.Domain(c)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, hosts)
}

func (h *DomainHandler) AddHost(c *gin.Context) {
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

	domainUid := c.Param("domainUID")

	host := &model.Host{
		IP: request.Ip,
	}

	_, err := h.domainService.AddHost(
		c.Request.Context(),
		domainUid,
		host,
		"UserInput",
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add host",
			"details": err.Error(),
		})
		log.Printf("Sending client 500 error response for adding host to domain %s with message %s", domainUid, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message":      "New user has been created",
		"created_host": nil,
	})
}
