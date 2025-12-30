package active_directory

import (
	"RedPaths-server/internal/repository/dgraphutil"
	"RedPaths-server/pkg/model/active_directory"
	"context"
	"fmt"
	"time"

	"github.com/dgraph-io/dgo/v210"
)

type DirectoryNodeRepository interface {
	// CRUD
	Create(ctx context.Context, tx *dgo.Txn, directoryNode active_directory.DirectoryNode) (*active_directory.DirectoryNode, error)
	Get(ctx context.Context, tx *dgo.Txn, uid string) (*active_directory.DirectoryNode, error)
	Delete(ctx context.Context, tx *dgo.Txn, directoryNodeUID string) error

	AddSecurityPrincipal(ctx context.Context, tx *dgo.Txn, directoryNodeUID, securityPrincipalUID string) error
	AddParentDirectorNode(ctx context.Context, tx *dgo.Txn, directoryNodeUID, parentDirectoryNodeUID string) error

	UpdateDirectoryNode(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*active_directory.DirectoryNode, error)
}

type DgraphDirectoryNodeRepository struct {
	DB *dgo.Dgraph
}

func NewDgraphDirectoryNodeRepository(db *dgo.Dgraph) *DgraphDirectoryNodeRepository {
	return &DgraphDirectoryNodeRepository{DB: db}
}

func (r *DgraphDirectoryNodeRepository) Create(ctx context.Context, tx *dgo.Txn, directoryNode *active_directory.DirectoryNode, actor string) (*active_directory.DirectoryNode, error) {
	dgraphutil.InitCreateMetadata(&directoryNode.RedPathsMetadata, actor)
	return dgraphutil.CreateEntity(ctx, tx, "DirectoryNode", directoryNode)
}

func (r *DgraphDirectoryNodeRepository) Get(ctx context.Context, tx *dgo.Txn, uid string) (*active_directory.DirectoryNode, error) {
	query := `
        query DirectoryNode($uid: string) {
            directorynode(func: uid($uid)) {
                uid
                name
            }
        }
    `
	return dgraphutil.GetEntityByUID[active_directory.DirectoryNode](ctx, tx, uid, "directorynode", query)
}

func (r *DgraphActiveDirectoryRepository) AddSecurityPrincipal(ctx context.Context, tx *dgo.Txn, directoryNodeUID, securityPrincipalUID string) error {
	relationName := "locates"
	err := dgraphutil.AddRelation(ctx, tx, directoryNodeUID, securityPrincipalUID, relationName)
	if err != nil {
		return fmt.Errorf("error while linking security principal %s to directory node %s with relation %s", securityPrincipalUID, directoryNodeUID, relationName)
	}
	return nil
}

func (r *DgraphActiveDirectoryRepository) AddParentDirectorNode(ctx context.Context, tx *dgo.Txn, directoryNodeUID, parentDirectoryNodeUID string) error {
	relationName := "parent"
	err := dgraphutil.AddRelation(ctx, tx, directoryNodeUID, parentDirectoryNodeUID, relationName)
	if err != nil {
		return fmt.Errorf("error while linking parent directory node %s to directory node %s with relation %s", parentDirectoryNodeUID, directoryNodeUID, relationName)
	}
	return nil
}

func (r *DgraphDirectoryNodeRepository) UpdateDirectoryNode(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*active_directory.DirectoryNode, error) {
	// legacy
	fields["updated_at"] = time.Now().Format(time.RFC3339)
	return dgraphutil.UpdateAndGet(ctx, tx, uid, actor, fields, r.Get)
}
