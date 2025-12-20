package history

import (
	"time"

	"github.com/google/uuid"
)

type Change struct {
	UID        uuid.UUID     `gorm:"type:uuid;primaryKey"`
	EntityType string        `gorm:"not null;index:idx_entity"`
	EntityUID  string        `gorm:"not null;index:idx_entity"`
	Changes    []FieldChange `gorm:"type:jsonb;not null;serializer:json"`
	ChangedAt  time.Time     `gorm:"not null"`

	ChangedBy    string
	ChangeReason string
}

type FieldChange struct {
	Field    string `json:"field"`
	OldValue any    `json:"old_value"`
	NewValue any    `json:"new_value"`
}
