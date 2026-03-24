package handlers

import (
	"RedPaths-server/internal/rest/requests"
	rpad "RedPaths-server/pkg/model/active_directory"
	"RedPaths-server/pkg/service/active_directory"
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

func (h *ActiveDirectoryHandler) AddActiveDirectoryDomain(c *gin.Context) {
	var request requests.AddDomainRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request to add a new domain",
			"details": err.Error(),
		})
		return
	}

	uid := c.Param("adUID")

	domain := &rpad.Domain{
		Name: request.Name,
	}

	createdDomain, err := h.activeDirectoryService.AddDomain(
		c.Request.Context(),
		uid,
		domain,
		request.AssertionContext,
		"user",
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add domain into active directory container",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "success",
		"message":      "New domain has been added to active directory",
		"added_domain": createdDomain,
	})
}

func (h *ActiveDirectoryHandler) GetActiveDirectoryDomains(c *gin.Context) {
	uid := c.Param("adUID")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Active Directory UID is required",
		})
		return
	}

	domains, err := h.activeDirectoryService.GetAllDomains(
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

func (h *ActiveDirectoryHandler) GetProjectActiveDirectory(c *gin.Context) {
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

func (h *ActiveDirectoryHandler) UpdateProjectActiveDirectory(c *gin.Context) {
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
