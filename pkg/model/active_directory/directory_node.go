package active_directory

import (
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/utils"
)

type DirectoryNode struct {
	// Internal
	UID   string   `json:"uid,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`

	// Specific
	Name              string `json:"directory_node.name,omitempty"`
	Description       string `json:"directory_node.description,omitempty"`
	DistinguishedName string `json:"directory_node.distinguished_name,omitempty"`
	NodeType          string `json:"directory_node.node_type,omitempty"` // OU | Container
	IsBuiltin         bool   `json:"directory_node.is_builtin,omitempty"`

	// Relations
	Parent     *utils.UIDRef   `json:"directory_node.parent,omitempty"`
	Locates    []*utils.UIDRef `json:"directory_node.locates,omitempty"`
	HasACL     *utils.UIDRef   `json:"directory_node.has_acl,omitempty"`
	HasGPOLink []*utils.UIDRef `json:"directory_node.has_gpo_link,omitempty"`

	// Meta
	RedPathsMetadata core.RedPathsMetadata `json:"-"`
}

func (dn *DirectoryNode) UnmarshalJSON(data []byte) error {
	type Alias DirectoryNode
	aux := (*Alias)(dn)
	return core.UnmarshalWithMetadata(data, aux, &dn.RedPathsMetadata)
}

func (dn DirectoryNode) MarshalJSON() ([]byte, error) {
	type Alias DirectoryNode
	return core.MarshalWithMetadata(Alias(dn), dn.RedPathsMetadata)
}
