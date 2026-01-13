package active_directory

import (
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/utils"
)

type Trust struct {

	// Internal
	UID   string   `json:"uid,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`

	// Specific
	TrustType    string `json:"trust.trust_type,omitempty"`
	Direction    string `json:"trust.direction,omitempty"`
	IsTransitive bool   `json:"trust.is_transitive,omitempty"`

	// Relations
	TrustedDomain *utils.UIDRef `json:"trust.trusted_domain,omitempty"`

	// Meta
	RedPathsMetadata core.RedPathsMetadata `json:"-"`
}

func (t *Trust) UnmarshalJSON(data []byte) error {
	type Alias Trust
	aux := (*Alias)(t)
	return core.UnmarshalWithMetadata(data, aux, &t.RedPathsMetadata)
}

func (t Trust) MarshalJSON() ([]byte, error) {
	type Alias Trust
	return core.MarshalWithMetadata(Alias(t), t.RedPathsMetadata)
}
