package handlers

import (
	"RedPaths-server/pkg/input"
	"RedPaths-server/pkg/service/redpaths"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type RedPathsModuleHandler struct {
	redPathsModuleService *redpaths.ModuleService
}

func NewRedPathsModuleHandler(redPathsModuleService *redpaths.ModuleService) *RedPathsModuleHandler {
	return &RedPathsModuleHandler{
		redPathsModuleService: redPathsModuleService,
	}
}

func (h *RedPathsModuleHandler) GetModules(c *gin.Context) {
	modules, err := h.redPathsModuleService.GetAll(c.Request.Context())
	if err != nil {
		log.Printf("failed to get all modulelib: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(modules) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "no redpaths modulelib found"})
	}
	c.JSON(http.StatusOK, modules)
}

func (h *RedPathsModuleHandler) GetModuleInheritanceGraph(c *gin.Context) {
	graph, err := h.redPathsModuleService.GetInheritanceGraph(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, graph)
}

func (h *RedPathsModuleHandler) RunModule(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

func (h *RedPathsModuleHandler) RunAttackVector(c *gin.Context) {
	moduleKey := c.Param("moduleKey")
	log.Printf("Run Vector")

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	params, err := input.ParseParameters(body)
	log.Printf(params.ProjectUID)
	if err != nil {
		log.Printf(string(body))
		log.Printf("failed to parse parameters: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	runUid, err := h.redPathsModuleService.RunAttackVector(c.Request.Context(), moduleKey, &params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"runUid": runUid,
	})
}

func (h *RedPathsModuleHandler) GetAttackVectorOptions(c *gin.Context) {
	moduleKey := c.Param("moduleKey")
	options, err := h.redPathsModuleService.GetOptionsForAttackVector(c.Request.Context(), moduleKey)
	if err != nil {
		log.Printf("failed to get options for attack vector: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, options)
}

func (h *RedPathsModuleHandler) GetModuleRuns(c *gin.Context) {
	projectUid := c.Param("projectUID")
	runMetadata, err := h.redPathsModuleService.GetAllRunMetadata(c.Request.Context(), projectUid)
	if err != nil {
		log.Printf("failed to get all module runs for project %s with error: %v", projectUid, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, runMetadata)
}

func (h *RedPathsModuleHandler) GetVectorRuns(c *gin.Context) {
	projectUid := c.Param("projectUID")
	vrunMetadata, err := h.redPathsModuleService.GetAllVectorRuns(c.Request.Context(), projectUid)
	if err != nil {
		log.Printf("failed to get all vector runs for project %s with error: %v", projectUid, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, vrunMetadata)
}

func (h *RedPathsModuleHandler) GetModuleOptions(c *gin.Context) {
	panic("not implemented")
}
