package model

import (
	"RedPaths-server/pkg/model/core"
)

type Service struct {
	// Internal
	UID   string   `json:"uid,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`

	// Specific
	Name string `json:"service.name,omitempty"`
	Port string `json:"service.port,omitempty"`

	// AD related
	SPNs                    []string `json:"service.spns,omitempty"`
	AccountName             string   `json:"account_name,omitempty"`
	SID                     string   `json:"sid,omitempty"`
	PasswordLastSet         int64    `json:"password_last_set,omitempty"`
	ConstrainedDelegation   []string `json:"constrained_delegation,omitempty"`
	UnconstrainedDelegation bool     `json:"unconstrained_delegation,omitempty"`
	DNSHostName             string   `json:"dns_host_name,omitempty"`
	WhenCreated             string   `json:"when_created,omitempty"`
	WhenChanged             string   `json:"when_changed,omitempty"`
	LastLogon               int64    `json:"last_logon,omitempty"`
	OperatingSystem         string   `json:"operating_system,omitempty"`
	Description             string   `json:"description,omitempty"`
	IsLegacy                bool     `json:"is_legacy,omitempty"`
	TrustedForDelegation    bool     `json:"trusted_for_delegation,omitempty"`
	AccountCanBeDelegated   bool     `json:"account_can_be_delegated,omitempty"`

	// Meta
	RedPathsMetadata core.RedPathsMetadata `json:"-"`
}

func (s *Service) UnmarshalJSON(data []byte) error {
	type Alias Service
	aux := (*Alias)(s)
	return core.UnmarshalWithMetadata(data, aux, &s.RedPathsMetadata)
}

func (s Service) MarshalJSON() ([]byte, error) {
	type Alias Service
	return core.MarshalWithMetadata(Alias(s), s.RedPathsMetadata)
}
