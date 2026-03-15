package active_directory

import (
	"RedPaths-server/internal/db"
	rperror "RedPaths-server/internal/error"
	"RedPaths-server/internal/repository/active_directory"
	"RedPaths-server/internal/repository/changes"
	"RedPaths-server/internal/repository/redpaths"
	"RedPaths-server/internal/utils"
	"RedPaths-server/pkg/model"
	rpad "RedPaths-server/pkg/model/active_directory"
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/core/res"
	utils2 "RedPaths-server/pkg/model/utils"
	"RedPaths-server/pkg/model/utils/assertion"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/dgo/v210/protos/api"
	"gorm.io/gorm"

	"github.com/dgraph-io/dgo/v210"
)

// ProjectService handles business logic for projects
type ProjectService struct {
	projectRepo active_directory.ProjectRepository

	hostService         HostService
	hostRepo            active_directory.HostRepository
	serviceRepo         active_directory.ServiceRepository
	domainRepo          active_directory.DomainRepository
	targetRepo          active_directory.TargetRepository
	userRepo            active_directory.UserRepository
	directoryNodeRepo   active_directory.DirectoryNodeRepository
	activeDirectoryRepo active_directory.ActiveDirectoryRepository
	assertionRepo       redpaths.AssertionRepository

	changeRepo changes.RedPathsChangeRepository
	db         *dgo.Dgraph
	pdb        *gorm.DB
}

// NewProjectService creates a new ProjectService instance
func NewProjectService(dgraphCon *dgo.Dgraph, postgresCon *gorm.DB) (*ProjectService, error) {

	return &ProjectService{
		db:                  dgraphCon,
		pdb:                 postgresCon,
		projectRepo:         active_directory.NewDgraphProjectRepository(dgraphCon),
		activeDirectoryRepo: active_directory.NewDgraphActiveDirectoryRepository(dgraphCon),
		domainRepo:          active_directory.NewDgraphDomainRepository(dgraphCon),
		hostRepo:            active_directory.NewDgraphHostRepository(dgraphCon),
		targetRepo:          active_directory.NewDgraphTargetRepository(dgraphCon),
		userRepo:            active_directory.NewDgraphUserRepository(dgraphCon),
		serviceRepo:         active_directory.NewDgraphServiceRepository(dgraphCon),
		directoryNodeRepo:   active_directory.NewDgraphDirectoryNodeRepository(dgraphCon),
		assertionRepo:       redpaths.NewDgraphAssertionRepository(dgraphCon),
	}, nil
}

func (s *ProjectService) AddActiveDirectory(
	ctx context.Context,
	assertionCtx assertion.Context,
	projectUID string, incomingActiveDirectory *rpad.ActiveDirectory, actor string,
) (*res.EntityResult[*rpad.ActiveDirectory], error) {
	var result *res.EntityResult[*rpad.ActiveDirectory]

	log.Printf("[AddActiveDirectory] forestName=%s, projectUID=%s, actor=%s",
		incomingActiveDirectory.ForestName, projectUID, actor)

	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		// Check if AD already exists
		existingAD, err := s.activeDirectoryRepo.FindByForestNameInProject(
			ctx, tx, projectUID, incomingActiveDirectory.ForestName,
		)
		if err != nil {
			return fmt.Errorf("checking existing AD: %w", err)
		}

		var ad *rpad.ActiveDirectory
		var assertions []*core.Assertion

		if existingAD != nil {
			// AD exists - reuse
			ad = existingAD
			log.Printf("[AddActiveDirectory] Reusing existing AD uid=%s", ad.UID)
		} else {
			// Create new AD
			ad, err = s.activeDirectoryRepo.Create(ctx, tx, incomingActiveDirectory, actor)
			if err != nil {
				return fmt.Errorf("creating AD: %w", err)
			}
			log.Printf("[AddActiveDirectory] Created new AD uid=%s", ad.UID)
		}

		// Create assertion (even if AD existed, link might be new)
		assertionBody := &core.Assertion{
			Predicate:           core.PredicateHasActiveDirectory,
			Method:              core.MethodDirectAdd,
			Source:              actor,
			Confidence:          assertionCtx.GetConfidence(),
			Status:              core.StatusValidated,
			Timestamp:           time.Now(),
			HasDiscoveredParent: true,
			MarkedAsHighValue:   assertionCtx.IsHighValue(),
			Subject:             &utils2.UIDRef{UID: projectUID, Type: "Project"},
			Object:              &utils2.UIDRef{UID: ad.UID, Type: "ActiveDirectory"},
		}

		createdAssertion, err := s.assertionRepo.Create(ctx, tx, assertionBody)
		if err != nil {
			return fmt.Errorf("creating assertion: %w", err)
		}
		assertions = append(assertions, createdAssertion)

		// Link assertion to project
		err = s.projectRepo.AddAssertion(ctx, tx, projectUID, createdAssertion.UID)
		if err != nil {
			return fmt.Errorf("linking assertion to project: %w", err)
		}

		// Build result
		result = &res.EntityResult[*rpad.ActiveDirectory]{
			Entity:     ad,
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
		return nil, fmt.Errorf("transaction failed: %w", err)
	}

	return result, nil
}

