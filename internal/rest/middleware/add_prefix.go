package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// middleware/field_mapping.go

func AddPrefixMiddleware(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rawFields map[string]interface{}

		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_JSON"})
			c.Abort()
			return
		}

		if err := json.Unmarshal(bodyBytes, &rawFields); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_JSON"})
			c.Abort()
			return
		}

		mappedFields := make(map[string]interface{})
		for key, value := range rawFields {
			mappedFields[prefix+"."+key] = value
		}

		// Body wiederherstellen falls andere Middleware ihn noch braucht
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		c.Set("mappedFields", mappedFields)
		c.Next()
	}
}
