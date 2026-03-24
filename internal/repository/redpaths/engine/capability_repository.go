package engine

import (
	dgraphutil2 "RedPaths-server/internal/repository/util/dgraph"
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/core/res"
	"RedPaths-server/pkg/model/engine"
	"RedPaths-server/pkg/schema"
	"context"
	"fmt"
	"time"

	"github.com/dgraph-io/dgo/v210"
)

type CapabilityRepository interface {
	// CRUD
	Create(ctx context.Context, tx *dgo.Txn, capability *engine.Capability, actor string) (*engine.Capability, error)
	Get(ctx context.Context, tx *dgo.Txn, uid string) (*engine.Capability, error)
	//Delete(ctx context.Context, tx *dgo.Txn, capabilityUID string) error
	Update(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*engine.Capability, error)

	// Finds
	//FindByDistinguishedNameInDomain(ctx context.Context, tx *dgo.Txn, domainUID string, dsName string) (*active_directory.DirectoryNode, error)

	GetAllByHostUID(ctx context.Context, tx *dgo.Txn, hostUID string) ([]*res.EntityResult[*engine.Capability], error)
	/*GetAllByDomainUID(ctx context.Context, tx *dgo.Txn, domainUID string) ([]*res.EntityResult[*engine.Capability], error)
	GetAllByGPOUID(ctx context.Context, tx *dgo.Txn, gpoUID string) ([]*res.EntityResult[*engine.Capability], error)
	GetALLByACEUID(ctx context.Context, tx *dgo.Txn, aceUID string) ([]*res.EntityResult[*engine.Capability], error)
	GetByProjectUID(ctx context.Context, tx *dgo.Txn, projectUID string) ([]*res.EntityResult[*engine.Capability], error)*/
}

type DgraphCapabilityRepository struct {
	DB *dgo.Dgraph
}

func NewDgraphCapabilityRepository(db *dgo.Dgraph) *DgraphCapabilityRepository {
	return &DgraphCapabilityRepository{DB: db}
}

func (r *DgraphCapabilityRepository) Create(ctx context.Context, tx *dgo.Txn, capability *engine.Capability, actor string) (*engine.Capability, error) {
	dgraphutil2.InitCreateMetadata(&capability.RedPathsMetadata, actor)
	return dgraphutil2.CreateEntity(ctx, tx, "Capability", capability)
}

func (r *DgraphCapabilityRepository) Update(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*engine.Capability, error) {
	// legacy
	fields["updated_at"] = time.Now().Format(time.RFC3339)
	return dgraphutil2.UpdateAndGet(ctx, tx, uid, actor, fields, r.Get)
}

func (r *DgraphCapabilityRepository) Get(ctx context.Context, tx *dgo.Txn, uid string) (*engine.Capability, error) {
	query := `
        query Capability($uid: string) {
            capability(func: uid($uid)) {
                uid
                capability.name
				capability.scope
            }
        }
    `
	return dgraphutil2.GetEntityByUID[engine.Capability](ctx, tx, uid, "capability", query)
}

func (r *DgraphCapabilityRepository) GetAllByHostUID(ctx context.Context, tx *dgo.Txn, hostUID string) ([]*res.EntityResult[*engine.Capability], error) {
	fields, err := schema.DefaultFields("Capability")
	if err != nil {
		return nil, fmt.Errorf("GetAllByHostUID: %w", err)
	}

	return dgraphutil2.GetEntitiesWithAssertions[*engine.Capability](
		ctx,
		tx,
		hostUID,
		core.PredicateDerives,
		"Capability",
		fields,
		"getHostsCapabilities",
	)
}

/*func (r *DgraphCapabilityRepository) Delete(ctx context.Context, tx *dgo.Txn, directoryNodeUID string) error {
	//TODO implement me
	panic("implement me")
}

func (r *DgraphCapabilityRepository) AddParentDirectorNode(ctx context.Context, tx *dgo.Txn, directoryNodeUID, parentDirectoryNodeUID string) error {
	//TODO implement me
	panic("implement me")
}



func (r *DgraphCapabilityRepository) GetByProjectUID(ctx context.Context, tx *dgo.Txn, projectUID string) ([]*res.EntityResult[*active_directory.DirectoryNode], error) {
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

	return dgraph.GetEntitiesWithAssertionsNHop[*active_directory.DirectoryNode](
		ctx, tx, projectUID,
		[]dgraph.HopConfig{
			{Predicate: core.PredicateHasActiveDirectory},
			{Predicate: core.PredicateHasDomain, ObjectType: "Domain"},
			{Predicate: core.PredicateContains, ObjectType: "DirectoryNode"},
		},

		fields, "getProjectDirectorNodes",
	)

}

func (r *DgraphCapabilityRepository) FindByDistinguishedNameInDomain(ctx context.Context, tx *dgo.Txn, domainUID string, dsName string) (*active_directory.DirectoryNode, error) {
	fields := []string{
		"uid",
		"domain.name",
		"created_at",
		"modified_at",
		"dgraph.type",
	}

	return dgraph.FindEntityByFieldViaAssertion[active_directory.DirectoryNode](
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


func (r *DgraphDirectoryNodeRepository) Create(ctx context.Context, tx *dgo.Txn, directoryNode *active_directory.DirectoryNode, actor string) (*active_directory.DirectoryNode, error) {
	dgraph.InitCreateMetadata(&directoryNode.RedPathsMetadata, actor)
	return dgraph.CreateEntity(ctx, tx, "DirectoryNode", directoryNode)
}

func (r *DgraphDirectoryNodeRepository) GetProjectActiveDirectory(ctx context.Context, tx *dgo.Txn, uid string) (*active_directory.DirectoryNode, error) {
	query := `
        query DirectoryNode($uid: string) {
            directorynode(func: uid($uid)) {
                uid
                directory_node.name
            }
        }
    `
	return dgraph.GetEntityByUID[active_directory.DirectoryNode](ctx, tx, uid, "directorynode", query)
}

func (r *DgraphDirectoryNodeRepository) Update(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*active_directory.DirectoryNode, error) {
	// legacy
	fields["updated_at"] = time.Now().Format(time.RFC3339)
	return dgraph.UpdateAndGet(ctx, tx, uid, actor, fields, r.GetProjectActiveDirectory)
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

	return dgraph.GetEntitiesWithAssertions[*active_directory.DirectoryNode](
		ctx,
		tx,
		domainUID,
		core.PredicateContains,
		"DirectoryNode",
		fields,
		"getDomainDirectoryNodes",
	)
}
*/
