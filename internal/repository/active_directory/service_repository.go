package active_directory

import (
	"RedPaths-server/internal/repository/dgraphutil"
	"RedPaths-server/pkg/model"
	"context"
	"fmt"
	"log"

	"github.com/dgraph-io/dgo/v210"
)

type ServiceRepository interface {
	//CRUD
	CreateWithObject(ctx context.Context, tx *dgo.Txn, model model.Service) (string, error)
	Get(ctx context.Context, tx *dgo.Txn, uid string) (*model.Service, error)
	UpdateService(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*model.Service, error)
	//Relations
	LinkToHost(ctx context.Context, tx *dgo.Txn, serviceUID, hostUID string) error
	GetByHostUID(ctx context.Context, tx *dgo.Txn, hostUID string) ([]*model.Service, error)
}

type DgraphServiceRepository struct {
	DB *dgo.Dgraph
}

func (r *DgraphServiceRepository) UpdateService(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*model.Service, error) {
	return dgraphutil.UpdateAndGet(ctx, tx, uid, actor, fields, r.Get)

}

func NewDgraphServiceRepository(db *dgo.Dgraph) *DgraphServiceRepository {
	return &DgraphServiceRepository{DB: db}
}

func (r *DgraphServiceRepository) CreateWithObject(ctx context.Context, tx *dgo.Txn, service model.Service) (string, error) {
	return dgraphutil.OldCreateEntity(ctx, tx, "Service", service)
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
func (r *DgraphServiceRepository) GetByHostUID(ctx context.Context, tx *dgo.Txn, hostUID string) ([]*model.Service, error) {
	fields := []string{
		"uid",
		"name",
		"port",
		"dgraph.type",
		"deployed_on_host { uid }",
	}

	services, err := dgraphutil.GetEntitiesByRelation[*model.Service](
		ctx,
		tx,
		"Service",
		"deployed_on_host",
		hostUID,
		fields,
	)
	if err != nil {
		return nil, err
	}

	log.Printf("Found %d services for host %s\n", len(services), hostUID)
	return services, nil
}
