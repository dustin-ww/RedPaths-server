package changes

import (
	"RedPaths-server/pkg/model/redpaths/history"
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	TableChanges = "redpaths_changes"
)

type RedPathsChangeRepository interface {
	Save(ctx context.Context, tx *gorm.DB, change *history.Change) error
	GetByEntity(ctx context.Context, tx *gorm.DB, entityType, entityUID string) ([]*history.Change, error)
}

type PostgresRedPathsChangesRepository struct{}

func NewPostgresRedPathsChangesRepository() *PostgresRedPathsChangesRepository {
	return &PostgresRedPathsChangesRepository{}
}

func (r *PostgresRedPathsChangesRepository) Save(
	ctx context.Context,
	tx *gorm.DB,
	change *history.Change,
) error {

	if change.UID == uuid.Nil {
		change.UID = uuid.New()
	}

	if change.ChangedAt.IsZero() {
		change.ChangedAt = time.Now().UTC()
	}

	return tx.WithContext(ctx).Create(change).Error
}

func (r *PostgresRedPathsChangesRepository) GetByEntity(ctx context.Context, tx *gorm.DB, entityType, entityUID string) ([]*history.Change, error) {
	//TODO implement me
	panic("implement me")
}
