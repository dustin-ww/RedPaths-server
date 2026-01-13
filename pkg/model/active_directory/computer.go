package active_directory

import (
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/utils"
)

type Computer struct {

	// Internal & Interface
	core.BasePrincipal

	// Specific
	Hostname           string `json:"computer.hostname,omitempty"`
	IsDomainController bool   `json:"computer.is_domain_controller,omitempty"`

	// Relations
	Represents    *utils.UIDRef `json:"computer.represents,omitempty"`
	HasDelegation *utils.UIDRef `json:"computer.has_delegation,omitempty"`
	HasSPN        *utils.UIDRef `json:"computer.has_spn,omitempty"`

	// Meta
	RedPathsMetadata core.RedPathsMetadata `json:"-"`
}

func (u *Computer) PrincipalType() PrincipalType {
	return PrincipalComputer
}

func (u *Computer) UnmarshalJSON(data []byte) error {
	type Alias Computer
	aux := (*Alias)(u)
	return core.UnmarshalWithMetadata(data, aux, &u.RedPathsMetadata)
}

func (u Computer) MarshalJSON() ([]byte, error) {
	type Alias Computer
	return core.MarshalWithMetadata(Alias(u), u.RedPathsMetadata)
}
