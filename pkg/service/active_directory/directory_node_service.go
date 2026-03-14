package active_directory

import (
	"RedPaths-server/internal/db"
	"RedPaths-server/internal/repository/active_directory"
	"RedPaths-server/internal/repository/dgraphutil"
	"RedPaths-server/internal/repository/redpaths"
	"RedPaths-server/internal/utils"
	rpad "RedPaths-server/pkg/model/active_directory"
	"RedPaths-server/pkg/model/active_directory/gpo"
	"RedPaths-server/pkg/model/active_directory/priv"
	"RedPaths-server/pkg/model/core"
	utils2 "RedPaths-server/pkg/model/utils"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/dgo/v210"
)

type DirectoryNodeService struct {
	domainRepo          active_directory.DomainRepository
	hostRepo            active_directory.HostRepository
	userRepo            active_directory.UserRepository
	activeDirectoryRepo active_directory.ActiveDirectoryRepository
	directoryNodeRepo   active_directory.DirectoryNodeRepository
	assertionRepo       redpaths.AssertionRepository
	aclRepo             active_directory.ACLRepository
	db                  *dgo.Dgraph
}

func NewDirectoryNodeService(dgraphCon *dgo.Dgraph) (*DirectoryNodeService, error) {
	domainRepo := active_directory.NewDgraphDomainRepository(dgraphCon)
	hostRepo := active_directory.NewDgraphHostRepository(dgraphCon)
	activeDirectoryRepo := active_directory.NewDgraphActiveDirectoryRepository(dgraphCon)
	userRepo := active_directory.NewDgraphUserRepository(dgraphCon)
	directoryNodeRepo := active_directory.NewDgraphDirectoryNodeRepository(dgraphCon)
	assertionRepo := redpaths.NewDgraphAssertionRepository(dgraphCon)

	return &DirectoryNodeService{
		db:                  dgraphCon,
		domainRepo:          domainRepo,
		hostRepo:            hostRepo,
		activeDirectoryRepo: activeDirectoryRepo,
		userRepo:            userRepo,
		directoryNodeRepo:   directoryNodeRepo,
		assertionRepo:       assertionRepo,
	}, nil
}

