package redpaths

import (
	"RedPaths-server/internal/db"
	"RedPaths-server/internal/recom"
	rp "RedPaths-server/internal/repository/redpaths"
	"RedPaths-server/pkg/interfaces"
	"RedPaths-server/pkg/model/redpaths"
	"RedPaths-server/pkg/model/redpaths/input"
	"context"
	"fmt"
	"log"

	"gorm.io/gorm"
)

type ModuleService struct {
	db                 *gorm.DB
	redPathsModuleRepo rp.RedPathsModuleRepository
	redPathsVectorRepo rp.RedPathsVectorRepository
	attackRunner       interfaces.ModuleExecutor // Add this back
	recommender        *recom.Engine
}

func NewModuleService(attackRunner interfaces.ModuleExecutor, recommender *recom.Engine, postgresCon *gorm.DB) (*ModuleService, error) {

	return &ModuleService{
		db:                 postgresCon,
		redPathsModuleRepo: rp.NewPostgresRedPathsModuleRepository(),
		redPathsVectorRepo: rp.NewPostgresRedPathsVectorRepository(),
		attackRunner:       attackRunner, // Store the executor
		recommender:        recommender,
	}, nil
}

func (s *ModuleService) GetInheritanceSubgraph(ctx context.Context, moduleKey string, direction rp.GraphDirection, maxDepth *int) (*redpaths.InheritanceGraph, error) {
	return db.ExecutePostgresRead(ctx, s.db, func(tx *gorm.DB) (*redpaths.InheritanceGraph, error) {
		return s.redPathsModuleRepo.GetInheritanceSubgraph(ctx, tx, moduleKey, direction, maxDepth)
	})
}

func (s *ModuleService) CreateWithObject(ctx context.Context, module *redpaths.Module) (string, error) {
	var attackID string
	err := db.ExecutePostgresInTransaction(ctx, s.db, func(tx *gorm.DB) error {

		// Check if module already exists
		exists, err := s.redPathsModuleRepo.CheckIfExistsByKey(ctx, tx, module.Key)
		if err != nil {
			return fmt.Errorf("failed to check if module exists: %w", err)
		}
		if exists {
			return fmt.Errorf("module with name '%s' already exists", module.Name)
		}

		attackID, err = s.redPathsModuleRepo.CreateWithObject(ctx, tx, module)
		if err != nil {
			return fmt.Errorf("failed to create redpaths module: %w", err)
		}

		if len(module.Options) != 0 {
			for _, option := range module.Options {
				err := s.redPathsModuleRepo.AddOption(ctx, tx, option)
				if err != nil {
					return fmt.Errorf("error while adding option '%s'", option.Key)
				}
			}
		}

		return nil
	})

	return attackID, err
}

func (s *ModuleService) CreateModuleRun(ctx context.Context, runMetadata *redpaths.ModuleRun) error {
	return db.ExecutePostgresInTransaction(ctx, s.db, func(tx *gorm.DB) error {
		err := s.redPathsModuleRepo.AddRun(ctx, tx, runMetadata)
		if err != nil {
			return err
		}
		return nil
	})
}

func (s *ModuleService) CreateModuleInheritanceEdges(ctx context.Context, inheritanceEdges []*redpaths.ModuleDependency) error {
	return db.ExecutePostgresInTransaction(ctx, s.db, func(tx *gorm.DB) error {
		for _, inheritanceEdge := range inheritanceEdges {
			exists, err := s.redPathsModuleRepo.CheckIfDependencyExits(ctx, tx, inheritanceEdge.PreviousModule, inheritanceEdge.NextModule)
			if err != nil {
				return fmt.Errorf("failed to check if inheritance edge exists: %w", err)
			}

			if exists {
				log.Printf("edge from '%s' to '%s' already exists\n", inheritanceEdge.PreviousModule, inheritanceEdge.NextModule)
				continue
			}

			_, err = s.redPathsModuleRepo.AddDependency(ctx, tx, inheritanceEdge.PreviousModule, inheritanceEdge.NextModule)
			if err != nil {
				return fmt.Errorf("failed to add inheritance edge: %w", err)
			}
		}

		return nil
	})
}

func (s *ModuleService) GetAll(ctx context.Context) ([]*redpaths.Module, error) {
	return db.ExecutePostgresRead(ctx, s.db, func(db *gorm.DB) ([]*redpaths.Module, error) {
		modules, err := s.redPathsModuleRepo.GetAll(ctx, db)
		if err != nil {
			return nil, err
		}

		for _, module := range modules {
			module.Options, err = s.redPathsModuleRepo.GetOptions(ctx, db, module.Key)
			if err != nil {
				return nil, fmt.Errorf("failed to get module options: %w", err)
			}
			module.DependencyVector, err = s.redPathsModuleRepo.GetOrderedDependencies(ctx, db, module.Key)
			if err != nil {
				return nil, fmt.Errorf("failed to get module dependency vector: %w", err)
			}
		}

		return modules, nil
	})
}

func (s *ModuleService) GetAllRunMetadata(ctx context.Context, projectUID string) ([]*redpaths.ModuleRun, error) {
	return db.ExecutePostgresRead(ctx, s.db, func(db *gorm.DB) ([]*redpaths.ModuleRun, error) {
		runs, err := s.redPathsModuleRepo.GetAllModuleRuns(ctx, db, projectUID)
		if err != nil {
			return nil, err
		}
		return runs, nil
	})
}

