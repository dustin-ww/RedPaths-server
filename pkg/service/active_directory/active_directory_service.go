package active_directory

import (
	"RedPaths-server/internal/db"
	"RedPaths-server/internal/repository/active_directory"
	"RedPaths-server/internal/repository/redpaths/engine"
	dgraphutil "RedPaths-server/internal/repository/util/dgraph"
	"RedPaths-server/internal/utils"
	rpap "RedPaths-server/pkg/model/active_directory"
	"RedPaths-server/pkg/model/active_directory/priv"
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/core/res"
	utils2 "RedPaths-server/pkg/model/utils"
	"RedPaths-server/pkg/model/utils/assertion"
	engine3 "RedPaths-server/pkg/service/catalog"
	engine4 "RedPaths-server/pkg/service/upsert"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/dgo/v210"
)

type ActiveDirectoryService struct {
	domainRepo          active_directory.DomainRepository
	hostRepo            active_directory.HostRepository
	activeDirectoryRepo active_directory.ActiveDirectoryRepository
	aclRepo             active_directory.ACLRepository
	assertionRepo       engine.AssertionRepository
	catalogService      *engine3.CatalogService

	directoryNodeService *DirectoryNodeService
	db                   *dgo.Dgraph
}

func NewActiveDirectoryService(dgraphCon *dgo.Dgraph) (*ActiveDirectoryService, error) {
	domainRepo := active_directory.NewDgraphDomainRepository(dgraphCon)
	hostRepo := active_directory.NewDgraphHostRepository(dgraphCon)
	activeDirectoryRepo := active_directory.NewDgraphActiveDirectoryRepository(dgraphCon)
	aclRepo := active_directory.NewDgraphDgraphACLRepository(dgraphCon)
	assertionRepo := engine.NewDgraphAssertionRepository(dgraphCon)
	catalogService := engine3.NewCatalogService(dgraphCon)
	directoryNodeService, err := NewDirectoryNodeService(dgraphCon)

	if err != nil {
		return nil, fmt.Errorf("NewDirectoryNodeService error: %w", err)
	}

	return &ActiveDirectoryService{
		db:                   dgraphCon,
		domainRepo:           domainRepo,
		hostRepo:             hostRepo,
		activeDirectoryRepo:  activeDirectoryRepo,
		aclRepo:              aclRepo,
		assertionRepo:        assertionRepo,
		catalogService:       catalogService,
		directoryNodeService: directoryNodeService,
	}, nil
}

func (s *ActiveDirectoryService) AddDomain(
	ctx context.Context,
	activeDirectoryUID string,
	incomingDomain *rpap.Domain,
	assertionCtx assertion.Context,
	actor string,
) (*res.EntityResult[*rpap.Domain], error) {

	var result *res.EntityResult[*rpap.Domain]

	log.Printf("[AddActiveDirectoryDomain] name=%s, activeDirectoryUID=%s, actor=%s",
		incomingDomain.Name, activeDirectoryUID, actor)

	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		// 1. Check if domain already exists in this AD
		existingDomain, err := s.domainRepo.FindByNameInActiveDirectory(
			ctx, tx, activeDirectoryUID, incomingDomain.Name,
		)
		if err != nil {
			return fmt.Errorf("checking existing domain: %w", err)
		}

		var domain *rpap.Domain
		var assertions []*core.Assertion

		if existingDomain != nil {
			// Domain exists - reuse
			domain = existingDomain
			log.Printf("[AddActiveDirectoryDomain] Reusing existing domain uid=%s", domain.UID)
		} else {
			// Create new Domain
			domain, err = s.domainRepo.Create(ctx, tx, incomingDomain, actor)
			if err != nil {
				return fmt.Errorf("creating domain: %w", err)
			}
			log.Printf("[AddActiveDirectoryDomain] Created domain uid=%s name=%s", domain.UID, domain.Name)

			// Create & Link ACL
			acl := priv.ACL{Owner: actor}
			createdACL, err := s.aclRepo.CreateACL(ctx, tx, &acl, actor)

			if err != nil {
				return fmt.Errorf("error while creating ACL for domain: %w", err)
			}

			err = s.aclRepo.LinkACLToEntity(ctx, tx, createdACL.UID, domain.UID)

			if err != nil {
				return fmt.Errorf("error while linking ACL to domain: %w", err)
			}
			log.Printf("Created and linked acl to domain")

			// Create & Link Default Directory Nodes
			dirNodes, err := s.directoryNodeService.CreateBuildDefaultDirectoryNodes(ctx, tx, "User", domain.UID)

			if err != nil {
				return fmt.Errorf("error while creating directory nodes: %w", err)
			}
			log.Printf("created directory nodes with length: ", len(dirNodes))
		}

		// 2. Create assertion (even if domain existed, link might be new)
		assertion := &core.Assertion{
			Predicate:  core.PredicateHasDomain,
			Method:     core.MethodDirectAdd,
			Source:     actor,
			Confidence: 1.0,
			Status:     core.StatusValidated,
			Timestamp:  time.Now(),
			Subject:    &utils2.UIDRef{UID: activeDirectoryUID},
			Object:     &utils2.UIDRef{UID: domain.UID},
		}

		createdAssertion, err := s.assertionRepo.Create(ctx, tx, assertion)
		if err != nil {
			return fmt.Errorf("creating assertion: %w", err)
		}
		assertions = append(assertions, createdAssertion)

		// 3. Optional: Link assertion to AD (if you have AddAssertion method)
		// err = s.activeDirectoryRepo.AddAssertion(ctx, tx, activeDirectoryUID, createdAssertion.UID)
		// if err != nil {
		//     return fmt.Errorf("linking assertion to AD: %w", err)
		// }

		// 4. Build result
		result = &res.EntityResult[*rpap.Domain]{
			Entity:     domain,
			Assertions: assertions,
			Metadata: &res.ResultMetadata{
				Source:         actor,
				ScanTimestamp:  time.Now(),
				EntityCount:    1,
				AssertionCount: len(assertions),
			},
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("AddActiveDirectoryDomain failed: %w", err)
	}

	return result, nil
}

