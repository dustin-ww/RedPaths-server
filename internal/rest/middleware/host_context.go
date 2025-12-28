package middleware

import (
	"RedPaths-server/pkg/service/active_directory"

	"net/http"

	"github.com/gin-gonic/gin"
)

func HostContext(projectService *active_directory.ProjectService) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectUID := c.Param("projectUID")
		hostUID := c.Param("hostUID")

		host, err := projectService.GetHostByProject(c.Request.Context(), projectUID, hostUID)

		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"error": "host not found",
			})
			return
		}

		c.Set("host", host)
		c.Next()
	}
}
