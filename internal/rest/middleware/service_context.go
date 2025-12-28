package middleware

import (
	"RedPaths-server/pkg/service/active_directory"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ServiceContext(hostService *active_directory.HostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		hostUID := c.Param("hostUID")
		serviceUID := c.Param("serviceUID")

		service, err := hostService.GetServiceByHost(c.Request.Context(), hostUID, serviceUID)

		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"error": "service not found",
			})
			return
		}

		c.Set("service", service)
		c.Next()
	}
}
