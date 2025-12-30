package active_directory

import (
	"RedPaths-server/pkg/model/active_directory"
	"context"

	"github.com/dgraph-io/dgo/v210"
)

type SecurityPrincipalRepository interface {
	// CRUD
	Create(ctx context.Context, tx *dgo.Txn, activeDirectory *active_directory.ActiveDirectory) (*active_directory.ActiveDirectory, error)
	Get(ctx context.Context, tx *dgo.Txn, uid string) (*active_directory.ActiveDirectory, error)
	UpdateActiveDirectory(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*active_directory.ActiveDirectory, error)
	Delete(ctx context.Context, tx *dgo.Txn, uid string) error

	// Relations
	AddUser(ctx context.Context, tx *dgo.Txn, securityPrincipalUID, userUID string) error
}
