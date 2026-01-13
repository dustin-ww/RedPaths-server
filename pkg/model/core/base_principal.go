package core

import "RedPaths-server/pkg/model/utils"

type BasePrincipal struct {
	UID         string `json:"uid,omitempty"`
	Name        string `json:"security_principal.name,omitempty"`
	SID         string `json:"security_principal.sid,omitempty"`
	Description string `json:"security_principal.description,omitempty"`

	// Relations
	Capabilities []*utils.UIDRef `json:"security_principal.has_capability,omitempty"`
	Owns         []*utils.UIDRef `json:"security_principal.owns,omitempty"`
	HasACL       *utils.UIDRef   `json:"security_principal.has_acl,omitempty"`

	DType []string `json:"dgraph.type,omitempty"`
}

func (b BasePrincipal) GetUID() string  { return b.UID }
func (b BasePrincipal) GetSID() string  { return b.SID }
func (b BasePrincipal) GetName() string { return b.Name }
