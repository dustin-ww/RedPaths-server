package redpaths

import (
	rperrors "RedPaths-server/internal/error"
	"RedPaths-server/pkg/model/redpaths"
	"context"
	"errors"
	"fmt"
	"log"

	"gorm.io/gorm"
)

const (
	TableModules            = "redpaths_modules"
	TableModuleDependencies = "redpaths_modules_dependencies"
	TableModuleOptions      = "redpaths_modules_options"
	TableModuleRuns         = "redpaths_modules_runs"
)

type GraphDirection string

const (
	GraphUpstream   GraphDirection = "upstream"
	GraphDownstream GraphDirection = "downstream"
	GraphBoth       GraphDirection = "both"
)

const upstreamCTE = `
WITH RECURSIVE graph AS (
	SELECT previous_module, next_module, 1 AS depth
	FROM redpaths_modules_dependencies
	WHERE next_module = $1

	UNION ALL

	SELECT d.previous_module, d.next_module, g.depth + 1
	FROM redpaths_modules_dependencies d
	JOIN graph g ON d.next_module = g.previous_module
	WHERE ($2::int IS NULL OR g.depth < $2)
)
SELECT DISTINCT previous_module, next_module FROM graph;
`

const downstreamCTE = `
WITH RECURSIVE graph AS (
	SELECT previous_module, next_module, 1 AS depth
	FROM redpaths_modules_dependencies
	WHERE previous_module = $1

	UNION ALL

	SELECT d.previous_module, d.next_module, g.depth + 1
	FROM redpaths_modules_dependencies d
	JOIN graph g ON d.previous_module = g.next_module
	WHERE ($2::int IS NULL OR g.depth < $2)
)
SELECT DISTINCT previous_module, next_module FROM graph;
`

type RedPathsModuleRepository interface {

	//CRUD
	GetAll(ctx context.Context, tx *gorm.DB) ([]*redpaths.Module, error)
	CreateWithObject(ctx context.Context, tx *gorm.DB, module *redpaths.Module) (string, error)
	Get(ctx context.Context, tx *gorm.DB, moduleKey string) (*redpaths.Module, error)
	CheckIfExistsByKey(ctx context.Context, tx *gorm.DB, key string) (bool, error)

	// module dependencies
	CheckIfDependencyExits(ctx context.Context, tx *gorm.DB, previousModuleKey, nextModuleKey string) (bool, error)
	AddDependency(ctx context.Context, tx *gorm.DB, previousModuleKey, nextModuleKey string) (string, error)
	GetAllDependencies(ctx context.Context, tx *gorm.DB) ([]*redpaths.ModuleDependency, error)
	GetOrderedDependencies(ctx context.Context, tx *gorm.DB, moduleKey string) ([]string, error)
	GetInheritanceSubgraph(ctx context.Context, tx *gorm.DB, moduleKey string, direction GraphDirection, maxDepth *int) (*redpaths.InheritanceGraph, error)

	// module options
	AddOption(ctx context.Context, tx *gorm.DB, moduleOption *redpaths.ModuleOption) error
	GetOptions(ctx context.Context, tx *gorm.DB, moduleKey string) ([]*redpaths.ModuleOption, error)

	// module history
	AddRun(ctx context.Context, tx *gorm.DB, runMetadata *redpaths.ModuleRun) error
	GetAllModuleRuns(ctx context.Context, tx *gorm.DB, projectUID string) ([]*redpaths.ModuleRun, error)
}

type PostgresRedPathsModuleRepository struct{}