// -----------------------------------------------------------------------------
// UpsertActiveDirectory
// -----------------------------------------------------------------------------

func (s *ActiveDirectoryService) UpsertActiveDirectory(
	ctx context.Context,
	input engine4.Input[*rpap.ActiveDirectory],
) (*res.EntityResult[*rpap.ActiveDirectory], error) {

	subjectUID, subjectType, hasParent := input.Resolved()

	var result *res.EntityResult[*rpap.ActiveDirectory]

	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		// --- Existence Check ---
		existence, err := s.activeDirectoryRepo.FindExisting(ctx, tx, input.ProjectUID, input.Entity)
		if err != nil {
			return fmt.Errorf("existence check failed: %w", err)
		}

		filters := active_directory.BuildActiveDirectoryFilter(input.Entity)
		var actualAD *rpap.ActiveDirectory

		switch existence.FoundVia {

		// --- Case 1: Not found → create new ---
		case dgraphutil.ExistenceSourceNotFound:
			createdAD, err := s.activeDirectoryRepo.Create(ctx, tx, input.Entity, input.Actor)
			if err != nil {
				return fmt.Errorf("creating active directory: %w", err)
			}
			actualAD = createdAD
			log.Printf("[UpsertActiveDirectory] Created uid=%s forest=%s", actualAD.UID, actualAD.ForestName)

		case dgraphutil.ExistenceSourceHierarchy,
			dgraphutil.ExistenceSourceProject:

			best := dgraphutil.BestCandidate(existence.Entities, filters, 0.5)

			if best == nil {
				// Candidates found but score too low → create new
				createdAD, err := s.activeDirectoryRepo.Create(ctx, tx, input.Entity, input.Actor)
				if err != nil {
					return fmt.Errorf("creating active directory (low score): %w", err)
				}
				actualAD = createdAD
				log.Printf("[UpsertActiveDirectory] Low score, created uid=%s", actualAD.UID)

			} else if best.Score >= 0.8 {
				// --- Case 2: High Confidence → Merge ---
				mergeFields := buildActiveDirectoryMergeFields(
					best.Result.Entity,
					input.Entity,
					input.AssertionCtx.GetConfidence(),
				)
				updated, err := s.activeDirectoryRepo.Update(
					ctx, tx,
					best.Result.Entity.UID,
					input.Actor,
					mergeFields,
				)
				if err != nil {
					return fmt.Errorf("merging active directory: %w", err)
				}
				actualAD = updated
				log.Printf("[UpsertActiveDirectory] Merged uid=%s score=%.2f",
					actualAD.UID, best.Score)

			} else {
				// --- Case 3: Medium Confidence (0.5–0.8) → Possible Duplicate ---
				log.Printf("[UpsertActiveDirectory] Possible duplicate uid=%s score=%.2f",
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
					Subject:             &utils2.UIDRef{UID: best.Result.Entity.UID, Type: "ActiveDirectory"},
					Object:              &utils2.UIDRef{UID: input.Entity.UID, Type: "ActiveDirectory"},
				}

				if _, err := s.assertionRepo.Create(ctx, tx, duplicateAssertion); err != nil {
					return fmt.Errorf("creating duplicate assertion: %w", err)
				}

				result = best.Result
				return nil
			}

		default:
			return fmt.Errorf("unhandled existence state: %s", existence.FoundVia)
		}

		// --- Create assertion ---
		assertionSchema := &core.Assertion{
			Predicate:           core.PredicateHasActiveDirectory,
			Method:              core.MethodDirectAdd,
			Source:              input.Actor,
			Confidence:          input.AssertionCtx.GetConfidence(),
			Status:              core.StatusValidated,
			Timestamp:           time.Now(),
			HasDiscoveredParent: hasParent,
			MarkedAsHighValue:   input.AssertionCtx.IsHighValue(),
			Subject:             &utils2.UIDRef{UID: subjectUID, Type: subjectType},
			Object:              &utils2.UIDRef{UID: actualAD.UID, Type: "ActiveDirectory"},
		}

		createdAssertion, err := s.assertionRepo.Create(ctx, tx, assertionSchema)
		if err != nil {
			return fmt.Errorf("creating assertion: %w", err)
		}

		result = &res.EntityResult[*rpap.ActiveDirectory]{
			Entity:     actualAD,
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
		return nil, fmt.Errorf("UpsertActiveDirectory failed: %w", err)
	}

	// --- Catalog Integration (outside the transaction) ---
	if result == nil || len(result.Assertions) == 0 {
		return result, nil
	}

	_, catalogErr := engine3.AddToCatalog(
		ctx,
		s.catalogService,
		input.ProjectUID,
		result.Entity.UID,
		"ActiveDirectory",
		result.Assertions[0],
		input.Actor,
	)
	if catalogErr != nil {
		log.Printf("[UpsertActiveDirectory] Warning: failed to add AD %s to catalog: %v",
			result.Entity.UID, catalogErr)
	}

	if hasParent {
		promoteErr := engine3.PromoteInCatalog(
			ctx,
			s.catalogService,
			input.ProjectUID,
			result.Entity.UID,
			"ActiveDirectory",
			core.PredicateHasActiveDirectory,
			input.Actor,
		)
		if promoteErr != nil {
			log.Printf("[UpsertActiveDirectory] Warning: failed to promote AD %s in catalog: %v",
				result.Entity.UID, promoteErr)
		}
	}

	return result, nil
}

