package enumeration

import (
	"RedPaths-server/modulelib/enumeration/internal"
	"RedPaths-server/pkg/adapter"
	"RedPaths-server/pkg/adapter/scan"
	"RedPaths-server/pkg/adapter/serializable"
	"RedPaths-server/pkg/interfaces"
	"RedPaths-server/pkg/interfaces/module"
	"RedPaths-server/pkg/model"
	"RedPaths-server/pkg/model/active_directory"
	"RedPaths-server/pkg/model/events"
	"RedPaths-server/pkg/model/redpaths/input"
	"RedPaths-server/pkg/model/rpsdk"
	plugin "RedPaths-server/pkg/module_exec"
	"RedPaths-server/pkg/sse"
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/antchfx/xmlquery"
)

// INITIALIZE MODULE AS REDPATHS PLUGIN
func init() {
	module := &NetworkExplorer{
		configKey: "NetworkExplorer",
	}
	plugin.RegisterPlugin(module)
}

type NetworkExplorer struct {
	// Internal
	configKey string
	// Services
	services *rpsdk.Services
	// Tool Adaptera
	logger *sse.SSELogger
}

func (n *NetworkExplorer) SetServices(services *rpsdk.Services) {
	n.services = services
}

func (n *NetworkExplorer) GetMetadata() *interfaces.ModuleMetadata {
	return &interfaces.ModuleMetadata{
		Name:        "NetworkEnumeration",
		Category:    "enumeration",
		Description: "Performs network-wide enumeration using Nmap to identify live hosts, open ports and services",
		Prerequisites: []*module.Prerequisite{
			{
				Type:        module.PrereqNetworkAccess,
				Name:        "Network Reachability",
				Description: "Target network must be reachable from the scanning interface",
				Required:    true,
				Conditions:  "network.reachable = true",
			},
			{
				Type:        module.PrereqKnowledge,
				Name:        "Target Network Range",
				Description: "CIDR or IP range to scan",
				Required:    true,
				Conditions:  "target.cidr != null",
			},
		},
		Provides: []*module.Capability{
			{
				Type:        "network_discovery",
				Name:        "Host Discovery",
				Description: "Discovers live hosts in the target network",
				Confidence:  0.95,
				Metadata: map[string]interface{}{
					"method": "icmp,tcp,syn",
				},
			},
			{
				Type:        "service_enumeration",
				Name:        "Service & Port Enumeration",
				Description: "Identifies open ports, protocols and running services",
				Confidence:  0.9,
				Metadata: map[string]interface{}{
					"ports":    "1-65535",
					"versions": true,
				},
			},
			{
				Type:        "os_fingerprinting",
				Name:        "Operating System Detection",
				Description: "Attempts to identify the operating system of discovered hosts",
				Confidence:  0.75,
			},
		},
		Risk:       3,
		Stealth:    2,
		Complexity: 3,
	}

}

func (n *NetworkExplorer) ConfigKey() string {
	return n.configKey
}

// ExecuteModule method called by module loader
func (n *NetworkExplorer) ExecuteModule(params *input.Parameter, logger *sse.SSELogger) error {
	n.logger = logger

	// Log start of module execution
	log.Printf("Executing module key: %s", n.ConfigKey)
	logger.Info("Starting module: %s", n.ConfigKey)

	sse.NewEvent(events.ScanStart).
		WithData("target_network", params.Inputs["network"]).
		WithData("ports", "1-1024").
		Log(logger)

	factory := adapter.GetAdapterFactory()
	scanTool := "nmap"
	scanAdapter, err := factory.GetScanAdapter(scanTool)

	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get scan adapter: %v", err))
		return err
	}

	scanCtx, scanCancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer scanCancel()

	targetNetwork := "127.0.0.1"
	log.Println("Using Target for network enumeration: " + targetNetwork)

	scanResult, err := scanAdapter.Scan(
		scanCtx,
		scan.WithTargets([]string{targetNetwork}),
		scan.WithPortRange("1-1024"),
		scan.WithServiceScan(),
		scan.WithScriptScan(),
		interfaces.WithTimeout(30*time.Minute),
	)

	if nmapResult, ok := scanResult.(*scan.NmapScanResult); ok {
		// entrypoint to evaluate scan results
		n.processScanResults(*nmapResult, params)
	} else {
		return fmt.Errorf("could not map scan result to nmap result: %v", scanResult)
	}

	// Log completion of module
	sse.NewEvent(events.ScanComplete).
		WithData("timestamp", time.Now().Unix()).
		Log(logger)

	return nil
}

