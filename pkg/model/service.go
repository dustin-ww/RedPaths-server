package model

import (
	"RedPaths-server/pkg/model/utils"
	"time"
)

type Service struct {
	// Internal
	UID   string   `json:"uid,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`

	// Specific
	Name string `json:"service.name,omitempty"`
	Port string `json:"service.port,omitempty"`

	// AD related
	SPNs                    []string `json:"spns,omitempty"`
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

	// Reverse
	DeployedOnHost *utils.UIDRef `json:"deployed_on_host,omitempty"`

	// History related
	DiscoveredAt time.Time `json:"discovered_at,omitempty"`
	DiscoveredBy string    `json:"discovered_by,omitempty"`
	LastSeenAt   time.Time `json:"last_seen_at,omitempty"`
	LastSeenBy   string    `json:"last_seen_by,omitempty"`
}

type ServiceBuilder struct {
	service Service
}

func NewServiceBuilder() *ServiceBuilder {
	return &ServiceBuilder{service: Service{}}
}

func (b *ServiceBuilder) WithUID(uid string) *ServiceBuilder {
	b.service.UID = uid
	return b
}

func (b *ServiceBuilder) WithName(name string) *ServiceBuilder {
	b.service.Name = name
	return b
}

func (b *ServiceBuilder) WithPort(port string) *ServiceBuilder {
	b.service.Port = port
	return b
}

func (b *ServiceBuilder) WithDType(dType []string) *ServiceBuilder {
	b.service.DType = dType
	return b
}

func (b *ServiceBuilder) WithRunsOnHosts(hosts *utils.UIDRef) *ServiceBuilder {
	b.service.DeployedOnHost = hosts
	return b
}

func (b *ServiceBuilder) WithSPNs(spns []string) *ServiceBuilder {
	b.service.SPNs = spns
	return b
}

func (b *ServiceBuilder) WithAccountName(accountName string) *ServiceBuilder {
	b.service.AccountName = accountName
	return b
}

func (b *ServiceBuilder) WithSID(sid string) *ServiceBuilder {
	b.service.SID = sid
	return b
}

func (b *ServiceBuilder) WithPasswordLastSet(timestamp int64) *ServiceBuilder {
	b.service.PasswordLastSet = timestamp
	return b
}

func (b *ServiceBuilder) WithConstrainedDelegation(delegation []string) *ServiceBuilder {
	b.service.ConstrainedDelegation = delegation
	return b
}

func (b *ServiceBuilder) WithUnconstrainedDelegation(delegation bool) *ServiceBuilder {
	b.service.UnconstrainedDelegation = delegation
	return b
}

func (b *ServiceBuilder) WithDNSHostName(hostName string) *ServiceBuilder {
	b.service.DNSHostName = hostName
	return b
}

func (b *ServiceBuilder) WithTimestamps(created, changed string) *ServiceBuilder {
	b.service.WhenCreated = created
	b.service.WhenChanged = changed
	return b
}

func (b *ServiceBuilder) WithLastLogon(timestamp int64) *ServiceBuilder {
	b.service.LastLogon = timestamp
	return b
}

func (b *ServiceBuilder) WithOperatingSystem(os string) *ServiceBuilder {
	b.service.OperatingSystem = os
	return b
}

func (b *ServiceBuilder) WithDescription(desc string) *ServiceBuilder {
	b.service.Description = desc
	return b
}

func (b *ServiceBuilder) MarkAsLegacy() *ServiceBuilder {
	b.service.IsLegacy = true
	return b
}

func (b *ServiceBuilder) WithDelegationFlags(trusted, accountDelegated bool) *ServiceBuilder {
	b.service.TrustedForDelegation = trusted
	b.service.AccountCanBeDelegated = accountDelegated
	return b
}

func (b *ServiceBuilder) Build() Service {
	return b.service
}
