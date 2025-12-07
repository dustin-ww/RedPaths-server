package redpaths

import "time"

type LogLevel string

const (
	DEBUG   LogLevel = "debug"
	INFO    LogLevel = "info"
	WARNING LogLevel = "warning"
	ERROR   LogLevel = "error"
)

type LogEntry struct {
	ID         string      `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	ProjectUID string      `json:"projectUID" gorm:"column:project_uid;index"`
	RunID      string      `json:"runID" gorm:"column:run_uid;index"`
	ModuleKey  string      `json:"moduleKey,omitempty" gorm:"column:module_key;index"`
	Level      LogLevel    `json:"level,omitempty" gorm:"column:log_level"`
	EventType  string      `json:"eventType,omitempty" gorm:"column:event_type"`
	Message    string      `json:"message,omitempty" gorm:"column:message"`
	Payload    interface{} `json:"payload,omitempty" gorm:"column:payload;type:jsonb"`
	Timestamp  time.Time   `json:"timestamp" gorm:"column:timestamp"`
}
