package active_directory

import (
	"RedPaths-server/internal/db"
	rperror "RedPaths-server/internal/error"
	"RedPaths-server/internal/repository/active_directory"
	"RedPaths-server/internal/repository/redpaths/changes"
	engine2 "RedPaths-server/internal/repository/redpaths/engine"
	"RedPaths-server/internal/repository/util/dgraph"
	"RedPaths-server/internal/utils"
	"RedPaths-server/pkg/model"
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/core/res"
	"RedPaths-server/pkg/model/engine"
	"RedPaths-server/pkg/model/redpaths/history"
	utils2 "RedPaths-server/pkg/model/utils"
	"RedPaths-server/pkg/model/utils/assertion"
	engine3 "RedPaths-server/pkg/service/catalog"
	engine5 "RedPaths-server/pkg/service/change"
	engine4 "RedPaths-server/pkg/service/upsert"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/dgo/v210"
	"gorm.io/gorm"
)

// -----------------------------------------------------------------------------
// HostService
// -----------------------------------------------------------------------------

type HostService struct {
	hostRepo       active_directory.HostRepository
	serviceRepo    active_directory.ServiceRepository
	projectRepo    active_directory.ProjectRepository
	domainRepo     active_directory.DomainRepository
	assertionRepo  engine2.AssertionRepository
	capabilityRepo engine2.CapabilityRepository
	catalogService *engine3.CatalogService

	changeRepo changes.RedPathsChangeRepository
	db         *dgo.Dgraph
	pdb        *gorm.DB
}

func NewHostService(dgraphCon *dgo.Dgraph, postgresCon *gorm.DB) (*HostService, error) {
	assertionRepo := engine2.NewDgraphAssertionRepository(dgraphCon)
	catalogService := engine3.NewCatalogService(dgraphCon)

	return &HostService{
		hostRepo:       active_directory.NewDgraphHostRepository(dgraphCon),
		serviceRepo:    active_directory.NewDgraphServiceRepository(dgraphCon),
		projectRepo:    active_directory.NewDgraphProjectRepository(dgraphCon),
		domainRepo:     active_directory.NewDgraphDomainRepository(dgraphCon),
		assertionRepo:  assertionRepo,
		capabilityRepo: engine2.NewDgraphCapabilityRepository(dgraphCon),
		changeRepo:     changes.NewPostgresRedPathsChangesRepository(postgresCon),
		catalogService: catalogService,
		db:             dgraphCon,
		pdb:            postgresCon,
	}, nil
}

// -----------------------------------------------------------------------------
// GetCapabilities
// -----------------------------------------------------------------------------

func (s *HostService) GetCapabilities(
	ctx context.Context,
	hostUID string,
) ([]*res.EntityResult[*engine.Capability], error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*res.EntityResult[*engine.Capability], error) {
		return s.capabilityRepo.GetAllByHostUID(ctx, tx, hostUID)
	})
}

// -----------------------------------------------------------------------------
// UpsertHost
// -----------------------------------------------------------------------------

