package redpaths

import (
	"RedPaths-server/pkg/model/redpaths"
	"RedPaths-server/pkg/model/utils/pagination"
	"RedPaths-server/pkg/model/utils/query"
	"context"
	"fmt"
	"log"
	"strings"

	"gorm.io/gorm"
)

const (
	TableModuleLogs = "redpaths_module_logs"
)

type RedPathsLogRepository interface {
	CreateLogEntry(ctx context.Context, tx *gorm.DB, event *redpaths.LogEntry) error
	GetLogsByProject(ctx context.Context, tx *gorm.DB, projectUID string) ([]*redpaths.LogEntry, error)
	GetLogsByRun(ctx context.Context, tx *gorm.DB, runUID string) ([]*redpaths.LogEntry, error)
	GetLogsByModule(ctx context.Context, tx *gorm.DB, moduleKey string) ([]*redpaths.LogEntry, error)
	GetEventTypeSet(ctx context.Context, tx *gorm.DB, projectUID string) ([]*string, error)
	GetRunUidSet(ctx context.Context, tx *gorm.DB, projectUID string) ([]*string, error)
	GetModuleKeySet(ctx context.Context, tx *gorm.DB, projectUID string) ([]*string, error)

	GetLogsByProjectWithOptions(ctx context.Context, tx *gorm.DB, projectUID string, opts *query.LogQueryOptions) (*pagination.PaginatedLogResult, error)
}

type PostgresRedPathsLogRepository struct{}

func NewPostgresrRedPathsLogRepository() *PostgresRedPathsLogRepository {
	return &PostgresRedPathsLogRepository{}
}

func (r *PostgresRedPathsLogRepository) CreateLogEntry(ctx context.Context, tx *gorm.DB, logEntry *redpaths.LogEntry) error {
	log.Printf("creating log entry for module %s", logEntry.ModuleKey)
	result := tx.WithContext(ctx).Table(TableModuleLogs).Create(logEntry)
	if result.Error != nil {
		return fmt.Errorf("create of log entry failed: %w", result.Error)
	}
	return nil
}

func (r *PostgresRedPathsLogRepository) GetLogsByProject(ctx context.Context, tx *gorm.DB, projectUID string) ([]*redpaths.LogEntry, error) {
	var logs []*redpaths.LogEntry

	result := tx.WithContext(ctx).
		Table(TableModuleLogs).
		Where("project_uid = ?", projectUID).
		Or("project_uid = ?", "SYSTEM").
		Order("id DESC").
		Find(&logs)

	if result.Error != nil {
		return nil, fmt.Errorf("fetching logs by project failed: %w", result.Error)
	}
	log.Printf("found %d logs", len(logs))
	return logs, nil
}

func (r *PostgresRedPathsLogRepository) GetLogsByRun(ctx context.Context, tx *gorm.DB, runUID string) ([]*redpaths.LogEntry, error) {
	var logs []*redpaths.LogEntry

	result := tx.WithContext(ctx).
		Table(TableModuleLogs).
		Where("run_uid = ?", runUID).
		Order("created_at ASC").
		Find(&logs)

	if result.Error != nil {
		return nil, fmt.Errorf("fetching logs by run failed: %w", result.Error)
	}

	return logs, nil
}

func (r *PostgresRedPathsLogRepository) GetLogsByModule(ctx context.Context, tx *gorm.DB, moduleKey string) ([]*redpaths.LogEntry, error) {
	var logs []*redpaths.LogEntry

	result := tx.WithContext(ctx).
		Table(TableModuleLogs).
		Where("module_key = ?", moduleKey).
		Order("created_at DESC").
		Find(&logs)

	if result.Error != nil {
		return nil, fmt.Errorf("fetching logs by module failed: %w", result.Error)
	}

	return logs, nil
}

func (r *PostgresRedPathsLogRepository) GetRunUidSet(ctx context.Context, tx *gorm.DB, projectUID string) ([]*string, error) {
	var runUids []*string

	err := tx.WithContext(ctx).
		Table(TableModuleLogs).
		Where("project_uid = ?", projectUID).
		Distinct("run_uid").
		Order("run_uid").
		Pluck("run_uid", &runUids).
		Error

	if err != nil {
		return nil, fmt.Errorf("error fetching run uids: %w", err)
	}

	return runUids, nil
}

