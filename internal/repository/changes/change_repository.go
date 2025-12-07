package changes

import (
	"RedPaths-server/pkg/model/redpaths/change"
	"context"

	"gorm.io/gorm"
)

const (
	TableChangeEvents  = "redpaths_change_event"
	TableNodeSnapshots = "redpaths_node_snapshot"
)

type RedPathsChangeRepository interface {

	//CRUD
	GetAllChangeEventsByProject(ctx context.Context, tx *gorm.DB, projectUID string) ([]*change.Event, error)
	GetAllEventsByTransaction(ctx context.Context, tx *gorm.DB, moduleRunID string) ([]*change.Event, error)
	CreateChangeEvent(ctx context.Context, tx *gorm.DB, event *change.Event) error
}

type PostgresRedPathsModuleRepository struct{}
