package core

import (
	"time"
)

type EntityResult[T any] struct {
	Entity     T               `json:"entity"`
	Assertions []*Assertion    `json:"assertions"`
	Metadata   *ResultMetadata `json:"metadata,omitempty"`
}

type ResultMetadata struct {
	Source         string    `json:"source"`
	ScanTimestamp  time.Time `json:"scan_timestamp"`
	EntityCount    int       `json:"entity_count"`
	AssertionCount int       `json:"assertion_count"`
}
