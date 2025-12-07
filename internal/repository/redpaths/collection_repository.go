package redpaths

import (
	"RedPaths-server/pkg/model/redpaths"
	"context"
	"fmt"
	"log"

	"gorm.io/gorm"
)

type RedPathsCollectionRepository interface {
	//CRUD
	GetAll(ctx context.Context, tx *gorm.DB) ([]*redpaths.Collection, error)
	Create(ctx context.Context, tx *gorm.DB, name string, description string) (uint, error)
	AddModule(ctx context.Context, tx *gorm.DB, collectionID, moduleKey string)
	GetModulesForCollection(ctx context.Context, tx *gorm.DB, collectionID uint) ([]redpaths.Module, error)
}

type PostgresRedPathsCollectionRepository struct {
}

func NewPostgresRedPathsCollectionRepository() *PostgresRedPathsCollectionRepository {
	return &PostgresRedPathsCollectionRepository{}
}

func (r *PostgresRedPathsCollectionRepository) GetAll(ctx context.Context, tx *gorm.DB) ([]*redpaths.Collection, error) {
	var collections []*redpaths.Collection

	err := tx.WithContext(ctx).Table("redpaths_collections").Find(&collections).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get all redpaths collections: %w", err)
	}

	return collections, nil
}

func (r *PostgresRedPathsCollectionRepository) GetModulesForCollection(ctx context.Context, tx *gorm.DB, collectionID uint) ([]redpaths.Module, error) {
	var moduleKeys []string

	err := tx.WithContext(ctx).
		Table("redpaths_collection_modules").
		Where("collection_id = ?", collectionID).
		Pluck("module_key", &moduleKeys).
		Error

	if err != nil {
		return nil, fmt.Errorf("failed to get module keys for collection %d: %w", collectionID, err)
	}

	if len(moduleKeys) == 0 {
		return []redpaths.Module{}, nil
	}

	// Module laden
	var modules []redpaths.Module

	err = tx.WithContext(ctx).
		Table("redpaths_modules").
		Where("key IN ?", moduleKeys).
		Find(&modules).
		Error

	if err != nil {
		return nil, fmt.Errorf("failed to get modulelib for collection %d: %w", collectionID, err)
	}

	return modules, nil
}

func (r *PostgresRedPathsCollectionRepository) Create(ctx context.Context, tx *gorm.DB, name string, description string) (uint, error) {
	collection := &redpaths.Collection{Name: name, Description: description}

	err := tx.WithContext(ctx).Table("redpaths_collections").Create(collection).Error
	if err != nil {
		log.Printf("failed to create redpaths collection: %v", err)
		return 0, err
	}

	return collection.ID, nil
}

func (r *PostgresRedPathsCollectionRepository) AddModule(ctx context.Context, tx *gorm.DB, collectionID, moduleKey string) {

}
