package interfaces

import (
	"RedPaths-server/pkg/model/redpaths/input"
	"RedPaths-server/pkg/model/rpsdk"
	"RedPaths-server/pkg/sse"
)

type RedPathsModule interface {
	ConfigKey() string
	ExecuteModule(params *input.Parameter, logger *sse.SSELogger) error
	SetServices(services *rpsdk.Services)
}
