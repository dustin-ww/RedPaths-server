package interfaces

import (
	"RedPaths-server/pkg/interfaces/module"
	"RedPaths-server/pkg/model/redpaths/input"
	"RedPaths-server/pkg/model/rpsdk"
	"RedPaths-server/pkg/sse"
)

type RedPathsModule interface {
	ConfigKey() string
	ExecuteModule(params *input.Parameter, logger *sse.SSELogger) error
	SetServices(services *rpsdk.Services)
	GetMetadata() *ModuleMetadata
}

type ModuleMetadata struct {
	Name          string
	Category      string
	Description   string
	Prerequisites []*module.Prerequisite
	Provides      []*module.Capability
	Risk          int
	Stealth       int
	Complexity    int
}