func (s *HostService) UpsertHost(
	ctx context.Context,
	input engine4.Input[*model.Host],
) (*res.EntityResult[*model.Host], error) {

	subjectUID, subjectType, hasParent := input.Resolved()

	var result *res.EntityResult[*model.Host]
	var pendingChange *history.Change // wird nach der Dgraph-Tx gespeichert

	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		existence, err := s.hostRepo.FindExisting(ctx, tx, input.ProjectUID, input.Entity)
		if err != nil {
			return fmt.Errorf("existence check failed: %w", err)
		}

		filters := active_directory.BuildHostFilter(input.Entity)
		var actualHost *model.Host

		switch existence.FoundVia {

		case dgraph.ExistenceSourceNotFound:
			createdHost, err := s.hostRepo.Create(ctx, tx, input.Entity, input.Actor)
			if err != nil {
				return fmt.Errorf("creating host: %w", err)
			}
			actualHost = createdHost
			log.Printf("[UpsertHost] Created uid=%s ip=%s", actualHost.UID, actualHost.IP)

			// Change: alle gesetzten Felder als "new_value", old_value nil
			pendingChange = engine5.BuildCreatedChange(actualHost, input.Actor)

		case dgraph.ExistenceSourceHierarchy,
			dgraph.ExistenceSourceProject:

			best := dgraph.BestCandidate(existence.Entities, filters, 0.5)

			if best == nil {
				createdHost, err := s.hostRepo.Create(ctx, tx, input.Entity, input.Actor)
				if err != nil {
					return fmt.Errorf("creating host (low score): %w", err)
				}
				actualHost = createdHost
				log.Printf("[UpsertHost] Low score, created uid=%s", actualHost.UID)

				pendingChange = engine5.BuildCreatedChange(actualHost, input.Actor)

			} else if best.Score >= 0.8 {
				mergeFields := buildMergeFields(
					best.Result.Entity,
					input.Entity,
					input.AssertionCtx.GetConfidence(),
				)
				updated, err := s.hostRepo.UpdateHost(
					ctx, tx,
					best.Result.Entity.UID,
					input.Actor,
					mergeFields,
				)
				if err != nil {
					return fmt.Errorf("merging host: %w", err)
				}
				actualHost = updated
				log.Printf("[UpsertHost] Merged uid=%s score=%.2f",
					actualHost.UID, best.Score)

				// Change: diff zwischen existing und incoming
				pendingChange = engine5.BuildUpdatedChange(
					best.Result.Entity,
					mergeFields,
					input.Actor,
					fmt.Sprintf("Merged from scan, confidence score=%.2f", best.Score),
				)

			} else {
				log.Printf("[UpsertHost] Possible duplicate uid=%s score=%.2f",
					best.Result.Entity.UID, best.Score)

				duplicateAssertion := &core.Assertion{
					Predicate:  core.PredicatePossibleDuplicate,
					Method:     core.MethodInferred,
					Source:     input.Actor,
					Confidence: best.Score,
					Status:     core.StatusTentative,
					Timestamp:  time.Now(),
					Note: fmt.Sprintf(
						"Possible duplicate detected with score %.2f — manual review required",
						best.Score,
					),
					HasDiscoveredParent: false,
					MarkedAsHighValue:   false,
					Subject:             &utils2.UIDRef{UID: best.Result.Entity.UID, Type: "Host"},
					Object:              &utils2.UIDRef{UID: input.Entity.UID, Type: "Host"},
				}

				if _, err := s.assertionRepo.Create(ctx, tx, duplicateAssertion); err != nil {
					return fmt.Errorf("creating duplicate assertion: %w", err)
				}

				// Change für das bestehende Entity (Status-Information)
				pendingChange = &history.Change{
					EntityType: "Host",
					EntityUID:  best.Result.Entity.UID,
					ChangeType: history.ChangeTypePossibleDup,
					ChangedBy:  input.Actor,
					ChangeReason: fmt.Sprintf(
						"Possible duplicate candidate detected, score=%.2f", best.Score,
					),
					Changes: []history.FieldChange{
						{
							Field:    "duplicate_candidate_uid",
							OldValue: nil,
							NewValue: input.Entity.UID,
						},
						{
							Field:    "duplicate_score",
							OldValue: nil,
							NewValue: best.Score,
						},
					},
				}

				result = best.Result
				return nil
			}

		default:
			return fmt.Errorf("unhandled existence state: %s", existence.FoundVia)
		}

		assertionSchema := &core.Assertion{
			Predicate:           core.PredicateHasHost,
			Method:              core.MethodDirectAdd,
			Source:              input.Actor,
			Confidence:          input.AssertionCtx.GetConfidence(),
			Status:              core.StatusValidated,
			Timestamp:           time.Now(),
			HasDiscoveredParent: hasParent,
			MarkedAsHighValue:   input.AssertionCtx.IsHighValue(),
			Subject:             &utils2.UIDRef{UID: subjectUID, Type: subjectType},
			Object:              &utils2.UIDRef{UID: actualHost.UID, Type: "Host"},
		}

		createdAssertion, err := s.assertionRepo.Create(ctx, tx, assertionSchema)
		if err != nil {
			return fmt.Errorf("creating assertion: %w", err)
		}

		result = &res.EntityResult[*model.Host]{
			Entity:     actualHost,
			Assertions: []*core.Assertion{createdAssertion},
			Metadata: &res.ResultMetadata{
				Source:         input.Actor,
				ScanTimestamp:  time.Now(),
				EntityCount:    1,
				AssertionCount: 1,
			},
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("UpsertHost failed: %w", err)
	}

	// --- Change History speichern (außerhalb der Dgraph-Tx, best-effort) ---
	if pendingChange != nil {
		s.saveChangeAsync(ctx, pendingChange)
	}

	// --- Catalog Integration ---
	if result == nil || len(result.Assertions) == 0 {
		return result, nil
	}

	_, catalogErr := engine3.AddToCatalog(
		ctx, s.catalogService,
		input.ProjectUID, result.Entity.UID, "Host",
		result.Assertions[0], input.Actor,
	)
	if catalogErr != nil {
		log.Printf("[UpsertHost] Warning: failed to add host %s to catalog: %v",
			result.Entity.UID, catalogErr)
	}

	if hasParent {
		promoteErr := engine3.PromoteInCatalog(
			ctx, s.catalogService,
			input.ProjectUID, result.Entity.UID, "Host",
			core.PredicateHasHost, input.Actor,
		)
		if promoteErr != nil {
			log.Printf("[UpsertHost] Warning: failed to promote host %s in catalog: %v",
				result.Entity.UID, promoteErr)
		}
	}

	return result, nil
}

