package active_directory

import "RedPaths-server/pkg/model/utils"

type BasePrincipal struct {
	ID          string `json:"uid,omitempty"`
	Name        string `json:"security_principal.name,omitempty"`
	SID         string `json:"security_principal.sid,omitempty"`
	Description string `json:"security_principal.description,omitempty"`

	// Relations
	Capabilities []*utils.UIDRef `json:"security_principal.has_capability,omitempty"`
	Owns         []*utils.UIDRef `json:"security_principal.owns,omitempty"`
	HasACL       *utils.UIDRef   `json:"security_principal.has_acl,omitempty"`
}
