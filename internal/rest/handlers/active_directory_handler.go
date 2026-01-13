package handlers

import (
	active_directory2 "RedPaths-server/pkg/model/active_directory"
	"RedPaths-server/pkg/service/active_directory"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ActiveDirectoryHandler struct {
	activeDirectoryService *active_directory.ActiveDirectoryService
}

func NewActiveDirectoryHandler(activeDirectoryService *active_directory.ActiveDirectoryService) *ActiveDirectoryHandler {
	return &ActiveDirectoryHandler{
		activeDirectoryService: activeDirectoryService,
	}
}

func (h *ActiveDirectoryHandler) AddDomain(c *gin.Context) {
	type AddDomainRequest struct {
		Name string `json:"name" binding:"required" validate:"required"`
	}

	var request AddDomainRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	uid := c.Param("projectUID")

	domain := &active_directory2.Domain{
		Name: request.Name,
	}

	_, err := h.activeDirectoryService.AddDomain(
		c.Request.Context(),
		uid,
		domain,
		"UserInput",
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add domain",
			"details": err.Error(),
		})
		fmt.Println(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "Domain successfully added",
		"name":   request.Name,
	})
}

func (h *ActiveDirectoryHandler) GetDomains(c *gin.Context) {
	uid := c.Param("activeDirectoryUID")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Active Directory UID is required",
		})
		return
	}

	domains, err := h.activeDirectoryService.GetDomainsByActiveDirectory(
		c.Request.Context(),
		uid,
	)

	if err != nil {
		errReturn := gin.H{
			"error":   "Failed to retrieve domains",
			"details": err.Error(),
		}
		log.Println(err)
		c.JSON(http.StatusInternalServerError, errReturn)
	}

	log.Println(domains)

	c.JSON(http.StatusOK, domains)
}

func (h *ActiveDirectoryHandler) Get(c *gin.Context) {
	uid := c.Param("adUID")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Active Directory UID is required",
		})
		return
	}

	project, err := h.activeDirectoryService.Get(
		c.Request.Context(),
		uid,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve active directory",
			"details": err.Error(),
		})
		return
	}
	log.Println(*project)
	c.JSON(http.StatusOK, project)
}

func (h *ActiveDirectoryHandler) UpdateActiveDirectory(c *gin.Context) {
	//project := restcontext.Project(c)
	uid := c.Param("adUID")
	var fieldsToUpdate map[string]interface{}

	if err := c.BindJSON(&fieldsToUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_JSON"})
		return
	}

	updatedActiveDirectory, err := h.activeDirectoryService.UpdateActiveDirectory(c.Request.Context(), uid, "UserInput", fieldsToUpdate)

	if err != nil {
		log.Printf("Sending 500 response while updating active directory because: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to update active directory",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Active Directory updated successfully",
		"updated_project": updatedActiveDirectory,
	})
}
