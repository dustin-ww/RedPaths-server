// internal/rest/handlers/change_handler.go
package handlers

import (
	"RedPaths-server/internal/repository/redpaths/changes"
	"RedPaths-server/pkg/service/change"

	"net/http"

	"github.com/gin-gonic/gin"
)

type ChangeHandler struct {
	changeService *change.ChangeService
}

func NewChangeHandler(changeService *change.ChangeService) *ChangeHandler {
	return &ChangeHandler{changeService: changeService}
}

func (h *ChangeHandler) GetChanges(c *gin.Context) {
	// entityType kommt aus der EntityType-Middleware via context value
	entityType := c.GetString("entityType")

	// entityUID: verschiedene Param-Namen je nach Entity
	entityUID := resolveEntityUID(c)

	if entityUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "entity uid missing"})
		return
	}
	if entityType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "entity type missing"})
		return
	}

	result, err := h.changeService.GetChangesByEntity(c.Request.Context(), entityType, entityUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *ChangeHandler) GetChangesWithOptions(c *gin.Context) {
	entityType := c.GetString("entityType")
	entityUID := resolveEntityUID(c)

	if entityUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "entity uid missing"})
		return
	}

	var opts changes.ChangeQueryOptions
	if err := c.ShouldBindJSON(&opts); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.changeService.GetChangesByEntityWithOptions(
		c.Request.Context(),
		entityType,
		entityUID,
		&opts,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// resolveEntityUID liest den UID-Param unabhängig vom Entity-spezifischen Namen
func resolveEntityUID(c *gin.Context) string {
	for _, param := range []string{"hostUID", "userUID", "domainUID", "dirNodeUID"} {
		if uid := c.Param(param); uid != "" {
			return uid
		}
	}
	return ""
}
