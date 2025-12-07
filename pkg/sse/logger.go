package sse

import (
	"RedPaths-server/pkg/model/events"
	"RedPaths-server/pkg/service"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"

	rpmodel "RedPaths-server/pkg/model/redpaths"
)

// Configurable constants
const (
	HeartbeatInterval = 30 * time.Second // Send heartbeats every 30 seconds
	MaxEventBacklog   = 1000             // Maximum events to return in logs query
)

// SSEEvent represents an event to be sent over Server-Sent Events
type SSEEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	ID      int         `json:"id"`
	RunID   string      `json:"runId"`
}

// SSELogger handles logging, storage and SSE dispatch
type SSELogger struct {
	ctx        context.Context
	cancel     context.CancelFunc
	clients    sync.Map // map[chan SSEEvent]rpmodel.LogLevel
	eventBus   chan SSEEvent
	eventID    int
	runID      string
	projectUID string
	moduleKey  string
	logService *service.LogService

	store     []rpmodel.LogEntry
	storeLock sync.RWMutex

	maxLogs    int
	pruneSize  int
	pruneIntv  time.Duration
	created    time.Time
	lastActive time.Time
}

// ErrorResponse standardizes error responses
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Details string `json:"details,omitempty"`
}

// Global variables for logger management
var (
	loggers        sync.Map // map[string]*SSELogger
	loggersMutex   sync.Mutex
	ssePostgresCon *gorm.DB // Must be initialized externally
)

// Init initializes the global DB connection for SSE
func Init(db *gorm.DB) {
	ssePostgresCon = db
	// Start the cleanup routine
	go cleanupInactiveLoggers()
}

// cleanupInactiveLoggers removes loggers that have been inactive for too long
func cleanupInactiveLoggers() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		var toRemove []string
		now := time.Now()

		loggers.Range(func(key, value interface{}) bool {
			runID := key.(string)
			logger := value.(*SSELogger)

			// If logger is inactive for more than 30 minutes, mark for removal
			if now.Sub(logger.lastActive) > 30*time.Minute {
				toRemove = append(toRemove, runID)
			}
			return true
		})

		// Remove inactive loggers
		for _, runID := range toRemove {
			if logger, ok := loggers.Load(runID); ok {
				logger.(*SSELogger).Close()
				loggers.Delete(runID)
				log.Printf("[SSE] Cleaned up inactive logger for runID: %s", runID)
			}
		}
	}
}

// GetLogger retrieves or creates a logger for the specified runID
func GetLogger(runID string, projectUID string, db *gorm.DB) *SSELogger {
	if runID == "" {
		runID = "system"
	}

	if db == nil && ssePostgresCon == nil {
		log.Printf("[SSE] No database connection available")
		return nil
	}

	if db == nil {
		db = ssePostgresCon
	}

	if val, ok := loggers.Load(runID); ok {
		if logger, ok := val.(*SSELogger); ok {
			logger.lastActive = time.Now()
			return logger
		}
		loggers.Delete(runID)
	}

	logger, err := NewSSELogger(runID, projectUID, db)
	if err != nil {
		log.Printf("[SSE] Failed to create logger for runID '%s': %v\n", runID, err)
		return nil
	}
	loggers.Store(runID, logger)
	return logger
}

// ForModule returns a new logger with module context
func (l *SSELogger) ForModule(moduleKey string) *SSELogger {
	// Create a shallow copy of the logger
	moduleCopy := *l
	moduleCopy.moduleKey = moduleKey
	return &moduleCopy
}

