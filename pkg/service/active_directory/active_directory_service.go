package active_directory

import (
	"RedPaths-server/internal/db"
	"RedPaths-server/internal/repository/active_directory"
	"RedPaths-server/internal/repository/redpaths"
	"RedPaths-server/internal/utils"
	rpap "RedPaths-server/pkg/model/active_directory"
	"RedPaths-server/pkg/model/active_directory/priv"
	"RedPaths-server/pkg/model/core"
	utils2 "RedPaths-server/pkg/model/utils"
	"RedPaths-server/pkg/model/utils/assertion"
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
	assertionRepo       redpaths.AssertionRepository

	directoryNodeService *DirectoryNodeService
	db                   *dgo.Dgraph
}

func NewActiveDirectoryService(dgraphCon *dgo.Dgraph) (*ActiveDirectoryService, error) {
	domainRepo := active_directory.NewDgraphDomainRepository(dgraphCon)
	hostRepo := active_directory.NewDgraphHostRepository(dgraphCon)
	activeDirectoryRepo := active_directory.NewDgraphActiveDirectoryRepository(dgraphCon)
	aclRepo := active_directory.NewDgraphDgraphACLRepository(dgraphCon)
	assertionRepo := redpaths.NewDgraphAssertionRepository(dgraphCon)
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
		directoryNodeService: directoryNodeService,
	}, nil
}

func (s *ActiveDirectoryService) AddDomain(
	ctx context.Context,
	activeDirectoryUID string,
	incomingDomain *rpap.Domain,
	assertionCtx assertion.Context,
	actor string,
) (*core.EntityResult[*rpap.Domain], error) {

	var result *core.EntityResult[*rpap.Domain]

	log.Printf("[AddDomain] name=%s, activeDirectoryUID=%s, actor=%s",
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
			log.Printf("[AddDomain] Reusing existing domain uid=%s", domain.UID)
		} else {
			// Create new Domain
			domain, err = s.domainRepo.Create(ctx, tx, incomingDomain, actor)
			if err != nil {
				return fmt.Errorf("creating domain: %w", err)
			}
			log.Printf("[AddDomain] Created domain uid=%s name=%s", domain.UID, domain.Name)

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
		result = &core.EntityResult[*rpap.Domain]{
			Entity:     domain,
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

	if err != nil {
		return nil, fmt.Errorf("AddDomain failed: %w", err)
	}

	return result, nil
}

func (s *ActiveDirectoryService) GetAllDomains(ctx context.Context, activeDirectoryUID string) ([]*core.EntityResult[*rpap.Domain], error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*core.EntityResult[*rpap.Domain], error) {
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
