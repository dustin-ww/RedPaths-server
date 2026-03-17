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
	"RedPaths-server/pkg/model/utils/assertion"
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

func init() {
	m := &NetworkExplorer{
		configKey: "NetworkExplorer",
	}
	plugin.RegisterPlugin(m)
}

type NetworkExplorer struct {
	configKey string
	services  *rpsdk.Services
	logger    *sse.SSELogger
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
				Metadata:    map[string]interface{}{"method": "icmp,tcp,syn"},
			},
			{
				Type:        "service_enumeration",
				Name:        "Service & Port Enumeration",
				Description: "Identifies open ports, protocols and running services",
				Confidence:  0.9,
				Metadata:    map[string]interface{}{"ports": "1-65535", "versions": true},
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

// Reusable assertion contexts — defined once, shared across all service calls.
// Scan-detected entities get 0.85 confidence; direct topology links get 0.95.
var (
	// assertCtxAD is used when creating an ActiveDirectory or Domain node from scan data.
	assertCtxAD = assertion.Context{
		Confidence: float64Ptr(0.85),
		Status:     strPtr("scan_detected"),
		HighValue:  boolPtr(false),
	}

	// assertCtxHost is used when linking a discovered host to a domain.
	assertCtxHost = assertion.Context{
		Confidence: float64Ptr(0.95),
		Status:     strPtr("scan_detected"),
		HighValue:  boolPtr(false),
	}

	// assertCtxService is used when attaching open-port services to a host.
	assertCtxService = assertion.Context{
		Confidence: float64Ptr(0.90),
		Status:     strPtr("scan_detected"),
		HighValue:  boolPtr(false),
	}
)

func float64Ptr(v float64) *float64 { return &v }
func strPtr(v string) *string       { return &v }
func boolPtr(v bool) *bool          { return &v }

func (n *NetworkExplorer) ExecuteModule(params *input.Parameter, logger *sse.SSELogger) error {
	n.logger = logger
	log.Printf("Executing module key: %s", n.configKey)
	logger.Info("Starting module: %s", n.configKey)

	sse.NewEvent(events.ScanStart).
		WithData("target_network", params.Inputs["network"]).
		WithData("ports", "1-1024").
		Log(logger)

	factory := adapter.GetAdapterFactory()
	scanAdapter, err := factory.GetScanAdapter("nmap")
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get scan adapter: %v", err))
		return err
	}

	scanCtx, scanCancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer scanCancel()

	// For Testing purposes
	targetNetwork := "127.0.0.1"

	log.Println("Using target for network enumeration: " + targetNetwork)

	scanResult, err := scanAdapter.Scan(
		scanCtx,
		scan.WithTargets([]string{targetNetwork}),
		scan.WithPortRange("1-1024"),
		scan.WithServiceScan(),
		scan.WithScriptScan(),
		interfaces.WithTimeout(30*time.Minute),
	)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	nmapResult, ok := scanResult.(*scan.NmapScanResult)
	if !ok {
		return fmt.Errorf("could not map scan result to nmap result: %v", scanResult)
	}

	if err := n.processScanResults(context.Background(), *nmapResult, params); err != nil {
		return fmt.Errorf("processing scan results failed: %w", err)
	}

	sse.NewEvent(events.ScanComplete).
		WithData("timestamp", time.Now().Unix()).
		Log(logger)

	return nil
}

// processScanResults builds the full hierarchy for every discovered host:
// Project → ActiveDirectory → Domain → Host → Services
func (n *NetworkExplorer) processScanResults(ctx context.Context, nmapResult scan.NmapScanResult, params *input.Parameter) error {
	document, err := nmapResult.GetXMLDocument()
	if err != nil {
		return fmt.Errorf("failed to parse nmap XML: %w", err)
	}

	// Local caches avoid redundant DB calls for hosts sharing a domain/forest.
	domainCache := make(map[string]string)          // domainName -> domainUID
	activeDirectoryCache := make(map[string]string) // forestRoot -> adUID

	for _, host := range nmapResult.GetNmapResult().Host {
		ip := host.Address[0].Addr
		xb := internal.NewXPathBuilder(ip)

		// 1. Extract domain name and forest root from nmap fingerprints
		domainName, strategy := n.extractDomainName(document, xb)
		forestRoot := n.extractForestRoot(document, ip)
		if forestRoot == "" {
			forestRoot = domainName
		}

		// 2. Ensure ActiveDirectory (forest level)
		var activeDirectoryUID string
		if forestRoot != "" {
			adUID, cached := activeDirectoryCache[forestRoot]
			if !cached {
				adUID, err = n.ensureActiveDirectory(ctx, params.ProjectUID, forestRoot)
				if err != nil {
					n.logger.Error("failed to ensure AD", "forest", forestRoot, "error", err)
				} else {
					activeDirectoryCache[forestRoot] = adUID
				}
			}
			activeDirectoryUID = adUID
		}

		// 3. Ensure Domain under ActiveDirectory
		var domainUID string
		if domainName != "" {
			uid, cached := domainCache[domainName]
			if !cached {
				uid, err = n.ensureDomain(ctx, activeDirectoryUID, domainName, strategy)
				if err != nil {
					n.logger.Error("failed to ensure domain", "domain", domainName, "error", err)
				} else {
					domainCache[domainName] = uid
				}
			}
			domainUID = uid
		}

		// 4. Build Host
		hostUID, err := n.buildHost(ctx, document, xb, ip, domainUID, params)
		if err != nil {
			n.logger.Error("failed to build host", "ip", ip, "error", err)
			continue
		}

		// 5. Build Services
		n.buildServices(ctx, host, hostUID)
	}

	return nil
}

