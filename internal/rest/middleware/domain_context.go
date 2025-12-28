package middleware

import (
	"net/http"

	"RedPaths-server/pkg/service/active_directory"

	"github.com/gin-gonic/gin"
)

func DomainContext(projectService *active_directory.ProjectService) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectUID := c.Param("projectUID")
		domainUID := c.Param("domainUID")

		domain, err := projectService.GetByUID(c.Request.Context(), projectUID, domainUID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"error": "domain not found in project",
			})
			return
		}
		c.Set("domain", domain)
		c.Next()
	}
}
