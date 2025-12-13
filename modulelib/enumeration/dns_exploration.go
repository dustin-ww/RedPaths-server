package enumeration

import (
	"RedPaths-server/pkg/interfaces"
	"RedPaths-server/pkg/model/redpaths/input"
	"RedPaths-server/pkg/model/rpsdk"
	plugin "RedPaths-server/pkg/module_exec"
	"RedPaths-server/pkg/sse"
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
	return nil
}

// THIS METHOD IS CALLED BY THE REDPATHS SERVER
func (n *DNSExplorer) ExecuteModule(params *input.Parameter, logger *sse.SSELogger) error {
	// INSERT MAIN LOGIC HERE
	return nil
}
