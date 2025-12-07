package service

import (
	"RedPaths-server/internal/db"
	rprepo "RedPaths-server/internal/repository/redpaths"
	"RedPaths-server/pkg/model/redpaths"
	"RedPaths-server/pkg/model/utils/pagination"
	"RedPaths-server/pkg/model/utils/query"
	"context"
	"fmt"

	"gorm.io/gorm"
)

type LogService struct {
	db              *gorm.DB
	redPathsLogRepo rprepo.RedPathsLogRepository
}

func NewLogService(postgresCon *gorm.DB) (*LogService, error) {

	return &LogService{
		db:              postgresCon,
		redPathsLogRepo: rprepo.NewPostgresrRedPathsLogRepository(),
	}, nil
}

func (s *LogService) CreateWithObject(ctx context.Context, log *redpaths.LogEntry) error {
	//if log.ModuleKey == "" || log.RunID == "" {
	//	return fmt.Errorf("error while creating log entry in log service: module key or run uid is empty")
	//}
	if log == nil {
		return fmt.Errorf("log entry is nil")
	}

	if log.ProjectUID == "" {
		log.ProjectUID = "SYSTEM"
	}

	// Re-enable validation to catch potential issues early

	return db.ExecutePostgresInTransaction(ctx, s.db, func(tx *gorm.DB) error {
		println("Storing log")
		println("For project: " + log.ProjectUID)
		return s.redPathsLogRepo.CreateLogEntry(ctx, tx, log)
	})
}

func (s *LogService) GetAllProjectLogs(ctx context.Context, projectUID string) ([]*redpaths.LogEntry, error) {
	return db.ExecutePostgresRead(ctx, s.db, func(db *gorm.DB) ([]*redpaths.LogEntry, error) {
		logEntries, err := s.redPathsLogRepo.GetLogsByProject(ctx, db, projectUID)
		if err != nil {
			return nil, err
		}
		return logEntries, nil
	})
}

func (s *LogService) GetProjectLogsWithOptions(ctx context.Context, projectUID string, opts *query.LogQueryOptions) (*pagination.PaginatedLogResult, error) {
	return db.ExecutePostgresRead(ctx, s.db, func(db *gorm.DB) (*pagination.PaginatedLogResult, error) {
		logEntries, err := s.redPathsLogRepo.GetLogsByProjectWithOptions(ctx, db, projectUID, opts)
		if err != nil {
			return nil, err
		}
		return logEntries, nil
	})
}

func (s *LogService) GetAllEventTypes(ctx context.Context, projectUID string) ([]*string, error) {
	return db.ExecutePostgresRead(ctx, s.db, func(db *gorm.DB) ([]*string, error) {
		eventTypes, err := s.redPathsLogRepo.GetEventTypeSet(ctx, db, projectUID)
		if err != nil {
			return nil, err
		}
		return eventTypes, nil
	})
}

func (s *LogService) GetModuleKeySet(ctx context.Context, projectUID string) ([]*string, error) {
	return db.ExecutePostgresRead(ctx, s.db, func(db *gorm.DB) ([]*string, error) {
		moduleKeys, err := s.redPathsLogRepo.GetModuleKeySet(ctx, db, projectUID)
		if err != nil {
			return nil, err
		}
		return moduleKeys, nil
	})
}
