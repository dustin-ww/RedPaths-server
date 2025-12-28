package middleware

import (
	"fmt"
	"net/http"

	"RedPaths-server/pkg/service/active_directory"

	"github.com/gin-gonic/gin"
)

func ProjectContext(projectService *active_directory.ProjectService) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectUID := c.Param("projectUID")
		fmt.Println("Project UID:", projectUID)

		project, err := projectService.Get(c.Request.Context(), projectUID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"error": "project not found",
			})
			fmt.Printf("Error IN CONTEXT: %v\n", err)
			return
		}

		fmt.Printf("got project: %v\n", project.UID)
		c.Set("project", project)
		c.Next()
	}
}
