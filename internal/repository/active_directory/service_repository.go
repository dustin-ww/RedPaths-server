package active_directory

import (
	"RedPaths-server/internal/repository/dgraphutil"
	"RedPaths-server/pkg/model"
	"RedPaths-server/pkg/model/core"
	"context"
	"fmt"
	"log"

	"github.com/dgraph-io/dgo/v210"
)

type ServiceRepository interface {
	//CRUD
	Create(ctx context.Context, tx *dgo.Txn, model *model.Service, actor string) (*model.Service, error)
	Get(ctx context.Context, tx *dgo.Txn, uid string) (*model.Service, error)
	UpdateService(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*model.Service, error)
	GetByHostUID(ctx context.Context, tx *dgo.Txn, hostUID string) ([]*core.EntityResult[*model.Service], error)
	//Relations
	/*	LinkToHost(ctx context.Context, tx *dgo.Txn, serviceUID, hostUID string) error
		GetByHostUID(ctx context.Context, tx *dgo.Txn, hostUID string) ([]*model.Service, error)*/
}

type DgraphServiceRepository struct {
	DB *dgo.Dgraph
}

func (r *DgraphServiceRepository) Create(ctx context.Context, tx *dgo.Txn, service *model.Service, actor string) (*model.Service, error) {
	dgraphutil.InitCreateMetadata(&service.RedPathsMetadata, actor)
	return dgraphutil.CreateEntity(ctx, tx, "service", service)
}

func (r *DgraphServiceRepository) UpdateService(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*model.Service, error) {
	return dgraphutil.UpdateAndGet(ctx, tx, uid, actor, fields, r.Get)
}

func NewDgraphServiceRepository(db *dgo.Dgraph) *DgraphServiceRepository {
	return &DgraphServiceRepository{DB: db}
}

func (r *DraphHostRepository) ServiceExistsByPortOnHost(ctx context.Context, tx *dgo.Txn, projectUID, ip string) (bool, error) {
	return dgraphutil.ExistsByFieldInProject(ctx, tx, projectUID, "Host", "ip", ip)
}

func (r *DgraphServiceRepository) Get(ctx context.Context, tx *dgo.Txn, uid string) (*model.Service, error) {
	query := `
        query Service($uid: string) {
            Service(func: uid($uid)) {
                uid
                name
				dgraph.type
				port
                deployed_on_host { uid }
            }
        }`

	return dgraphutil.GetEntityByUID[model.Service](ctx, tx, uid, "service", query)
}

func (r *DgraphServiceRepository) LinkToHost(ctx context.Context, tx *dgo.Txn, serviceUID, hostUID string) error {
	relationName := "deployed_on_host"
	log.Printf("LINKING: service %s and host %s", serviceUID, hostUID)
	err := dgraphutil.AddRelation(ctx, tx, serviceUID, hostUID, relationName)
	if err != nil {
		return fmt.Errorf("error while reverse linking service %s to host %s with relation name %s", serviceUID, hostUID, relationName)
	}
	return nil
}

// TODO -> history runs on hosts (remove s)
func (r *DgraphServiceRepository) GetByHostUID(ctx context.Context, tx *dgo.Txn, hostUID string) ([]*core.EntityResult[*model.Service], error) {
	fields := []string{
		"uid",
		"service.name",
		"service.port",
		"created_at",
		"modified_at",
		"discovered_at",
		"discovered_by",
		"validated_at",
		"validated_by",
		"dgraph.type",
	}

	return dgraphutil.GetEntitiesWithAssertions[*model.Service](
		ctx,
		tx,
		hostUID,
		core.PredicateRuns,
		"Service",
		fields,
		"getHostServices",
	)
}
