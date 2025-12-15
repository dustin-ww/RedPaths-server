// In service/attack_vector_service.go
package redpaths

import (
	"RedPaths-server/internal/recom"
	rp "RedPaths-server/internal/repository/redpaths"
	"RedPaths-server/pkg/interfaces"
	"RedPaths-server/pkg/model/events"
	"RedPaths-server/pkg/model/redpaths"
	"RedPaths-server/pkg/model/redpaths/input"
	"RedPaths-server/pkg/sse"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RunAttackVector runs an attack vector starting with the target module
func RunAttackVector(ctx context.Context, postgresCon *gorm.DB, targetModuleKey string, params *input.Parameter, executor interfaces.ModuleExecutor, recommender *recom.Engine, moduleService *ModuleService) (string, error) {
	// Generate a unique run ID
	vectorRunID := uuid.New().String()
	log.Println("Starting Execution with vectorRunID: " + vectorRunID)

	log.Println("Starting Execution by building subgraph for vector run entry")
	depth := 10
	subGraph, err := moduleService.GetInheritanceSubgraph(ctx, targetModuleKey, rp.GraphDownstream, &depth)
	if err != nil {
		return "", fmt.Errorf("failed to get subgraph: %w", err)
	}
	vectorRun := redpaths.NewVectorRunBuilder().WithRunUID(vectorRunID).WithProjectUID(params.ProjectUID).WithGraph(subGraph).Build()
	err = moduleService.CreateVectorRun(ctx, vectorRun)
	if err != nil {
		return "", fmt.Errorf("failed to create vector run: %w", err)
	}

	// Initialize parameters if nil
	if params == nil {
		params = &input.Parameter{}
	}
	params.RunID = vectorRunID

	log.Println("Starting Execution with projectUID: " + params.ProjectUID)
	// Initialize the logger for this run
	logger := sse.GetLogger(vectorRunID, params.ProjectUID, postgresCon)
	if logger == nil {
		return "", fmt.Errorf("failed to initialize logger for run %s", vectorRunID)
	}
	defer func() {
		// Optional: Keep logger active for a while to allow clients to fetch final logs
		// In production we rely on the automatic cleanup of inactive loggers
	}()

	// Log the start of the attack vector execution
	sse.NewEvent(events.ModuleStart).
		WithData("runId", vectorRunID).
		WithData("timestamp", time.Now().Unix()).
		WithData("module", targetModuleKey)

	// Create a module-specific logger
	moduleLogger := logger.ForModule(targetModuleKey)
	moduleLogger.Info("Starting module execution")

	// Initialize module service
	moduleService, err = NewModuleService(nil, nil, postgresCon)
	if err != nil {
		logger.Error("Failed to create module service", map[string]interface{}{
			"error": err.Error(),
		})
		return "", fmt.Errorf("failed to create module service: %w", err)
	}

	// Get the attack vector modules
	moduleDependencies, err := moduleService.GetAttackVectorByKey(ctx, targetModuleKey)
	if err != nil {
		logger.Error("Failed to get attack vector", map[string]interface{}{
			"moduleKey": targetModuleKey,
			"error":     err.Error(),
		})
		return "", fmt.Errorf("failed to get attack vector: %w", err)
	}

	// Log number of modules to execute
	logger.Info(fmt.Sprintf("Attack vector has %d modules to execute", len(moduleDependencies)))

	// Track total execution time
	//startTime := time.Now()

	// Execute each module in the attack vector
	var lastExecutedModule *redpaths.Module
	for i, module := range moduleDependencies {
		moduleRunID := uuid.New().String()
		moduleRunSuccessful := false

		// Create module-specific logger for each module
		currentModuleLogger := logger.ForModule(module.Key)

		currentModuleLogger.Info(fmt.Sprintf("Executing module %d/%d: %s",
			i+1, len(moduleDependencies), module.Name),
			map[string]interface{}{
				"moduleKey":    module.Key,
				"moduleIndex":  i + 1,
				"totalModules": len(moduleDependencies),
			})

		// Execute the module with progress monitoring
		moduleStartTime := time.Now()

		err := executor.ExecuteModule(module.Key, params, currentModuleLogger)
		if err != nil {
			moduleRunSuccessful = true
		}
		executionTime := time.Since(moduleStartTime)

		if err != nil {
			currentModuleLogger.Error(fmt.Sprintf("Failed to execute module: %s", module.Name),
				map[string]interface{}{
					"moduleKey":     module.Key,
					"error":         err.Error(),
					"executionTime": executionTime.String(),
				})

			// Send run_error event
			sse.NewEvent(events.ModuleError).
				WithData("runId", vectorRunID).
				WithData("timestamp", time.Now().Unix()).
				WithData("error", err.Error()).
				WithData("executionTime", executionTime.Seconds()).
				WithData("failed", true)

			return "", fmt.Errorf("failed to execute module %s: %w", module.Key, err)
		}

		var moduleRun *redpaths.ModuleRun
		moduleRun, err = redpaths.NewModuleRunBuilder().ModuleKey(module.Key).RunUID(moduleRunID).ProjectUID(params.ProjectUID).WasSuccessful(moduleRunSuccessful).Parameters(nil).Targets(nil).VectorRunUID(vectorRunID).Build()
		err = moduleService.CreateModuleRun(ctx, moduleRun)
		if err != nil {
			return "", fmt.Errorf("error while creating new module run metadata in vector runner engine: %s", err)
		}

		currentModuleLogger.Info(fmt.Sprintf("Successfully executed module: %s", module.Name),
			map[string]interface{}{
				"moduleKey":     module.Key,
				"executionTime": executionTime.String(),
			})

		// Send module_complete event
		sse.NewEvent(events.ModuleComplete).
			WithData("runId", vectorRunID).
			WithData("timestamp", time.Now().Unix()).
			WithData("executionTime", executionTime.Seconds()).
			WithData("failed", true)

		lastExecutedModule = module

	}

	if lastExecutedModule != nil && recommender != nil {
		recommendation := recommender.Calculate(lastExecutedModule)
		if recommendation != nil {
			logger.SendRecommendation(recommendation.ConfigKey())
			moduleLogger.Info("Sent recommendation to client", map[string]interface{}{
				"recommendedModule": recommendation.ConfigKey(),
			})
			log.Printf("Sent recommendation to client: %s for run: %s, with last executed module: %s", recommendation.ConfigKey(), vectorRunID, lastExecutedModule.Name)
		}
	} else {
		log.Printf("No recommended module found for run: %s", vectorRunID)
	}

	moduleLogger.Info("Attack vector execution completed successfully")
	return vectorRunID, nil
}