// NewSSELogger initializes a new SSELogger
func NewSSELogger(runID string, projectUID string, db *gorm.DB) (*SSELogger, error) {
	if runID == "" {
		return nil, fmt.Errorf("runID required")
	}

	svc, err := service.NewLogService(db)
	if err != nil {
		return nil, fmt.Errorf("log service init failed: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	logger := &SSELogger{
		ctx:        ctx,
		cancel:     cancel,
		eventBus:   make(chan SSEEvent, 100),
		runID:      runID,
		logService: svc,
		maxLogs:    10000,
		pruneSize:  2000,
		pruneIntv:  10 * time.Second,
		created:    time.Now(),
		lastActive: time.Now(),
		projectUID: projectUID,
	}

	go logger.dispatchEvents()
	go logger.pruneLoop()

	return logger, nil
}

// Close cleans up resources
func (l *SSELogger) Close() {
	l.cancel()
	close(l.eventBus)
	l.clients.Range(func(key, _ interface{}) bool {
		ch := key.(chan SSEEvent)
		close(ch)
		l.clients.Delete(ch)
		return true
	})
}

// pruneLoop periodically trims the in-memory store
func (l *SSELogger) pruneLoop() {
	ticker := time.NewTicker(l.pruneIntv)
	defer ticker.Stop()

	for {
		select {
		case <-l.ctx.Done():
			return
		case <-ticker.C:
			l.storeLock.Lock()
			if len(l.store) > l.maxLogs {
				l.store = l.store[l.pruneSize:]
			}
			l.storeLock.Unlock()
		}
	}
}

// RegisterClient registers a new SSE client
func (l *SSELogger) RegisterClient(minLevel rpmodel.LogLevel) chan SSEEvent {
	ch := make(chan SSEEvent, 50)
	l.clients.Store(ch, minLevel)
	l.lastActive = time.Now()

	// Backfill
	go func() {
		l.storeLock.RLock()
		defer l.storeLock.RUnlock()
		for _, entry := range l.store {
			if entry.Level >= minLevel {
				ch <- SSEEvent{Type: "log", Payload: entry, ID: 0, RunID: l.runID}
			}
		}
	}()

	return ch
}

// UnregisterClient removes a client
func (l *SSELogger) UnregisterClient(ch chan SSEEvent) {
	l.clients.Delete(ch)
	close(ch)
	l.lastActive = time.Now()
}

// Info logs an INFO level message
func (l *SSELogger) Info(msg string, payload ...interface{}) {
	var data interface{}
	if len(payload) > 0 {
		data = payload[0]
	}
	l.Log(rpmodel.INFO, msg, data)
}

// Debug logs a DEBUG level message
func (l *SSELogger) Debug(msg string, payload ...interface{}) {
	var data interface{}
	if len(payload) > 0 {
		data = payload[0]
	}
	l.Log(rpmodel.DEBUG, msg, data)
}

// Warning logs a WARNING level message
func (l *SSELogger) Warning(msg string, payload ...interface{}) {
	var data interface{}
	if len(payload) > 0 {
		data = payload[0]
	}
	l.Log(rpmodel.WARNING, msg, data)
}

// Error logs an ERROR level message
func (l *SSELogger) Error(msg string, payload ...interface{}) {
	var data interface{}
	if len(payload) > 0 {
		data = payload[0]
	}
	l.Log(rpmodel.ERROR, msg, data)
}

// Log creates a new log entry
func (l *SSELogger) Log(level rpmodel.LogLevel, msg string, payload interface{}) {
	l.lastActive = time.Now()

	entry := rpmodel.LogEntry{
		RunID:      l.runID,
		ProjectUID: l.projectUID,
		ModuleKey:  l.moduleKey,
		Level:      level,
		Message:    msg,
		Payload:    payload,
		Timestamp:  time.Now(),
	}

	// Use background context as fallback if logger context is already canceled
	ctx := l.ctx
	if ctx.Err() != nil {
		ctx = context.Background()
	}

	if err := l.logService.CreateWithObject(ctx, &entry); err != nil {
		log.Printf("[SSELogger] DB write failed: %v", err)
	}

	l.storeLock.Lock()
	l.store = append(l.store, entry)
	l.storeLock.Unlock()

	l.sendEvent("log", entry)
}

// Event sends a custom event
func (l *SSELogger) Event(eventType events.EventType, payload interface{}) {
	l.lastActive = time.Now()

	entry := rpmodel.LogEntry{
		RunID:      l.runID,
		ModuleKey:  l.moduleKey,
		EventType:  eventType.String(),
		Payload:    payload,
		Timestamp:  time.Now(),
		ProjectUID: l.projectUID,
	}

	// Use background context as fallback if logger context is already canceled
	ctx := l.ctx
	if ctx.Err() != nil {
		ctx = context.Background()
	}

	if err := l.logService.CreateWithObject(ctx, &entry); err != nil {
		log.Printf("[SSELogger] DB write failed: %v", err)
	}

	l.storeLock.Lock()
	l.store = append(l.store, entry)
	l.storeLock.Unlock()

	l.sendEvent(eventType.String(), entry)
}

// sendEvent queues a new event
func (l *SSELogger) sendEvent(eventType string, payload interface{}) {
	l.eventID++
	ev := SSEEvent{Type: eventType, Payload: payload, ID: l.eventID, RunID: l.runID}
	select {
	case l.eventBus <- ev:
	case <-l.ctx.Done():
		log.Printf("[SSELogger] Failed to send event, context canceled")
	default:
		log.Printf("[SSELogger] Event bus full, dropping event")
	}
}

// dispatchEvents fans out events to clients
func (l *SSELogger) dispatchEvents() {
	for ev := range l.eventBus {
		l.clients.Range(func(key, val interface{}) bool {
			ch := key.(chan SSEEvent)
			minLevel := val.(rpmodel.LogLevel)
			if e, ok := ev.Payload.(rpmodel.LogEntry); ok && e.Level < minLevel {
				return true
			}
			select {
			case ch <- ev:
			default:
				// Channel is full, consider client slow and remove it
				log.Printf("[SSELogger] Client channel full, removing slow client")
				l.UnregisterClient(ch)
			}
			return true
		})
	}
}
