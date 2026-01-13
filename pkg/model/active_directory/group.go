package active_directory

import (
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/utils"
)

type Group struct {
	core.BasePrincipal

	// Specific
	GroupScope      string   `json:"group.group_scope,omitempty"`
	GroupType       string   `json:"group.group_type,omitempty"`
	IsPrivileged    bool     `json:"group.is_privileged,omitempty"`
	IsBuiltIn       bool     `json:"group.is_built_in,omitempty"`
	Privileges      []string `json:"group.privileges,omitempty"`
	CanDCSync       bool     `json:"group.can_dcsync,omitempty"`
	CanRDP          bool     `json:"group.can_rdp,omitempty"`
	CanLogonLocally bool     `json:"group.can_logon_locally,omitempty"`

	// Risk Management
	RiskScore   int      `json:"group.risk_score,omitempty"`
	RiskReasons []string `json:"group.risk_reasons,omitempty"`

	// Relations
	HasMember []*utils.UIDRef `json:"group.has_member,omitempty"`
	MemberOf  []*utils.UIDRef `json:"group.member_of,omitempty"`

	// Meta
	RedPathsMetadata core.RedPathsMetadata `json:"-"`
}

func (g Group) PrincipalType() PrincipalType {
	return PrincipalGroup
}

func (g *Group) UnmarshalJSON(data []byte) error {
	type Alias Group
	aux := (*Alias)(g)
	return core.UnmarshalWithMetadata(data, aux, &g.RedPathsMetadata)
}

func (g Group) MarshalJSON() ([]byte, error) {
	type Alias Group
	return core.MarshalWithMetadata(Alias(g), g.RedPathsMetadata)
}
