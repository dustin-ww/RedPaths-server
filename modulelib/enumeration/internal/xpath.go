package internal

import "fmt"

const (
	hostXPath     = "//host[address/@addr='%s']"
	hostNameXPath = "ports/port[@portid='389']/service[@name='ldap']/@hostname"

	ldapExtraInfoXPath = "ports/port[@portid='%s']/service[@name='ldap']/@extrainfo"
	sslCertCommonName  = "ports/port[@portid='%s']/script[@id='ssl-cert']/table[@key='subject']/elem[@key='commonName']"
	sslCertDomainComp  = "ports/port[@portid='%s']/script[@id='ssl-cert']/table[@key='issuer']/elem[@key='domainComponent']"
	sslCertSANDNS      = "ports/port[@portid='%s']/script[@id='ssl-cert']/table[@key='extensions']/table/elem[contains(@value,'DNS:')]"
)

type XPathBuilder struct {
	hostIP   string
	hostBase string
}

func NewXPathBuilder(ip string) *XPathBuilder {
	return &XPathBuilder{
		hostIP:   ip,
		hostBase: fmt.Sprintf("//host[address/@addr='%s']", ip),
	}
}

func (b *XPathBuilder) Host() string     { return b.hostBase }
func (b *XPathBuilder) Hostname() string { return hostNameXPath }

// ── SSL / LDAP ────────────────────────────────────────────────────────────────

func (b *XPathBuilder) LDAPExtraInfo(portID string) string {
	return fmt.Sprintf("%s/%s", b.hostBase, fmt.Sprintf(ldapExtraInfoXPath, portID))
}
func (b *XPathBuilder) SSLCertCommonName(portID string) string {
	return fmt.Sprintf("%s/%s", b.hostBase, fmt.Sprintf(sslCertCommonName, portID))
}
func (b *XPathBuilder) SSLCertDomainComponent(portID string) string {
	return fmt.Sprintf("%s/%s", b.hostBase, fmt.Sprintf(sslCertDomainComp, portID))
}
func (b *XPathBuilder) SSLCertSANDNS(portID string) string {
	return fmt.Sprintf("%s/%s", b.hostBase, fmt.Sprintf(sslCertSANDNS, portID))
}

// ── Identity ──────────────────────────────────────────────────────────────────

func (b *XPathBuilder) DNSComputerNameRDP() string {
	return fmt.Sprintf("%s/script[@id='rdp-ntlm-info']/table/elem[@key='DNS_Computer_Name']", b.hostBase)
}
func (b *XPathBuilder) DNSComputerNameSQL() string {
	return fmt.Sprintf("%s/script[@id='ms-sql-ntlm-info']/table/table/elem[@key='DNS_Computer_Name']", b.hostBase)
}
func (b *XPathBuilder) NetBIOSComputerNameRDP() string {
	return fmt.Sprintf("%s/script[@id='rdp-ntlm-info']/table/elem[@key='NetBIOS_Computer_Name']", b.hostBase)
}
func (b *XPathBuilder) NetBIOSComputerNameSQL() string {
	return fmt.Sprintf("%s/script[@id='ms-sql-ntlm-info']/table/table/elem[@key='NetBIOS_Computer_Name']", b.hostBase)
}
func (b *XPathBuilder) DNSTreeNameRDP() string {
	return fmt.Sprintf("%s/script[@id='rdp-ntlm-info']/table/elem[@key='DNS_Tree_Name']", b.hostBase)
}
func (b *XPathBuilder) DNSTreeNameSQL() string {
	return fmt.Sprintf("%s/script[@id='ms-sql-ntlm-info']/table/table/elem[@key='DNS_Tree_Name']", b.hostBase)
}

// ── OS ────────────────────────────────────────────────────────────────────────

func (b *XPathBuilder) SMBOS() string {
	return fmt.Sprintf("%s/hostscript/script[@id='smb-os-discovery']/elem[@key='os']", b.hostBase)
}
func (b *XPathBuilder) ServiceOSType() string {
	return fmt.Sprintf("%s/ports/port/service[@ostype!='']/@ostype", b.hostBase)
}
func (b *XPathBuilder) ProductVersionRDP() string {
	return fmt.Sprintf("%s/script[@id='rdp-ntlm-info']/table/elem[@key='Product_Version']", b.hostBase)
}
func (b *XPathBuilder) ProductVersionSQL() string {
	return fmt.Sprintf("%s/script[@id='ms-sql-ntlm-info']/table/table/elem[@key='Product_Version']", b.hostBase)
}
func (b *XPathBuilder) SMBFQDN() string {
	return fmt.Sprintf("%s/hostscript/script[@id='smb-os-discovery']/elem[@key='fqdn']", b.hostBase)
}

// ── DC detection — individual port checks ────────────────────────────────────
// Used by isDomainController() in the module to count matching ports.

// OpenPort returns an XPath that matches if the given port is open.
func (b *XPathBuilder) OpenPort(portID string) string {
	return fmt.Sprintf("%s/ports/port[@portid='%s' and state/@state='open']", b.hostBase, portID)
}

// DCCertOID matches if the ssl-cert DomainController OID value is present.
// The OID 1.3.6.1.4.1.311.20.2 is present on many Windows certs, but only
// DC certs have the value "DomainController" encoded in the cert extensions.
// We check for the OID name element AND the adjacent value element containing
// "DomainController" as a sibling in the same table row.
func (b *XPathBuilder) DCCertOID() string {
	return fmt.Sprintf(
		"%s/ports/port/script[@id='ssl-cert']/table[@key='extensions']/table[elem[@key='name' and text()='1.3.6.1.4.1.311.20.2'] and elem[@key='value' and contains(text(),'DomainController')]]",
		b.hostBase,
	)
}

// LDAPService matches if an LDAP service is running on the given port.
func (b *XPathBuilder) LDAPService(portID string) string {
	return fmt.Sprintf("%s/ports/port[@portid='%s']/service[@name='ldap']", b.hostBase, portID)
}
