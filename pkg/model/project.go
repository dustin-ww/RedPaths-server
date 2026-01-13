package model

import (
	"RedPaths-server/pkg/model/utils"
	"time"
)

type Project struct {
	// Internal
	UID   string   `json:"uid,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`

	// Specific
	Name        string   `json:"project.name,omitempty"`
	Tags        []string `json:"project.tags,omitempty"`
	Description string   `json:"project.description,omitempty"`

	// Relations
	HasAD                     []*utils.UIDRef `json:"has_ad,omitempty"`
	HashHostWithUnknownDomain []*utils.UIDRef `json:"has_unknown_domain_host,omitempty"`

	// Meta
	HasTarget []Target  `json:"project.has_target,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}
