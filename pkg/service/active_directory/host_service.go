package active_directory

import (
	"RedPaths-server/internal/db"
	rperror "RedPaths-server/internal/error"
	"RedPaths-server/internal/repository/active_directory"
	"RedPaths-server/internal/repository/redpaths"
	"RedPaths-server/internal/utils"
	"RedPaths-server/pkg/model"
	"RedPaths-server/pkg/model/core"
	utils2 "RedPaths-server/pkg/model/utils"
	"RedPaths-server/pkg/model/utils/assertion"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/dgo/v210"
)

type HostService struct {
	hostRepo      active_directory.HostRepository
	serviceRepo   active_directory.ServiceRepository
	projectRepo   active_directory.ProjectRepository
	domainRepo    active_directory.DomainRepository
	assertionRepo redpaths.AssertionRepository
	db            *dgo.Dgraph
}

func NewHostService(dgraphCon *dgo.Dgraph) (*HostService, error) {

	hostRepo := active_directory.NewDgraphHostRepository(dgraphCon)
	serviceRepo := active_directory.NewDgraphServiceRepository(dgraphCon)
	projectRepo := active_directory.NewDgraphProjectRepository(dgraphCon)
	domainRepo := active_directory.NewDgraphDomainRepository(dgraphCon)
	assertionRepo := redpaths.NewDgraphAssertionRepository(dgraphCon)

	return &HostService{
		hostRepo:      hostRepo,
		serviceRepo:   serviceRepo,
		projectRepo:   projectRepo,
		domainRepo:    domainRepo,
		assertionRepo: assertionRepo,
		db:            dgraphCon}, nil
}

func (s *HostService) AddService(
	ctx context.Context,
	assertionCtx assertion.Context,
	hostUID string,
	incomingService *model.Service, actor string) (*core.EntityResult[model.Service], error) {
	log.Println("Starting adding service")
	var result *core.EntityResult[model.Service]

	log.Printf("Adding service %s to host %s", actor, incomingService.Name)
	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		// TODO: EXISTS CHECK
		/*exists, existingUID, err := dgraphutil.ExistsByFieldOnParent(
			ctx,
			tx,
			hostUID,
			"Host",
			"Service",
			"has_service",
			"port",
			service.Port,
		)
		if err != nil {
			return fmt.Errorf("failed to check service existence: %w", err)
		}

		if exists {
			log.Printf("Service already exists")
			serviceUID = existingUID
		} else {*/
		service, err := s.serviceRepo.Create(ctx, tx, incomingService, actor)
		if err != nil {
			return fmt.Errorf("failed to create service: %w", err)
		}

		assertionSchema := &core.Assertion{
			Predicate:           core.PredicateRuns,
			Method:              core.MethodDirectAdd,
			Source:              actor,
			Confidence:          assertionCtx.GetConfidence(),
			Status:              core.StatusValidated,
			Timestamp:           time.Now(),
			HasDiscoveredParent: true,
			MarkedAsHighValue:   assertionCtx.IsHighValue(),
			Subject:             &utils2.UIDRef{UID: hostUID, Type: "Host"},
			Object:              &utils2.UIDRef{UID: service.UID, Type: "Service"},
		}

		var assertions []*core.Assertion
		createdAssertion, err := s.assertionRepo.Create(ctx, tx, assertionSchema)
		if err != nil {
			return fmt.Errorf("error while creating assertion with uid: %w with error: %w", err)
		}

		log.Printf("Created assertion with uid: %s", createdAssertion.UID)

		/*if err := s.serviceRepo.LinkToHost(ctx, tx, serviceUID, hostUID); err != nil {
			return fmt.Errorf("failed to reverse link service to host: %w", err)
		}*/

		assertions = append(assertions, createdAssertion)

		result = &core.EntityResult[model.Service]{
			Entity:     *service,
			Assertions: assertions,
			Metadata: &core.ResultMetadata{
				Source:         actor,
				ScanTimestamp:  time.Now(),
				EntityCount:    1,
				AssertionCount: len(assertions),
			},
		}
		return nil
	})
	return result, err
}

func (s *HostService) CreateWithUnknownDomain(ctx context.Context, host *model.Host, projectUID string, actor string) (string, error) {
	var hostUID string
	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {

		var err error
		hostUID, err = s.hostRepo.Create(ctx, tx, host, actor)
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

func (s *HostService) GetAllServicesByHost(ctx context.Context, hostUID string) ([]*core.EntityResult[*model.Service], error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*core.EntityResult[*model.Service], error) {
		return s.serviceRepo.GetByHostUID(ctx, tx, hostUID)
	})
}

func (s *HostService) GetServiceByHost(ctx context.Context, hostUID, serviceUID string) (*model.Service, error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) (*model.Service, error) {
		services, err := s.serviceRepo.GetByHostUID(ctx, tx, hostUID)
		if err != nil {
			log.Printf("Failed to get service by host uid %s: %v", hostUID, err)
			return nil, err
		}

		for _, service := range services {
			if service.Entity.UID == serviceUID {
				return service.Entity, nil
			}
		}
		log.Printf("Service not found by host uid %s", hostUID)
		return nil, rperror.ErrNotFound
	})
}

func (s *HostService) UpdateHost(ctx context.Context, uid, actor string, fields map[string]interface{}) (*model.Host, error) {
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

	return db.ExecuteInTransactionWithResult[*model.Host](ctx, s.db, func(tx *dgo.Txn) (*model.Host, error) {
		return s.hostRepo.UpdateHost(ctx, tx, uid, actor, fields)
	})
}
