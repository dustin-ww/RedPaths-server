package active_directory

import (
	"RedPaths-server/internal/db"
	"RedPaths-server/internal/repository/active_directory"
	"RedPaths-server/internal/repository/redpaths/engine"
	"RedPaths-server/pkg/model/active_directory/priv"
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/core/res"
	utils2 "RedPaths-server/pkg/model/utils"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/dgo/v210"
)

type ACLService struct {
	domainRepo          active_directory.DomainRepository
	hostRepo            active_directory.HostRepository
	activeDirectoryRepo active_directory.ActiveDirectoryRepository
	aclRepo             active_directory.ACLRepository
	assertionRepo       engine.AssertionRepository
	db                  *dgo.Dgraph
}

func NewACLService(dgraphCon *dgo.Dgraph) (*ACLService, error) {
	domainRepo := active_directory.NewDgraphDomainRepository(dgraphCon)
	hostRepo := active_directory.NewDgraphHostRepository(dgraphCon)
	activeDirectoryRepo := active_directory.NewDgraphActiveDirectoryRepository(dgraphCon)
	aclRepo := active_directory.NewDgraphDgraphACLRepository(dgraphCon)
	assertionRepo := engine.NewDgraphAssertionRepository(dgraphCon)

	return &ACLService{
		db:                  dgraphCon,
		domainRepo:          domainRepo,
		hostRepo:            hostRepo,
		activeDirectoryRepo: activeDirectoryRepo,
		aclRepo:             aclRepo,
		assertionRepo:       assertionRepo,
	}, nil
}

func (s *ACLService) AddACE(
	ctx context.Context,
	aclID string,
	incomingACE *priv.ACE,
	actor string,
) (*res.EntityResult[*priv.ACE], error) {

	var result *res.EntityResult[*priv.ACE]

	log.Printf("[AddACE] name=%s, ACLUID=%s, actor=%s",
		incomingACE.Name, aclID, actor)

	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		// 1. Check if domain already exists in this AD
		//TODO
		/*existingDomain, err := s.domainRepo.FindByNameInActiveDirectory(
			ctx, tx, activeDirectoryUID, incomingDomain.Name,
		)
		if err != nil {
			return fmt.Errorf("checking existing domain: %w", err)
		}*/

		//var ace *rpap.Domain
		var assertions []*core.Assertion

		/*		if existingDomain != nil {
				// Domain exists - reuse
				domain = existingDomain
				log.Printf("[AddActiveDirectoryDomain] Reusing existing domain uid=%s", domain.UID)
			} else {*/
		// Create new Domain
		ace, err := s.aclRepo.CreateACE(ctx, tx, incomingACE, actor)
		if err != nil {
			return fmt.Errorf("creating domain: %w", err)
		}
		log.Printf("[AddACE] Created ace uid=%s name=%s", ace.UID, ace.Name)

		if err != nil {
			return fmt.Errorf("error while linking ACL to domain: %w", err)
		}
		log.Printf("Created and linked acl to domain")

		// 2. Create assertion (even if domain existed, link might be new)
		assertion := &core.Assertion{
			Predicate:  core.PredicateContains,
			Method:     core.MethodDirectAdd,
			Source:     actor,
			Confidence: 1.0,
			Status:     core.StatusValidated,
			Timestamp:  time.Now(),
			Subject:    &utils2.UIDRef{UID: aclID},
			Object:     &utils2.UIDRef{UID: ace.UID},
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
		result = &res.EntityResult[*priv.ACE]{
			Entity:     ace,
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
		return nil, fmt.Errorf("AddACE failed: %w", err)
	}

	return result, nil
}

func (s *ACLService) AddADRight(
	ctx context.Context,
	aceID string,
	incomingADRight *priv.ADRight,
	actor string,
) (*res.EntityResult[*priv.ADRight], error) {

	var result *res.EntityResult[*priv.ADRight]

	log.Printf("[AddACE] name=%s, ACE_UID=%s, actor=%s",
		incomingADRight.Name, aceID, actor)

	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		// 1. Check if domain already exists in this AD
		//TODO
		/*existingDomain, err := s.domainRepo.FindByNameInActiveDirectory(
			ctx, tx, activeDirectoryUID, incomingDomain.Name,
		)
		if err != nil {
			return fmt.Errorf("checking existing domain: %w", err)
		}*/

		//var ace *rpap.Domain
		var assertions []*core.Assertion

		/*		if existingDomain != nil {
				// Domain exists - reuse
				domain = existingDomain
				log.Printf("[AddActiveDirectoryDomain] Reusing existing domain uid=%s", domain.UID)
			} else {*/
		// Create new Domain
		adRight, err := s.aclRepo.CreateADRight(ctx, tx, incomingADRight, actor)
		if err != nil {
			return fmt.Errorf("creating ad right: %w", err)
		}
		log.Printf("[AddACE] Created ad right uid=%s name=%s", adRight.UID, adRight.Name)

		// 2. Create assertion (even if domain existed, link might be new)
		assertion := &core.Assertion{
			Predicate:  core.PredicateContains,
			Method:     core.MethodDirectAdd,
			Source:     actor,
			Confidence: 1.0,
			Status:     core.StatusValidated,
			Timestamp:  time.Now(),
			Subject:    &utils2.UIDRef{UID: aceID},
			Object:     &utils2.UIDRef{UID: adRight.UID},
		}

		createdAssertion, err := s.assertionRepo.Create(ctx, tx, assertion)
		if err != nil {
			return fmt.Errorf("creating assertion: %w", err)
		}
		assertions = append(assertions, createdAssertion)

		// Build result
		result = &res.EntityResult[*priv.ADRight]{
			Entity:     adRight,
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
		return nil, fmt.Errorf("AddADRight failed: %w", err)
	}

	return result, nil
}

func (s *ACLService) GetAllACE(ctx context.Context, aclUID string) ([]*res.EntityResult[*priv.ACE], error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*res.EntityResult[*priv.ACE], error) {
		return s.aclRepo.GetAllACEByACL(ctx, tx, aclUID)
	})
}

/*func (s *ACLService) UpdateProjectActiveDirectory(ctx context.Context, uid, actor string, fields map[string]interface{}) (*rpap.ActiveDirectory, error) {
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

/*	return db.ExecuteInTransactionWithResult[*rpap.ActiveDirectory](ctx, s.db, func(tx *dgo.Txn) (*rpap.ActiveDirectory, error) {
		return s.activeDirectoryRepo.Update(ctx, tx, uid, actor, fields)
	})
}*/

func (s *ACLService) GetACL(ctx context.Context, aclUID string) (*priv.ACL, error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) (*priv.ACL, error) {
		return s.aclRepo.GetACL(ctx, tx, aclUID)
	})
}