// TODO: refactor to dedicated service
func (s *ModuleService) GetAllVectorRuns(ctx context.Context, projectUID string) ([]*redpaths.VectorRun, error) {
	return db.ExecutePostgresRead(ctx, s.db, func(db *gorm.DB) ([]*redpaths.VectorRun, error) {
		vruns, err := s.redPathsVectorRepo.GetAllVectorRuns(ctx, db, projectUID)
		if err != nil {
			return nil, err
		}
		return vruns, nil
	})
}

func (s *ModuleService) CreateVectorRun(ctx context.Context, vectorMetadata *redpaths.VectorRun) error {
	return db.ExecutePostgresInTransaction(ctx, s.db, func(tx *gorm.DB) error {
		err := s.redPathsVectorRepo.AddRun(ctx, tx, vectorMetadata)
		if err != nil {
			return err
		}
		return nil
	})
}

func (s *ModuleService) GetAttackVectorByKey(ctx context.Context, moduleKey string) ([]*redpaths.Module, error) {
	return db.ExecutePostgresRead(ctx, s.db, func(db *gorm.DB) ([]*redpaths.Module, error) {
		var attackVector []*redpaths.Module

		module, err := s.redPathsModuleRepo.Get(ctx, db, moduleKey)
		if err != nil {
			return nil, fmt.Errorf("error while fetching all dependencies of redpaths modulelib for graph edges %s", err)
		}

		if module.DependencyVector == nil {
			module.DependencyVector, err = s.redPathsModuleRepo.GetOrderedDependencies(ctx, db, moduleKey)
		}

		for _, dependencyKey := range module.DependencyVector {
			dep, err := s.redPathsModuleRepo.Get(ctx, db, dependencyKey)
			if err != nil {
				return nil, fmt.Errorf("error while fetching all dependencies of redpaths modulelib for graph edges %s", err)
			}
			attackVector = append(attackVector, dep)
		}
		return append(attackVector, module), nil
	})
}

//func (s *ModuleService) GetAttackVector(ctx context.Context, moduleKey string) ([]*redpaths.Module, error) {
//	return db.ExecutePostgresRead(ctx, s.db, func(db *gorm.DB) (*redpaths.Module, error) {
//		s.redPathsModuleRepo.GetOrderedDependencies()
//	})
//}

func (s *ModuleService) GetInheritanceGraph(ctx context.Context) (*redpaths.InheritanceGraph, error) {
	return db.ExecutePostgresRead(ctx, s.db, func(db *gorm.DB) (*redpaths.InheritanceGraph, error) {
		var inheritanceGraph redpaths.InheritanceGraph

		modules, err := s.redPathsModuleRepo.GetAll(ctx, db)
		if err != nil {
			return nil, fmt.Errorf("error while fetching all redpaths modulelib %s", err)
		}
		inheritanceGraph.Nodes = modules
		edges, err := s.redPathsModuleRepo.GetAllDependencies(ctx, db)
		if err != nil {
			return nil, fmt.Errorf("error while fetching all dependencies of redpaths modulelib for graph edges %s", err)
		}
		inheritanceGraph.Edges = edges
		return &inheritanceGraph, nil
	})
}

func (s *ModuleService) GetInheritanceGraphForModule(ctx context.Context, moduleKey string) (*redpaths.InheritanceGraph, error) {
	return db.ExecutePostgresRead(ctx, s.db, func(db *gorm.DB) (*redpaths.InheritanceGraph, error) {
		var inheritanceGraph redpaths.InheritanceGraph

		modules, err := s.redPathsModuleRepo.GetAll(ctx, db)
		if err != nil {
			return nil, fmt.Errorf("error while fetching all redpaths modulelib %s", err)
		}
		inheritanceGraph.Nodes = modules
		edges, err := s.redPathsModuleRepo.GetAllDependencies(ctx, db)
		if err != nil {
			return nil, fmt.Errorf("error while fetching all dependencies of redpaths modulelib for graph edges %s", err)
		}
		inheritanceGraph.Edges = edges
		return &inheritanceGraph, nil
	})
}

func (s *ModuleService) RunAttackVector(ctx context.Context, key string, params *input.Parameter) (string, error) {
	var runUid string
	err := db.ExecutePostgresInTransaction(ctx, s.db, func(tx *gorm.DB) error {
		// Use the attackRunner that was injected into the service
		if s.attackRunner == nil {
			return fmt.Errorf("error while executing attack vector: the runner engine seems to be nil")
		}
		var err error
		// TODO: Change Params
		runUid, err = RunAttackVector(ctx, s.db, key, params, s.attackRunner, s.recommender, s)
		if err != nil {
			return err
		}
		return nil
	})
	return runUid, err
}

func (s *ModuleService) GetOptionsForAttackVector(ctx context.Context, moduleKey string) ([]*redpaths.ModuleOption, error) {
	return db.ExecutePostgresRead(ctx, s.db, func(db *gorm.DB) ([]*redpaths.ModuleOption, error) {
		modules, err := s.GetAttackVectorByKey(ctx, moduleKey)
		log.Println(moduleKey)
		log.Println(len(modules))
		if err != nil {
			return nil, err
		}

		for _, module := range modules {
			module.Options, err = s.redPathsModuleRepo.GetOptions(ctx, db, module.Key)
			if err != nil {
				return nil, fmt.Errorf("failed to get module options: %w", err)
			}
		}

		seenKeys := make(map[string]struct{})
		uniqueOptions := make([]*redpaths.ModuleOption, 0)

		for _, module := range modules {
			for _, option := range module.Options {
				log.Println(option.Key)
				key := option.Key
				if _, exists := seenKeys[key]; !exists {
					seenKeys[key] = struct{}{}
					uniqueOptions = append(uniqueOptions, option)
				}
			}
		}
		return uniqueOptions, nil
	})

}
