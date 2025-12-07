package handlers

import (
	"RedPaths-server/pkg/service/active_directory"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ServerHandler struct {
	projectService *active_directory.ProjectService
}

func NewServerHandler() *ServerHandler {
	return &ServerHandler{}
}

func (h *ServerHandler) GetHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "hello",
	})
}
