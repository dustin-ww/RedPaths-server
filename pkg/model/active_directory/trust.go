package active_directory

import (
	"RedPaths-server/pkg/model"
	"RedPaths-server/pkg/model/utils"
)

type Trust struct {
	UID   string   `json:"uid,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`

	// AD related
	TrustType     string       `json:"trust.trust_type,omitempty"`
	Direction     string       `json:"trust.direction,omitempty"`
	IsTransitive  bool         `json:"trust.is_transitive,omitempty"`
	TrustedDomain utils.UIDRef `json:"trust.trusted_domain,omitempty"`

	RedPathsMetadata model.RedPathsMetadata `json:"-"`
}

func (t *Trust) UnmarshalJSON(data []byte) error {
	type Alias Trust
	aux := (*Alias)(t)
	return model.UnmarshalWithMetadata(data, aux, &t.RedPathsMetadata)
}

func (t Trust) MarshalJSON() ([]byte, error) {
	type Alias Trust
	return model.MarshalWithMetadata(Alias(t), t.RedPathsMetadata)
}
