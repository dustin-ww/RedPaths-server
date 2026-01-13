package active_directory

import (
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/utils"
)

type ServiceAccount struct {

	// Internal & Interface
	core.BasePrincipal

	// Specific
	SAMAccountName string          `json:"user.sam_account_name,omitempty"`
	UPN            string          `json:"user.upn,omitempty"`
	IsDisabled     bool            `json:"user.is_disabled,omitempty"`
	IsLocked       bool            `json:"user.is_locked,omitempty"`
	HasSPN         []*utils.UIDRef `json:"service_account.has_spn,omitempty"`
	MemberOf       []*utils.UIDRef `json:"group.member_of,omitempty"`

	// Meta
	RedPathsMetadata core.RedPathsMetadata `json:"-"`
}

func (u *ServiceAccount) PrincipalType() PrincipalType {
	return PrincipalServiceAccount
}

func (u *ServiceAccount) UnmarshalJSON(data []byte) error {
	type Alias ServiceAccount
	aux := (*Alias)(u)
	return core.UnmarshalWithMetadata(data, aux, &u.RedPathsMetadata)
}

func (u ServiceAccount) MarshalJSON() ([]byte, error) {
	type Alias ServiceAccount
	return core.MarshalWithMetadata(Alias(u), u.RedPathsMetadata)
}
