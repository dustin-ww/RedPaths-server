package active_directory

import (
	"RedPaths-server/internal/repository/dgraphutil"
	"RedPaths-server/pkg/model/active_directory"
	"RedPaths-server/pkg/model/core"
	"context"
	"time"

	"github.com/dgraph-io/dgo/v210"
)

type ActiveDirectoryRepository interface {
	// CRUD
	Create(ctx context.Context, tx *dgo.Txn, activeDirectory *active_directory.ActiveDirectory, actor string) (*active_directory.ActiveDirectory, error)
	Get(ctx context.Context, tx *dgo.Txn, uid string) (*active_directory.ActiveDirectory, error)
	Update(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*active_directory.ActiveDirectory, error)
	Delete(ctx context.Context, tx *dgo.Txn, uid string) error

	// With Assertions
	GetByProjectUID(ctx context.Context, tx *dgo.Txn, projectUID string) ([]*core.EntityResult[*active_directory.ActiveDirectory], error)
	FindByForestNameInProject(ctx context.Context, tx *dgo.Txn, projectUID, adForestName string) (*active_directory.ActiveDirectory, error)
}

type DgraphActiveDirectoryRepository struct {
	DB *dgo.Dgraph
}

func NewDgraphActiveDirectoryRepository(db *dgo.Dgraph) *DgraphActiveDirectoryRepository {
	return &DgraphActiveDirectoryRepository{DB: db}
}

func (r *DgraphActiveDirectoryRepository) FindByForestNameInProject(
	ctx context.Context,
	tx *dgo.Txn,
	projectUID, adForestName string,
) (*active_directory.ActiveDirectory, error) {

	fields := []string{
		"uid",
		"active_directory.forest_name",
		"active_directory.forest_functional_level",
		"created_at",
		"modified_at",
		"dgraph.type",
	}

	return dgraphutil.FindEntityByFieldViaAssertion[active_directory.ActiveDirectory](
		ctx,
		tx,
		projectUID,
		core.PredicateHasActiveDirectory,
		"ActiveDirectory",
		"active_directory.forest_name",
		adForestName,
		fields,
	)
}

func (r *DgraphActiveDirectoryRepository) GetByProjectUID(
	ctx context.Context,
	tx *dgo.Txn,
	projectUID string,
) ([]*core.EntityResult[*active_directory.ActiveDirectory], error) {

	fields := []string{
		"uid",
		"active_directory.forest_name",
		"active_directory.forest_functional_level",
		"created_at",
		"modified_at",
		"discovered_at",
		"discovered_by",
		"validated_at",
		"validated_by",
		"dgraph.type",
	}

	return dgraphutil.GetEntitiesWithAssertions[*active_directory.ActiveDirectory](
		ctx,
		tx,
		projectUID,
		core.PredicateHasActiveDirectory,
		"ActiveDirectory",
		fields,
		"getProjectADs",
	)
}

/*func (r *DgraphActiveDirectoryRepository) GetByProjectUID(ctx context.Context, tx *dgo.Txn, projectUID string) ([]*active_directory.ActiveDirectory, error) {
	fields := []string{
		"uid",
		"forest_name",
		"forest_functional_level",
		"dgraph.type",
		"~project.has_ad { uid }",
	}

	activeDirectoryForests, err := dgraphutil.GetEntitiesByRelation[*active_directory.ActiveDirectory](
		ctx,
		tx,
		"ActiveDirectory",
		"~project.has_ad",
		projectUID,
		fields,
	)
	if err != nil {
		return nil, err
	}

	log.Printf("Found %d ads for project %s\n", len(activeDirectoryForests), projectUID)
	return activeDirectoryForests, nil
}*/

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
                active_directory.forest_name,
				active_directory.forest_functional_level,
            }
        }
    `
	return dgraphutil.GetEntityByUID[active_directory.ActiveDirectory](ctx, tx, uid, "activedirectory", query)
}

func (r *DgraphActiveDirectoryRepository) Update(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*active_directory.ActiveDirectory, error) {
	// legacy
	fields["updated_at"] = time.Now().Format(time.RFC3339)
	return dgraphutil.UpdateAndGet(ctx, tx, uid, actor, fields, r.Get)
}

func (r *DgraphActiveDirectoryRepository) Delete(ctx context.Context, tx *dgo.Txn, uid string) error {
	panic("implement me")
}
