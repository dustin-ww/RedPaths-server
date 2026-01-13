package active_directory

import (
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/utils"
)

type ActiveDirectory struct {

	// Internal
	UID   string   `json:"uid,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`

	//Specific
	ForestName            string `json:"active_directory.forest_name,omitempty"`
	ForestFunctionalLevel string `json:"active_directory.forest_functional_level,omitempty"`

	// Relations
	HasDomain []*utils.UIDRef `json:"active_directory.has_domain,omitempty"`

	// Meta
	RedPathsMetadata core.RedPathsMetadata `json:"-"`
}

func (ad *ActiveDirectory) UnmarshalJSON(data []byte) error {
	type Alias ActiveDirectory
	aux := (*Alias)(ad)
	return core.UnmarshalWithMetadata(data, aux, &ad.RedPathsMetadata)
}

func (ad ActiveDirectory) MarshalJSON() ([]byte, error) {
	type Alias ActiveDirectory
	return core.MarshalWithMetadata(Alias(ad), ad.RedPathsMetadata)
}
