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
	"RedPaths-server/pkg/service/upsert"
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
	m := &NetworkExplorer{configKey: "NetworkExplorer"}
	plugin.RegisterPlugin(m)
}

type NetworkExplorer struct {
	configKey string
	services  *rpsdk.Services
	logger    *sse.SSELogger
}

func (n *NetworkExplorer) SetServices(services *rpsdk.Services) { n.services = services }
func (n *NetworkExplorer) ConfigKey() string                    { return n.configKey }

func (n *NetworkExplorer) GetMetadata() *interfaces.ModuleMetadata {
	return &interfaces.ModuleMetadata{
		Name:        "NetworkEnumeration",
		Category:    "enumeration",
		Description: "Performs network-wide enumeration using Nmap to identify live hosts, open ports and services",
		Prerequisites: []*module.Prerequisite{
			{Type: module.PrereqNetworkAccess, Name: "Network Reachability", Required: true, Conditions: "network.reachable = true"},
			{Type: module.PrereqKnowledge, Name: "Target Network Range", Required: true, Conditions: "target.cidr != null"},
		},
		Provides: []*module.Capability{
			{Type: "network_discovery", Name: "Host Discovery", Confidence: 0.95, Metadata: map[string]interface{}{"method": "icmp,tcp,syn"}},
			{Type: "service_enumeration", Name: "Service & Port Enumeration", Confidence: 0.9, Metadata: map[string]interface{}{"ports": "1-65535", "versions": true}},
			{Type: "os_fingerprinting", Name: "Operating System Detection", Confidence: 0.75},
		},
		Risk: 3, Stealth: 2, Complexity: 3,
	}
}

// ── Assertion contexts ────────────────────────────────────────────────────────

var (
	assertCtxAD = assertion.Context{
		Confidence: float64Ptr(0.85),
		Status:     strPtr("scan_detected"),
		HighValue:  boolPtr(false),
	}
	assertCtxHost = assertion.Context{
		Confidence: float64Ptr(0.95),
		Status:     strPtr("scan_detected"),
		HighValue:  boolPtr(false),
	}
	assertCtxService = assertion.Context{
		Confidence: float64Ptr(0.90),
		Status:     strPtr("scan_detected"),
		HighValue:  boolPtr(false),
	}
)

func float64Ptr(v float64) *float64 { return &v }
func strPtr(v string) *string       { return &v }
func boolPtr(v bool) *bool          { return &v }

// ── DC port scoring ───────────────────────────────────────────────────────────

type dcPortRule struct {
	port       string
	score      int
	definitive bool
}

var dcPortRules = []dcPortRule{
	{port: "88", score: 3, definitive: true},
	{port: "389", score: 2},
	{port: "636", score: 2},
	{port: "3268", score: 2},
	{port: "3269", score: 2},
	{port: "464", score: 1},
	{port: "53", score: 1},
	{port: "445", score: 0},
}

const dcScoreThreshold = 3

func isDomainController(document *xmlquery.Node, xb *internal.XPathBuilder) (bool, int, []string) {
	score := 0
	var matched []string

	for _, rule := range dcPortRules {
		if xmlquery.FindOne(document, xb.OpenPort(rule.port)) != nil {
			matched = append(matched, rule.port)
			score += rule.score
			if rule.definitive {
				return true, score, matched
			}
		}
	}

	return score >= dcScoreThreshold, score, matched
}

// ── ExecuteModule ─────────────────────────────────────────────────────────────

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

	targetNetwork := "127.0.0.1" // Testing only

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

	log.Printf("[DEBUG] ExecuteModule: scan done, starting processScanResults")

	if err := n.processScanResults(context.Background(), *nmapResult, params); err != nil {
		return fmt.Errorf("processing scan results failed: %w", err)
	}

	sse.NewEvent(events.ScanComplete).WithData("timestamp", time.Now().Unix()).Log(logger)
	return nil
}

// ── processScanResults ────────────────────────────────────────────────────────

