package change

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type NodeSnapshot struct {
	NodeUID   string    `json:"node_uid" gorm:"primaryKey"`
	Data      JSONB     `json:"data" gorm:"type:jsonb;not null"`
	Edges     JSONBArr  `json:"edges" gorm:"type:jsonb;not null"`
	Version   int64     `json:"version" gorm:"not null;default:0"`
	UpdatedAt time.Time `json:"updated_at" gorm:"not null;default:now()"`
}

type JSONBArr []map[string]interface{}

func (j JSONBArr) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONBArr) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONBArr value: %v", value)
	}
	return json.Unmarshal(bytes, &j)
}

type NodeSnapshotBuilder struct {
	s NodeSnapshot
}

func NewNodeSnapshotBuilder() *NodeSnapshotBuilder {
	return &NodeSnapshotBuilder{
		s: NodeSnapshot{UpdatedAt: time.Now()},
	}
}

// SetNodeUID sets the UID of the node
func (b *NodeSnapshotBuilder) SetNodeUID(uid string) *NodeSnapshotBuilder {
	b.s.NodeUID = uid
	return b
}

// SetData sets the JSONB snapshot of node attributes
func (b *NodeSnapshotBuilder) SetData(val JSONB) *NodeSnapshotBuilder {
	b.s.Data = val
	return b
}

// SetEdges sets the JSONBArr for edges (each item: {"predicate": ..., "target_uid": ...})
func (b *NodeSnapshotBuilder) SetEdges(arr JSONBArr) *NodeSnapshotBuilder {
	b.s.Edges = arr
	return b
}

// SetVersion sets the version number of the snapshot
func (b *NodeSnapshotBuilder) SetVersion(v int64) *NodeSnapshotBuilder {
	b.s.Version = v
	return b
}

// SetUpdatedAt overrides the update timestamp
func (b *NodeSnapshotBuilder) SetUpdatedAt(ts time.Time) *NodeSnapshotBuilder {
	b.s.UpdatedAt = ts
	return b
}

// Build finalizes and returns the constructed NodeSnapshot
func (b *NodeSnapshotBuilder) Build() NodeSnapshot {
	return b.s
}
