package model

import (
	"RedPaths-server/pkg/model/utils"
	"time"
)

type Project struct {
	// Internal
	UID                       string         `json:"uid,omitempty"`
	Name                      string         `json:"name,omitempty"`
	Tags                      []string       `json:"tags,omitempty"`
	Description               string         `json:"description,omitempty"`
	CreatedAt                 time.Time      `json:"created_at,omitempty"`
	UpdatedAt                 time.Time      `json:"updated_at,omitempty"`
	HasTarget                 []Target       `json:"has_target,omitempty"`
	HasDomain                 []Domain       `json:"has_domain,omitempty"`
	HashHostWithUnknownDomain []utils.UIDRef `json:"has_unknown_domain_host,omitempty"`
	DType                     []string       `json:"dgraph.type,omitempty"`
}