func (s *ProjectService) GetAllActiveDirectories(ctx context.Context, projectUID string) ([]*res.EntityResult[*rpad.ActiveDirectory], error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*res.EntityResult[*rpad.ActiveDirectory], error) {
		return s.activeDirectoryRepo.GetByProjectUID(ctx, tx, projectUID)
	})
}

func (s *ProjectService) GetAllDirectoryNodes(ctx context.Context, projectUID string) ([]*res.EntityResult[*rpad.DirectoryNode], error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*res.EntityResult[*rpad.DirectoryNode], error) {
		return s.directoryNodeRepo.GetByProjectUID(ctx, tx, projectUID)
	})
}

func (s *ProjectService) GetAllDomains(ctx context.Context, projectUID string) ([]*res.EntityResult[*rpad.Domain], error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*res.EntityResult[*rpad.Domain], error) {
		return s.domainRepo.GetAllByProjectUID(ctx, tx, projectUID)
	})
}

// CreateTarget creates a new target and links it to a project.
// TODO implement cidr
func (s *ProjectService) CreateTarget(ctx context.Context, projectUID, ip, note string, cidr int) (string, error) {
	log.Println("CREATE TARGET WITH: " + projectUID + ip + note)

	var targetUID string

	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		uid, err := s.createTarget(ctx, tx, cidr, ip, note)
		if err != nil {
			return err
		}
		targetUID = uid

		if err := s.linkTargetToProject(ctx, tx, projectUID, targetUID); err != nil {
			return err
		}

		return nil
	})

	return targetUID, err
}

func (s *ProjectService) createTarget(ctx context.Context, tx *dgo.Txn, cidr int, ip, note string) (string, error) {
	log.Println(cidr)
	target := map[string]interface{}{
		"uid":         "_:target",
		"ip":          ip,
		"cidr":        cidr,
		"note":        note,
		"dgraph.type": "Target",
	}

	targetJSON, err := json.Marshal(target)
	if err != nil {
		return "", fmt.Errorf("failed to marshal target: %w", err)
	}

	mu := &api.Mutation{SetJson: targetJSON}
	assigned, err := tx.Mutate(ctx, mu)
	if err != nil {
		return "", fmt.Errorf("mutation failed: %w", err)
	}

	log.Println("Mutation succeeded")
	return assigned.Uids["target"], nil
}

func (s *ProjectService) linkTargetToProject(ctx context.Context, tx *dgo.Txn, projectUID, targetUID string) error {
	nquad := fmt.Sprintf("<%s> <has_target> <%s> .", projectUID, targetUID)

	mu := &api.Mutation{
		SetNquads: []byte(nquad),
	}
	_, err := tx.Mutate(ctx, mu)
	return err
}

// Create creates a new project.
func (s *ProjectService) Create(ctx context.Context, incomingProject *model.Project) (*model.Project, error) {
	var createdProject *model.Project
	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		var err error
		createdProject, err = s.projectRepo.Create(ctx, tx, incomingProject)
		if err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}
		return nil
	})
	return createdProject, err
}

// GetOverviewForAll retrieves overview information for all projects.
func (s *ProjectService) GetOverviewForAll(ctx context.Context) ([]*model.Project, error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*model.Project, error) {
		return s.projectRepo.GetAll(ctx, tx)
	})
}

