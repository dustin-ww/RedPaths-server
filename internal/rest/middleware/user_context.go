package middleware

import (
	"RedPaths-server/pkg/service/active_directory"
	"net/http"

	"github.com/gin-gonic/gin"
)

func UserContext(projectService *active_directory.ProjectService) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectUID := c.Param("projectUID")
		userUID := c.Param("userUID")

		user, err := projectService.GetUserInProject(c.Request.Context(), projectUID, userUID)

		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"error": "user not found",
			})
			return
		}

		c.Set("user", user)
		c.Next()
	}
}