func (r *PostgresRedPathsModuleRepository) GetInheritanceSubgraph(
	ctx context.Context,
	tx *gorm.DB,
	moduleKey string,
	direction GraphDirection,
	maxDepth *int,
) (*redpaths.InheritanceGraph, error) {

	var edges []*redpaths.ModuleDependency
	var queries []string

	// Parameter für SQL
	var maxDepthVal interface{}
	if maxDepth != nil {
		maxDepthVal = *maxDepth
	} else {
		maxDepthVal = nil
	}

	// Build query based on direction
	switch direction {
	case GraphUpstream:
		queries = []string{upstreamCTE}
	case GraphDownstream:
		queries = []string{downstreamCTE}
	case GraphBoth:
		// Upstream + Downstream zusammenführen
		queries = []string{upstreamCTE, downstreamCTE}
	default:
		return nil, fmt.Errorf("unsupported graph direction: %s", direction)
	}

	edgesMap := make(map[string]*redpaths.ModuleDependency)

	for _, query := range queries {
		var tmpEdges []*redpaths.ModuleDependency
		if err := tx.WithContext(ctx).
			Raw(query, moduleKey, maxDepthVal).
			Scan(&tmpEdges).Error; err != nil {
			return nil, fmt.Errorf("failed to get subgraph: %w", err)
		}

		// Duplikate vermeiden
		for _, e := range tmpEdges {
			key := e.PreviousModule + "->" + e.NextModule
			if _, exists := edgesMap[key]; !exists {
				edgesMap[key] = e
			}
		}
	}

	// Map zu Slice
	edges = make([]*redpaths.ModuleDependency, 0, len(edgesMap))
	keysSet := map[string]struct{}{moduleKey: {}}
	for _, e := range edgesMap {
		edges = append(edges, e)
		keysSet[e.PreviousModule] = struct{}{}
		keysSet[e.NextModule] = struct{}{}
	}

	// Module laden
	var modules []*redpaths.Module
	if err := tx.WithContext(ctx).
		Table(TableModules).
		Where("key IN ?", mapKeys(keysSet)).
		Find(&modules).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch modules for subgraph: %w", err)
	}

	return &redpaths.InheritanceGraph{
		Nodes: modules,
		Edges: edges,
	}, nil
}

func mapKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (r *PostgresRedPathsModuleRepository) GetOrderedDependencies(ctx context.Context, tx *gorm.DB, moduleKey string) ([]string, error) {
	// SQL statement for recursive Common Table Expression (CTE)
	// This mimics the WITH RECURSIVE functionality from PostgreSQL
	// See: https://www.dylanpaulus.com/posts/postgres-is-a-graph-database/
	query := `
        WITH RECURSIVE dependent_modules AS (
            SELECT previous_module
            FROM redpaths_modules_dependencies
            WHERE next_module = ?
            
            UNION
            
            SELECT e.previous_module
            FROM redpaths_modules_dependencies e
            JOIN dependent_modules dm ON e.next_module = dm.previous_module
        )
        SELECT m.key
        FROM redpaths_modules m
        JOIN dependent_modules dm ON m.key = dm.previous_module
    `

	var moduleKeys []string

	if err := tx.WithContext(ctx).Raw(query, moduleKey).Scan(&moduleKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to get dependency key list: %w", err)
	}

	return moduleKeys, nil
}

func (r *PostgresRedPathsModuleRepository) AddOption(ctx context.Context, tx *gorm.DB, moduleOption *redpaths.ModuleOption) error {
	result := tx.WithContext(ctx).Table(TableModuleOptions).Create(&moduleOption)
	if result.Error != nil {
		return fmt.Errorf("create failed: %w", result.Error)
	}
	return nil
}

func (r *PostgresRedPathsModuleRepository) AddDependency(ctx context.Context, tx *gorm.DB, previousModuleKey, nextModuleKey string) (string, error) {
	dependency := &redpaths.ModuleDependency{PreviousModule: previousModuleKey, NextModule: nextModuleKey}
	result := tx.WithContext(ctx).Table(TableModuleDependencies).Create(&dependency)
	if result.Error != nil {
		return "", fmt.Errorf("create failed: %w", result.Error)
	}
	//TODO: Change
	return dependency.PreviousModule, nil
}

func (r *PostgresRedPathsModuleRepository) GetAllDependencies(ctx context.Context, tx *gorm.DB) ([]*redpaths.ModuleDependency, error) {
	var dependencies []*redpaths.ModuleDependency

	err := tx.WithContext(ctx).Table(TableModuleDependencies).Find(&dependencies).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get all edges: %w", err)
	}

	if len(dependencies) == 0 {
		log.Println("no edges found in database")
	}
	return dependencies, nil
}