func (s *HostService) saveChangeAsync(ctx context.Context, change *history.Change) {
	go func() {
		saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := db.ExecutePostgresInTransaction(saveCtx, s.pdb, func(tx *gorm.DB) error {
			return s.changeRepo.Save(saveCtx, tx, change)
		})
		if err != nil {
			log.Printf("[ChangeHistory] Warning: failed to save change for entity=%s uid=%s: %v",
				change.EntityType, change.EntityUID, err)
		}
	}()
}

// -----------------------------------------------------------------------------
// buildMergeFields
// -----------------------------------------------------------------------------

// buildMergeFields builds the update fields for a host merge.
// Only overwrites a field if the incoming value is non-empty and
// either the existing value is empty or the incoming confidence is high enough.
func buildMergeFields(
	existing *model.Host,
	incoming *model.Host,
	incomingConfidence float64,
) map[string]interface{} {
	fields := map[string]interface{}{
		"last_seen_at": time.Now(),
	}

	// IP: overwrite if incoming is set and confidence >= 0.8
	if incoming.IP != "" && (existing.IP == "" || incomingConfidence >= 0.8) {
		fields["host.ip"] = incoming.IP
	}

	// DNS: overwrite if incoming is set and existing is empty
	if incoming.DNSHostName != "" && existing.DNSHostName == "" {
		fields["host.dns_host_name"] = incoming.DNSHostName
	}

	// OS: overwrite if incoming is set and existing is empty
	if incoming.OperatingSystem != "" && existing.OperatingSystem == "" {
		fields["host.operating_system"] = incoming.OperatingSystem
	}

	// OS Version: overwrite if incoming is set and existing is empty
	if incoming.OperatingSystemVersion != "" && existing.OperatingSystemVersion == "" {
		fields["host.operating_system_version"] = incoming.OperatingSystemVersion
	}

	// Hostname: overwrite if incoming is set and existing is empty
	if incoming.Hostname != "" && existing.Hostname == "" {
		fields["host.hostname"] = incoming.Hostname
	}

	// Distinguished Name: overwrite if incoming is set and existing is empty
	if incoming.DistinguishedName != "" && existing.DistinguishedName == "" {
		fields["host.distinguished_name"] = incoming.DistinguishedName
	}

	// IsDomainController: only set if true — once a DC, always a DC
	if incoming.IsDomainController && !existing.IsDomainController {
		fields["host.is_domain_controller"] = true
	}

	return fields
}

// -----------------------------------------------------------------------------
// AddService
// -----------------------------------------------------------------------------

func (s *HostService) AddService(
	ctx context.Context,
	assertionCtx assertion.Context,
	hostUID string,
	incomingService *model.Service,
	actor string,
) (*res.EntityResult[model.Service], error) {
	log.Printf("[AddService] Adding service %s to host %s", incomingService.Name, hostUID)

	var result *res.EntityResult[model.Service]

	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
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

		createdAssertion, err := s.assertionRepo.Create(ctx, tx, assertionSchema)
		if err != nil {
			return fmt.Errorf("creating service assertion: %w", err)
		}

		log.Printf("[AddService] Created assertion uid=%s", createdAssertion.UID)

		result = &res.EntityResult[model.Service]{
			Entity:     *service,
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

	return result, err
}

// -----------------------------------------------------------------------------
// GetAllServicesByHost / GetServiceByHost
// -----------------------------------------------------------------------------

func (s *HostService) GetAllServicesByHost(
	ctx context.Context,
	hostUID string,
) ([]*res.EntityResult[*model.Service], error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*res.EntityResult[*model.Service], error) {
		return s.serviceRepo.GetByHostUID(ctx, tx, hostUID)
	})
}

func (s *HostService) GetServiceByHost(
	ctx context.Context,
	hostUID string,
	serviceUID string,
) (*model.Service, error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) (*model.Service, error) {
		services, err := s.serviceRepo.GetByHostUID(ctx, tx, hostUID)
		if err != nil {
			log.Printf("[GetServiceByHost] Failed for hostUID=%s: %v", hostUID, err)
			return nil, err
		}
		for _, service := range services {
			if service.Entity.UID == serviceUID {
				return service.Entity, nil
			}
		}
		log.Printf("[GetServiceByHost] Not found serviceUID=%s hostUID=%s", serviceUID, hostUID)
		return nil, rperror.ErrNotFound
	})
}

// -----------------------------------------------------------------------------
// UpdateHost
// -----------------------------------------------------------------------------

func (s *HostService) UpdateHost(
	ctx context.Context,
	uid string,
	actor string,
	fields map[string]interface{},
) (*model.Host, error) {
	if uid == "" {
		return nil, utils.ErrUIDRequired
	}
	return db.ExecuteInTransactionWithResult[*model.Host](ctx, s.db, func(tx *dgo.Txn) (*model.Host, error) {
		return s.hostRepo.UpdateHost(ctx, tx, uid, actor, fields)
	})
}