func (n *NetworkExplorer) ensureActiveDirectory(ctx context.Context, projectUID, forestRoot string) (string, error) {
	//existing, err := n.services.ProjectService.GetActiveDirectoryByForest(ctx, projectUID, forestRoot)
	//if err == nil && existing != nil {
	//	log.Printf("[NetworkExplorer] Reusing existing AD uid=%s forest=%s", existing.UID, forestRoot)
	//	return existing.UID, nil
	//}

	assertCtxAD = assertion.Context{
		Confidence: float64Ptr(0.85),
		Status:     strPtr("scan_detected"),
		HighValue:  boolPtr(false),
	}

	ad := &active_directory.ActiveDirectory{ForestName: forestRoot}
	adUID, err := n.services.ProjectService.AddActiveDirectory(ctx, assertCtxAD, projectUID, ad, n.configKey)
	if err != nil {
		return "", fmt.Errorf("failed to create ActiveDirectory for forest %s: %w", forestRoot, err)
	}

	sse.NewEvent(events.DomainDiscovered).
		WithData("type", "active_directory").
		WithData("forest", forestRoot).
		WithData("timestamp", time.Now().Unix()).
		Log(n.logger)

	log.Printf("[NetworkExplorer] Created AD uid=%s forest=%s", adUID, forestRoot)
	return adUID.Entity.UID, nil
}

func (n *NetworkExplorer) ensureDomain(ctx context.Context, activeDirectoryUID, domainName, strategy string) (string, error) {
	incomingDomain := &active_directory.Domain{Name: domainName}

	result, err := n.services.ActiveDirectoryService.AddDomain(
		ctx,
		activeDirectoryUID,
		incomingDomain,
		assertCtxAD,
		n.configKey,
	)
	if err != nil {
		return "", fmt.Errorf("AddDomain failed for %s: %w", domainName, err)
	}

	sse.NewEvent(events.DomainDiscovered).
		WithData("domain", domainName).
		WithData("strategy", strategy).
		WithData("timestamp", time.Now().Unix()).
		Log(n.logger)

	log.Printf("[NetworkExplorer] Domain ensured: name=%s uid=%s", domainName, result.Entity.UID)
	return result.Entity.UID, nil
}

func (n *NetworkExplorer) buildHost(
	ctx context.Context,
	document *xmlquery.Node,
	xb *internal.XPathBuilder,
	ip, domainUID string,
	params *input.Parameter,
) (string, error) {
	if ip == "" {
		return "", fmt.Errorf("IP cannot be empty")
	}
	if params == nil {
		return "", fmt.Errorf("parameters cannot be nil")
	}

	hostBuilder := model.NewHostBuilder().WithIP(ip)

	if hostNode := xmlquery.FindOne(document, xb.Host()); hostNode != nil {
		if hostnameNode := xmlquery.FindOne(hostNode, xb.Hostname()); hostnameNode != nil {
			hostname := hostnameNode.InnerText()
			log.Printf("[NetworkExplorer] Hostname %s resolved for ip %s", hostname, ip)
			hostBuilder.WithName(hostname)
		}
	}

	host, err := hostBuilder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build host model: %w", err)
	}

	var hostUID string

	if domainUID != "" {
		result, err := n.services.DomainService.AddHost(ctx, assertCtxHost, domainUID, host, n.configKey)
		if err != nil {
			return "", fmt.Errorf("failed to add host to domain: %w", err)
		}
		hostUID = result.Entity.UID
		log.Printf("[NetworkExplorer] Host ip=%s linked to domain uid=%s → host uid=%s", ip, domainUID, hostUID)
	} else {
		hostUID, err = n.services.HostService.CreateWithUnknownDomain(ctx, host, params.ProjectUID, n.configKey)
		if err != nil {
			return "", fmt.Errorf("failed to create host without domain: %w", err)
		}
		log.Printf("[NetworkExplorer] Host ip=%s created without domain → uid=%s", ip, hostUID)
	}

	sse.NewEvent(events.HostDiscovered).
		WithData("ip", ip).
		WithData("timestamp", time.Now().Unix()).
		Log(n.logger)

	return hostUID, nil
}

