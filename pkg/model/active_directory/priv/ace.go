package priv

import (
	"RedPaths-server/pkg/model/core"
)

type ACE struct {

	// Internal
	UID   string   `json:"uid,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`

	//Specific
	Name       string `json:"ace.name,omitempty"`
	AccessType string `json:"ace.accesss_type,omitempty"`
	Inherit    bool   `json:"ace.inherit,omitempty"`
	AppliesTo  string `json:"ace.applies_to,omitempty"`

	// Meta
	RedPathsMetadata core.RedPathsMetadata `json:"-"`
}

func (ace *ACE) UnmarshalJSON(data []byte) error {
	type Alias ACE
	aux := (*Alias)(ace)
	return core.UnmarshalWithMetadata(data, aux, &ace.RedPathsMetadata)
}

func (ace ACE) MarshalJSON() ([]byte, error) {
	type Alias ACE
	return core.MarshalWithMetadata(Alias(ace), ace.RedPathsMetadata)
}
