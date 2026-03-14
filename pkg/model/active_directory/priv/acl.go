package priv

import (
	"RedPaths-server/pkg/model/core"
)

type ACL struct {

	// Internal
	UID   string   `json:"uid,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`

	//Specific
	Name  string `json:"acl.name,omitempty"`
	Owner string `json:"acl.owner,omitempty"`

	// Meta
	RedPathsMetadata core.RedPathsMetadata `json:"-"`
}

func (acl *ACL) UnmarshalJSON(data []byte) error {
	type Alias ACL
	aux := (*Alias)(acl)
	return core.UnmarshalWithMetadata(data, aux, &acl.RedPathsMetadata)
}

func (acl ACL) MarshalJSON() ([]byte, error) {
	type Alias ACL
	return core.MarshalWithMetadata(Alias(acl), acl.RedPathsMetadata)
}
