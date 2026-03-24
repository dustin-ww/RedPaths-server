// internal/rest/handlers/helpers.go
package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// handleCatalogGet ist ein generischer Helper für alle Catalog-GET-Methoden.
// Er extrahiert die projectUID, ruft die Service-Funktion auf und gibt das
// Ergebnis als JSON zurück.
func handleCatalogGet[T any](
	c *gin.Context,
	paramName string,
	errorMsg string,
	serviceFn func(ctx context.Context, uid string) (T, error),
) {
	uid := c.Param(paramName)
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": paramName + " is required",
		})
		return
	}

	result, err := serviceFn(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   errorMsg,
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

func EntityType(entityType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("entityType", entityType)
		c.Next()
	}
}
