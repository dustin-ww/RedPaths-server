// core/interfaces/module_executor.go
package interfaces

import (
	"RedPaths-server/pkg/model/redpaths/input"
	"RedPaths-server/pkg/sse"
)

type ModuleExecutor interface {
	ExecuteModule(key string, params *input.Parameter, logger *sse.SSELogger) error
}
