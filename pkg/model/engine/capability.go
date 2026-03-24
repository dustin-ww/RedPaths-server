package engine

import "RedPaths-server/pkg/model/core"

// Capability represents a concrete action an agent can perform,
// derived from AD rights, host attributes, CVEs, or service configs.
type Capability struct {
	// Internal
	UID   string   `json:"uid,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`

	//Specific
	Name         string     `json:"capability.name"`
	Scope        ScopeType  `json:"capability.scope"`
	SourceType   SourceType `json:"capability.source_type"`
	Precondition string     `json:"capability.precondition,omitempty"`
	RiskLevel    int        `json:"capability.risk_level"`

	// Meta
	RedPathsMetadata core.RedPathsMetadata `json:"-"`
}

type ScopeType string

const (
	ScopeDomain  ScopeType = "Domain"
	ScopeHost    ScopeType = "Host"
	ScopeService ScopeType = "Service"
)

type SourceType string

const (
	SourceAD      SourceType = "AD"
	SourceHost    SourceType = "Host"
	SourceCVE     SourceType = "CVE"
	SourceService SourceType = "Service"
)

func (c *Capability) UnmarshalJSON(data []byte) error {
	type Alias Capability
	aux := (*Alias)(c)
	return core.UnmarshalWithMetadata(data, aux, &c.RedPathsMetadata)
}

func (c Capability) MarshalJSON() ([]byte, error) {
	type Alias Capability
	return core.MarshalWithMetadata(Alias(c), c.RedPathsMetadata)
}
