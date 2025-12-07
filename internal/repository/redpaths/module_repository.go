package redpaths

import (
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
)

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

	// module options
	AddOption(ctx context.Context, tx *gorm.DB, moduleOption *redpaths.ModuleOption) error
	GetOptions(ctx context.Context, tx *gorm.DB, moduleKey string) ([]*redpaths.ModuleOption, error)

	// module history

}

type PostgresRedPathsModuleRepository struct{}

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
				return nil, fmt.Errorf("module not found: %s", moduleKey)
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