func (n *NetworkExplorer) processScanResults(nmapResult scan.NmapScanResult, params *input.Parameter) {
	// Iterate through each host from nmap output XML
	// Structure: Build domain -> Build host -> Build services
	for _, host := range nmapResult.GetNmapResult().Host {

		var domainUID string
		var hostUID string

		// try to build/get domain from host
		domainUID, err := n.tryToBuildDomain(nmapResult, host.Address[0].Addr, params)
		if err != nil {
			n.logger.Error("failed to build domain", "ip", host.Address[0].Addr, "error", err)
		}

		// BUILD HOST
		hostUID, err = n.buildHost(nmapResult, host.Address[0].Addr, params, domainUID)
		if err != nil {
			n.logger.Error("failed to build host", "ip", host.Address[0].Addr, "error", err)
		}

		// BUILD SERVICES
		n.buildServices(host, hostUID)

	}
}

func (n *NetworkExplorer) buildHost(nmapResult scan.NmapScanResult, ip string, params *input.Parameter, domainUID string) (string, error) {
	if ip == "" {
		return "", fmt.Errorf("IP cannot be empty")
	}

	if params == nil {
		return "", fmt.Errorf("parameters cannot be nil")
	}

	document, err := nmapResult.GetXMLDocument()
	if err != nil {
		n.logger.Error("failed to parse nmap XML document", "ip", ip, "error", err)
		return "", err
	}

	xpathBuilder := internal.NewXPathBuilder(ip)
	hostBuilder := model.NewHostBuilder()
	hostBuilder.WithIP(ip)

	// Extract hostname
	hostPath := xpathBuilder.Host()
	if node := xmlquery.FindOne(document, hostPath); node != nil {
		hostnamePath := xpathBuilder.Hostname()
		if hostnameNode := xmlquery.FindOne(node, hostnamePath); hostnameNode != nil {
			hostname := hostnameNode.InnerText()
			log.Printf("FOUND HOSTNAME: " + hostname)
			hostBuilder.WithName(hostname)
		} else {
			log.Printf("NO HOSTNAME FOUND")
		}
	}

	host, err := hostBuilder.Build()

	sse.NewEvent(events.HostDiscovered).
		WithData("timestamp", time.Now().Unix()).
		Log(n.logger)

	if err != nil {
		return "", fmt.Errorf("failed to build host: %v", err)
	}

	ctx := context.Background()

	// Create the host in the appropriate domain
	var hostUID string
	if domainUID != "" {
		log.Printf("Using domain UID: %s", domainUID)
		hostUID, err = n.services.DomainService.AddHost(ctx, domainUID, host, n.ConfigKey())
		if err != nil {
			return "", fmt.Errorf("failed to add host to domain: %v", err)
		}
	} else {
		log.Printf("Using no domain UID because UID is: %s", domainUID)
		hostUID, err = n.services.HostService.CreateWithUnknownDomain(ctx, host, params.ProjectUID, n.ConfigKey())
		if err != nil {
			return "", fmt.Errorf("failed to create host: %v", err)
		}
	}

	return hostUID, nil
}

func (n *NetworkExplorer) tryToBuildDomain(nmapResult scan.NmapScanResult, ip string, params *input.Parameter) (string, error) {
	if ip == "" {
		return "", fmt.Errorf("IP cannot be empty")
	}

	if params == nil {
		return "", fmt.Errorf("parameters cannot be nil")
	}

	document, err := nmapResult.GetXMLDocument()
	if err != nil {
		return "", fmt.Errorf("failed to parse nmap XML document: %v", err)
	}

	xpathBuilder := internal.NewXPathBuilder(ip)

	ports := []string{"389", "636", "3268", "3269"}

	strategies := []struct {
		name     string
		getXPath func(port string) string
		extract  func(string) string
	}{
		{
			name:     "LDAP Info",
			getXPath: xpathBuilder.LDAPExtraInfo,
			extract:  extractDomainFromExtrainfo,
		},
		{
			name:     "Certificate Common Name",
			getXPath: xpathBuilder.SSLCertCommonName,
			extract:  extractDomainFromFQDN,
		},
		{
			name:     "SAN-DNS",
			getXPath: xpathBuilder.SSLCertSANDNS,
			extract: func(text string) string {
				return extractDomainFromFQDN(strings.TrimPrefix(text, "DNS:"))
			},
		},
		{
			name:     "issuer domainComponent",
			getXPath: xpathBuilder.SSLCertDomainComponent,
			extract: func(text string) string {
				return text + ".local"
			},
		},
	}

	var domain string
	var usedStrategy string

	for _, strategy := range strategies {
		for _, port := range ports {
			xpath := strategy.getXPath(port)
			//log.Printf("Trying XPath", "strategy", strategy.name, "port", port, "xpath", xpath)

			nodes, err := xmlquery.QueryAll(document, xpath)
			if err != nil {
				//	log.Printf("XPath error", "strategy", strategy.name, "xpath", xpath, "error", err)
				continue
			}

			if len(nodes) > 0 && nodes[0] != nil {
				domain = strategy.extract(nodes[0].InnerText())
				if domain != "" {
					usedStrategy = strategy.name
					break
				}
			}
		}
		if domain != "" {
			break
		}
	}

	if domain == "" {
		return "", fmt.Errorf("could not determine domain from nmap results")
	}

	domainBuilder := active_directory.NewDomainBuilder()
	domainBuilder.WithName(domain)
	builtDomain := domainBuilder.Build()

	ctx := context.Background()

	sse.NewEvent(events.DomainDiscovered).
		WithData("timestamp", time.Now().Unix()).
		WithData("strategy", usedStrategy).
		Log(n.logger)

	log.Printf("PROJECT UIIDDDDDD: " + params.ProjectUID)
	addedDomainID, err := n.services.ProjectService.AddDomain(ctx, params.ProjectUID, &builtDomain, n.ConfigKey())
	if err != nil {
		log.Printf("failed to add domain to project: %v", err)
		n.logger.Error("failed to create domain for host",
			"domain", domain,
			"ip", ip,
			"project", params.ProjectUID,
			"error", err)
		return "", err
	}

	n.logger.Info("created domain for host",
		"domain", domain,
		"ip", ip,
		"strategy", usedStrategy,
		"domainID", addedDomainID)

	log.Printf("added domain: %s with uid %s", domain, addedDomainID)
	return addedDomainID, nil
}