func (n *NetworkExplorer) processScanResults(
	ctx context.Context,
	nmapResult scan.NmapScanResult,
	params *input.Parameter,
) error {
	document, err := nmapResult.GetXMLDocument()
	if err != nil {
		return fmt.Errorf("failed to parse nmap XML: %w", err)
	}

	hosts := nmapResult.GetNmapResult().Host
	log.Printf("[DEBUG] processScanResults: hostCount=%d projectUID=%s",
		len(hosts), params.ProjectUID)

	if len(hosts) == 0 {
		log.Printf("[DEBUG] WARNING: host list empty — nothing to process")
		return nil
	}

	adCache := make(map[string]string)
	domainCache := make(map[string]string)

	for i, host := range hosts {
		ip := host.Address[0].Addr
		log.Printf("[DEBUG] ── host[%d] ip=%s ──────────────────────────", i, ip)

		xb := internal.NewXPathBuilder(ip)

		domainName, strategy := n.extractDomainName(document, xb)
		forestRoot := n.extractForestRoot(xb, document)
		if forestRoot == "" {
			forestRoot = domainName
		}

		log.Printf("[DEBUG] host[%d] ip=%s domainName=%q forestRoot=%q strategy=%q",
			i, ip, domainName, forestRoot, strategy)

		// ── ActiveDirectory (Forest) ──────────────────────────────────────────
		var adUID string
		if forestRoot != "" {
			if cached, ok := adCache[forestRoot]; ok {
				adUID = cached
				log.Printf("[DEBUG] host[%d] AD cache hit forest=%s uid=%s", i, forestRoot, adUID)
			} else {
				log.Printf("[DEBUG] host[%d] calling upsertActiveDirectory forest=%s", i, forestRoot)
				adUID, err = n.upsertActiveDirectory(ctx, params.ProjectUID, forestRoot)
				if err != nil {
					log.Printf("[ERROR] host[%d] upsertActiveDirectory failed forest=%s err=%v",
						i, forestRoot, err)
				} else {
					adCache[forestRoot] = adUID
					log.Printf("[DEBUG] host[%d] upsertActiveDirectory ok forest=%s uid=%s",
						i, forestRoot, adUID)
				}
			}
		} else {
			log.Printf("[DEBUG] host[%d] ip=%s no forestRoot — skipping AD upsert", i, ip)
		}

		// ── Domain ───────────────────────────────────────────────────────────
		var domainUID string
		if domainName != "" {
			if cached, ok := domainCache[domainName]; ok {
				domainUID = cached
				log.Printf("[DEBUG] host[%d] Domain cache hit domain=%s uid=%s", i, domainName, domainUID)
			} else {
				log.Printf("[DEBUG] host[%d] calling upsertDomain domain=%s adUID=%s", i, domainName, adUID)
				domainUID, err = n.upsertDomain(ctx, params.ProjectUID, adUID, domainName, strategy)
				if err != nil {
					log.Printf("[ERROR] host[%d] upsertDomain failed domain=%s err=%v",
						i, domainName, err)
				} else {
					domainCache[domainName] = domainUID
					log.Printf("[DEBUG] host[%d] upsertDomain ok domain=%s uid=%s",
						i, domainName, domainUID)
				}
			}
		} else {
			log.Printf("[DEBUG] host[%d] ip=%s no domainName — host will be orphaned", i, ip)
		}

		// ── Host ─────────────────────────────────────────────────────────────
		log.Printf("[DEBUG] host[%d] calling upsertHost ip=%s domainUID=%s", i, ip, domainUID)
		hostUID, err := n.upsertHost(ctx, document, xb, ip, domainUID, params)
		if err != nil {
			log.Printf("[ERROR] host[%d] upsertHost failed ip=%s err=%v", i, ip, err)
			continue
		}
		log.Printf("[DEBUG] host[%d] upsertHost ok ip=%s uid=%s", i, ip, hostUID)

		// ── Services ─────────────────────────────────────────────────────────
		log.Printf("[DEBUG] host[%d] calling upsertServices hostUID=%s", i, hostUID)
		n.upsertServices(ctx, host, hostUID)
	}

	log.Printf("[DEBUG] processScanResults: done")
	return nil
}

// ── upsertActiveDirectory ─────────────────────────────────────────────────────

func (n *NetworkExplorer) upsertActiveDirectory(
	ctx context.Context,
	projectUID string,
	forestRoot string,
) (string, error) {
	log.Printf("[DEBUG] upsertActiveDirectory: projectUID=%s forestRoot=%s", projectUID, forestRoot)

	result, err := n.services.ActiveDirectoryService.UpsertActiveDirectory(
		ctx,
		upsert.Input[*active_directory.ActiveDirectory]{
			Entity:       &active_directory.ActiveDirectory{ForestName: forestRoot},
			ProjectUID:   projectUID,
			ParentUID:    nil,
			ParentType:   "Project",
			AssertionCtx: assertCtxAD,
			Actor:        n.configKey,
		},
	)
	if err != nil {
		log.Printf("[ERROR] upsertActiveDirectory: UpsertActiveDirectory returned err=%v", err)
		return "", fmt.Errorf("UpsertActiveDirectory failed for forest %s: %w", forestRoot, err)
	}

	log.Printf("[DEBUG] upsertActiveDirectory: result uid=%s", result.Entity.UID)

	sse.NewEvent(events.DomainDiscovered).
		WithData("type", "active_directory").
		WithData("forest", forestRoot).
		WithData("timestamp", time.Now().Unix()).
		Log(n.logger)

	log.Printf("[NetworkExplorer] AD upserted uid=%s forest=%s", result.Entity.UID, forestRoot)
	return result.Entity.UID, nil
}

