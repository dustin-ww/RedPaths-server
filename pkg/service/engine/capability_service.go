package engine

import (
	"RedPaths-server/internal/db"
	"RedPaths-server/internal/repository/active_directory"
	engine2 "RedPaths-server/internal/repository/redpaths/engine"
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/core/res"
	"RedPaths-server/pkg/model/engine"
	utils2 "RedPaths-server/pkg/model/utils"
	"RedPaths-server/pkg/model/utils/assertion"
	active_directory2 "RedPaths-server/pkg/service/active_directory"
	"RedPaths-server/pkg/service/catalog"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/dgo/v210"
	"gorm.io/gorm"
)

type CapabilityService struct {
	hostRepo       active_directory.HostRepository
	serviceRepo    active_directory.ServiceRepository
	projectRepo    active_directory.ProjectRepository
	domainRepo     active_directory.DomainRepository
	assertionRepo  engine2.AssertionRepository
	capabilityRepo engine2.CapabilityRepository

	projectService active_directory2.ProjectService
	catalogService catalog.CatalogService
	db             *dgo.Dgraph
}

func NewCapabilityService(dgraphCon *dgo.Dgraph, postgresCon *gorm.DB) (*CapabilityService, error) {

	hostRepo := active_directory.NewDgraphHostRepository(dgraphCon)
	serviceRepo := active_directory.NewDgraphServiceRepository(dgraphCon)
	projectRepo := active_directory.NewDgraphProjectRepository(dgraphCon)
	domainRepo := active_directory.NewDgraphDomainRepository(dgraphCon)
	assertionRepo := engine2.NewDgraphAssertionRepository(dgraphCon)
	capabilityRepo := engine2.NewDgraphCapabilityRepository(dgraphCon)
	// postgres is not needed here
	projectService, err := active_directory2.NewProjectService(dgraphCon, postgresCon)
	catalogService := catalog.NewCatalogService(dgraphCon)

	if err != nil {
		return nil, fmt.Errorf("error creating project service in capability service: %v", err)
	}

	return &CapabilityService{
		hostRepo:       hostRepo,
		serviceRepo:    serviceRepo,
		projectRepo:    projectRepo,
		domainRepo:     domainRepo,
		assertionRepo:  assertionRepo,
		capabilityRepo: capabilityRepo,
		projectService: *projectService,
		catalogService: *catalogService,
		db:             dgraphCon}, nil
}

func (s *CapabilityService) GetCapabilitiesFromCatalog(
	ctx context.Context,
	projectUID string,
) ([]*res.EntityResult[*engine.Capability], error) {
	log.Printf("[DEBUG] GetCatalogCapabilities called with projectUID: %s", projectUID)
	return catalog.GetFromCatalog[*engine.Capability](ctx, &s.catalogService, projectUID, "Capability")
}

func (s *CapabilityService) CreateAndLinkCapability(
	ctx context.Context,
	assertionCtx assertion.Context,
	incomingCapability *engine.Capability,
	subjectUID,
	subjectType,
	projectUID,
	actor string) (*res.EntityResult[engine.Capability], error) {

	var result *res.EntityResult[engine.Capability]

	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		capability, err := s.capabilityRepo.Create(ctx, tx, incomingCapability, actor)
		if err != nil {
			return fmt.Errorf("failed to create capability: %w", err)
		}

		assertionSchema := &core.Assertion{
			Predicate:           core.PredicateDerives,
			Method:              core.MethodDirectAdd,
			Source:              actor,
			Confidence:          assertionCtx.GetConfidence(),
			Status:              core.StatusValidated,
			Timestamp:           time.Now(),
			HasDiscoveredParent: true,
			MarkedAsHighValue:   assertionCtx.IsHighValue(),
			Subject:             &utils2.UIDRef{UID: subjectUID, Type: subjectType},
			Object:              &utils2.UIDRef{UID: capability.UID, Type: "Capability"},
		}

		createdAssertion, err := s.assertionRepo.Create(ctx, tx, assertionSchema)
		if err != nil {
			return fmt.Errorf("failed to create assertion: %w", err)
		}

		result = &res.EntityResult[engine.Capability]{
			Entity:     *capability,
			Assertions: []*core.Assertion{createdAssertion},
			Metadata: &res.ResultMetadata{
				Source:         actor,
				ScanTimestamp:  time.Now(),
				EntityCount:    1,
				AssertionCount: 1,
			},
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create and link capability: %w", err)
	}

	_, err = s.projectService.AddEntityToProjectCatalog(
		ctx, result.Assertions[0], projectUID, result.Entity.UID, actor,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add capability to project catalog: %w", err)
	}

	return result, nil
}
