package handlers

import (
	"RedPaths-server/pkg/model"
	"RedPaths-server/pkg/service/active_directory"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
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
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add host",
			"details": err.Error(),
		})
		fmt.Println(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "host successfully added",
		"ip":     request.Ip,
	})
}