func (s *DirectoryNodeService) AddSecurityPrincipal(
	ctx context.Context,
	directoryNodeUID string,
	incomingSecurityPrincipal rpad.SecurityPrincipal,
	actor string,
) (*core.EntityResult[rpad.SecurityPrincipal], error) {

	log.Println("[AddSecurityPrincipal]")

	var result *core.EntityResult[rpad.SecurityPrincipal]

	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {

		var (
			securityPrincipal rpad.SecurityPrincipal
			assertions        []*core.Assertion
			err               error
		)

		switch p := incomingSecurityPrincipal.(type) {

		case *rpad.User:
			// TODO: optional Existenzprüfung (z.B. by SID / UPN)
			securityPrincipal, err = s.userRepo.Create(ctx, tx, p, actor)
			if err != nil {
				return fmt.Errorf("creating user: %w", err)
			}

		/*
			case *rpad.Group:
				securityPrincipal, err = s.groupRepo.Create(ctx, tx, p, actor)
				if err != nil {
					return fmt.Errorf("creating group: %w", err)
				}

			case *rpad.Computer:
				securityPrincipal, err = s.computerRepo.Create(ctx, tx, p, actor)
				if err != nil {
					return fmt.Errorf("creating computer: %w", err)
				}
		*/

		default:
			return fmt.Errorf("unsupported security principal type: %T", p)
		}

		log.Printf(
			"[AddSecurityPrincipal] Created %s uid=%s",
			securityPrincipal.PrincipalType(),
			securityPrincipal.GetUID(),
		)

		// Create assertion: DirectoryNode CONTAINS SecurityPrincipal
		assertion := &core.Assertion{
			Predicate:           core.PredicateContains,
			Method:              core.MethodDirectAdd,
			Source:              actor,
			Confidence:          1.0,
			Status:              core.StatusValidated,
			Timestamp:           time.Now(),
			HasDiscoveredParent: true,
			MarkedAsHighValue:   false,
			Subject: &utils2.UIDRef{
				UID:  directoryNodeUID,
				Type: "DirectoryNode",
			},
			Object: &utils2.UIDRef{
				UID:  securityPrincipal.GetUID(),
				Type: string(securityPrincipal.PrincipalType()),
			},
		}

		createdAssertion, err := s.assertionRepo.Create(ctx, tx, assertion)
		if err != nil {
			return fmt.Errorf("creating assertion: %w", err)
		}
		assertions = append(assertions, createdAssertion)

		// Build result (identisch zu AddDirectoryNode)
		result = &core.EntityResult[rpad.SecurityPrincipal]{
			Entity:     securityPrincipal,
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
		return nil, fmt.Errorf("AddSecurityPrincipal failed: %w", err)
	}

	return result, nil
}

func (s *DirectoryNodeService) AddGPOLink(
	ctx context.Context,
	directoryNodeUID string,
	incomingGPOLink *gpo.Link,
	actor string,
) (*core.EntityResult[*gpo.Link], error) {

	panic("implement me")
}

func (s *DirectoryNodeService) GetDirectoryNodeSecurityPrincipals(ctx context.Context, directoryNodeUID string) ([]*rpad.SecurityPrincipal, error) {
	panic("implement me")
}

func (s *DirectoryNodeService) GetDirectoryNodeACL(ctx context.Context, directoryNodeUID string) (*priv.ACL, error) {
	return db.ExecuteInTransactionWithResult[*priv.ACL](ctx, s.db, func(tx *dgo.Txn) (*priv.ACL, error) {
		return s.aclRepo.GetByDirectoryNodeUID(ctx, tx, directoryNodeUID)
	})
}

func (s *DirectoryNodeService) UpdateDirectoryNode(ctx context.Context, uid, actor string, fields map[string]interface{}) (*rpad.DirectoryNode, error) {
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

	return db.ExecuteInTransactionWithResult[*rpad.DirectoryNode](ctx, s.db, func(tx *dgo.Txn) (*rpad.DirectoryNode, error) {
		return s.directoryNodeRepo.Update(ctx, tx, uid, actor, fields)
	})
}

func (s *DirectoryNodeService) CreateBuildDefaultDirectoryNodes(ctx context.Context, tx *dgo.Txn, actor, domainUID string) ([]*rpad.DirectoryNode, error) {

	defaultDirNodes := []*rpad.DirectoryNode{
		{
			Name:     "Users",
			NodeType: rpad.DirectoryNodeTypeOU,
			Parent:   &utils2.UIDRef{UID: domainUID},
		},
		{
			Name:     "Computers",
			NodeType: rpad.DirectoryNodeTypeOU,
			Parent:   &utils2.UIDRef{UID: domainUID},
		},
		{
			Name:     "Builtin",
			NodeType: rpad.DirectoryNodeTypeContainer,
			Parent:   &utils2.UIDRef{UID: domainUID},
		},
		{
			Name:     "Domain Controllers",
			NodeType: rpad.DirectoryNodeTypeOU,
			Parent:   &utils2.UIDRef{UID: domainUID},
		},
	}

	var createdDefaultDirNodes []*rpad.DirectoryNode

	for _, dirNode := range defaultDirNodes {

		createdDirNode, err := s.directoryNodeRepo.Create(ctx, tx, dirNode, actor)
		createdDefaultDirNodes = append(createdDefaultDirNodes, createdDirNode)

		if err != nil {
			return nil, fmt.Errorf("error while creating directory node: %w", err)
		}

		assertion := &core.Assertion{
			Predicate:           core.PredicateContains,
			Method:              core.MethodDirectAdd,
			Source:              actor,
			Confidence:          1.0,
			Status:              core.StatusValidated,
			Timestamp:           time.Now(),
			HasDiscoveredParent: true,
			MarkedAsHighValue:   false,
			Subject: &utils2.UIDRef{
				UID:  domainUID,
				Type: "Domain",
			},
			Object: &utils2.UIDRef{
				UID:  createdDirNode.UID,
				Type: "Default Directory Node",
			},
		}

		createdAssertion, err := s.assertionRepo.Create(ctx, tx, assertion)
		if err != nil {
			return nil, fmt.Errorf("error while creating assertion with uid: %w with error: %w", err)
		}
		log.Printf("Created assertion with uid: %s", createdAssertion.UID)
	}
	log.Printf("Created default-dir-nodes: %v", createdDefaultDirNodes)
	return createdDefaultDirNodes, nil

}

func (s *DirectoryNodeService) GetAllDirectoryNodesInDomain(
	ctx context.Context,
	domainUID string,
) ([]*core.EntityResult[*rpad.DirectoryNode], error) {

	fields := []string{
		"uid",
		"directory_node.name",
		"directory_node.description",
		"directory_node.distinguished_name",
		"directory_node.node_type",
		"directory_node.is_builtin",
		"dgraph.type",
		"discovered_by",
		"discovered_at",
		"last_seen_at",
		"last_seen_by",
	}

	// 1. Erste Ebene: Domain --contains--> DirectoryNodes
	rootNodes, err := db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*core.EntityResult[*rpad.DirectoryNode], error) {
		return dgraphutil.GetEntitiesWithAssertions[*rpad.DirectoryNode](
			ctx, tx, domainUID,
			core.PredicateContains,
			"DirectoryNode",
			fields,
			"getDomainRootDirectoryNodes",
		)
	})
	if err != nil {
		return nil, fmt.Errorf("fetching root directory nodes for domain %s: %w", domainUID, err)
	}

	allResults := make([]*core.EntityResult[*rpad.DirectoryNode], 0, len(rootNodes))
	allResults = append(allResults, rootNodes...)

	// 2. Ab hier rekursiv: DirectoryNode --parent--> DirectoryNode
	for _, rootNode := range rootNodes {
		children, err := s.GetAllChildrenRecursive(ctx, rootNode.Entity.UID)
		if err != nil {
			return nil, fmt.Errorf("fetching children of %s: %w", rootNode.Entity.UID, err)
		}
		allResults = append(allResults, children...)
	}

	return allResults, nil
}

