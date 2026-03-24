// pkg/model/redpaths/history/change.go
package history

import (
	"time"

	"github.com/google/uuid"
)

type ChangeType string

const (
	ChangeTypeCreated     ChangeType = "created"
	ChangeTypeUpdated     ChangeType = "updated"
	ChangeTypePossibleDup ChangeType = "possible_duplicate"
)

type Change struct {
	ID           uuid.UUID     `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	EntityType   string        `gorm:"column:entity_type;not null"    json:"entity_type"`
	EntityUID    string        `gorm:"column:entity_uid;not null"     json:"entity_uid"`
	ChangeType   ChangeType    `gorm:"column:change_type;not null"    json:"change_type"`
	Changes      []FieldChange `gorm:"type:jsonb;serializer:json"     json:"changes"`
	ChangedAt    time.Time     `gorm:"column:changed_at;not null"     json:"changed_at"`
	ChangedBy    string        `gorm:"column:changed_by"              json:"changed_by"`
	ChangeReason string        `gorm:"column:change_reason"           json:"change_reason"`
}
type FieldChange struct {
	Field    string `json:"field"`
	OldValue any    `json:"old_value"`
	NewValue any    `json:"new_value"`
}
