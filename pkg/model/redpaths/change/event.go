package change

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"time"
)

type Event struct {
	EventID uuid.UUID `json:"event_id" gorm:"type:uuid;primaryKey"`

	SubjectUID string `json:"subject_uid" gorm:"not null"`
	EventType  string `json:"event_type" gorm:"type:text;not null"`

	Predicate *string `json:"predicate,omitempty"`
	OldValue  *JSONB  `json:"old_value,omitempty"`
	NewValue  *JSONB  `json:"new_value,omitempty"`

	OldTargetUID *string `json:"old_target_uid,omitempty"`
	NewTargetUID *string `json:"new_target_uid,omitempty"`

	ChangedAt     time.Time `json:"changed_at" gorm:"not null;default:now()"`
	ChangedBy     *string   `json:"changed_by,omitempty"`
	TransactionID uuid.UUID `json:"transaction_id" gorm:"type:uuid;not null"`
}

type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: %v", value)
	}
	return json.Unmarshal(bytes, &j)
}

// ChangeEventBuilder provides a fluent API to construct ChangeEvent instances.
type EventBuilder struct {
	e Event
}

// NewChangeEventBuilder initializes a builder with default EventID and ChangedAt
func NewChangeEventBuilder() *EventBuilder {
	return &EventBuilder{
		e: Event{
			EventID:   uuid.New(),
			ChangedAt: time.Now(),
		},
	}
}

// SetSubjectUID sets the Dgraph UID of the subject (source node)
func (b *EventBuilder) SetSubjectUID(uid string) *EventBuilder {
	b.e.SubjectUID = uid
	return b
}

// SetEventType sets the type of change (e.g., node_create, attribute_update)
func (b *EventBuilder) SetEventType(eventType string) *EventBuilder {
	b.e.EventType = eventType
	return b
}

// SetPredicate sets the attribute or edge label for update/add/remove events
func (b *EventBuilder) SetPredicate(pred string) *EventBuilder {
	b.e.Predicate = &pred
	return b
}

// SetOldValue sets the previous JSON value of an attribute
func (b *EventBuilder) SetOldValue(val JSONB) *EventBuilder {
	b.e.OldValue = &val
	return b
}

// SetNewValue sets the new JSON value of an attribute
func (b *EventBuilder) SetNewValue(val JSONB) *EventBuilder {
	b.e.NewValue = &val
	return b
}

// SetOldTargetUID sets the previous target UID for an edge removal
func (b *EventBuilder) SetOldTargetUID(uid string) *EventBuilder {
	b.e.OldTargetUID = &uid
	return b
}

// SetNewTargetUID sets the new target UID for an edge addition
func (b *EventBuilder) SetNewTargetUID(uid string) *EventBuilder {
	b.e.NewTargetUID = &uid
	return b
}

// SetChangedBy annotates who or what service initiated the change
func (b *EventBuilder) SetChangedBy(by string) *EventBuilder {
	b.e.ChangedBy = &by
	return b
}

// SetTransactionID groups multiple events under one transaction
func (b *EventBuilder) SetTransactionID(txID uuid.UUID) *EventBuilder {
	b.e.TransactionID = txID
	return b
}

// Build finalizes the ChangeEvent construction
func (b *EventBuilder) Build() Event {
	// Ensure TransactionID is set
	if b.e.TransactionID == uuid.Nil {
		b.e.TransactionID = uuid.New()
	}
	return b.e
}
