package model

import (
	"RedPaths-server/pkg/model/utils"
	"errors"
	"time"
)

type Host struct {
	// Internal
	UID                string          `json:"uid,omitempty"`
	IP                 string          `json:"ip"`
	Name               string          `json:"name,omitempty"`
	IsDomainController bool            `json:"is_domain_controller,omitempty"`
	BelongsToDomain    []*utils.UIDRef `json:"belongs_to_domain,omitempty"`
	HasService         []*utils.UIDRef `json:"has_service,omitempty"`
	DType              []string        `json:"dgraph.type,omitempty"`
	InternalCreatedAt  time.Time       `json:"internal_created_at,omitempty"`
	// AD related
	DistinguishedName      string    `json:"distinguishedName"`
	ObjectGUID             string    `json:"objectGUID"`
	ObjectSid              string    `json:"objectSid"`
	SAMAccountName         string    `json:"sAMAccountName"`
	DNSHostName            string    `json:"dNSHostName"`
	OperatingSystem        string    `json:"operatingSystem"`
	OperatingSystemVersion string    `json:"operatingSystemVersion"`
	LastLogonTimestamp     time.Time `json:"lastLogonTimestamp"`
	WhenCreated            time.Time `json:"whenCreated"`
	WhenChanged            time.Time `json:"whenChanged"`
	UserAccountControl     int       `json:"userAccountControl"`

	// History related
	DiscoveredAt time.Time `json:"discovered_at,omitempty"`
	DiscoveredBy string    `json:"discovered_by,omitempty"`
	LastSeenAt   time.Time `json:"last_seen_at,omitempty"`
	LastSeenBy   string    `json:"last_seen_by,omitempty"`
}

type HostBuilder struct {
	host *Host
}

func NewHostBuilder() *HostBuilder {
	return &HostBuilder{
		host: &Host{
			DType:             []string{"host"},
			InternalCreatedAt: time.Now(),
		},
	}
}

func (b *HostBuilder) WithServices(services []*utils.UIDRef) *HostBuilder {
	b.host.HasService = services
	return b
}

func (b *HostBuilder) AddService(service *Service) *HostBuilder {
	if service != nil && service.UID != "" {
		b.host.HasService = append(b.host.HasService, &utils.UIDRef{UID: service.UID})
	}
	return b
}

func (b *HostBuilder) AddServiceUID(uid string) *HostBuilder {
	if uid != "" {
		b.host.HasService = append(b.host.HasService, &utils.UIDRef{UID: uid})
	}
	return b
}

func (b *HostBuilder) AddServiceUIDs(uids ...string) *HostBuilder {
	for _, uid := range uids {
		b.AddServiceUID(uid)
	}
	return b
}

func (b *HostBuilder) WithUID(uid string) *HostBuilder {
	b.host.UID = uid
	return b
}

func (b *HostBuilder) WithIP(ip string) *HostBuilder {
	b.host.IP = ip
	return b
}

func (b *HostBuilder) WithName(name string) *HostBuilder {
	b.host.Name = name
	return b
}

func (b *HostBuilder) AsDomainController() *HostBuilder {
	b.host.IsDomainController = true
	return b
}

func (b *HostBuilder) WithDomain(domainUID string) *HostBuilder {
	if domainUID != "" {
		b.host.BelongsToDomain = []*utils.UIDRef{{UID: domainUID}}
	}
	return b
}
func (b *HostBuilder) WithDistinguishedName(dn string) *HostBuilder {
	b.host.DistinguishedName = dn
	return b
}

func (b *HostBuilder) WithObjectGUID(guid string) *HostBuilder {
	b.host.ObjectGUID = guid
	return b
}

func (b *HostBuilder) WithObjectSid(sid string) *HostBuilder {
	b.host.ObjectSid = sid
	return b
}

func (b *HostBuilder) WithSAMAccountName(name string) *HostBuilder {
	b.host.SAMAccountName = name
	return b
}

func (b *HostBuilder) WithDNSHostName(hostname string) *HostBuilder {
	b.host.DNSHostName = hostname
	return b
}

func (b *HostBuilder) WithOperatingSystem(os string) *HostBuilder {
	b.host.OperatingSystem = os
	return b
}

func (b *HostBuilder) WithOperatingSystemVersion(version string) *HostBuilder {
	b.host.OperatingSystemVersion = version
	return b
}

func (b *HostBuilder) WithLastLogonTimestamp(timestamp time.Time) *HostBuilder {
	b.host.LastLogonTimestamp = timestamp
	return b
}

func (b *HostBuilder) WithWhenCreated(created time.Time) *HostBuilder {
	b.host.WhenCreated = created
	return b
}

func (b *HostBuilder) WithWhenChanged(changed time.Time) *HostBuilder {
	b.host.WhenChanged = changed
	return b
}

func (b *HostBuilder) WithUserAccountControl(uac int) *HostBuilder {
	b.host.UserAccountControl = uac
	return b
}

func (b *HostBuilder) Build() (*Host, error) {
	// Validate required fields
	if b.host.IP == "" {
		return nil, errors.New("IP is a required field")
	}

	return b.host, nil
}
