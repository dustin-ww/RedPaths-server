package redpaths

import (
	"RedPaths-server/pkg/model/redpaths"
	"context"
	"fmt"

	"gorm.io/gorm"
)

const (
	TableVectorRuns = "redpaths_vector_runs"
)

type RedPathsVectorRepository interface {

	// module history
	AddRun(ctx context.Context, tx *gorm.DB, runMetadata *redpaths.VectorRun) error
	GetAllVectorRuns(ctx context.Context, tx *gorm.DB, projectUID string) ([]*redpaths.VectorRun, error)
}

type PostgresRedPathsVectorRepository struct {
}

func NewPostgresRedPathsVectorRepository() *PostgresRedPathsVectorRepository {
	return &PostgresRedPathsVectorRepository{}
}

func (r *PostgresRedPathsVectorRepository) AddRun(ctx context.Context, tx *gorm.DB, runMetadata *redpaths.VectorRun) error {

	result := tx.WithContext(ctx).Table(TableVectorRuns).Create(&runMetadata)

	if err := result.Error; err != nil {
		return fmt.Errorf("failed to store new vector run: %w", err)
	}

	return nil
}

func (r *PostgresRedPathsVectorRepository) GetAllVectorRuns(ctx context.Context, tx *gorm.DB, projectUID string) ([]*redpaths.VectorRun, error) {
	var runs []*redpaths.VectorRun

	result := tx.WithContext(ctx).
		Table(TableVectorRuns).
		Where("project_uid = ?", projectUID).
		Find(&runs)

	if err := result.Error; err != nil {
		return nil, fmt.Errorf("failed to get vector runs for project: %s with error: %s", projectUID, err)
	}

	return runs, nil
}
