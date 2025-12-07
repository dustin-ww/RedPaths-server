package attacks

import (
	"RedPaths-server/pkg/model/redpaths/input"
	"RedPaths-server/pkg/model/rpsdk"
	plugin "RedPaths-server/pkg/module_exec"
	"RedPaths-server/pkg/sse"
)

type PrinterNightmare struct {
	configKey string
}

func (n *PrinterNightmare) ConfigKey() string {
	//TODO implement me
	return n.configKey
}

func (n *PrinterNightmare) SetServices(services *rpsdk.Services) {
}

func (n *PrinterNightmare) DependsOn() int {
	//TODO implement me
	return 0
}

func (n *PrinterNightmare) ExecuteModule(params *input.Parameter, logger *sse.SSELogger) error {
	return nil
}

// INIT
func init() {
	module := &PrinterNightmare{
		configKey: "PrinterNightmare",
	}
	plugin.RegisterPlugin(module)
}