// -----------------------------------------------------------------------------
// buildActiveDirectoryMergeFields
// -----------------------------------------------------------------------------

func buildActiveDirectoryMergeFields(
	existing *rpap.ActiveDirectory,
	incoming *rpap.ActiveDirectory,
	incomingConfidence float64,
) map[string]interface{} {
	fields := map[string]interface{}{
		"last_seen_at": time.Now(),
	}

	// Forest name: unique identity — overwrite only if existing is empty
	if incoming.ForestName != "" && existing.ForestName == "" {
		fields["active_directory.forest_name"] = incoming.ForestName
	}

	// Forest functional level: update if incoming is set and confidence is high or existing is empty
	if incoming.ForestFunctionalLevel != "" &&
		(existing.ForestFunctionalLevel == "" || incomingConfidence >= 0.8) {
		fields["active_directory.forest_functional_level"] = incoming.ForestFunctionalLevel
	}

	return fields
}

// -----------------------------------------------------------------------------
// GetAllDomains
// -----------------------------------------------------------------------------

func (s *ActiveDirectoryService) GetAllDomains(ctx context.Context, activeDirectoryUID string) ([]*res.EntityResult[*rpap.Domain], error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*res.EntityResult[*rpap.Domain], error) {
		return s.domainRepo.GetAllByActiveDirectoryUID(ctx, tx, activeDirectoryUID)
	})
}

func (s *ActiveDirectoryService) UpdateActiveDirectory(ctx context.Context, uid, actor string, fields map[string]interface{}) (*rpap.ActiveDirectory, error) {
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

	return db.ExecuteInTransactionWithResult[*rpap.ActiveDirectory](ctx, s.db, func(tx *dgo.Txn) (*rpap.ActiveDirectory, error) {
		return s.activeDirectoryRepo.Update(ctx, tx, uid, actor, fields)
	})
}

func (s *ActiveDirectoryService) Get(ctx context.Context, activeDirectoryUID string) (*rpap.ActiveDirectory, error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) (*rpap.ActiveDirectory, error) {
		return s.activeDirectoryRepo.Get(ctx, tx, activeDirectoryUID)
	})
}
