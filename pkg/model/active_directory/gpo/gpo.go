package gpo

import (
	"RedPaths-server/pkg/model/core"
)

type GPO struct {

	// Internal
	UID   string   `json:"uid,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`

	//Specific
	Name        string `json:"gpo.name"`
	Description string `json:"gpo.description,omitempty"`

	// Relations
	/*	Links []*utils.UIDRef `json:"active_directory.has_domain,omitempty"`

		Grants
		Contains
		HasAcls*/

	// Meta
	RedPathsMetadata core.RedPathsMetadata `json:"-"`
}

func (gpo *GPO) UnmarshalJSON(data []byte) error {
	type Alias GPO
	aux := (*Alias)(gpo)
	return core.UnmarshalWithMetadata(data, aux, &gpo.RedPathsMetadata)
}

func (gpo GPO) MarshalJSON() ([]byte, error) {
	type Alias GPO
	return core.MarshalWithMetadata(Alias(gpo), gpo.RedPathsMetadata)
}
