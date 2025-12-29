package active_directory

import (
	"RedPaths-server/internal/repository/dgraphutil"
	"RedPaths-server/pkg/model/active_directory"
	"context"
	"fmt"
	"time"

	"github.com/dgraph-io/dgo/v210"
)

type ActiveDirectoryRepository interface {
	// CRUD
	Create(ctx context.Context, tx *dgo.Txn, activeDirectory *active_directory.ActiveDirectory) (*active_directory.ActiveDirectory, error)
	Get(ctx context.Context, tx *dgo.Txn, uid string) (*active_directory.ActiveDirectory, error)
	UpdateActiveDirectory(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*active_directory.ActiveDirectory, error)
	Delete(ctx context.Context, tx *dgo.Txn, uid string) error

	// Relations
	AddDomain(ctx context.Context, tx *dgo.Txn, activeDirectoryUID, domainUID string) error
}

type DgraphActiveDirectoryRepository struct {
	DB *dgo.Dgraph
}

func NewDgraphActiveDirectoryRepository(db *dgo.Dgraph) *DgraphActiveDirectoryRepository {
	return &DgraphActiveDirectoryRepository{DB: db}
}

// Create adds a new project to the database
func (r *DgraphActiveDirectoryRepository) Create(ctx context.Context, tx *dgo.Txn, activeDirectory *active_directory.ActiveDirectory, actor string) (*active_directory.ActiveDirectory, error) {
	dgraphutil.InitCreateMetadata(&activeDirectory.RedPathsMetadata, actor)
	return dgraphutil.CreateEntity(ctx, tx, "ActiveDirectory", activeDirectory)
}

func (r *DgraphActiveDirectoryRepository) Get(ctx context.Context, tx *dgo.Txn, uid string) (*active_directory.ActiveDirectory, error) {
	query := `
        query ActiveDirectory($uid: string) {
            activedirectory(func: uid($uid)) {
                uid
                name
            }
        }
    `
	return dgraphutil.GetEntityByUID[active_directory.ActiveDirectory](ctx, tx, uid, "activedirectory", query)
}

func (r *DgraphActiveDirectoryRepository) UpdateActiveDirectory(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*active_directory.ActiveDirectory, error) {
	// legacy
	fields["updated_at"] = time.Now().Format(time.RFC3339)
	return dgraphutil.UpdateAndGet(ctx, tx, uid, actor, fields, r.Get)
}

func (r *DgraphActiveDirectoryRepository) Delete(ctx context.Context, tx *dgo.Txn, uid string) error {
	panic("implement me")
}

// AddDomain connects a domain to a project
func (r *DgraphActiveDirectoryRepository) AddDomain(ctx context.Context, tx *dgo.Txn, projectUID, domainUID string) error {
	relationName := "has_domain"
	err := dgraphutil.AddRelation(ctx, tx, projectUID, domainUID, relationName)
	if err != nil {
		return fmt.Errorf("error while linking domain %s to project %s with relation %s", domainUID, projectUID, relationName)
	}
	return nil
}