func extractDomainFromExtrainfo(info string) string {
	re := regexp.MustCompile(`Domain:\s*([a-zA-Z0-9.-]+)`)
	match := re.FindStringSubmatch(info)
	if len(match) > 1 {
		return strings.Replace(match[1], ".local0.", ".local", 1)
	}
	return ""
}

func extractDomainFromFQDN(fqdn string) string {
	if parts := strings.SplitN(fqdn, ".", 2); len(parts) == 2 {
		return parts[1]
	}
	return ""
}

func (n *NetworkExplorer) isHostDomainDiscovered(result serializable.Host) bool {
	return true
}

// BUILD DOMAIN

func (n *NetworkExplorer) buildDomainName(nmapResult scan.NmapScanResult) string {
	ports := []string{"389", "636", "3268", "3269"}

	ipAddr := "10.3.10.10"

	for _, port := range ports {
		xpath := fmt.Sprintf("//host/address[@addr='%s']/../ports/port[@portid='%s']/script[@id='ssl-cert']/table[@key='subject']/elem[@key='commonName']",
			ipAddr, port)

		value, err := nmapResult.QueryValue(xpath)
		if err == nil && value != "" {
			return value
		}

	}

	return ""
}

func (n *NetworkExplorer) buildServices(host serializable.Host, hostID string) {
	for _, port := range host.Ports.Port {
		if port.State.State == "open" {
			serviceBuilder := model.NewServiceBuilder()
			serviceBuilder.WithName(port.Service.Name)
			serviceBuilder.WithPort(port.Portid)
			service := serviceBuilder.Build()
			sse.NewEvent(events.ServiceDetected).
				WithData("timestamp", time.Now().Unix()).
				WithData("port", port.Portid).
				Log(n.logger)
			uid, err := n.services.HostService.AddService(context.Background(), hostID, service)
			if err != nil {
				log.Printf("error creating service in module network explorer: %v", err)
				return
			}

			log.Printf("created service in module network explorer: %s", uid)
		}

	}

}

func (n *NetworkExplorer) isDomainController(nmapResult scan.NmapScanResult, ip string) bool {
	document, err := nmapResult.GetXMLDocument()
	if err != nil {
		n.logger.Error(fmt.Sprintf("[Network Explorer] Error detecting domain controller: %v", err))
		return false
	}

	xpathBuilder := internal.NewXPathBuilder(ip)

	dcXPath := xpathBuilder.IsDomainController()

	if node := xmlquery.FindOne(document, dcXPath); node != nil {
		return true
	}

	dcPorts := map[string]bool{
		"53":   true, // DNS
		"88":   true, // Kerberos
		"389":  true, // LDAP
		"445":  true, // SMB
		"464":  true, // Kerberos password history
		"636":  true, // LDAPS
		"3268": true, // Global Catalog
		"3269": true, // Global Catalog over SSL
	}

	matchCount := 0

	for portID := range dcPorts {
		portXPath := fmt.Sprintf("%s/ports/port[@portid='%s' and state/@state='open']", xpathBuilder.Host(), portID)
		if node := xmlquery.FindOne(document, portXPath); node != nil {
			matchCount++

			if portID == "88" {
				return true
			}
		}
	}

	serviceTypes := []string{"ldap", "kerberos", "msrpc"}
	for _, serviceType := range serviceTypes {
		serviceXPath := fmt.Sprintf("%s/ports/port[state/@state='open']/service[@name='%s']", xpathBuilder.Host(), serviceType)
		nodes := xmlquery.Find(document, serviceXPath)
		matchCount += len(nodes)
	}

	return matchCount >= 3
}