// ── upsertDomain ──────────────────────────────────────────────────────────────

func (n *NetworkExplorer) upsertDomain(
	ctx context.Context,
	projectUID string,
	adUID string,
	domainName string,
	strategy string,
) (string, error) {
	var parentUID *string
	parentType := "Project"
	if adUID != "" {
		parentUID = &adUID
		parentType = "ActiveDirectory"
	}

	log.Printf("[DEBUG] upsertDomain: projectUID=%s adUID=%s domainName=%s parentType=%s",
		projectUID, adUID, domainName, parentType)

	result, err := n.services.DomainService.UpsertDomain(
		ctx,
		upsert.Input[*active_directory.Domain]{
			Entity:       &active_directory.Domain{Name: domainName},
			ProjectUID:   projectUID,
			ParentUID:    parentUID,
			ParentType:   parentType,
			AssertionCtx: assertCtxAD,
			Actor:        n.configKey,
		},
	)
	if err != nil {
		log.Printf("[ERROR] upsertDomain: UpsertDomain returned err=%v", err)
		return "", fmt.Errorf("UpsertDomain failed for %s: %w", domainName, err)
	}

	log.Printf("[DEBUG] upsertDomain: result uid=%s", result.Entity.UID)

	sse.NewEvent(events.DomainDiscovered).
		WithData("domain", domainName).
		WithData("strategy", strategy).
		WithData("timestamp", time.Now().Unix()).
		Log(n.logger)

	log.Printf("[NetworkExplorer] Domain upserted name=%s uid=%s strategy=%s",
		domainName, result.Entity.UID, strategy)
	return result.Entity.UID, nil
}

// ── upsertHost ────────────────────────────────────────────────────────────────

func (n *NetworkExplorer) upsertHost(
	ctx context.Context,
	document *xmlquery.Node,
	xb *internal.XPathBuilder,
	ip string,
	domainUID string,
	params *input.Parameter,
) (string, error) {
	if ip == "" {
		return "", fmt.Errorf("IP cannot be empty")
	}
	if params == nil {
		return "", fmt.Errorf("parameters cannot be nil")
	}

	log.Printf("[DEBUG] upsertHost: ip=%s domainUID=%s projectUID=%s", ip, domainUID, params.ProjectUID)

	b := model.NewHostBuilder().WithIP(ip)

	// ── Name ──────────────────────────────────────────────────────────────────
	if hostNode := xmlquery.FindOne(document, xb.Host()); hostNode != nil {
		if node := xmlquery.FindOne(hostNode, xb.Hostname()); node != nil {
			if v := strings.TrimSpace(node.InnerText()); v != "" {
				b.WithName(v)
			}
		}
	}
	if netbios := firstText(document, xb.NetBIOSComputerNameRDP(), xb.NetBIOSComputerNameSQL()); netbios != "" {
		b.WithName(netbios)
	}

	// ── DNS FQDN ──────────────────────────────────────────────────────────────
	if fqdn := firstText(document, xb.DNSComputerNameRDP(), xb.DNSComputerNameSQL(), xb.SMBFQDN()); fqdn != "" {
		b.WithDNSHostName(fqdn)
	}

	// ── Operating System ──────────────────────────────────────────────────────
	if osStr := firstText(document, xb.SMBOS()); osStr != "" {
		b.WithOperatingSystem(osStr)
	} else if osType := firstText(document, xb.ServiceOSType()); osType != "" {
		b.WithOperatingSystem(osType)
	}

	// ── OS Version ────────────────────────────────────────────────────────────
	if version := firstText(document, xb.ProductVersionRDP(), xb.ProductVersionSQL()); version != "" {
		b.WithOperatingSystemVersion(version)
	}

	// ── Domain Controller ─────────────────────────────────────────────────────
	dc, score, matchedPorts := isDomainController(document, xb)
	if dc {
		b.AsDomainController()
		log.Printf("[NetworkExplorer] %s: DC detected (score=%d, ports=%v)", ip, score, matchedPorts)
	} else {
		log.Printf("[NetworkExplorer] %s: not a DC (score=%d, ports=%v)", ip, score, matchedPorts)
	}

	host, err := b.Build()
	if err != nil {
		log.Printf("[ERROR] upsertHost: Build failed ip=%s err=%v", ip, err)
		return "", fmt.Errorf("failed to build host model: %w", err)
	}

	log.Printf("[DEBUG] upsertHost: host built ip=%s hostname=%s os=%q dc=%v",
		host.IP, host.DNSHostName, host.OperatingSystem, host.IsDomainController)

	var parentUID *string
	if domainUID != "" {
		parentUID = &domainUID
	}

	log.Printf("[DEBUG] upsertHost: calling HostService.UpsertHost ip=%s parentUID=%v",
		ip, parentUID)

	result, err := n.services.HostService.UpsertHost(
		ctx,
		upsert.Input[*model.Host]{
			Entity:       host,
			ProjectUID:   params.ProjectUID,
			ParentUID:    parentUID,
			ParentType:   "Domain",
			AssertionCtx: assertCtxHost,
			Actor:        n.configKey,
		},
	)
	if err != nil {
		log.Printf("[ERROR] upsertHost: HostService.UpsertHost failed ip=%s err=%v", ip, err)
		return "", fmt.Errorf("failed to upsert host: %w", err)
	}

	log.Printf("[DEBUG] upsertHost: ok ip=%s uid=%s", ip, result.Entity.UID)

	sse.NewEvent(events.HostDiscovered).
		WithData("ip", ip).
		WithData("hostname", host.DNSHostName).
		WithData("os", host.OperatingSystem).
		WithData("os_version", host.OperatingSystemVersion).
		WithData("dc", host.IsDomainController).
		WithData("timestamp", time.Now().Unix()).
		Log(n.logger)

	log.Printf("[NetworkExplorer] Host upserted ip=%s dns=%s os=%q dc=%v uid=%s",
		ip, host.DNSHostName, host.OperatingSystem, host.IsDomainController, result.Entity.UID)

	return result.Entity.UID, nil
}

