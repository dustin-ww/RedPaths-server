package priv

import "RedPaths-server/pkg/model/core"

type ADRight struct {

	// Internal
	UID   string   `json:"uid,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`

	//Specific
	Name      string `json:"ad_right.name,omitempty"`
	Category  string `json:"ad_right.category,omitempty"`
	RistLevel int    `json:"ace.inherit,omitempty"`

	// Meta
	RedPathsMetadata core.RedPathsMetadata `json:"-"`
}

func (adr *ADRight) UnmarshalJSON(data []byte) error {
	type Alias ADRight
	aux := (*Alias)(adr)
	return core.UnmarshalWithMetadata(data, aux, &adr.RedPathsMetadata)
}

func (adr ADRight) MarshalJSON() ([]byte, error) {
	type Alias ADRight
	return core.MarshalWithMetadata(Alias(adr), adr.RedPathsMetadata)
}
