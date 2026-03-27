package handlers

import (
	restcontext "RedPaths-server/internal/rest/context"
	"RedPaths-server/internal/rest/requests"
	"RedPaths-server/pkg/model"
	rpadmodel "RedPaths-server/pkg/model/active_directory"
	"RedPaths-server/pkg/model/utils/assertion"
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

	log.Println("PROJECT DATE " + projectsOverviews[0].RedPathsMetadata.CreatedAt.String())
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

	project := &model.Project{
		Name: request.Name,
	}

	projectUID, err := h.projectService.Create(c.Request.Context(), project)

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
		"created": project,
	})
}

// UpdateProject updates fields of an existing project.
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	project := restcontext.Project(c)
	mappedFields := c.MustGet("mappedFields").(map[string]interface{})

	updatedProject, err := h.projectService.UpdateProject(
		c.Request.Context(), project.UID, "UserInput", mappedFields,
	)

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

// AddProjectActiveDirectory adds a new Active Directory forest to a project.
func (h *ProjectHandler) AddProjectActiveDirectory(c *gin.Context) {

	var request requests.CreateActiveDirectory

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request to create a new active directory forest",
			"details": err.Error(),
		})
		return
	}

	ac := assertion.FromRequest(request.AssertionContext)

	activeDirectory := &rpadmodel.ActiveDirectory{
		ForestName:            request.ForestName,
		ForestFunctionalLevel: request.ForestFunctionalLevel,
	}

	projectUid := c.Param("projectUID")
	createdAD, err := h.projectService.AddActiveDirectory(
		c.Request.Context(),
		ac,
		projectUid,
		activeDirectory,
		"UserInput")

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add a new active directory into project",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":                   "success",
		"message":                  "New active directory has been created",
		"created_active_directory": createdAD,
	})
}

func (h *ProjectHandler) GetCatalogDomains(c *gin.Context) {
	handleCatalogGet(c, "projectUID", "Failed to retrieve domains from catalog",
		h.projectService.GetAllDomainsFromCatalog)
}

func (h *ProjectHandler) GetPlacedDomains(c *gin.Context) {
	handleCatalogGet(c, "projectUID", "Failed to retrieve placed domains",
		h.projectService.GetAllDomains)
}

func (h *ProjectHandler) GetCatalogHosts(c *gin.Context) {
	handleCatalogGet(c, "projectUID", "Failed to retrieve hosts from catalog",
		h.projectService.GetAllHostsFromCatalog)
}

func (h *ProjectHandler) GetPlacedHosts(c *gin.Context) {
	handleCatalogGet(c, "projectUID", "Failed to retrieve placed domains",
		h.projectService.GetHostsByProject)
}

func (h *ProjectHandler) GetCatalogUsers(c *gin.Context) {
	handleCatalogGet(c, "projectUID", "Failed to retrieve users from catalog",
		h.projectService.GetAllUsersFromCatalog)
}

//func (h *ProjectHandler) GetPlacedUsers(c *gin.Context) {
//	handleCatalogGet(c, "projectUID", "Failed to retrieve users from catalog",
//		h.projectService.GetUsersByProject)
//}

func (h *ProjectHandler) GetCatalogServices(c *gin.Context) {
	handleCatalogGet(c, "projectUID", "Failed to retrieve services from catalog",
		h.projectService.GetAllServicesFromCatalog)
}

func (h *ProjectHandler) GetPlacedServices(c *gin.Context) {
	handleCatalogGet(c, "projectUID", "Failed to retrieve users from catalog",
		h.projectService.GetServicesByProject)
}

func (h *ProjectHandler) GetCatalogDirectoryNodes(c *gin.Context) {
	handleCatalogGet(c, "projectUID", "Failed to retrieve directory nodes from catalog",
		h.projectService.GetAllDirectoryNodesFromCatalog)
}

func (h *ProjectHandler) GetDirectoryNodes(c *gin.Context) {
	handleCatalogGet(c, "projectUID", "Failed to retrieve directory nodes",
		h.projectService.GetAllDirectoryNodes)
}

func (h *ProjectHandler) GetProjectActiveDirectories(c *gin.Context) {
	handleCatalogGet(c, "projectUID", "Failed to retrieve active directories",
		h.projectService.GetAllActiveDirectories)
}

func (h *ProjectHandler) GetTargets(c *gin.Context) {
	handleCatalogGet(c, "projectUID", "Failed to retrieve targets",
		h.projectService.GetTargets)
}

func (h *ProjectHandler) Get(c *gin.Context) {
	handleCatalogGet(c, "projectUID", "Failed to retrieve project",
		h.projectService.Get)
}

func (h *ProjectHandler) GetOrphanedDomains(c *gin.Context) {
	handleCatalogGet(c, "projectUID", "Failed to retrieve orphaned domains from catalog",
		h.projectService.GetOrphanedDomainsFromCatalog)
}

func (h *ProjectHandler) GetOrphanedHosts(c *gin.Context) {
	handleCatalogGet(c, "projectUID", "Failed to retrieve orphaned hosts from catalog",
		h.projectService.GetOrphanedHostsFromCatalog)
}

func (h *ProjectHandler) GetOrphanedUsers(c *gin.Context) {
	handleCatalogGet(c, "projectUID", "Failed to retrieve orphaned users from catalog",
		h.projectService.GetOrphanedUsersFromCatalog)
}
