package active_directory

import (
	"RedPaths-server/internal/db"
	"RedPaths-server/internal/repository/active_directory"
	"RedPaths-server/internal/repository/redpaths"
	"RedPaths-server/pkg/model/active_directory/gpo"
	"RedPaths-server/pkg/model/core"
	utils2 "RedPaths-server/pkg/model/utils"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/dgo/v210"
)

type GPOService struct {
	domainRepo        active_directory.DomainRepository
	hostRepo          active_directory.HostRepository
	directoryNodeRepo active_directory.DirectoryNodeRepository
	assertionRepo     redpaths.AssertionRepository
	gpoRepo           active_directory.GPORepository
	db                *dgo.Dgraph
}

func NewGPOService(dgraphCon *dgo.Dgraph) (*GPOService, error) {
	domainRepo := active_directory.NewDgraphDomainRepository(dgraphCon)
	hostRepo := active_directory.NewDgraphHostRepository(dgraphCon)
	directoryNodeRepo := active_directory.NewDgraphDirectoryNodeRepository(dgraphCon)
	assertionRepo := redpaths.NewDgraphAssertionRepository(dgraphCon)
	gpoRepo := active_directory.NewDgraphGPORepository(dgraphCon)

	return &GPOService{
		db:                dgraphCon,
		domainRepo:        domainRepo,
		hostRepo:          hostRepo,
		directoryNodeRepo: directoryNodeRepo,
		assertionRepo:     assertionRepo,
		gpoRepo:           gpoRepo,
	}, nil
}

// LinkGPOToContainer typ = Domain or DirectoryNode
func (s *GPOService) LinkGPOToContainer(
	ctx context.Context,
	containerUID,
	containerTyp string,
	incomingGPOLink *gpo.Link,
	actor string,
) (*core.EntityResult[*gpo.Link], error) {

	log.Println("[AddGPOLink] start")

	if incomingGPOLink.LinksTo == nil {
		return nil, fmt.Errorf("incoming GPO link has no LinksTo reference")
	}

	var result *core.EntityResult[*gpo.Link]

	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {

		// 1. Check if link already exists
		existingLink, err := s.gpoRepo.FindGPOLinkByGPOName(
			ctx, tx, containerUID, incomingGPOLink.LinksTo.Name,
		)
		if err != nil {
			return fmt.Errorf("find existing GPO link: %w", err)
		}

		if existingLink.Entity != nil {
			log.Println("[AddGPOLink] GPO already linked to container")
			result = existingLink
			return nil
		}

		existingLinkEntity, err := s.gpoRepo.CreateLink(ctx, tx, incomingGPOLink, actor)

		log.Println("[AddGPOLink] GPO not linked yet, creating link objects. Checking if gpo exists")

		// 2. Ensure GPO exists
		existingGPO, err := s.gpoRepo.FindGPOByNameInContainer(
			ctx, tx, containerUID, incomingGPOLink.LinksTo.Name,
		)
		if err != nil {
			return fmt.Errorf("error while finding GPO in container: %w", err)
		}

		if existingGPO == nil {
			log.Println("[AddGPOLink] GPO does not exist, creating")
			existingGPO, err = s.gpoRepo.CreateGPO(
				ctx, tx, incomingGPOLink.LinksTo, actor,
			)
			if err != nil {
				return fmt.Errorf("error while trying to create new gpo: %w", err)
			}
		}

		// 3. Link gpo to GPOLink
		err = s.gpoRepo.AddGPOToLink(
			ctx, tx, existingLinkEntity.UID, existingGPO.UID,
		)
		if err != nil {
			return fmt.Errorf("add GPO link: %w", err)
		}

		// 4. Create assertion as link between domain and gpolink
		assertion := &core.Assertion{
			Predicate:           core.PredicateHasGPOLink,
			Method:              core.MethodDirectAdd,
			Source:              actor,
			Confidence:          1.0,
			Status:              core.StatusValidated,
			Timestamp:           time.Now(),
			HasDiscoveredParent: true,
			MarkedAsHighValue:   false,
			Subject:             &utils2.UIDRef{UID: containerUID, Type: containerTyp},
			Object:              &utils2.UIDRef{UID: existingLinkEntity.UID, Type: "GPOLink"},
		}

		createdAssertion, err := s.assertionRepo.Create(ctx, tx, assertion)
		if err != nil {
			return fmt.Errorf("error while creating assertion during adding new gpo to domain: %w", err)
		}

		result = &core.EntityResult[*gpo.Link]{
			Entity:     existingLinkEntity,
			Assertions: []*core.Assertion{createdAssertion},
			Metadata: &core.ResultMetadata{
				Source:         actor,
				ScanTimestamp:  time.Now(),
				EntityCount:    1,
				AssertionCount: 1,
			},
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *GPOService) CreateAndLinkGPO(ctx context.Context, sourceObjectUID string, incomingGPOLink gpo.Link, actor string) (*gpo.Link, error) {
	panic("implement me")
}
