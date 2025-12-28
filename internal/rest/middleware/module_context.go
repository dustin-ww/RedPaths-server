package middleware

import (
	"RedPaths-server/pkg/service/redpaths"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ModuleContext(moduleService *redpaths.ModuleService) gin.HandlerFunc {
	return func(c *gin.Context) {
		moduleKey := c.Param("moduleKey")
		module, err := moduleService.GetModuleByKeyIfExists(c.Request.Context(), moduleKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"error": "module not found",
			})
			return
		}

		c.Set("module", module)
		c.Next()
	}
}
