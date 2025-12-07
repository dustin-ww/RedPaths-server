package model

import (
	"RedPaths-server/pkg/model/utils"
	"time"
)

type Domain struct {
	// Internal
	UID              string         `json:"uid,omitempty"`
	Name             string         `json:"name,omitempty"`
	BelongsToProject utils.UIDRef   `json:"belongs_to_project,omitempty"`
	HasHost          []utils.UIDRef `json:"has_host,omitempty"`
	HasUser          []utils.UIDRef `json:"has_user,omitempty"`
	DType            []string       `json:"dgraph.type,omitempty"`
	// AD related ..
	DNSName             string         `json:"dns_name,omitempty"`
	NetBiosName         string         `json:"net_bios_name,omitempty"`
	DomainGUID          string         `json:"domain_guid,omitempty"`
	DomainSID           string         `json:"domain_sid,omitempty"`
	DomainFunctionLevel string         `json:"domain_function_level,omitempty"`
	ForestFunctionLevel string         `json:"forest_function_level,omitempty"`
	FSMORoleOwners      []string       `json:"fsmo_role_owners,omitempty"`
	SecurityPolicies    utils.UIDRef   `json:"security_policies,omitempty"`
	TrustRelationships  []utils.UIDRef `json:"trust_relationships,omitempty"`
	Created             time.Time      `json:"created,omitempty"`
	LastModified        time.Time      `json:"last_modified,omitempty"`
	LinkedGPOs          []string       `json:"linked_gpos,omitempty"`
	DefaultContainers   []string       `json:"default_containers,omitempty"`
}

type SecurityPolicy struct {
	MinPasswordLength int `json:"min_pwd_length,omitempty"`
	PasswordHistory   int `json:"pwd_history_length,omitempty"`
	LockoutThreshold  int `json:"lockout_threshold,omitempty"`
	LockoutDuration   int `json:"lockout_duration,omitempty"`
}

type Trust struct {
	TrustedDomain string `json:"trusted_domain,omitempty"`
	Direction     string `json:"direction,omitempty"`  // inbound, outbound, bidirectional
	TrustType     string `json:"trust_type,omitempty"` // parent-child, cross-forest, external
	IsTransitive  bool   `json:"is_transitive,omitempty"`
}

type DomainBuilder struct {
	domain Domain
}

func NewDomainBuilder() *DomainBuilder {
	return &DomainBuilder{
		domain: Domain{
			DType:        []string{"Domain"},
			Created:      time.Now(),
			LastModified: time.Now(),
		},
	}
}

func (b *DomainBuilder) WithUID(uid string) *DomainBuilder {
	b.domain.UID = uid
	return b
}

func (b *DomainBuilder) WithName(name string) *DomainBuilder {
	b.domain.Name = name
	return b
}

func (b *DomainBuilder) WithProject(project utils.UIDRef) *DomainBuilder {
	b.domain.BelongsToProject = project
	return b
}

func (b *DomainBuilder) AddHost(host utils.UIDRef) *DomainBuilder {
	b.domain.HasHost = append(b.domain.HasHost, host)
	return b
}

func (b *DomainBuilder) AddUser(user utils.UIDRef) *DomainBuilder {
	b.domain.HasUser = append(b.domain.HasUser, user)
	return b
}

func (b *DomainBuilder) WithDType(dtype []string) *DomainBuilder {
	b.domain.DType = dtype
	return b
}

func (b *DomainBuilder) WithDNSName(dnsName string) *DomainBuilder {
	b.domain.DNSName = dnsName
	return b
}

func (b *DomainBuilder) WithNetBiosName(netBiosName string) *DomainBuilder {
	b.domain.NetBiosName = netBiosName
	return b
}

func (b *DomainBuilder) WithDomainGUID(guid string) *DomainBuilder {
	b.domain.DomainGUID = guid
	return b
}

func (b *DomainBuilder) WithDomainSID(sid string) *DomainBuilder {
	b.domain.DomainSID = sid
	return b
}

