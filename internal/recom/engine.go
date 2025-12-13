package recom

import (
	"RedPaths-server/pkg/interfaces"
	"RedPaths-server/pkg/model/redpaths"

	"gorm.io/gorm"
)

type Engine struct {
	db      *gorm.DB
	modules map[string]interfaces.RedPathsModule
}

func NewEngine(db *gorm.DB) *Engine {
	return &Engine{
		db:      db,
		modules: make(map[string]interfaces.RedPathsModule),
	}
}

func (e *Engine) RegisterModule(key string, module interfaces.RedPathsModule) {
	e.modules[key] = module
}

func (e *Engine) Calculate(lastModule *redpaths.Module) interfaces.RedPathsModule {
	for _, impl := range e.modules {
		if impl.ConfigKey() == "PrinterNightmare" {
			return impl
		}
	}
	return nil
}
