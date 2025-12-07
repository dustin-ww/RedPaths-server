package handlers

import (
	"RedPaths-server/pkg/model/utils/query"
	"RedPaths-server/pkg/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type LogHandler struct {
	logService *service.LogService
}

func NewLogHandler(logService *service.LogService) *LogHandler {
	return &LogHandler{
		logService: logService,
	}
}

func (h *LogHandler) GetLogTypes(c *gin.Context) {
	uid := c.Param("projectUID")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Project UID is required",
		})
		return
	}

	types, err := h.logService.GetAllEventTypes(
		c.Request.Context(),
		uid)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve log types",
			"details": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, types)
}

func (h *LogHandler) GetModuleKeySet(c *gin.Context) {
	uid := c.Param("projectUID")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Project UID is required",
		})
		return
	}

	types, err := h.logService.GetModuleKeySet(
		c.Request.Context(),
		uid)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve log types",
			"details": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, types)
}

func (h *LogHandler) GetLogs(c *gin.Context) {
	uid := c.Param("projectUID")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Project UID is required",
		})
		return
	}

	logs, err := h.logService.GetAllProjectLogs(
		c.Request.Context(),
		uid,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve logs",
			"details": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, logs)
}

func (h *LogHandler) GetLogsWithOptions(c *gin.Context) {
	uid := c.Param("projectUID")

	var request query.LogQueryOptions
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Project UID is required",
		})
		return
	}

	logs, err := h.logService.GetProjectLogsWithOptions(
		c.Request.Context(),
		uid,
		&request,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve logs with specified query",
			"details": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, logs)
}

/*CreateLogEntry(ctx context.Context, tx *gorm.DB, event *redpaths.LogEntry) error
GetLogsByProject(ctx context.Context, tx *gorm.DB, projectUID string) ([]*redpaths.LogEntry, error)
GetLogsByRun(ctx context.Context, tx *gorm.DB, runUID string) ([]*redpaths.LogEntry, error)
GetLogsByModule(ct*/
