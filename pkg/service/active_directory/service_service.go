package active_directory

import (
	"RedPaths-server/internal/db"
	"RedPaths-server/internal/repository/active_directory"
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
