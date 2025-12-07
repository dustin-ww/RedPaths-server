package sse

import (
	"RedPaths-server/pkg/model/events"
	rpmodel "RedPaths-server/pkg/model/redpaths"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

// SSEHandler establishes an SSE stream
func SSEHandler(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get parameters
	query := r.URL.Query()
	runID := query.Get("runId")
	minLevelStr := query.Get("level")

	// Default to INFO level if not specified
	minLevel := rpmodel.INFO
	if minLevelStr != "" {
		switch minLevelStr {
		case string(rpmodel.DEBUG):
			minLevel = rpmodel.DEBUG
		case string(rpmodel.INFO):
			minLevel = rpmodel.INFO
		case string(rpmodel.WARNING):
			minLevel = rpmodel.WARNING
		case string(rpmodel.ERROR):
			minLevel = rpmodel.ERROR
		}
	}

	// Get or create logger
	logger := GetLogger(runID, "", ssePostgresCon)
	if logger == nil {
		sendErrorResponse(w, "Failed to create logger", http.StatusInternalServerError, "")
		return
	}

	// Register client with specified minimum level
	client := logger.RegisterClient(minLevel)
	defer logger.UnregisterClient(client)

	ctx := r.Context()

	// Initial connected event
	fmt.Fprintf(w, "event: connected\n")
	fmt.Fprintf(w, "data: {\"status\":\"connected\", \"runId\":\"%s\", \"minLevel\":\"%s\"}\n\n", runID, minLevel)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Setup heartbeat ticker
	heartbeat := time.NewTicker(HeartbeatInterval)
	defer heartbeat.Stop()

	// Event loop
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			// Send heartbeat event
			fmt.Fprintf(w, "event: heartbeat\n")
			fmt.Fprintf(w, "data: {\"timestamp\":%d}\n\n", time.Now().Unix())
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case ev, ok := <-client:
			if !ok {
				return
			}
			data, err := json.Marshal(ev)
			if err != nil {
				log.Printf("[SSE] JSON marshal failed: %v", err)
				continue
			}
			fmt.Fprintf(w, "event: %s\n", ev.Type)
			fmt.Fprintf(w, "data: %s\n\n", data)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

// TriggerEventHandler triggers an event manually
func TriggerEventHandler(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get parameters
	query := r.URL.Query()
	eventType := query.Get("type")
	message := query.Get("msg")
	runID := query.Get("runId")

	if eventType == "" || message == "" {
		sendErrorResponse(w, "Missing parameters", http.StatusBadRequest, "Both 'type' and 'msg' are required")
		return
	}

	logger := GetLogger(runID, "", ssePostgresCon)
	if logger == nil {
		sendErrorResponse(w, "Logger not available", http.StatusInternalServerError, "")
		return
	}

	// Create payload with timestamp
	payload := map[string]interface{}{
		"message":   message,
		"source":    "HTTP-Trigger",
		"timestamp": time.Now().Unix(),
	}

	// Handle different event types
	switch eventType {
	case string(rpmodel.DEBUG):
		logger.Debug(message, payload)
	case string(rpmodel.INFO):
		logger.Info(message, payload)
	case string(rpmodel.WARNING):
		logger.Warning(message, payload)
	case string(rpmodel.ERROR):
		logger.Error(message, payload)
	default:
		logger.Event(events.EventType(eventType), payload)
	}

	response := map[string]interface{}{
		"status":    "success",
		"eventType": eventType,
		"runId":     runID,
		"timestamp": time.Now().Unix(),
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// LogsHandler lists recent logs with pagination support
func LogsHandler(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get parameters
	query := r.URL.Query()
	runID := query.Get("runId")

	logger := GetLogger(runID, "", ssePostgresCon)
	if logger == nil {
		sendErrorResponse(w, "Logger not found", http.StatusNotFound, "")
		return
	}

	// Parse filters
	levelFilter := query.Get("level")
	moduleFilter := query.Get("module")
	eventTypeFilter := query.Get("type")

	// Parse pagination parameters
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")
	/*sinceStr := query.Get("since")
	untilStr := query.Get("until")*/

	// Default pagination values
	limit := 100
	offset := 0
	//var since, until int64

	// Parse limit
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
			if limit > MaxEventBacklog {
				limit = MaxEventBacklog
			}
		}
	}

	// Parse offset
	if offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	/*	// Parse since timestamp
		if sinceStr != "" {
			if parsed, err := strconv.ParseInt(sinceStr, 10, 64); err == nil {
				since = parsed
			}
		}

		// Parse until timestamp
		if untilStr != "" {
			if parsed, err := strconv.ParseInt(untilStr, 10, 64); err == nil {
				until = parsed
			}
		}
	*/
	// Lock for reading
	logger.storeLock.RLock()
	defer logger.storeLock.RUnlock()

	// Filter and collect results
	var results []rpmodel.LogEntry
	totalCount := 0
	resultCount := 0

	for _, entry := range logger.store {
		// Apply filters
		if levelFilter != "" && string(entry.Level) != levelFilter {
			continue
		}
		if moduleFilter != "" && entry.ModuleKey != moduleFilter {
			continue
		}
		if eventTypeFilter != "" && entry.EventType != eventTypeFilter {
			continue
		}
		/*	if since > 0 && entry.Timestamp < since {
				continue
			}
			if until > 0 && entry.Timestamp > until {
				continue
			}*/

		totalCount++

		// Apply pagination
		if totalCount <= offset {
			continue
		}

		if resultCount < limit {
			results = append(results, entry)
			resultCount++
		}
	}

	// Prepare response with metadata
	response := map[string]interface{}{
		"logs":      results,
		"count":     len(results),
		"total":     totalCount,
		"offset":    offset,
		"limit":     limit,
		"timestamp": time.Now().Unix(),
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[SSE] Error encoding response: %v", err)
	}
}

// sendErrorResponse sends a standardized error response
func sendErrorResponse(w http.ResponseWriter, message string, statusCode int, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response := ErrorResponse{
		Error:   message,
		Code:    statusCode,
		Details: details,
	}
	json.NewEncoder(w).Encode(response)
}
