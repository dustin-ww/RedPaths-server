// middleware/strip_prefix.go

package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func StripDgraphPrefixMiddleware(c *gin.Context) {
	// Response abfangen
	rec := &responseRecorder{
		ResponseWriter: c.Writer,
		body:           &bytes.Buffer{},
		statusCode:     http.StatusOK,
	}
	c.Writer = rec

	// Handler ausführen
	c.Next()

	// Nur JSON transformieren
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		c.Writer = rec.ResponseWriter
		rec.ResponseWriter.WriteHeader(rec.statusCode)
		rec.ResponseWriter.Write(rec.body.Bytes())
		return
	}

	// JSON parsen und Prefixe strippen
	stripped, err := stripPrefixesFromJSON(rec.body.Bytes())
	if err != nil {
		rec.ResponseWriter.WriteHeader(rec.statusCode)
		rec.ResponseWriter.Write(rec.body.Bytes())
		return
	}

	rec.ResponseWriter.Header().Set("Content-Type", "application/json")
	rec.ResponseWriter.WriteHeader(rec.statusCode)
	rec.ResponseWriter.Write(stripped)
}

type responseRecorder struct {
	gin.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}

func (r *responseRecorder) WriteHeaderNow() {
	// Gin-spezifisch: noch nicht schreiben, wir buffern
}

func stripPrefixesFromJSON(data []byte) ([]byte, error) {
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return json.Marshal(stripValue(raw))
}

func stripValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return stripMap(val)
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = stripValue(item)
		}
		return result
	default:
		return v
	}
}

func stripMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		if k == "dgraph.type" {
			continue
		}
		newKey := k
		if idx := strings.LastIndex(k, "."); idx != -1 {
			newKey = k[idx+1:]
		}
		result[newKey] = stripValue(v)
	}
	return result
}
