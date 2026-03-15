package model

import "RedPaths-server/pkg/model/core"

type Project struct {
	// Internal
	UID   string   `json:"uid,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`

	// Specific
	Name        string   `json:"project.name,omitempty"`
	Tags        []string `json:"project.tags,omitempty"`
	Description string   `json:"project.description,omitempty"`

	// Targets
	HasTarget []Target `json:"project.has_target,omitempty"`

	// Meta
	RedPathsMetadata core.RedPathsMetadata `json:"-"`
}

func (p *Project) UnmarshalJSON(data []byte) error {
	type Alias Project
	aux := (*Alias)(p)
	return core.UnmarshalWithMetadata(data, aux, &p.RedPathsMetadata)
}

func (p Project) MarshalJSON() ([]byte, error) {
	type Alias Project
	return core.MarshalWithMetadata(Alias(p), p.RedPathsMetadata)
}
