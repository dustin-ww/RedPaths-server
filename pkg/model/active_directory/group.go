package active_directory

import (
	"RedPaths-server/pkg/model"
	"RedPaths-server/pkg/model/utils"
)

type Group struct {
	BasePrincipal

	// internal
	UID         string   `json:"uid,omitempty"`
	Name        string   `json:"group.name,omitempty"`
	Description string   `json:"group.description,omitempty"`
	DType       []string `json:"dgraph.type,omitempty"`

	// AD-related
	GroupScope      string `json:"group.group_scope,omitempty"`
	GroupType       string `json:"group.group_type,omitempty"`
	IsPrivileged    bool   `json:"group.is_privileged,omitempty"`
	IsBuiltIn       bool   `json:"group.is_built_in,omitempty"`
	CanDCSync       bool   `json:"group.can_dcsync,omitempty"`
	CanRDP          bool   `json:"group.can_rdp,omitempty"`
	CanLogonLocally bool   `json:"group.can_logon_locally,omitempty"`

	// Risk Management
	RiskScore   int      `json:"group.risk_score,omitempty"`
	RiskReasons []string `json:"group.risk_reasons,omitempty"`

	// Relations
	HasMember []*utils.UIDRef `json:"group.has_member,omitempty"`
	MemberOf  []*utils.UIDRef `json:"group.member_of,omitempty"`

	RedPathsMetadata model.RedPathsMetadata `json:"-"`
}

func (g Group) PrincipalType() PrincipalType {
	return PrincipalGroup
}

func (g *Group) UnmarshalJSON(data []byte) error {
	type Alias Group
	aux := (*Alias)(g)
	return model.UnmarshalWithMetadata(data, aux, &g.RedPathsMetadata)
}

func (g Group) MarshalJSON() ([]byte, error) {
	type Alias Group
	return model.MarshalWithMetadata(Alias(g), g.RedPathsMetadata)
}
