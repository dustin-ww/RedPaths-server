package active_directory

import "RedPaths-server/pkg/model"

type ActiveDirectory struct {
	UID                   string                 `json:"uid,omitempty"`
	ForestName            string                 `json:"forest_name,omitempty"`
	ForestFunctionalLevel string                 `json:"forest_functional_level,omitempty"`
	RedPathsMetadata      model.RedPathsMetadata `json:"rp_metadata,omitempty"`
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
