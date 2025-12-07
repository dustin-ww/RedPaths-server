package module_exec

import (
	"RedPaths-server/pkg/model/redpaths/input"
	"RedPaths-server/pkg/sse"
	"fmt"
)

// ExecuteModule executes a registered module by key
func (r *Registry) ExecuteModule(key string, params *input.Parameter, moduleLogger *sse.SSELogger) error {
	impl, exists := r.implementations[key]
	if !exists {
		return fmt.Errorf("[Executor] No implementation found for module key: %s", key)
	}

	if moduleLogger == nil {
		moduleLogger = sse.GetLogger(params.RunID, params.ProjectUID, nil)
		if moduleLogger == nil {
			return fmt.Errorf("[Executor] Failed to create logger for module execution")
		}
	}

	return impl.ExecuteModule(params, moduleLogger)
}

// ExecuteModule is a global shortcut to execute a module
/*func ExecuteModule(key string, params *input.Parameter, moduleLogger *sse.SSELogger) error {
	return GlobalRegistry.ExecuteModule(key, params, moduleLogger)
}
*/