func (r *PostgresRedPathsModuleRepository) GetAll(ctx context.Context, tx *gorm.DB) ([]*redpaths.Module, error) {
	var modules []*redpaths.Module

	err := tx.WithContext(ctx).Table(TableModules).Find(&modules).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get all modulelib: %w", err)
	}

	if len(modules) == 0 {
		log.Println("no modulelib found in database")
	}
	return modules, nil
}

func NewPostgresRedPathsModuleRepository() *PostgresRedPathsModuleRepository {
	return &PostgresRedPathsModuleRepository{}
}

func (r *PostgresRedPathsModuleRepository) CheckIfDependencyExits(ctx context.Context, tx *gorm.DB, previousModuleKey, nextModuleKey string) (bool, error) {
	query := tx.WithContext(ctx)
	var count int64
	err := query.Table(TableModuleDependencies).
		Where("previous_module = ?", previousModuleKey).
		Where("next_module = ?", nextModuleKey).
		Count(&count).
		Error
	if err != nil {
		return false, fmt.Errorf("failed to check if module exists by key: %w", err)
	}
	return count > 0, nil
}

func (r *PostgresRedPathsModuleRepository) CheckIfExistsByKey(ctx context.Context, tx *gorm.DB, key string) (bool, error) {
	query := tx.WithContext(ctx)
	var count int64
	err := query.Table(TableModules).
		Where("key = ?", key).
		Count(&count).
		Error
	if err != nil {
		return false, fmt.Errorf("failed to check if module exists by key: %w", err)
	}
	return count > 0, nil
}

func (r *PostgresRedPathsModuleRepository) CreateWithObject(ctx context.Context, tx *gorm.DB, module *redpaths.Module) (string, error) {
	result := tx.WithContext(ctx).Table(TableModules).Create(&module)
	if result.Error != nil {
		return "", fmt.Errorf("create failed: %w", result.Error)
	}
	return module.AttackID, nil
}

func (r *PostgresRedPathsModuleRepository) Get(ctx context.Context, tx *gorm.DB, moduleKey string) (*redpaths.Module, error) {
	{
		var module redpaths.Module

		tx := tx.WithContext(ctx)

		err := tx.Table(TableModules).
			First(&module, "key = ?", moduleKey).
			Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, rperrors.ErrNotFound
			}
			return nil, fmt.Errorf("database error: %w", err)
		}
		return &module, nil
	}
}

func (r *PostgresRedPathsModuleRepository) GetOptions(ctx context.Context, tx *gorm.DB, moduleKey string) ([]*redpaths.ModuleOption, error) {
	if moduleKey == "" {
		return nil, errors.New("moduleKey cannot be empty")
	}

	var options []*redpaths.ModuleOption

	result := tx.WithContext(ctx).
		Table(TableModuleOptions).
		Where("module_key = ?", moduleKey).
		Find(&options)

	if err := result.Error; err != nil {
		return nil, fmt.Errorf("failed to fetch module options: %w", err)
	}

	if options == nil {
		options = []*redpaths.ModuleOption{}
	}

	return options, nil
}

func (r *PostgresRedPathsModuleRepository) AddRun(ctx context.Context, tx *gorm.DB, runMetadata *redpaths.ModuleRun) error {
	if runMetadata.ModuleKey == "" {
		return fmt.Errorf("moduleKey cannot be empty")
	}

	result := tx.WithContext(ctx).Table(TableModuleRuns).Create(&runMetadata)

	if err := result.Error; err != nil {
		return fmt.Errorf("failed register new module run: %w", err)
	}

	return nil
}

func (r *PostgresRedPathsModuleRepository) GetAllModuleRuns(ctx context.Context, tx *gorm.DB, projectUID string) ([]*redpaths.ModuleRun, error) {
	var runs []*redpaths.ModuleRun

	result := tx.WithContext(ctx).
		Table(TableModuleRuns).
		Where("project_uid = ?", projectUID).
		Find(&runs)

	if err := result.Error; err != nil {
		return nil, fmt.Errorf("failed to get module runs for project: %s with error: %s", projectUID, err)
	}

	return runs, nil
}
