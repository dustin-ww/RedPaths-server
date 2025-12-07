package active_directory

import (
	"RedPaths-server/internal/db"
	"RedPaths-server/internal/repository/active_directory"
	"RedPaths-server/pkg/model"
	"context"
	"fmt"
	"log"

	"github.com/dgraph-io/dgo/v210"
)

type HostService struct {
	hostRepo    active_directory.HostRepository
	serviceRepo active_directory.ServiceRepository
	projectRepo active_directory.ProjectRepository
	domainRepo  active_directory.DomainRepository
	db          *dgo.Dgraph
}

func NewHostService(dgraphCon *dgo.Dgraph) (*HostService, error) {

	hostRepo := active_directory.NewDgraphHostRepository(dgraphCon)
	serviceRepo := active_directory.NewDgraphServiceRepository(dgraphCon)
	projectRepo := active_directory.NewDgraphProjectRepository(dgraphCon)
	domainRepo := active_directory.NewDgraphDomainRepository(dgraphCon)

	return &HostService{
		hostRepo:    hostRepo,
		serviceRepo: serviceRepo,
		projectRepo: projectRepo,
		domainRepo:  domainRepo,
		db:          dgraphCon}, nil
}

func (s *HostService) AddService(ctx context.Context, hostUID string, service model.Service) (string, error) {
	var serviceUID string
	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		var err error
		serviceUID, err = s.serviceRepo.CreateWithObject(ctx, tx, service)
		if err != nil {
			return fmt.Errorf("failed to create service: %w", err)
		}

		if err := s.hostRepo.AddService(ctx, tx, hostUID, serviceUID); err != nil {
			return fmt.Errorf("failed to link service to host: %w", err)
		}

		if err := s.serviceRepo.LinkToHost(ctx, tx, serviceUID, hostUID); err != nil {
			return fmt.Errorf("failed to reverse link domain to host: %w", err)
		}
		return nil
	})
	return serviceUID, err

}

func (s *HostService) CreateWithUnknownDomain(ctx context.Context, host *model.Host, projectUID string) (string, error) {
	var hostUID string
	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {

		var err error
		hostUID, err = s.hostRepo.Create(ctx, tx, host)
		log.Printf("Creating Host with uid %s with unknown domain in project with uid %s", hostUID, projectUID)
		if err != nil {
			return fmt.Errorf("failed to create host: %w", err)
		}

		if err := s.projectRepo.AddHostWithUnknownDomain(ctx, tx, projectUID, hostUID); err != nil {
			return fmt.Errorf("failed to reverse link unknown domain host to project: %w", err)
		}

		return nil
	})
	return hostUID, err
}

func (s *HostService) GetHostServices(ctx context.Context, hostUID string) ([]*model.Service, error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*model.Service, error) {
		return s.serviceRepo.GetByHostUID(ctx, tx, hostUID)
	})
}
