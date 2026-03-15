package active_directory

import (
	"RedPaths-server/internal/db"
	"RedPaths-server/internal/repository/active_directory"
	"RedPaths-server/internal/repository/redpaths"
	"RedPaths-server/internal/utils"
	"RedPaths-server/pkg/model"
	rpad "RedPaths-server/pkg/model/active_directory"
	"RedPaths-server/pkg/model/active_directory/gpo"
	"RedPaths-server/pkg/model/active_directory/priv"
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/core/res"
	utils2 "RedPaths-server/pkg/model/utils"
	"RedPaths-server/pkg/model/utils/assertion"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/dgo/v210"
)

type DomainService struct {
	domainRepo        active_directory.DomainRepository
	hostRepo          active_directory.HostRepository
	directoryNodeRepo active_directory.DirectoryNodeRepository
	assertionRepo     redpaths.AssertionRepository
	aclRepo           active_directory.ACLRepository
	gpoRepo           active_directory.GPORepository

	db *dgo.Dgraph
}

func NewDomainService(dgraphCon *dgo.Dgraph) (*DomainService, error) {
	domainRepo := active_directory.NewDgraphDomainRepository(dgraphCon)
	hostRepo := active_directory.NewDgraphHostRepository(dgraphCon)
	directoryNodeRepo := active_directory.NewDgraphDirectoryNodeRepository(dgraphCon)
	assertionRepo := redpaths.NewDgraphAssertionRepository(dgraphCon)
	aclRepo := active_directory.NewDgraphDgraphACLRepository(dgraphCon)
	gpoRepo := active_directory.NewDgraphGPORepository(dgraphCon)

	return &DomainService{
		db:                dgraphCon,
		domainRepo:        domainRepo,
		hostRepo:          hostRepo,
		directoryNodeRepo: directoryNodeRepo,
		assertionRepo:     assertionRepo,
		aclRepo:           aclRepo,
		gpoRepo:           gpoRepo,
	}, nil
}

func (s *DomainService) AddHost(
	ctx context.Context,
	assertionCtx assertion.Context,
	domainUID string,
	host *model.Host,
	actor string,
) (*res.EntityResult[*model.Host], error) {

	log.Println("[AddHost]")

	var result *res.EntityResult[*model.Host]

	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		existingHost, err := s.hostRepo.FindByIPInDomain(ctx, tx, domainUID, host.IP)
		if err != nil {
			return fmt.Errorf("checking existing host: %w", err)
		}

		var actualHost *model.Host
		var assertions []*core.Assertion

		if existingHost != nil {
			actualHost = existingHost
			log.Printf(
				"[AddHost] Reusing existing host uid=%s ip=%s",
				actualHost.UID,
				actualHost.IP,
			)
		} else {
			actualHostUID, err := s.hostRepo.Create(ctx, tx, host, actor)
			if err != nil {
				return fmt.Errorf("creating host: %w", err)
			}
			actualHost = host
			actualHost.UID = actualHostUID

			log.Printf(
				"[AddHost] Created host uid=%s ip=%s name=%s",
				actualHost.UID,
				actualHost.IP,
				actualHost.Name,
			)
		}

		assertionSchema := &core.Assertion{
			Predicate:           core.PredicateHasHost,
			Method:              core.MethodDirectAdd,
			Source:              actor,
			Confidence:          assertionCtx.GetConfidence(),
			Status:              core.StatusValidated,
			Timestamp:           time.Now(),
			HasDiscoveredParent: true,
			MarkedAsHighValue:   assertionCtx.IsHighValue(),
			Subject:             &utils2.UIDRef{UID: domainUID, Type: "Domain"},
			Object:              &utils2.UIDRef{UID: actualHost.UID, Type: "Host"},
		}

		createdAssertion, err := s.assertionRepo.Create(ctx, tx, assertionSchema)
		if err != nil {
			return fmt.Errorf("creating assertion: %w", err)
		}
		assertions = append(assertions, createdAssertion)

		result = &res.EntityResult[*model.Host]{
			Entity:     actualHost,
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
		return nil, fmt.Errorf("AddHost failed: %w", err)
	}

	return result, nil
}

