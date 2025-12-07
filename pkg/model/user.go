package model

import "time"

type User struct {
	UID             string   `json:"uid,omitempty"`
	Name            string   `json:"name,omitempty"`
	NTLMHash        string   `json:"ntlm_hash,omitempty"`
	Password        string   `json:"password,omitempty"`
	IsAdmin         bool     `json:"is_admin,omitempty"`
	BelongsToDomain Domain   `json:"belongs_to_domain,omitempty"`
	DType           []string `json:"dgraph.type,omitempty"`

	// History related
	DiscoveredAt time.Time `json:"discovered_at,omitempty"`
	DiscoveredBy string    `json:"discovered_by,omitempty"`
	LastSeenAt   time.Time `json:"last_seen_at,omitempty"`
	LastSeenBy   string    `json:"last_seen_by,omitempty"`
}
