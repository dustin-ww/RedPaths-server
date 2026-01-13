package core

import (
	"encoding/json"
	"time"
)

type RedPathsMetadata struct {
	// history related
	DiscoveredAt time.Time `json:"discovered_at,omitempty"`
	DiscoveredBy string    `json:"discovered_by,omitempty"`
	LastSeenAt   time.Time `json:"last_seen_at,omitempty"`
	LastSeenBy   string    `json:"last_seen_by,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	ModifiedAt   time.Time `json:"modified_at,omitempty"`
	ValidatedAt  time.Time `json:"last_validated_at,omitempty"`
	ValidatedBy  string    `json:"last_validated_by,omitempty"`
}

// UnmarshalWithMetadata unmarshals JSON and extracts RPMetadata
func UnmarshalWithMetadata(data []byte, target interface{}, metadata *RedPathsMetadata) error {
	if err := json.Unmarshal(data, target); err != nil {
		return err
	}

	var aux struct {
		CreatedAt    time.Time `json:"created_at,omitempty"`
		ModifiedAt   time.Time `json:"modified_at,omitempty"`
		DiscoveredAt time.Time `json:"discovered_at,omitempty"`
		DiscoveredBy string    `json:"discovered_by,omitempty"`
		LastSeenAt   time.Time `json:"last_seen_at,omitempty"`
		LastSeenBy   string    `json:"last_seen_by,omitempty"`
		ValidatedAt  time.Time `json:"validated_at,omitempty"`
		ValidatedBy  string    `json:"validated_by,omitempty"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	*metadata = RedPathsMetadata{
		CreatedAt:    aux.CreatedAt,
		ModifiedAt:   aux.ModifiedAt,
		DiscoveredAt: aux.DiscoveredAt,
		DiscoveredBy: aux.DiscoveredBy,
		LastSeenAt:   aux.LastSeenAt,
		LastSeenBy:   aux.LastSeenBy,
		ValidatedAt:  aux.ValidatedAt,
		ValidatedBy:  aux.ValidatedBy,
	}

	return nil
}

// MarshalWithMetadata marshals target with embedded metadata
func MarshalWithMetadata(target interface{}, metadata RedPathsMetadata) ([]byte, error) {
	targetBytes, err := json.Marshal(target)
	if err != nil {
		return nil, err
	}

	var targetMap map[string]interface{}
	if err := json.Unmarshal(targetBytes, &targetMap); err != nil {
		return nil, err
	}

	if !metadata.ValidatedAt.IsZero() {
		targetMap["validated_at"] = metadata.CreatedAt
	}
	if metadata.ValidatedBy != "" {
		targetMap["validated_by"] = metadata.CreatedAt
	}
	if !metadata.CreatedAt.IsZero() {
		targetMap["created_at"] = metadata.CreatedAt
	}
	if !metadata.ModifiedAt.IsZero() {
		targetMap["modified_at"] = metadata.ModifiedAt
	}
	if !metadata.DiscoveredAt.IsZero() {
		targetMap["discovered_at"] = metadata.DiscoveredAt
	}
	if metadata.DiscoveredBy != "" {
		targetMap["discovered_by"] = metadata.DiscoveredBy
	}
	if !metadata.LastSeenAt.IsZero() {
		targetMap["last_seen_at"] = metadata.LastSeenAt
	}
	if metadata.LastSeenBy != "" {
		targetMap["last_seen_by"] = metadata.LastSeenBy
	}

	return json.Marshal(targetMap)
}
