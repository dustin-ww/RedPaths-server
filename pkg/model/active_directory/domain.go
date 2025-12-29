package active_directory

import (
	"RedPaths-server/pkg/model"
	"RedPaths-server/pkg/model/redpaths/history"
	"RedPaths-server/pkg/model/utils"
	"log"
	"time"
)

type Domain struct {
	// Internal
	UID         string   `json:"uid,omitempty"`
	Name        string   `json:"domain.name,omitempty"`
	Description string   `json:"domain.description,omitempty"`
	DType       []string `json:"dgraph.type,omitempty"`

	// AD related
	DNSName               string          `json:"domain.dns_name,omitempty"`
	NetBiosName           string          `json:"domain.netbios_name,omitempty"`
	DomainGUID            string          `json:"domain.domain_guid,omitempty"`
	DomainSID             string          `json:"domnain.domain_sid,omitempty"`
	DomainFunctionalLevel string          `json:"domain.functional_level,omitempty"`
	ForestFunctionalLevel string          `json:"domain.forest_functional_level,omitempty"`
	FSMORoleOwners        []string        `json:"domain.fsmo_role_owners,omitempty"`
	LinkedGPOs            []string        `json:"domain.linked_gpos,omitempty"`
	DefaultContainers     []string        `json:"domain.default_containers,omitempty"`
	ContainsDirNodes      []*utils.UIDRef `json:"domain.contains_dir_nodes,omitempty"`
	HasPrincipals         []*utils.UIDRef `json:"domain.has_principals,omitempty"`
	HasTrust              []*utils.UIDRef `json:"domain.has_trust,omitempty"`
	HasACL                []*utils.UIDRef `json:"domain.has_acl,omitempty"`
	HasGPOLink            []*utils.UIDRef `json:"domain.has_gpo_link,omitempty"`
	HasSecurityPolicy     *utils.UIDRef   `json:"domain.has_security_policy,omitempty"`

	RedPathsMetadata model.RedPathsMetadata `json:"-"`
}

func (d *Domain) UnmarshalJSON(data []byte) error {
	type Alias Domain
	aux := (*Alias)(d)
	return model.UnmarshalWithMetadata(data, aux, &d.RedPathsMetadata)
}

func (d Domain) MarshalJSON() ([]byte, error) {
	type Alias Domain
	return model.MarshalWithMetadata(Alias(d), d.RedPathsMetadata)
}

func (d *Domain) EntityUID() string {
	return d.UID
}

func (d *Domain) EntityType() string {
	return "Domain"
}

func (d *Domain) Diff(other any) []history.FieldChange {
	o, ok := other.(*Domain)
	if !ok || o == nil {
		return nil
	}

	var changes []history.FieldChange

	if d.Name != o.Name {
		log.Println("NOT THE SAME")
		changes = append(changes, history.FieldChange{
			Field:    "name",
			OldValue: d.Name,
			NewValue: o.Name,
		})
	}

	if d.DNSName != o.DNSName {
		changes = append(changes, history.FieldChange{
			Field:    "description",
			OldValue: d.DNSName,
			NewValue: o.DNSName,
		})
	}

	return changes
}

type DomainBuilder struct {
	domain Domain
}

func NewDomainBuilder() *DomainBuilder {
	return &DomainBuilder{
		domain: Domain{
			DType:        []string{"Domain"},
			CreatedAt:    time.Now(),
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
	b.domain.CreatedAt = created
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
