package changes

import (
	"RedPaths-server/pkg/model/redpaths/history"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	TableChanges = "redpaths_changes"
)

type ChangeQueryOptions struct {
	ChangeTypes []string
	StartTime   *time.Time
	EndTime     *time.Time
	Page        int
	PageSize    int
	SortOrder   string
}

type PaginatedChangeResult struct {
	Changes    []*history.Change
	TotalCount int64
	Page       int
	PageSize   int
	TotalPages int
}

type RedPathsChangeRepository interface {
	Save(ctx context.Context, tx *gorm.DB, change *history.Change) error
	GetByEntity(ctx context.Context, tx *gorm.DB, entityType, entityUID string) ([]*history.Change, error)
	GetByEntityWithOptions(ctx context.Context, tx *gorm.DB, entityType, entityUID string, opts *ChangeQueryOptions) (*PaginatedChangeResult, error)
}

type PostgresRedPathsChangesRepository struct {
	PDB *gorm.DB
}

func NewPostgresRedPathsChangesRepository(pdb *gorm.DB) *PostgresRedPathsChangesRepository {
	return &PostgresRedPathsChangesRepository{
		PDB: pdb,
	}
}

func (r *PostgresRedPathsChangesRepository) Save(
	ctx context.Context,
	tx *gorm.DB,
	change *history.Change,
) error {

	if change.ID == uuid.Nil {
		change.ID = uuid.New()
	}

	if change.ChangedAt.IsZero() {
		change.ChangedAt = time.Now().UTC()
	}

	return tx.WithContext(ctx).Table(TableChanges).Create(change).Error
}

func (r *PostgresRedPathsChangesRepository) GetByEntity(
	ctx context.Context,
	tx *gorm.DB,
	entityType, entityUID string,
) ([]*history.Change, error) {
	var result []*history.Change

	err := tx.WithContext(ctx).
		Table(TableChanges).
		Where("entity_type = ? AND entity_uid = ?", entityType, entityUID).
		Order("changed_at DESC").
		Find(&result).Error

	if err != nil {
		return nil, fmt.Errorf("fetching changes for entity %s/%s failed: %w", entityType, entityUID, err)
	}

	return result, nil
}

func (r *PostgresRedPathsChangesRepository) GetByEntityWithOptions(
	ctx context.Context,
	tx *gorm.DB,
	entityType, entityUID string,
	opts *ChangeQueryOptions,
) (*PaginatedChangeResult, error) {

	if opts == nil {
		opts = &ChangeQueryOptions{}
	}
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.PageSize < 1 {
		opts.PageSize = 50
	}
	if opts.PageSize > 500 {
		opts.PageSize = 500
	}
	if opts.SortOrder == "" {
		opts.SortOrder = "DESC"
	}

	q := tx.WithContext(ctx).
		Table(TableChanges).
		Where("entity_type = ? AND entity_uid = ?", entityType, entityUID)

	if len(opts.ChangeTypes) > 0 {
		q = q.Where("change_type IN ?", opts.ChangeTypes)
	}
	if opts.StartTime != nil {
		q = q.Where("changed_at >= ?", opts.StartTime)
	}
	if opts.EndTime != nil {
		q = q.Where("changed_at <= ?", opts.EndTime)
	}

	var totalCount int64
	if err := q.Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("counting changes failed: %w", err)
	}

	orderClause := fmt.Sprintf("changed_at %s", strings.ToUpper(opts.SortOrder))
	offset := (opts.Page - 1) * opts.PageSize

	var changes []*history.Change
	err := q.Order(orderClause).
		Offset(offset).
		Limit(opts.PageSize).
		Find(&changes).Error

	if err != nil {
		return nil, fmt.Errorf("fetching changes for entity %s/%s failed: %w", entityType, entityUID, err)
	}

	totalPages := int((totalCount + int64(opts.PageSize) - 1) / int64(opts.PageSize))

	return &PaginatedChangeResult{
		Changes:    changes,
		TotalCount: totalCount,
		Page:       opts.Page,
		PageSize:   opts.PageSize,
		TotalPages: totalPages,
	}, nil
}