// Get retrieves a project by its UID.
func (s *ProjectService) Get(ctx context.Context, projectUID string) (*model.Project, error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) (*model.Project, error) {
		return s.projectRepo.Get(ctx, tx, projectUID)
	})
}

// UpdateProject updates specified fields of a project.
func (s *ProjectService) UpdateProject(ctx context.Context, uid, actor string, fields map[string]interface{}) (*model.Project, error) {
	if uid == "" {
		return nil, utils.ErrUIDRequired
	}

	allowed := map[string]bool{"name": true, "description": true}
	protected := map[string]bool{"uid": true, "created_at": true, "updated_at": true, "type": true}

	for field := range fields {
		if protected[field] {
			return nil, fmt.Errorf("%w: %s", utils.ErrFieldProtected, field)
		}
		if !allowed[field] {
			return nil, fmt.Errorf("%w: %s", utils.ErrFieldNotAllowed, field)
		}
	}

	return db.ExecuteInTransactionWithResult[*model.Project](ctx, s.db, func(tx *dgo.Txn) (*model.Project, error) {
		return s.projectRepo.UpdateProject(ctx, tx, uid, actor, fields)
	})
}

// GetTargets retrieves all targets associated with a project.
func (s *ProjectService) GetTargets(ctx context.Context, projectUID string) ([]*model.Target, error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*model.Target, error) {
		targets, _ := s.projectRepo.GetTargets(ctx, tx, projectUID)
		log.Println(len(targets))
		return s.projectRepo.GetTargets(ctx, tx, projectUID)
	})
}

func (s *ProjectService) DeleteProject(ctx context.Context, projectUID string) error {
	return db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		return s.projectRepo.Delete(ctx, tx, projectUID)
	})
}

func (s *ProjectService) GetHostsByProject(ctx context.Context, projectUID string) ([]*res.EntityResult[*model.Host], error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*res.EntityResult[*model.Host], error) {
		return s.hostRepo.GetByProjectIncludingDomains(ctx, tx, projectUID)
	})
}

func (s *ProjectService) GetServicesByProject(ctx context.Context, projectUID string) ([]*res.EntityResult[*model.Service], error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*res.EntityResult[*model.Service], error) {
		return s.serviceRepo.GetByProjectUID(ctx, tx, projectUID)
	})
}

// TODO: Direct query
/*func (s *ProjectService) GetHostByProject(ctx context.Context, projectUID, hostUID string) (*model.Host, error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) (*model.Host, error) {
		hosts, err := s.hostRepo.GetByProjectIncludingDomains(ctx, tx, projectUID)

		if err != nil {
			return nil, fmt.Errorf("failed to get host by project: %w", err)
		}

		for _, host := range hosts {
			if host.UID == hostUID {
				return host, nil
			}
		}
		return nil, rperror.ErrNotFound
	})
}*/

func (s *ProjectService) GetAllUserInProject(ctx context.Context, projectUID string) ([]*rpad.User, error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*rpad.User, error) {
		return s.userRepo.GetByProjectIncludingDomains(ctx, tx, projectUID)

	})
}

func (s *ProjectService) GetUserInProject(ctx context.Context, projectUID, userUID string) (*rpad.User, error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) (*rpad.User, error) {
		users, err := s.userRepo.GetByProjectIncludingDomains(ctx, tx, projectUID)

		if err != nil {
			return nil, fmt.Errorf("failed to get user by project: %w", err)
		}

		for _, user := range users {
			if user.UID == userUID {
				return user, nil
			}
		}
		log.Println("Not user found for given user UID")
		return nil, rperror.ErrNotFound
	})
}

func (s *ProjectService) GetAllActiveDirectoriesInProject(ctx context.Context, projectUID, userUID string) (*rpad.User, error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) (*rpad.User, error) {
		users, err := s.userRepo.GetByProjectIncludingDomains(ctx, tx, projectUID)

		if err != nil {
			return nil, fmt.Errorf("failed to get user by project: %w", err)
		}

		for _, user := range users {
			if user.UID == userUID {
				return user, nil
			}
		}
		log.Println("Not user found for given user UID")
		return nil, rperror.ErrNotFound
	})
}