func (r *PostgresRedPathsLogRepository) GetModuleKeySet(ctx context.Context, tx *gorm.DB, projectUID string) ([]*string, error) {
	var moduleKeys []*string

	err := tx.WithContext(ctx).
		Table(TableModuleLogs).
		Where("project_uid = ?", projectUID).
		Distinct("module_key").
		Order("module_key").
		Pluck("module_key", &moduleKeys).
		Error

	if err != nil {
		return nil, fmt.Errorf("error fetching module key set: %w", err)
	}

	return moduleKeys, nil
}

func (r *PostgresRedPathsLogRepository) GetEventTypeSet(ctx context.Context, tx *gorm.DB, projectUID string) ([]*string, error) {
	var eventTypes []*string

	err := tx.WithContext(ctx).
		Table(TableModuleLogs).
		Where("project_uid = ?", projectUID).
		Distinct("event_type").
		Order("event_type").
		Pluck("event_type", &eventTypes).
		Error

	if err != nil {
		return nil, fmt.Errorf("error fetching event types: %w", err)
	}

	return eventTypes, nil
}

/*curl -X GET "http://localhost:8081/project/0x1/logs/query"   -H "Content-Type: application/json"   -d '{
"page": 1,
"pageSize": 1
}'
*/

func (r *PostgresRedPathsLogRepository) GetLogsByProjectWithOptions(
	ctx context.Context,
	tx *gorm.DB,
	projectUID string,
	opts *query.LogQueryOptions,
) (*pagination.PaginatedLogResult, error) {

	// Default Values
	if opts == nil {
		opts = &query.LogQueryOptions{}
	}
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.PageSize < 1 {
		opts.PageSize = 50 // Default
	}
	if opts.PageSize > 1000 {
		opts.PageSize = 1000 // Max
	}
	if opts.SortBy == "" {
		opts.SortBy = "timestamp"
	}
	if opts.SortOrder == "" {
		opts.SortOrder = "DESC"
	}

	// Base Query
	query := tx.WithContext(ctx).Table(TableModuleLogs)

	// Project Filter
	query = query.Where("project_uid = ? OR project_uid = ?", projectUID, "SYSTEM")

	// Search Term
	if opts.SearchTerm != "" {
		searchPattern := "%" + strings.ToLower(opts.SearchTerm) + "%"
		query = query.Where(
			"LOWER(message) LIKE ? OR LOWER(module_key) LIKE ?",
			searchPattern, searchPattern,
		)
	}

	// Event Type Filter
	if len(opts.EventTypes) > 0 {
		query = query.Where("event_type IN ?", opts.EventTypes)
	}

	// Module Key Filter
	if len(opts.ModuleKeys) > 0 {
		query = query.Where("module_key IN ?", opts.ModuleKeys)
	}

	// Time Range Filter
	if opts.StartTime != nil {
		query = query.Where("timestamp >= ?", opts.StartTime)
	}
	if opts.EndTime != nil {
		query = query.Where("timestamp <= ?", opts.EndTime)
	}

	// Total Count
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("counting logs failed: %w", err)
	}

	// Sorting
	orderClause := fmt.Sprintf("%s %s", opts.SortBy, strings.ToUpper(opts.SortOrder))
	query = query.Order(orderClause)

	// Pagination
	offset := (opts.Page - 1) * opts.PageSize
	query = query.Offset(offset).Limit(opts.PageSize)

	// Fetch Data
	var logs []*redpaths.LogEntry
	if err := query.Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("fetching logs failed: %w", err)
	}

	// Create Response
	totalPages := int((totalCount + int64(opts.PageSize) - 1) / int64(opts.PageSize))

	return &pagination.PaginatedLogResult{
		Logs:       logs,
		TotalCount: totalCount,
		Page:       opts.Page,
		PageSize:   opts.PageSize,
		TotalPages: totalPages,
	}, nil
}