func (s *DirectoryNodeService) GetAllChildDirectoryNodes(
	ctx context.Context,
	parentUID string,
) ([]*core.EntityResult[*rpad.DirectoryNode], error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*core.EntityResult[*rpad.DirectoryNode], error) {

		fields := []string{
			"uid",
			"directory_node.name",
			"directory_node.description",
			"directory_node.distinguished_name",
			"directory_node.node_type",
			"directory_node.is_builtin",
			"dgraph.type",
			"discovered_by",
			"discovered_at",
			"last_seen_at",
			"last_seen_by",
		}

		return dgraphutil.GetEntitiesWithAssertions[*rpad.DirectoryNode](
			ctx,
			tx,
			parentUID,
			core.PredicateParent,
			"DirectoryNode",
			fields,
			"getChildDirectoryNodes",
		)
	})
}

func (s *DirectoryNodeService) GetAllChildrenRecursive(
	ctx context.Context,
	rootUID string,
) ([]*core.EntityResult[*rpad.DirectoryNode], error) {

	var allResults []*core.EntityResult[*rpad.DirectoryNode]
	queue := []string{rootUID}

	for len(queue) > 0 {
		currentUID := queue[0]
		queue = queue[1:]

		children, err := s.GetAllChildDirectoryNodes(ctx, currentUID)
		if err != nil {
			return nil, fmt.Errorf("fetching children of %s: %w", currentUID, err)
		}

		allResults = append(allResults, children...)

		for _, child := range children {
			queue = append(queue, child.Entity.UID)
		}
	}

	return allResults, nil
}

func (s *DirectoryNodeService) AddChildDirectoryNode(
	ctx context.Context,
	parentDirectoryNodeUID string,
	incomingChildDirectoryNode *rpad.DirectoryNode,
	actor string,
) (*core.EntityResult[rpad.DirectoryNode], error) {

	log.Println("[AddChildDirectoryNode] Adding child directory node")

	var result *core.EntityResult[rpad.DirectoryNode]

	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {

		var (
			assertions []*core.Assertion
			err        error
		)

		createdChildDirNode, err := s.directoryNodeRepo.Create(ctx, tx, incomingChildDirectoryNode, actor)

		log.Printf(
			"[AddChildDirectoryNode] Created uid=%s",
			createdChildDirNode.UID,
		)

		// Create assertion
		assertion := &core.Assertion{
			Predicate:           core.PredicateParent,
			Method:              core.MethodDirectAdd,
			Source:              actor,
			Confidence:          1.0,
			Status:              core.StatusValidated,
			Timestamp:           time.Now(),
			HasDiscoveredParent: true,
			MarkedAsHighValue:   false,
			Subject: &utils2.UIDRef{
				UID:  parentDirectoryNodeUID,
				Type: "Domain|DirectoryNode",
			},
			Object: &utils2.UIDRef{
				UID:  createdChildDirNode.UID,
				Type: "DirectoryNode",
			},
		}

		createdAssertion, err := s.assertionRepo.Create(ctx, tx, assertion)
		if err != nil {
			return fmt.Errorf("creating assertion: %w", err)
		}
		assertions = append(assertions, createdAssertion)

		result = &core.EntityResult[rpad.DirectoryNode]{
			Entity:     *createdChildDirNode,
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
		return nil, fmt.Errorf("AddChildDirectoryNode failed: %w", err)
	}

	return result, nil
}
