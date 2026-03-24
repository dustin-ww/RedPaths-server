package change

import (
	"RedPaths-server/internal/db"
	"RedPaths-server/internal/repository/redpaths/changes"
	"RedPaths-server/pkg/model/redpaths/history"
	"context"

	"gorm.io/gorm"
)

type ChangeService struct {
	db                 *gorm.DB
	redPathsChangeRepo changes.RedPathsChangeRepository
}

func NewChangeService(postgresCon *gorm.DB) (*ChangeService, error) {

	return &ChangeService{
		db:                 postgresCon,
		redPathsChangeRepo: changes.NewPostgresRedPathsChangesRepository(postgresCon),
	}, nil
}

func (s *ChangeService) GetChangesByEntity(
	ctx context.Context,
	entityType, entityUID string,
) ([]*history.Change, error) {
	return db.ExecutePostgresRead(ctx, s.db, func(tx *gorm.DB) ([]*history.Change, error) {
		return s.redPathsChangeRepo.GetByEntity(ctx, tx, entityType, entityUID)
	})
}

func (s *ChangeService) GetChangesByEntityWithOptions(
	ctx context.Context,
	entityType, entityUID string,
	opts *changes.ChangeQueryOptions,
) (*changes.PaginatedChangeResult, error) {
	return db.ExecutePostgresRead(ctx, s.db, func(tx *gorm.DB) (*changes.PaginatedChangeResult, error) {
		return s.redPathsChangeRepo.GetByEntityWithOptions(ctx, tx, entityType, entityUID, opts)
	})
}

func (s *ChangeService) SaveChange(
	ctx context.Context,
	change *history.Change,
) error {
	return db.ExecutePostgresInTransaction(ctx, s.db, func(tx *gorm.DB) error {
		return s.redPathsChangeRepo.Save(ctx, tx, change)
	})
}
