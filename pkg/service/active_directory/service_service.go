package active_directory

import (
	"RedPaths-server/internal/db"
	"RedPaths-server/internal/repository/active_directory"
	"RedPaths-server/internal/utils"
	"RedPaths-server/pkg/model"
	"context"

	"github.com/dgraph-io/dgo/v210"
)

type ServiceService struct {
	serviceRepo active_directory.ServiceRepository
	db          *dgo.Dgraph
}

func NewServiceService(dgraphCon *dgo.Dgraph) (*ServiceService, error) {

	return &ServiceService{
		db:          dgraphCon,
		serviceRepo: active_directory.NewDgraphServiceRepository(dgraphCon),
	}, nil
}

func (s *ServiceService) GetHostServices(ctx context.Context, hostUID string) ([]*model.Service, error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*model.Service, error) {
		return s.serviceRepo.GetByHostUID(ctx, tx, hostUID)
	})
}

func (s *ServiceService) UpdateService(ctx context.Context, uid, actor string, fields map[string]interface{}) (*model.Service, error) {
	if uid == "" {
		return nil, utils.ErrUIDRequired
	}

	/*allowed := map[string]bool{"name": true, "description": true}
	protected := map[string]bool{"uid": true, "created_at": true, "updated_at": true, "type": true}

	for field := range fields {
		if protected[field] {
			return nil, fmt.Errorf("%w: %s", utils.ErrFieldProtected, field)
		}
		if !allowed[field] {
			return nil, fmt.Errorf("%w: %s", utils.ErrFieldNotAllowed, field)
		}
	}*/

	return db.ExecuteInTransactionWithResult[*model.Service](ctx, s.db, func(tx *dgo.Txn) (*model.Service, error) {
		return s.serviceRepo.UpdateService(ctx, tx, uid, actor, fields)
	})
}