func (s *DomainService) AddDirectoryNode(
	ctx context.Context,
	assertionCtx assertion.Context,
	domainUID string,
	incomingDirectoryNode *rpad.DirectoryNode,
	actor string,
) (*res.EntityResult[*rpad.DirectoryNode], error) {

	log.Println("[AddDirectoryNode]")

	var result *res.EntityResult[*rpad.DirectoryNode]

	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {

		// Check if DirectoryNode already exists in domain
		existingDirectoryNode, err := s.directoryNodeRepo.
			FindByDistinguishedNameInDomain(
				ctx,
				tx,
				domainUID,
				incomingDirectoryNode.DistinguishedName,
			)
		if err != nil {
			return fmt.Errorf("checking existing directory node: %w", err)
		}

		var directoryNode *rpad.DirectoryNode
		var acl *priv.ACL
		var assertions []*core.Assertion

		if existingDirectoryNode != nil {
			// Reuse existing node
			directoryNode = existingDirectoryNode
			log.Printf(
				"[AddDirectoryNode] Reusing existing directory node uid=%s dn=%s",
				directoryNode.UID,
				directoryNode.DistinguishedName,
			)
		} else {
			// Create DirectoryNode
			directoryNode, err = s.directoryNodeRepo.Create(
				ctx,
				tx,
				incomingDirectoryNode,
				actor,
			)

			if err != nil {
				return fmt.Errorf("creating directory node: %w", err)
			}

			log.Printf(
				"[AddDirectoryNode] Created directory node uid=%s dn=%s",
				directoryNode.UID,
				directoryNode.DistinguishedName,
			)

			// Create & Link ACL
			aclEntity := &priv.ACL{}
			acl, err = s.aclRepo.CreateACL(ctx, tx, aclEntity, actor)

			if err != nil {
				return fmt.Errorf("error while creating ACL for directory node: %w", err)
			}

			err = s.aclRepo.LinkACLToEntity(ctx, tx, acl.UID, directoryNode.UID)

			if err != nil {
				return fmt.Errorf("error while linking ACL to directory node: %w", err)
			}
			log.Printf("Created and linked acl to directory node")
		}

		// Create assertion
		assertionEntity := &core.Assertion{
			Predicate:           core.PredicateContains,
			Method:              core.MethodDirectAdd,
			Source:              actor,
			Confidence:          assertionCtx.GetConfidence(),
			Status:              core.StatusValidated,
			Timestamp:           time.Now(),
			HasDiscoveredParent: true,
			MarkedAsHighValue:   assertionCtx.IsHighValue(),
			Subject:             &utils2.UIDRef{UID: domainUID, Type: "Domain"},
			Object:              &utils2.UIDRef{UID: directoryNode.UID, Type: "DirectoryNode"},
		}

		createdAssertion, err := s.assertionRepo.Create(ctx, tx, assertionEntity)
		if err != nil {
			return fmt.Errorf("creating assertion: %w", err)
		}
		assertions = append(assertions, createdAssertion)

		result = &res.EntityResult[*rpad.DirectoryNode]{
			Entity:     directoryNode,
			Assertions: assertions,
			ACL:        acl,
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
		return nil, fmt.Errorf("AddDirectoryNode failed: %w", err)
	}

	return result, nil
}

func (s *DomainService) GetLinkedGPOs(ctx context.Context, domainUID string) (*res.GPOQueryResult, error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) (*res.GPOQueryResult, error) {
		return s.gpoRepo.GetGPOResultsByDomain(ctx, tx, domainUID)
	})
}

