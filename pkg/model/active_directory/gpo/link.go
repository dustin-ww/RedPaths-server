package gpo

import (
	"RedPaths-server/pkg/model/core"
)

type Link struct {

	// Internal
	UID   string   `json:"uid,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`

	//Specific
	LinkOrder  int  `json:"gpo.link_order"`
	IsEnforced bool `json:"gpo.is_enforced,omitempty"`
	IsEnabled  bool `json:"gpo.is_enabled,omitempty"`

	// Relations
	/*	Links []*utils.UIDRef `json:"active_directory.has_domain,omitempty"`

		Grants
		Contains
		HasAcls*/

	// Meta
	RedPathsMetadata core.RedPathsMetadata `json:"-"`
}

func (gl *Link) UnmarshalJSON(data []byte) error {
	type Alias Link
	aux := (*Alias)(gl)
	return core.UnmarshalWithMetadata(data, aux, &gl.RedPathsMetadata)
}

func (gl Link) MarshalJSON() ([]byte, error) {
	type Alias Link
	return core.MarshalWithMetadata(Alias(gl), gl.RedPathsMetadata)
}
