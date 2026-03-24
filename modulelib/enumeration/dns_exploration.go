package enumeration

import (
	"RedPaths-server/pkg/interfaces"
	"RedPaths-server/pkg/interfaces/module"
	"RedPaths-server/pkg/model"
	"RedPaths-server/pkg/model/redpaths/input"
	"RedPaths-server/pkg/model/rpsdk"
	"RedPaths-server/pkg/model/utils/assertion"
	plugin "RedPaths-server/pkg/module_exec"
	engine4 "RedPaths-server/pkg/service/upsert"
	"RedPaths-server/pkg/sse"
	"context"
	"fmt"
)

// INIT
func init() {
	module := &DNSExplorer{
		configKey: "DNSExplorer",
	}
	plugin.RegisterPlugin(module)

}

type DNSExplorer struct {
	// Internal
	configKey string
	// Services
	services *rpsdk.Services
	// Tool Adaptera
	logger *sse.SSELogger
}

func (n *DNSExplorer) ConfigKey() string {
	return n.configKey
}

func (n *DNSExplorer) SetServices(services *rpsdk.Services) {
	n.services = services
}

func (n *DNSExplorer) GetMetadata() *interfaces.ModuleMetadata {
	return &interfaces.ModuleMetadata{
		Name:        "DNSExploration",
		Category:    "enumeration",
		Description: "Performs DNS enumeration to discover domains, subdomains, records and infrastructure relationships",
		Prerequisites: []*module.Prerequisite{
			{
				Type:        module.PrereqKnowledge,
				Name:        "Target Domain",
				Description: "Fully qualified domain name to enumerate",
				Required:    true,
				Conditions:  "domain.fqdn != null",
			},
			{
				Type:        module.PrereqNetworkAccess,
				Name:        "DNS Resolution",
				Description: "Ability to perform DNS queries against recursive or authoritative resolvers",
				Required:    true,
				Conditions:  "dns.resolution = allowed",
			},
		},
		Provides: []*module.Capability{
			{
				Type:        "dns_enumeration",
				Name:        "DNS Record Enumeration",
				Description: "Enumerates common DNS record types (A, AAAA, MX, NS, TXT, SRV)",
				Confidence:  0.95,
				Metadata: map[string]interface{}{
					"record_types": []string{"A", "AAAA", "MX", "NS", "TXT", "SRV"},
				},
			},
			{
				Type:        "subdomain_discovery",
				Name:        "Subdomain Discovery",
				Description: "Discovers subdomains via brute-force and passive techniques",
				Confidence:  0.85,
				Metadata: map[string]interface{}{
					"methods": []string{"bruteforce", "zone_transfer", "passive"},
				},
			},
			{
				Type:        "infrastructure_mapping",
				Name:        "Infrastructure Mapping",
				Description: "Maps DNS data to underlying hosts and services",
				Confidence:  0.8,
			},
		},
		Risk:       2,
		Stealth:    6,
		Complexity: 2,
	}
}

func (n *DNSExplorer) ExecuteModule(params *input.Parameter, logger *sse.SSELogger) error {
	ctx := context.Background()
	n.logger = logger

	projectUID := params.ProjectUID

	n.logger.Info("DNSExplorer", "Starting UpsertHost test")

	// --- Test 1: Neuen Host erstellen ---
	n.logger.Info("DNSExplorer", "Test 1: Creating new host")
	result1, err := n.services.HostService.UpsertHost(ctx, engine4.Input[*model.Host]{
		ProjectUID: projectUID,
		Actor:      "dns-explorer-test",
		Entity: &model.Host{
			IP:              "192.168.1.100",
			Hostname:        "DC01",
			OperatingSystem: "Windows Server 2022",
		},
		AssertionCtx: assertion.NewContext(),
	})
	if err != nil {
		n.logger.Error("DNSExplorer", fmt.Sprintf("Test 1 failed: %v", err))
		return err
	}
	n.logger.Info("DNSExplorer", fmt.Sprintf("Test 1 OK: created host uid=%s", result1.Entity.UID))

	// --- Test 2: Gleichen Host nochmal → Merge erwartet ---
	n.logger.Info("DNSExplorer", "Test 2: Merging existing host")
	result2, err := n.services.HostService.UpsertHost(ctx, engine4.Input[*model.Host]{
		ProjectUID: projectUID,
		Actor:      "dns-explorer-test",
		Entity: &model.Host{
			IP:                     "192.168.1.100",
			Hostname:               "DC01",
			OperatingSystem:        "Windows Server 2022",
			OperatingSystemVersion: "21H2",
			IsDomainController:     true,
		},
		AssertionCtx: assertion.NewContext(),
	})
	if err != nil {
		n.logger.Error("DNSExplorer", fmt.Sprintf("Test 2 failed: %v", err))
		return err
	}
	n.logger.Info("DNSExplorer", fmt.Sprintf("Test 2 OK: merged host uid=%s", result2.Entity.UID))

	// --- Test 3: Possible Duplicate ---
	n.logger.Info("DNSExplorer", "Test 3: Possible duplicate")
	result3, err := n.services.HostService.UpsertHost(ctx, engine4.Input[*model.Host]{
		ProjectUID: projectUID,
		Actor:      "dns-explorer-test",
		Entity: &model.Host{
			IP:       "192.168.1.100",
			Hostname: "DC01-ALT",
		},
		AssertionCtx: assertion.NewContext(),
	})
	if err != nil {
		n.logger.Error("DNSExplorer", fmt.Sprintf("Test 3 failed: %v", err))
		return err
	}
	if result3 != nil {
		n.logger.Info("DNSExplorer", fmt.Sprintf("Test 3 OK: possible duplicate flagged uid=%s", result3.Entity.UID))
	}

	n.logger.Info("DNSExplorer", "All UpsertHost tests completed")
	return nil
}