// ── upsertServices ────────────────────────────────────────────────────────────

func (n *NetworkExplorer) upsertServices(ctx context.Context, host serializable.Host, hostUID string) {
	if hostUID == "" {
		return
	}

	log.Printf("[DEBUG] upsertServices: hostUID=%s portCount=%d", hostUID, len(host.Ports.Port))

	for _, port := range host.Ports.Port {
		if port.State.State != "open" {
			continue
		}

		service := model.NewServiceBuilder().
			WithName(port.Service.Name).
			WithPort(port.Portid).
			Build()

		_, err := n.services.HostService.AddService(
			ctx,
			assertCtxService,
			hostUID,
			service,
			n.configKey,
		)
		if err != nil {
			log.Printf("[ERROR] upsertServices: AddService failed port=%s host=%s err=%v",
				port.Portid, hostUID, err)
			continue
		}

		log.Printf("[DEBUG] upsertServices: ok port=%s service=%s host=%s",
			port.Portid, port.Service.Name, hostUID)

		sse.NewEvent(events.ServiceDetected).
			WithData("port", port.Portid).
			WithData("service", port.Service.Name).
			WithData("product", port.Service.Product).
			WithData("version", port.Service.Version).
			WithData("timestamp", time.Now().Unix()).
			Log(n.logger)
	}
}

// ── Domain / Forest extraction ────────────────────────────────────────────────

func (n *NetworkExplorer) extractDomainName(document *xmlquery.Node, xb *internal.XPathBuilder) (string, string) {
	ldapPorts := []string{"389", "636", "3268", "3269"}

	strategies := []struct {
		name     string
		getXPath func(port string) string
		extract  func(string) string
	}{
		{name: "LDAP ExtraInfo", getXPath: xb.LDAPExtraInfo, extract: extractDomainFromExtrainfo},
		{name: "SSL Cert CommonName", getXPath: xb.SSLCertCommonName, extract: extractDomainFromFQDN},
		{
			name:     "SSL Cert SAN-DNS",
			getXPath: xb.SSLCertSANDNS,
			extract:  func(text string) string { return extractDomainFromFQDN(strings.TrimPrefix(text, "DNS:")) },
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

func (n *NetworkExplorer) extractForestRoot(xb *internal.XPathBuilder, document *xmlquery.Node) string {
	if node := xmlquery.FindOne(document, xb.DNSTreeNameRDP()); node != nil {
		if v := strings.TrimSpace(node.InnerText()); v != "" {
			return v
		}
	}
	if node := xmlquery.FindOne(document, xb.DNSTreeNameSQL()); node != nil {
		if v := strings.TrimSpace(node.InnerText()); v != "" {
			return v
		}
	}
	return ""
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func firstText(document *xmlquery.Node, xpaths ...string) string {
	for _, xpath := range xpaths {
		if node := xmlquery.FindOne(document, xpath); node != nil {
			if v := strings.TrimSpace(node.InnerText()); v != "" {
				return v
			}
		}
	}
	return ""
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
	fqdn = strings.TrimPrefix(fqdn, "*.")
	if parts := strings.SplitN(fqdn, ".", 2); len(parts) == 2 {
		return parts[1]
	}
	return ""
}
