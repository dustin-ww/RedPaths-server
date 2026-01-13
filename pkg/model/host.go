package model

import (
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/utils"
	"errors"
	"time"
)

type Host struct {

	// Internal
	UID   string   `json:"uid,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`

	// Specific
	Name        string `json:"host.name,omitempty"`
	Hostname    string `json:"host.hostname,omitempty"`
	Description string `json:"host.description,omitempty"`
	// PRIMARY Identifier
	IP                 string `json:"host.ip"`
	IsDomainController bool   `json:"host.is_domain_controller,omitempty"`

	DistinguishedName string `json:"host.distinguished_name"`
	/*	ObjectGUID             string    `json:"object_guid"`
		ObjectSid              string    `json:"object_sid"`
		SAMAccountName         string    `json:"sam_account_name"`*/
	DNSHostName            string    `json:"host.dns_host_name"`
	OperatingSystem        string    `json:"host.operating_system"`
	OperatingSystemVersion string    `json:"host.operating_system_version"`
	LastLogonTimestamp     time.Time `json:"host.last_logon_timestamp"`
	UserAccountControl     int       `json:"host.user_account_control"`

	// Relations
	Runs   []*utils.UIDRef `json:"host.runs,omitempty"`
	HasACL []*utils.UIDRef `json:"host.has_acl,omitempty"`

	// Meta
	RedPathsMetadata core.RedPathsMetadata `json:"-"`
}

func (u *Host) UnmarshalJSON(data []byte) error {
	type Alias Host
	aux := (*Alias)(u)
	return core.UnmarshalWithMetadata(data, aux, &u.RedPathsMetadata)
}

func (u Host) MarshalJSON() ([]byte, error) {
	type Alias Host
	return core.MarshalWithMetadata(Alias(u), u.RedPathsMetadata)
}

type HostBuilder struct {
	host *Host
}

func NewHostBuilder() *HostBuilder {
	return &HostBuilder{
		host: &Host{
			DType: []string{"host"},
		},
	}
}

func (b *HostBuilder) WithServices(services []*utils.UIDRef) *HostBuilder {
	b.host.Runs = services
	return b
}

func (b *HostBuilder) AddService(service *Service) *HostBuilder {
	if service != nil && service.UID != "" {
		b.host.Runs = append(b.host.Runs, &utils.UIDRef{UID: service.UID})
	}
	return b
}

func (b *HostBuilder) AddServiceUID(uid string) *HostBuilder {
	if uid != "" {
		b.host.Runs = append(b.host.Runs, &utils.UIDRef{UID: uid})
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

func (b *HostBuilder) WithDistinguishedName(dn string) *HostBuilder {
	b.host.DistinguishedName = dn
	return b
}

/*
	func (b *HostBuilder) WithObjectGUID(guid string) *HostBuilder {
		b.host = guid
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
*/
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

/*func (b *HostBuilder) WithWhenCreated(created time.Time) *HostBuilder {
	b.host.WhenCreated = created
	return b
}

func (b *HostBuilder) WithWhenChanged(changed time.Time) *HostBuilder {
	b.host.WhenChanged = changed
	return b
}*/

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
