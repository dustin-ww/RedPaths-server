package active_directory

import (
	"RedPaths-server/internal/db"
	"RedPaths-server/internal/repository/active_directory"
	"RedPaths-server/internal/repository/redpaths/engine"
	dgraphutil "RedPaths-server/internal/repository/util/dgraph"
	"RedPaths-server/internal/utils"
	"RedPaths-server/pkg/model"
	rpad "RedPaths-server/pkg/model/active_directory"
	"RedPaths-server/pkg/model/active_directory/gpo"
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

type DomainService struct {
	domainRepo           active_directory.DomainRepository
	hostRepo             active_directory.HostRepository
	directoryNodeRepo    active_directory.DirectoryNodeRepository
	userRepo             active_directory.UserRepository
	assertionRepo        engine.AssertionRepository
	aclRepo              active_directory.ACLRepository
	gpoRepo              active_directory.GPORepository
	catalogService       *engine3.CatalogService
	directoryNodeService *DirectoryNodeService // neu

	db *dgo.Dgraph
}

func NewDomainService(dgraphCon *dgo.Dgraph) (*DomainService, error) {
	domainRepo := active_directory.NewDgraphDomainRepository(dgraphCon)
	hostRepo := active_directory.NewDgraphHostRepository(dgraphCon)
	directoryNodeRepo := active_directory.NewDgraphDirectoryNodeRepository(dgraphCon)
	userRepo := active_directory.NewDgraphUserRepository(dgraphCon)
	assertionRepo := engine.NewDgraphAssertionRepository(dgraphCon)
	aclRepo := active_directory.NewDgraphDgraphACLRepository(dgraphCon)
	gpoRepo := active_directory.NewDgraphGPORepository(dgraphCon)
	catalogService := engine3.NewCatalogService(dgraphCon)
	directoryNodeService, _ := NewDirectoryNodeService(dgraphCon)

	return &DomainService{
		db:                   dgraphCon,
		domainRepo:           domainRepo,
		hostRepo:             hostRepo,
		directoryNodeRepo:    directoryNodeRepo,
		userRepo:             userRepo,
		assertionRepo:        assertionRepo,
		aclRepo:              aclRepo,
		gpoRepo:              gpoRepo,
		catalogService:       catalogService,
		directoryNodeService: directoryNodeService,
	}, nil
}

func (s *DomainService) AddHost(
	ctx context.Context,
	assertionCtx assertion.Context,
	domainUID string,
	host *model.Host,
	actor string,
) (*res.EntityResult[*model.Host], error) {

	log.Println("[AddDomainHost]")

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
				"[AddDomainHost] Reusing existing host uid=%s ip=%s",
				actualHost.UID,
				actualHost.IP,
			)
		} else {
			actualHostUID, err := s.hostRepo.Create(ctx, tx, host, actor)
			if err != nil {
				return fmt.Errorf("creating host: %w", err)
			}
			actualHost = host
			actualHost.UID = actualHostUID.UID

			log.Printf(
				"[AddDomainHost] Created host uid=%s ip=%s name=%s",
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
		return nil, fmt.Errorf("AddDomainHost failed: %w", err)
	}

	return result, nil
}

