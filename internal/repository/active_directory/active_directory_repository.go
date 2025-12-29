package active_directory

import (
	"RedPaths-server/pkg/model"
	"RedPaths-server/pkg/model/active_directory"
	"context"

	"github.com/dgraph-io/dgo/v210"
)

// ActiveDirectoryRepository defines operations for project data access
type ActiveDirectoryRepository interface {
	// CRUD
	Create(ctx context.Context, tx *dgo.Txn, name string) (string, error)
	Get(ctx context.Context, tx *dgo.Txn, uid string) (*active_directory.ActiveDirectory, error)
	GetAll(ctx context.Context, tx *dgo.Txn) ([]*active_directory.ActiveDirectory, error)

	Delete(ctx context.Context, tx *dgo.Txn, uid string) error
	UpdateActiveDirectory(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*model.Project, error)

	// Relations
	AddDomain(ctx context.Context, tx *dgo.Txn, projectUID, domainUID string) error
	AddTarget(ctx context.Context, tx *dgo.Txn, projectUID, targetUID string) error
	AddHostWithUnknownDomain(ctx context.Context, tx *dgo.Txn, projectUID, hostUID string) error
	AddUser(ctx context.Context, tx *dgo.Txn, projectUID, userUID string) error
}
