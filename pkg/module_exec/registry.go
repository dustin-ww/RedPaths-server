package module_exec

import (
	"RedPaths-server/internal/config"
	"RedPaths-server/internal/recom"
	"RedPaths-server/pkg/interfaces"
	"RedPaths-server/pkg/model/redpaths"
	"RedPaths-server/pkg/model/rpsdk"
	redpaths2 "RedPaths-server/pkg/service/redpaths"
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/dgraph-io/dgo/v210"
	"gorm.io/gorm"
)

type Registry struct {
	modules              map[string]*redpaths.Module
	implementations      map[string]interfaces.RedPathsModule
	moduleService        *redpaths2.ModuleService
	serviceFactory       func() *rpsdk.Services
	initialized          bool
	pendingModules       map[string]*pendingModuleInfo
	mu                   sync.RWMutex // Race Condition Protection
	RecommendationEngine *recom.Engine
}

type pendingModuleInfo struct {
	module   *redpaths.Module
	inherits []*redpaths.ModuleDependency
}

var GlobalRegistry = &Registry{
	modules:         make(map[string]*redpaths.Module),
	implementations: make(map[string]interfaces.RedPathsModule),
	pendingModules:  make(map[string]*pendingModuleInfo),
	initialized:     false,
}

func InitializeRegistry(postgresCon *gorm.DB, dgraphCon *dgo.Dgraph) error {
	GlobalRegistry.mu.Lock()
	defer GlobalRegistry.mu.Unlock()

	if GlobalRegistry.initialized {
		log.Println("Registry already initialized, skipping initialization")
		return nil
	}

	recomEngine := recom.NewEngine(postgresCon)
	moduleService, err := redpaths2.NewModuleService(GlobalRegistry, recomEngine, postgresCon)
	if err != nil {
		return fmt.Errorf("failed to create module service: %w", err)
	}

	GlobalRegistry.moduleService = moduleService
	GlobalRegistry.RecommendationEngine = recomEngine
	GlobalRegistry.serviceFactory = func() *rpsdk.Services {
		return rpsdk.NewServicesContainer(dgraphCon, postgresCon)
	}
	GlobalRegistry.initialized = true

	return nil
}

func RegisterPlugin(module interfaces.RedPathsModule) error {
	log.Printf("--- Starting the RedPaths Module Loading Process...")

	configKey := module.ConfigKey()
	moduleConfig, inherits, err := config.ModuleFromConfig(configKey)
	if err != nil {
		return fmt.Errorf("failed to load module config: %w", err)
	}

	GlobalRegistry.mu.Lock()
	defer GlobalRegistry.mu.Unlock()

	GlobalRegistry.modules[moduleConfig.Key] = moduleConfig
	GlobalRegistry.implementations[moduleConfig.Key] = module

	if GlobalRegistry.initialized {
		services := GlobalRegistry.serviceFactory()
		module.SetServices(services)

		GlobalRegistry.pendingModules[moduleConfig.Key] = &pendingModuleInfo{
			module:   moduleConfig,
			inherits: inherits,
		}
	} else {
		GlobalRegistry.pendingModules[moduleConfig.Key] = &pendingModuleInfo{
			module:   moduleConfig,
			inherits: inherits,
		}
		log.Printf("Registry not initialized. Module %s will be persisted later.", moduleConfig.Key)
	}

	log.Printf("--- Finished the RedPaths Module Loading Process for %s", moduleConfig.Key)
	return nil
}

func CompleteRegistration() error {
	GlobalRegistry.mu.RLock()
	if !GlobalRegistry.initialized {
		GlobalRegistry.mu.RUnlock()
		return fmt.Errorf("cannot complete registration: Registry not initialized")
	}
	GlobalRegistry.mu.RUnlock()

	log.Println("Phase 0: Setting services for all registered modules...")

	GlobalRegistry.mu.Lock()
	for key, impl := range GlobalRegistry.implementations {
		services := GlobalRegistry.serviceFactory()
		impl.SetServices(services)
		log.Printf("Set services for module: %s", key)
	}
	GlobalRegistry.mu.Unlock()

	log.Println("Phase 1: Creating all modules in database...")

	GlobalRegistry.mu.Lock()
	pendingModules := make(map[string]*pendingModuleInfo)
	for k, v := range GlobalRegistry.pendingModules {
		pendingModules[k] = v
	}
	GlobalRegistry.mu.Unlock()

	for key, info := range pendingModules {
		ctx := context.Background()
		_, err := GlobalRegistry.moduleService.CreateWithObject(ctx, info.module)
		if err != nil {
			log.Printf("Error persisting module %s: %v", key, err)
		} else {
			log.Printf("Successfully registered module: %s", key)
		}
	}

	log.Println("Phase 1 complete. All modules created in database.")

	log.Println("Phase 2: Creating all module dependencies...")

	for key, info := range pendingModules {
		if len(info.inherits) > 0 {
			log.Printf("Registering dependencies for module %s", key)
			err := GlobalRegistry.moduleService.CreateModuleInheritanceEdges(context.Background(), info.inherits)
			if err != nil {
				log.Printf("Failed to register dependencies for %s: %v", key, err)
			} else {
				log.Printf("Successfully registered dependencies for module: %s", key)
			}
		}
	}

	log.Println("Phase 3: Registering modules with recommendation engine...")

	GlobalRegistry.mu.Lock()
	for key, impl := range GlobalRegistry.implementations {
		GlobalRegistry.RecommendationEngine.RegisterModule(key, impl)
		log.Printf("Registered module %s with recommendation engine", key)
	}
	GlobalRegistry.mu.Unlock()

	GlobalRegistry.mu.Lock()
	GlobalRegistry.pendingModules = make(map[string]*pendingModuleInfo)
	GlobalRegistry.mu.Unlock()

	log.Println("Module registration complete!")
	return nil
}

func persistModule(module *redpaths.Module, inherits []*redpaths.ModuleDependency, moduleService *redpaths2.ModuleService) error {
	ctx := context.Background()

	if _, err := moduleService.CreateWithObject(ctx, module); err != nil {
		return fmt.Errorf("failed to register module in database: %w", err)
	}

	if len(inherits) > 0 {
		if err := moduleService.CreateModuleInheritanceEdges(ctx, inherits); err != nil {
			return fmt.Errorf("failed to register inheritance references in database: %w", err)
		}
	}

	return nil
}

func GetAll() []*redpaths.Module {
	GlobalRegistry.mu.RLock()
	defer GlobalRegistry.mu.RUnlock()

	modules := make([]*redpaths.Module, 0, len(GlobalRegistry.modules))
	for _, module := range GlobalRegistry.modules {
		modules = append(modules, module)
	}
	return modules
}

func GetModule(key string) *redpaths.Module {
	GlobalRegistry.mu.RLock()
	defer GlobalRegistry.mu.RUnlock()

	return GlobalRegistry.modules[key]
}
