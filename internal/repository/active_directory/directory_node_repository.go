package active_directory

import (
	dgraphutil2 "RedPaths-server/internal/repository/util/dgraph"
	"RedPaths-server/pkg/model/active_directory"
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/core/res"
	"context"
	"time"

	"github.com/dgraph-io/dgo/v210"
)

type DirectoryNodeRepository interface {
	// CRUD
	Create(ctx context.Context, tx *dgo.Txn, directoryNode *active_directory.DirectoryNode, actor string) (*active_directory.DirectoryNode, error)
	Get(ctx context.Context, tx *dgo.Txn, uid string) (*active_directory.DirectoryNode, error)
	Delete(ctx context.Context, tx *dgo.Txn, directoryNodeUID string) error
	Update(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*active_directory.DirectoryNode, error)

	// Finds
	FindByDistinguishedNameInDomain(ctx context.Context, tx *dgo.Txn, domainUID string, dsName string) (*active_directory.DirectoryNode, error)

	GetAllByDomainUID(ctx context.Context, tx *dgo.Txn, domainUID string) ([]*res.EntityResult[*active_directory.DirectoryNode], error)

	GetByProjectUID(ctx context.Context, tx *dgo.Txn, projectUID string) ([]*res.EntityResult[*active_directory.DirectoryNode], error)
	FindExisting(ctx context.Context, tx *dgo.Txn, projectUID string, node *active_directory.DirectoryNode) (*dgraphutil2.ExistenceResult[*active_directory.DirectoryNode], error)
}

type DgraphDirectoryNodeRepository struct {
	DB *dgo.Dgraph
}

func (r *DgraphDirectoryNodeRepository) Delete(ctx context.Context, tx *dgo.Txn, directoryNodeUID string) error {
	//TODO implement me
	panic("implement me")
}

func (r *DgraphDirectoryNodeRepository) AddParentDirectorNode(ctx context.Context, tx *dgo.Txn, directoryNodeUID, parentDirectoryNodeUID string) error {
	//TODO implement me
	panic("implement me")
}

func (r *DgraphDirectoryNodeRepository) GetByProjectUID(ctx context.Context, tx *dgo.Txn, projectUID string) ([]*res.EntityResult[*active_directory.DirectoryNode], error) {
	fields := []string{
		"uid",
		"directory_node.name",
		"directory_node.description",
		"directory_node.distinguished_name",
		"directory_node.node_type",
		"directory_node.is_builtin",
		"created_at",
		"modified_at",
		"dgraph.type",
	}

	return dgraphutil2.GetEntitiesWithAssertionsNHop[*active_directory.DirectoryNode](
		ctx, tx, projectUID,
		[]dgraphutil2.HopConfig{
			{Predicate: core.PredicateHasActiveDirectory},
			{Predicate: core.PredicateHasDomain, ObjectType: "Domain"},
			{Predicate: core.PredicateContains, ObjectType: "DirectoryNode"},
		},

		fields, "getProjectDirectorNodes",
	)

}

func (r *DgraphDirectoryNodeRepository) FindByDistinguishedNameInDomain(ctx context.Context, tx *dgo.Txn, domainUID string, dsName string) (*active_directory.DirectoryNode, error) {
	fields := []string{
		"uid",
		"domain.name",
		"created_at",
		"modified_at",
		"dgraph.type",
	}

	return dgraphutil2.FindEntityByFieldViaAssertion[active_directory.DirectoryNode](
		ctx,
		tx,
		domainUID,
		core.PredicateContains,
		"DirectoryNode",
		"directory_node.distinguished_name",
		dsName,
		fields,
	)
}

func NewDgraphDirectoryNodeRepository(db *dgo.Dgraph) *DgraphDirectoryNodeRepository {
	return &DgraphDirectoryNodeRepository{DB: db}
}

func (r *DgraphDirectoryNodeRepository) Create(ctx context.Context, tx *dgo.Txn, directoryNode *active_directory.DirectoryNode, actor string) (*active_directory.DirectoryNode, error) {
	dgraphutil2.InitCreateMetadata(&directoryNode.RedPathsMetadata, actor)
	return dgraphutil2.CreateEntity(ctx, tx, "DirectoryNode", directoryNode)
}

func (r *DgraphDirectoryNodeRepository) Get(ctx context.Context, tx *dgo.Txn, uid string) (*active_directory.DirectoryNode, error) {
	query := `
        query DirectoryNode($uid: string) {
            directorynode(func: uid($uid)) {
                uid
                directory_node.name
            }
        }
    `
	return dgraphutil2.GetEntityByUID[active_directory.DirectoryNode](ctx, tx, uid, "directorynode", query)
}

func (r *DgraphDirectoryNodeRepository) Update(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*active_directory.DirectoryNode, error) {
	// legacy
	fields["updated_at"] = time.Now().Format(time.RFC3339)
	return dgraphutil2.UpdateAndGet(ctx, tx, uid, actor, fields, r.Get)
}

var directoryNodeHierarchyHops = []dgraphutil2.HopConfig{
	{Predicate: core.PredicateHasActiveDirectory},
	{Predicate: core.PredicateHasDomain, ObjectType: "Domain"},
	{Predicate: core.PredicateContains, ObjectType: "DirectoryNode"},
}

var directoryNodeFields = []string{
	"uid",
	"directory_node.name",
	"directory_node.description",
	"directory_node.distinguished_name",
	"directory_node.node_type",
	"directory_node.object_class",
	"directory_node.is_builtin",
	"directory_node.is_protected",
	"dgraph.type",
}

func BuildDirectoryNodeFilter(node *active_directory.DirectoryNode) []dgraphutil2.UniqueFieldFilter {
	return []dgraphutil2.UniqueFieldFilter{
		{Field: "directory_node.distinguished_name", Value: node.DistinguishedName},
		{Field: "directory_node.name", Value: node.Name},
	}
}

// FindExisting performs a two-phase existence check for a DirectoryNode.
//
// Phase 1: Searches via the project hierarchy (Project → AD → Domain → DirectoryNode).
// Phase 2: Falls back to direct project-level search for orphaned nodes.
func (r *DgraphDirectoryNodeRepository) FindExisting(
	ctx context.Context,
	tx *dgo.Txn,
	projectUID string,
	node *active_directory.DirectoryNode,
) (*dgraphutil2.ExistenceResult[*active_directory.DirectoryNode], error) {

	filters := BuildDirectoryNodeFilter(node)

	return dgraphutil2.CheckEntityExists[*active_directory.DirectoryNode](
		ctx, tx,
		projectUID,
		"DirectoryNode",
		filters,
		dgraphutil2.FilterModeOR,
		directoryNodeFields,
		directoryNodeHierarchyHops,
	)
}

func (r *DgraphDirectoryNodeRepository) GetAllByDomainUID(ctx context.Context, tx *dgo.Txn, domainUID string) ([]*res.EntityResult[*active_directory.DirectoryNode], error) {
	fields := []string{
		"uid",
		"directory_node.name",
		"directory_node.description",
		"directory_node.distinguished_name",
		"directory_node.node_type",
		"directory_node.is_builtin",
		"dgraph.type",
		"discovered_by",
		"discovered_at",
		"last_seen_at",
		"last_seen_by",
	}

	return dgraphutil2.GetEntitiesWithAssertions[*active_directory.DirectoryNode](
		ctx,
		tx,
		domainUID,
		core.PredicateContains,
		"DirectoryNode",
		fields,
		"getDomainDirectoryNodes",
	)
}
