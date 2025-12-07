package changes

import (
	"RedPaths-server/pkg/model/redpaths/history"
	"context"

	"gorm.io/gorm"
)

const (
	TableChangeEvents  = "redpaths_change_event"
	TableNodeSnapshots = "redpaths_node_snapshot"
)

type RedPathsChangeRepository interface {

	//CRUD
	GetAllChangeEventsByProject(ctx context.Context, tx *gorm.DB, projectUID string) ([]*history.Event, error)
	GetAllEventsByTransaction(ctx context.Context, tx *gorm.DB, moduleRunID string) ([]*history.Event, error)
	CreateChangeEvent(ctx context.Context, tx *gorm.DB, event *history.Event) error
}

type PostgresRedPathsModuleRepository struct{}
