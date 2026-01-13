package handlers

import (
	restcontext "RedPaths-server/internal/rest/context"
	rpadmodel "RedPaths-server/pkg/model/active_directory"
	"RedPaths-server/pkg/service/active_directory"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ProjectHandler struct {
	projectService *active_directory.ProjectService
}

// NewProjectHandler creates a new ProjectHandler with the given ProjectService.
func NewProjectHandler(projectService *active_directory.ProjectService) *ProjectHandler {
	return &ProjectHandler{
		projectService: projectService,
	}
}

// GetProjectOverviews returns overviews of all projects.
func (h *ProjectHandler) GetProjectOverviews(c *gin.Context) {
	projectsOverviews, err := h.projectService.GetOverviewForAll(c.Request.Context())

	if err != nil {
		errReturn := gin.H{
			"error":   "Failed to get overview items for all projects",
			"details": err.Error(),
		}

		c.JSON(http.StatusInternalServerError, errReturn)
		return
	}
	c.JSON(http.StatusOK, projectsOverviews)
}

// Get returns a single project by its UID.
func (h *ProjectHandler) Get(c *gin.Context) {
	uid := c.Param("projectUID")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Project UID is required",
		})
		return
	}

	project, err := h.projectService.Get(c.Request.Context(), uid)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve project",
			"details": err.Error(),
		})
		return
	}
	log.Println(*project)
	c.JSON(http.StatusOK, project)
}

// Delete removes a project by its UID.
func (h *ProjectHandler) Delete(c *gin.Context) {
	uid := c.Param("projectUID")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Project UID is required",
		})
		return
	}

	err := h.projectService.DeleteProject(c.Request.Context(), uid)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete project",
			"details": err.Error(),
		})
		return
	}
	c.Status(http.StatusOK)
}

// AddDomainWithHosts is a placeholder for adding a domain with hosts to a project.
func (h *ProjectHandler) AddDomainWithHosts(c *gin.Context) {
	panic("implement me")
}

// GetTargets returns all targets of a project.
func (h *ProjectHandler) GetTargets(c *gin.Context) {
	uid := c.Param("projectUID")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Project UID is required",
		})
		return
	}

	project, err := h.projectService.GetTargets(c.Request.Context(), uid)

	if err != nil {
		errReturn := gin.H{
			"error":   "Failed to retrieve targets",
			"details": err.Error(),
		}

		if err.Error() == "targets not found" {
			c.JSON(http.StatusNotFound, errReturn)
		} else {
			c.JSON(http.StatusInternalServerError, errReturn)
		}
		return
	}

	c.JSON(http.StatusOK, project)
}

// CreateTarget creates a new target for a project.
func (h *ProjectHandler) CreateTarget(c *gin.Context) {
	type CreateTargetRequest struct {
		IP   string `json:"ip" binding:"required" validate:"required"`
		Note string `json:"note"`
		CIDR int    `json:"cidr"`
	}

	var request CreateTargetRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	uid := c.Param("projectUID")
	fmt.Println("UID:", uid)

	target, err := h.projectService.CreateTarget(c.Request.Context(), uid, request.IP, request.Note, request.CIDR)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create target",
			"details": err.Error(),
		})
		fmt.Println(err)
		return
	}

	c.JSON(http.StatusOK, target)
}

// CreateProject creates a new project.
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	type CreateProjectRequest struct {
		Name string `json:"name" binding:"required"`
	}

	var request CreateProjectRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	projectUID, err := h.projectService.Create(c.Request.Context(), request.Name)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create project",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"uid":     projectUID,
		"message": "Project created successfully",
	})
}

// UpdateProject updates fields of an existing project.
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	project := restcontext.Project(c)
	var fieldsToUpdate map[string]interface{}

	if err := c.BindJSON(&fieldsToUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_JSON"})
		return
	}

	updatedProject, err := h.projectService.UpdateProject(c.Request.Context(), project.UID, "UserInput", fieldsToUpdate)

	if err != nil {
		log.Printf("Sending 500 response while updating project because: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to update project",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Project updated successfully",
		"updated_project": updatedProject,
	})
}

// AddActiveDirectory adds a new Active Directory forest to a project.
func (h *ProjectHandler) AddActiveDirectory(c *gin.Context) {
	type CreateActiveDirectory struct {
		ForestName string `json:"forest_name" binding:"required"`
	}

	var request CreateActiveDirectory

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request to create a new active directory forest",
			"details": err.Error(),
		})
		return
	}

	activeDirectory := &rpadmodel.ActiveDirectory{
		ForestName: request.ForestName,
	}

	projectUid := c.Param("projectUID")
	createdAD, err := h.projectService.AddActiveDirectory(c.Request.Context(), projectUid, activeDirectory, "UserInput")

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add a new active directory into project",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":       "success",
		"message":      "New active directory has been created",
		"created_user": createdAD,
	})
}

// GetActiveDirectories retrieves all active directories for a project.
func (h *ProjectHandler) GetActiveDirectories(c *gin.Context) {
	uid := c.Param("projectUID")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Project UID is required",
		})
		return
	}

	activeDirectories, err := h.projectService.GetAllActiveDirectories(c.Request.Context(), uid)

	if err != nil {
		errReturn := gin.H{
			"error":   "Failed to retrieve active directories",
			"details": err.Error(),
		}
		log.Println(err)
		c.JSON(http.StatusInternalServerError, errReturn)
		return
	}

	c.JSON(http.StatusOK, activeDirectories)
}

// GetHosts retrieves all hosts for a project.
func (h *ProjectHandler) GetHosts(c *gin.Context) {
	uid := c.Param("projectUID")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Project UID is required",
		})
		return
	}

	domains, err := h.projectService.GetHostsByProject(c.Request.Context(), uid)

	if err != nil {
		errReturn := gin.H{
			"error":   "Failed to retrieve hosts",
			"details": err.Error(),
		}
		c.JSON(http.StatusInternalServerError, errReturn)
		return
	}

	c.JSON(http.StatusOK, domains)
}

// GetUsers retrieves all users for a project.
func (h *ProjectHandler) GetUsers(c *gin.Context) {
	uid := c.Param("projectUID")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Project UID is required",
		})
		return
	}

	domains, err := h.projectService.GetAllUserInProject(c.Request.Context(), uid)

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
