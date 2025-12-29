package active_directory

import (
	"RedPaths-server/pkg/model"
	"RedPaths-server/pkg/model/utils"
)

type ActiveDirectory struct {
	UID                   string          `json:"uid,omitempty"`
	DType                 []string        `json:"dgraph.type,omitempty"`
	ForestName            string          `json:"active_directory.forest_name,omitempty"`
	ForestFunctionalLevel string          `json:"active_directory.forest_functional_level,omitempty"`
	HasDomain             []*utils.UIDRef `json:"active_directory.has_domain,omitempty"`

	RedPathsMetadata model.RedPathsMetadata `json:"-"`
}

func (ad *ActiveDirectory) UnmarshalJSON(data []byte) error {
	type Alias ActiveDirectory
	aux := (*Alias)(ad)
	return model.UnmarshalWithMetadata(data, aux, &ad.RedPathsMetadata)
}

func (ad ActiveDirectory) MarshalJSON() ([]byte, error) {
	type Alias ActiveDirectory
	return model.MarshalWithMetadata(Alias(ad), ad.RedPathsMetadata)
}