func (s *DomainService) LinkGPO(ctx context.Context, assertionCtx assertion.Context, incomingGPOLink *gpo.Link, incomingGPO *gpo.GPO, domainUID, actor string) (*res.GPOResult[*gpo.Link], error) {
	log.Println("[AddGPOLink]")

	var result *res.GPOResult[*gpo.Link]
	var linkedGPO *gpo.GPO

	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {

		// Check if gpo already exists in domain scope
		existingGPO, err := s.domainRepo.GetGPOIfKnown(ctx, tx, incomingGPO.Name)
		if err != nil {
			return fmt.Errorf("error while checking existing directory node: %w", err)
		}

		if len(existingGPO) == 0 {
			log.Println("[AddGPOLink] No existing GPO found. Creating new one")
			linkedGPO, err = s.gpoRepo.CreateGPO(ctx, tx, incomingGPO, actor)
			if err != nil {
				return fmt.Errorf("error while creating gpo: %w", err)
			}
		} else {
			// names of gpos are unique, so is only one result gpo
			log.Println("Found GPO in Domain. Using existing one...")
			linkedGPO = existingGPO[0]
		}

		var gpoLink *gpo.Link
		var gpoLinkAssertion []*core.Assertion
		var gpoAssertion []*core.Assertion

		//TODO: Exist Check for links

		gpoLink, err = s.gpoRepo.CreateLink(
			ctx,
			tx,
			incomingGPOLink,
			actor,
		)

		if err != nil {
			return fmt.Errorf("failed creating gpo link: %w", err)
		}

		log.Printf(
			"[AddGPOLink] Created gpo link to domain uid=%s",
			gpoLink.UID,
		)
		// Create assertion to gpo link
		assertionEntity := &core.Assertion{
			Predicate:           core.PredicateHasGPOLink,
			Method:              core.MethodDirectAdd,
			Source:              actor,
			Confidence:          assertionCtx.GetConfidence(),
			Status:              core.StatusValidated,
			Timestamp:           time.Now(),
			HasDiscoveredParent: true,
			MarkedAsHighValue:   assertionCtx.IsHighValue(),
			Subject:             &utils2.UIDRef{UID: domainUID, Type: "Domain"},
			Object:              &utils2.UIDRef{UID: gpoLink.UID, Type: "GPOLink"},
		}

		createdAssertion, err := s.assertionRepo.Create(ctx, tx, assertionEntity)
		if err != nil {
			return fmt.Errorf("creating assertion from domain to gpo link: %w", err)
		}
		gpoLinkAssertion = append(gpoLinkAssertion, createdAssertion)

		// Create assertion to gpo
		assertionEntity = &core.Assertion{
			Predicate:           core.PredicateLinksTo,
			Method:              core.MethodDirectAdd,
			Source:              actor,
			Confidence:          assertionCtx.GetConfidence(),
			Status:              core.StatusValidated,
			Timestamp:           time.Now(),
			HasDiscoveredParent: true,
			MarkedAsHighValue:   assertionCtx.IsHighValue(),
			Subject:             &utils2.UIDRef{UID: gpoLink.UID, Type: "GPOLink"},
			Object:              &utils2.UIDRef{UID: linkedGPO.UID, Type: "GPO"},
		}

		createdAssertion, err = s.assertionRepo.Create(ctx, tx, assertionEntity)
		if err != nil {
			return fmt.Errorf("creating assertion: %w", err)
		}
		gpoAssertion = append(gpoAssertion, createdAssertion)

		result = &res.GPOResult[*gpo.Link]{
			GPOLink:           gpoLink,
			GPOLinkAssertions: gpoLinkAssertion,
			GPO:               linkedGPO,
			GPOAssertions:     gpoAssertion,
			Metadata: &res.ResultMetadata{
				Source:         actor,
				ScanTimestamp:  time.Now(),
				EntityCount:    2,
				AssertionCount: len(gpoAssertion) + len(gpoLinkAssertion),
			},
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("AddDirectoryNode failed: %w", err)
	}

	return result, nil

}

func (s *DomainService) GetDomainHosts(ctx context.Context, domainUID string) ([]*res.EntityResult[*model.Host], error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*res.EntityResult[*model.Host], error) {
		return s.hostRepo.GetAllByDomainUID(ctx, tx, domainUID)
	})
}

func (s *DomainService) GetDomainGPOs(ctx context.Context, domainUID string) (*res.GPOQueryResult, error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) (*res.GPOQueryResult, error) {
		return s.gpoRepo.GetGPOResultsByDomain(ctx, tx, domainUID)
	})
}

func (s *DomainService) GetDomainDirectoryNodes(ctx context.Context, domainUID string) ([]*res.EntityResult[*rpad.DirectoryNode], error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*res.EntityResult[*rpad.DirectoryNode], error) {
		return s.directoryNodeRepo.GetAllByDomainUID(ctx, tx, domainUID)
	})
}

func (s *DomainService) UpdateDomain(ctx context.Context, uid, actor string, fields map[string]interface{}) (*rpad.Domain, error) {
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

	return db.ExecuteInTransactionWithResult[*rpad.Domain](ctx, s.db, func(tx *dgo.Txn) (*rpad.Domain, error) {
		return s.domainRepo.Update(ctx, tx, uid, actor, fields)
	})
}