func (s *DomainService) AddUser(
	ctx context.Context,
	assertionCtx assertion.Context,
	projectUID string,
	domainUID string,
	incomingUser *rpad.User,
	actor string,
) (*res.EntityResult[*rpad.User], error) {

	log.Println("[AddDomainUser]")

	var result *res.EntityResult[*rpad.User]

	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		existingUsers, err := s.userRepo.GetByDomainUID(ctx, tx, domainUID)
		if err != nil {
			return fmt.Errorf("checking existing users: %w", err)
		}

		var actualUser *rpad.User

		for _, u := range existingUsers {
			if (incomingUser.SID != "" && u.SID == incomingUser.SID) ||
				(incomingUser.SAMAccountName != "" && u.SAMAccountName == incomingUser.SAMAccountName) ||
				(incomingUser.UPN != "" && u.UPN == incomingUser.UPN) {
				actualUser = u
				log.Printf(
					"[AddDomainUser] Reusing existing user uid=%s name=%s",
					actualUser.UID,
					actualUser.Name,
				)
				break
			}
		}

		if actualUser == nil {
			createdUser, err := s.userRepo.Create(ctx, tx, incomingUser, actor)
			if err != nil {
				return fmt.Errorf("creating user: %w", err)
			}
			actualUser = createdUser
			log.Printf(
				"[AddDomainUser] Created user uid=%s name=%s",
				actualUser.UID,
				actualUser.Name,
			)
		}

		assertionSchema := &core.Assertion{
			Predicate:           core.PredicateHasUser,
			Method:              core.MethodDirectAdd,
			Source:              actor,
			Confidence:          assertionCtx.GetConfidence(),
			Status:              core.StatusValidated,
			Timestamp:           time.Now(),
			HasDiscoveredParent: true,
			MarkedAsHighValue:   assertionCtx.IsHighValue(),
			Subject:             &utils2.UIDRef{UID: domainUID, Type: "Domain"},
			Object:              &utils2.UIDRef{UID: actualUser.UID, Type: "User"},
		}

		createdAssertion, err := s.assertionRepo.Create(ctx, tx, assertionSchema)
		if err != nil {
			return fmt.Errorf("creating assertion: %w", err)
		}

		result = &res.EntityResult[*rpad.User]{
			Entity:     actualUser,
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
		return nil, fmt.Errorf("AddDomainUser failed: %w", err)
	}

	if result != nil && len(result.Assertions) > 0 {
		if _, catalogErr := engine3.AddToCatalog(
			ctx, s.catalogService,
			projectUID, result.Entity.UID, "User",
			result.Assertions[0], actor,
		); catalogErr != nil {
			log.Printf("[AddDomainUser] Warning: failed to add user %s to catalog: %v",
				result.Entity.UID, catalogErr)
		}
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

	log.Println("[AddDomainDirectoryNode]")

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
				"[AddDomainDirectoryNode] Reusing existing directory node uid=%s dn=%s",
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
				"[AddDomainDirectoryNode] Created directory node uid=%s dn=%s",
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
		return nil, fmt.Errorf("AddDomainDirectoryNode failed: %w", err)
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
		return nil, fmt.Errorf("AddDomainDirectoryNode failed: %w", err)
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

// -----------------------------------------------------------------------------
// UpsertDomain
// -----------------------------------------------------------------------------

func (s *DomainService) UpsertDomain(
	ctx context.Context,
	input engine4.Input[*rpad.Domain],
) (*res.EntityResult[*rpad.Domain], error) {

	subjectUID, subjectType, hasParent := input.Resolved()

	var result *res.EntityResult[*rpad.Domain]

	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		// --- Existence Check ---
		existence, err := s.domainRepo.FindExisting(ctx, tx, input.ProjectUID, input.Entity)
		if err != nil {
			return fmt.Errorf("existence check failed: %w", err)
		}

		filters := active_directory.BuildDomainFilter(input.Entity)
		var actualDomain *rpad.Domain

		switch existence.FoundVia {

		// --- Case 1: Not found → create new ---
		case dgraphutil.ExistenceSourceNotFound:
			createdDomain, err := s.domainRepo.Create(ctx, tx, input.Entity, input.Actor)
			if err != nil {
				return fmt.Errorf("creating domain: %w", err)
			}
			actualDomain = createdDomain
			log.Printf("[UpsertDomain] Created uid=%s name=%s", actualDomain.UID, actualDomain.Name)

			defaultDirNodes, err := s.directoryNodeService.CreateBuildDefaultDirectoryNodes(ctx, tx, input.Actor, actualDomain.UID)
			if err != nil {
				return fmt.Errorf("creating default directory nodes for domain: %w", err)
			}
			log.Printf("[UpsertDomain] Created %d default directory nodes for domain uid=%s", len(defaultDirNodes), actualDomain.UID)

		case dgraphutil.ExistenceSourceHierarchy,
			dgraphutil.ExistenceSourceProject:

			best := dgraphutil.BestCandidate(existence.Entities, filters, 0.5)

			if best == nil {
				// Candidates found but score too low → create new
				createdDomain, err := s.domainRepo.Create(ctx, tx, input.Entity, input.Actor)
				if err != nil {
					return fmt.Errorf("creating domain (low score): %w", err)
				}
				actualDomain = createdDomain
				log.Printf("[UpsertDomain] Low score, created uid=%s", actualDomain.UID)

				defaultDirNodes, err := s.directoryNodeService.CreateBuildDefaultDirectoryNodes(ctx, tx, input.Actor, actualDomain.UID)
				if err != nil {
					return fmt.Errorf("creating default directory nodes for domain (low score): %w", err)
				}
				log.Printf("[UpsertDomain] Created %d default directory nodes for domain uid=%s", len(defaultDirNodes), actualDomain.UID)

			} else if best.Score >= 0.8 {
				// --- Case 2: High Confidence → Merge ---
				mergeFields := buildDomainMergeFields(
					best.Result.Entity,
					input.Entity,
					input.AssertionCtx.GetConfidence(),
				)
				updated, err := s.domainRepo.Update(
					ctx, tx,
					best.Result.Entity.UID,
					input.Actor,
					mergeFields,
				)
				if err != nil {
					return fmt.Errorf("merging domain: %w", err)
				}
				actualDomain = updated
				log.Printf("[UpsertDomain] Merged uid=%s score=%.2f",
					actualDomain.UID, best.Score)

			} else {
				// --- Case 3: Medium Confidence (0.5–0.8) → Possible Duplicate ---
				log.Printf("[UpsertDomain] Possible duplicate uid=%s score=%.2f",
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
					Subject:             &utils2.UIDRef{UID: best.Result.Entity.UID, Type: "Domain"},
					Object:              &utils2.UIDRef{UID: input.Entity.UID, Type: "Domain"},
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
			Predicate:           core.PredicateHasDomain,
			Method:              core.MethodDirectAdd,
			Source:              input.Actor,
			Confidence:          input.AssertionCtx.GetConfidence(),
			Status:              core.StatusValidated,
			Timestamp:           time.Now(),
			HasDiscoveredParent: hasParent,
			MarkedAsHighValue:   input.AssertionCtx.IsHighValue(),
			Subject:             &utils2.UIDRef{UID: subjectUID, Type: subjectType},
			Object:              &utils2.UIDRef{UID: actualDomain.UID, Type: "Domain"},
		}

		createdAssertion, err := s.assertionRepo.Create(ctx, tx, assertionSchema)
		if err != nil {
			return fmt.Errorf("creating assertion: %w", err)
		}

		result = &res.EntityResult[*rpad.Domain]{
			Entity:     actualDomain,
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
		return nil, fmt.Errorf("UpsertDomain failed: %w", err)
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
		"Domain",
		result.Assertions[0],
		input.Actor,
	)
	if catalogErr != nil {
		log.Printf("[UpsertDomain] Warning: failed to add domain %s to catalog: %v",
			result.Entity.UID, catalogErr)
	}

	if hasParent {
		promoteErr := engine3.PromoteInCatalog(
			ctx,
			s.catalogService,
			input.ProjectUID,
			result.Entity.UID,
			"Domain",
			core.PredicateHasDomain,
			input.Actor,
		)
		if promoteErr != nil {
			log.Printf("[UpsertDomain] Warning: failed to promote domain %s in catalog: %v",
				result.Entity.UID, promoteErr)
		}
	}

	return result, nil
}

// -----------------------------------------------------------------------------
// buildDomainMergeFields
// -----------------------------------------------------------------------------

func buildDomainMergeFields(
	existing *rpad.Domain,
	incoming *rpad.Domain,
	incomingConfidence float64,
) map[string]interface{} {
	fields := map[string]interface{}{
		"last_seen_at": time.Now(),
	}

	// Name: overwrite if incoming set and existing empty
	if incoming.Name != "" && existing.Name == "" {
		fields["domain.name"] = incoming.Name
	}

	// DNS name: overwrite if incoming set and existing empty
	if incoming.DNSName != "" && existing.DNSName == "" {
		fields["domain.dns_name"] = incoming.DNSName
	}

	// NetBIOS name: overwrite if incoming set and existing empty
	if incoming.NetBiosName != "" && existing.NetBiosName == "" {
		fields["domain.netbios_name"] = incoming.NetBiosName
	}

	// GUID: truly unique identifier — only set once, never overwrite
	if incoming.DomainGUID != "" && existing.DomainGUID == "" {
		fields["domain.domain_guid"] = incoming.DomainGUID
	}

	// SID: truly unique identifier — only set once, never overwrite
	if incoming.DomainSID != "" && existing.DomainSID == "" {
		fields["domain.domain_sid"] = incoming.DomainSID
	}

	// Functional level: overwrite if incoming set, high confidence, or existing empty
	if incoming.DomainFunctionalLevel != "" &&
		(existing.DomainFunctionalLevel == "" || incomingConfidence >= 0.8) {
		fields["domain.functional_level"] = incoming.DomainFunctionalLevel
	}

	// Forest functional level: same rule
	if incoming.ForestFunctionalLevel != "" &&
		(existing.ForestFunctionalLevel == "" || incomingConfidence >= 0.8) {
		fields["domain.forest_functional_level"] = incoming.ForestFunctionalLevel
	}

	return fields
}

// -----------------------------------------------------------------------------
// UpdateDomain
// -----------------------------------------------------------------------------

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