func (n *NetworkExplorer) buildServices(ctx context.Context, host serializable.Host, hostUID string) {
	if hostUID == "" {
		return
	}

	for _, port := range host.Ports.Port {
		if port.State.State != "open" {
			continue
		}

		serviceBuilder := model.NewServiceBuilder().
			WithName(port.Service.Name).
			WithPort(port.Portid)

		// TODO
		//if port.Service.Product != "" {
		//	serviceBuilder.WithProduct(port.Service.Product)
		//}
		//if port.Service.Version != "" {
		//	serviceBuilder.WithVersion(port.Service.Version)
		//}

		service := serviceBuilder.Build()

		_, err := n.services.HostService.AddService(ctx, assertCtxService, hostUID, service, n.configKey)
		if err != nil {
			log.Printf("[NetworkExplorer] error adding service port=%s host=%s: %v", port.Portid, hostUID, err)
			continue
		}

		sse.NewEvent(events.ServiceDetected).
			WithData("port", port.Portid).
			WithData("service", port.Service.Name).
			WithData("timestamp", time.Now().Unix()).
			Log(n.logger)

		log.Printf("[NetworkExplorer] Service %s:%s added to host uid=%s", port.Service.Name, port.Portid, hostUID)
	}
}

// Domain / Forest extraction

func (n *NetworkExplorer) extractDomainName(document *xmlquery.Node, xb *internal.XPathBuilder) (string, string) {
	ldapPorts := []string{"389", "636", "3268", "3269"}

	strategies := []struct {
		name     string
		getXPath func(port string) string
		extract  func(string) string
	}{
		{
			name:     "LDAP ExtraInfo",
			getXPath: xb.LDAPExtraInfo,
			extract:  extractDomainFromExtrainfo,
		},
		{
			name:     "SSL Cert CommonName",
			getXPath: xb.SSLCertCommonName,
			extract:  extractDomainFromFQDN,
		},
		{
			name:     "SSL Cert SAN-DNS",
			getXPath: xb.SSLCertSANDNS,
			extract: func(text string) string {
				return extractDomainFromFQDN(strings.TrimPrefix(text, "DNS:"))
			},
		},
		{
			name:     "SSL Cert DomainComponent",
			getXPath: xb.SSLCertDomainComponent,
			extract: func(text string) string {
				if text == "" {
					return ""
				}
				return text + ".local"
			},
		},
	}

	for _, s := range strategies {
		for _, port := range ldapPorts {
			nodes, err := xmlquery.QueryAll(document, s.getXPath(port))
			if err != nil || len(nodes) == 0 || nodes[0] == nil {
				continue
			}
			domain := s.extract(nodes[0].InnerText())
			if domain != "" {
				log.Printf("[NetworkExplorer] Domain=%s via strategy=%s port=%s", domain, s.name, port)
				return domain, s.name
			}
		}
	}

	return "", ""
}

func (n *NetworkExplorer) extractForestRoot(document *xmlquery.Node, ip string) string {
	xpaths := []string{
		fmt.Sprintf("//host[address/@addr='%s']//script[@id='rdp-ntlm-info']/table/elem[@key='DNS_Tree_Name']", ip),
		fmt.Sprintf("//host[address/@addr='%s']//script[@id='ms-sql-ntlm-info']/table/table/elem[@key='DNS_Tree_Name']", ip),
	}
	for _, xpath := range xpaths {
		if node := xmlquery.FindOne(document, xpath); node != nil {
			if v := strings.TrimSpace(node.InnerText()); v != "" {
				return v
			}
		}
	}
	return ""
}

// String helpers

func extractDomainFromExtrainfo(info string) string {
	re := regexp.MustCompile(`Domain:\s*([a-zA-Z0-9.-]+)`)
	match := re.FindStringSubmatch(info)
	if len(match) > 1 {
		return strings.Replace(match[1], ".local0.", ".local", 1)
	}
	return ""
}

func extractDomainFromFQDN(fqdn string) string {
	fqdn = strings.TrimPrefix(fqdn, "*.")
	if parts := strings.SplitN(fqdn, ".", 2); len(parts) == 2 {
		return parts[1]
	}
	return ""
}