func (b *DomainBuilder) WithDomainFunctionLevel(level string) *DomainBuilder {
	b.domain.DomainFunctionLevel = level
	return b
}

func (b *DomainBuilder) WithForestFunctionLevel(level string) *DomainBuilder {
	b.domain.ForestFunctionLevel = level
	return b
}

func (b *DomainBuilder) WithFSMORoleOwners(owners []string) *DomainBuilder {
	b.domain.FSMORoleOwners = owners
	return b
}

func (b *DomainBuilder) AddFSMORoleOwner(owner string) *DomainBuilder {
	b.domain.FSMORoleOwners = append(b.domain.FSMORoleOwners, owner)
	return b
}

func (b *DomainBuilder) WithSecurityPolicies(policies utils.UIDRef) *DomainBuilder {
	b.domain.SecurityPolicies = policies
	return b
}

func (b *DomainBuilder) AddTrustRelationship(trust utils.UIDRef) *DomainBuilder {
	b.domain.TrustRelationships = append(b.domain.TrustRelationships, trust)
	return b
}

func (b *DomainBuilder) WithCreated(created time.Time) *DomainBuilder {
	b.domain.Created = created
	return b
}

func (b *DomainBuilder) WithLastModified(lastModified time.Time) *DomainBuilder {
	b.domain.LastModified = lastModified
	return b
}

func (b *DomainBuilder) WithLinkedGPOs(gpos []string) *DomainBuilder {
	b.domain.LinkedGPOs = gpos
	return b
}

func (b *DomainBuilder) AddLinkedGPO(gpo string) *DomainBuilder {
	b.domain.LinkedGPOs = append(b.domain.LinkedGPOs, gpo)
	return b
}

func (b *DomainBuilder) WithDefaultContainers(containers []string) *DomainBuilder {
	b.domain.DefaultContainers = containers
	return b
}

func (b *DomainBuilder) AddDefaultContainer(container string) *DomainBuilder {
	b.domain.DefaultContainers = append(b.domain.DefaultContainers, container)
	return b
}

func (b *DomainBuilder) Build() Domain {
	return b.domain
}

type SecurityPolicyBuilder struct {
	policy SecurityPolicy
}

func NewSecurityPolicyBuilder() *SecurityPolicyBuilder {
	return &SecurityPolicyBuilder{}
}

func (b *SecurityPolicyBuilder) WithMinPasswordLength(length int) *SecurityPolicyBuilder {
	b.policy.MinPasswordLength = length
	return b
}

func (b *SecurityPolicyBuilder) WithPasswordHistory(length int) *SecurityPolicyBuilder {
	b.policy.PasswordHistory = length
	return b
}

func (b *SecurityPolicyBuilder) WithLockoutThreshold(threshold int) *SecurityPolicyBuilder {
	b.policy.LockoutThreshold = threshold
	return b
}

func (b *SecurityPolicyBuilder) WithLockoutDuration(duration int) *SecurityPolicyBuilder {
	b.policy.LockoutDuration = duration
	return b
}

func (b *SecurityPolicyBuilder) Build() SecurityPolicy {
	return b.policy
}

type TrustBuilder struct {
	trust Trust
}

func NewTrustBuilder() *TrustBuilder {
	return &TrustBuilder{}
}

func (b *TrustBuilder) WithTrustedDomain(domain string) *TrustBuilder {
	b.trust.TrustedDomain = domain
	return b
}

func (b *TrustBuilder) WithDirection(direction string) *TrustBuilder {
	b.trust.Direction = direction
	return b
}

func (b *TrustBuilder) WithTrustType(trustType string) *TrustBuilder {
	b.trust.TrustType = trustType
	return b
}

func (b *TrustBuilder) WithTransitivity(isTransitive bool) *TrustBuilder {
	b.trust.IsTransitive = isTransitive
	return b
}

func (b *TrustBuilder) Build() Trust {
	return b.trust
}
