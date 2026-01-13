package active_directory

import (
	"RedPaths-server/internal/db"
	rperror "RedPaths-server/internal/error"
	"RedPaths-server/internal/repository/active_directory"
	"RedPaths-server/internal/repository/changes"
	"RedPaths-server/internal/utils"
	"RedPaths-server/pkg/model"
	rpad "RedPaths-server/pkg/model/active_directory"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/dgraph-io/dgo/v210/protos/api"
	"gorm.io/gorm"

	"github.com/dgraph-io/dgo/v210"
)

// ProjectService handles business logic for projects
type ProjectService struct {
	projectRepo         active_directory.ProjectRepository
	adRepo              active_directory.ActiveDirectoryRepository
	hostRepo            active_directory.HostRepository
	targetRepo          active_directory.TargetRepository
	userRepo            active_directory.UserRepository
	activeDirectoryRepo active_directory.ActiveDirectoryRepository

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
		adRepo:              active_directory.NewDgraphActiveDirectoryRepository(dgraphCon),
		hostRepo:            active_directory.NewDgraphHostRepository(dgraphCon),
		targetRepo:          active_directory.NewDgraphTargetRepository(dgraphCon),
		userRepo:            active_directory.NewDgraphUserRepository(dgraphCon),
		changeRepo:          changes.NewPostgresRedPathsChangesRepository(),
		activeDirectoryRepo: active_directory.NewDgraphActiveDirectoryRepository(dgraphCon),
	}, nil
}

func (s *ProjectService) AddActiveDirectory(ctx context.Context, projectUID string, incomingActiveDirectory *rpad.ActiveDirectory, actor string) (*rpad.ActiveDirectory, error) {
	var createdAD *rpad.ActiveDirectory
	log.Printf("[AddActiveDirectory] incomingAD.Name=%s, projectUID=%s", incomingActiveDirectory.ForestName, projectUID)
	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		existingActiveDirectory, err := s.activeDirectoryRepo.FindByForestNameInProject(ctx, tx, projectUID, incomingActiveDirectory.ForestName)
		if err != nil {
			return fmt.Errorf("error while checking if active directory exists: %v", err)
		}

		if existingActiveDirectory != nil {
			createdAD = existingActiveDirectory
			log.Printf("[AddActiveDirectory] active directory with forest name %s already exists in project with uid: %s", incomingActiveDirectory.ForestName, projectUID)
			return nil
		}
		createdAD, err = s.activeDirectoryRepo.Create(ctx, tx, incomingActiveDirectory, actor)

		err = s.projectRepo.AddActiveDirectory(ctx, tx, projectUID, createdAD.UID)
		if err != nil {
			return fmt.Errorf("error while creating new active directory: %v", err)
		}
		return nil
	})

	return createdAD, err
}

/*func (s *ProjectService) AddDomain(ctx context.Context, projectUID string, incomingDomain *active_directory2.Domain, actor string) (string, error) {
	var domainUID string

	log.Printf("[AddDomain] incomingDomain.Name=%s, projectUID=%s", incomingDomain.Name, projectUID)
	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		// check if incomingDomain already exists
		existingDomain, err := s.getDomainByNameIfExists(ctx, tx, projectUID, incomingDomain.Name)
		if err != nil {
			return fmt.Errorf("incomingDomain existence check failed: %w", err)
		}

		// Build changes
		if existingDomain != nil {
			change := utils.BuildChange(existingDomain, incomingDomain,
				utils.WithActor("scanner"),
				utils.WithReason("sync"),
			)

			if change != nil {
				err := s.changeRepo.Save(ctx, s.pdb, change)
				if err != nil {
					return fmt.Errorf("changeRepo save failed: %w", err)
				}
			}

			domainUID = existingDomain.UID
			return nil
		}
		return s.createAndLinkDomain(ctx, tx, incomingDomain, projectUID, &domainUID, actor)
	})

	return domainUID, err
}*/

/*func (s *ProjectService) getDomainByNameIfExists(ctx context.Context, tx *dgo.Txn, projectUID, domainName string) (*active_directory2.Domain, error) {
	domain, err := s.domainRepo.GetByNameInProject(ctx, tx, projectUID, domainName)
	if err != nil {
		if errors.Is(err, rperror.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get domain by name %s: %w", domainName, err)
	}
	log.Printf("domain with name %s already exists. Skipping!", domainName)
	return domain, nil
}

func (s *ProjectService) GetDomainInProjectByUID(ctx context.Context, projectUID, domainUID string) (*active_directory2.Domain, error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) (*active_directory2.Domain, error) {
		domain, err := s.domainRepo.GetByUIDInProject(ctx, tx, projectUID, domainUID)
		if err != nil {
			if errors.Is(err, rperror.ErrNotFound) {
				return nil, nil
			}
			return nil, fmt.Errorf("failed to get domain by uid %s: %w", domainUID, err)
		}
		return domain, nil
	})
}*/

/*func (s *ProjectService) createAndLinkDomain(
	ctx context.Context,
	tx *dgo.Txn,
	domain *active_directory2.Domain,
	projectUID string,
	domainUIDOut *string,
	actor string,
) error {
	var err error

	*domainUIDOut, err = s.domainRepo.CreateWithObject(ctx, tx, domain, actor)
	if err != nil {
		return fmt.Errorf("failed to create domain: %w", err)
	}

	if err := s.projectRepo.AddDomain(ctx, tx, projectUID, *domainUIDOut); err != nil {
		return fmt.Errorf("failed to link domain: %w", err)
	}

	if err := s.domainRepo.AddToProject(ctx, tx, *domainUIDOut, projectUID); err != nil {
		return fmt.Errorf("failed to reverse link domain to project: %w", err)
	}

	return nil
}
*/
/*func (s *ProjectService) GetProjectDomains(ctx context.Context, projectUID string) ([]*active_directory2.Domain, error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*active_directory2.Domain, error) {
		return s.domainRepo.GetAllByActiveDirectoryUID(ctx, tx, projectUID)
	})
}
*/
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
func (s *ProjectService) Create(ctx context.Context, name string) (string, error) {
	var projectUID string
	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		var err error
		projectUID, err = s.projectRepo.Create(ctx, tx, name)
		if err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}
		return nil
	})
	return projectUID, err
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

func (s *ProjectService) GetAllActiveDirectories(ctx context.Context, projectUID string) ([]*rpad.ActiveDirectory, error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*rpad.ActiveDirectory, error) {
		return s.adRepo.GetByProjectUID(ctx, tx, projectUID)
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

func (s *ProjectService) GetHostsByProject(ctx context.Context, projectUID string) ([]*model.Host, error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*model.Host, error) {
		return s.hostRepo.GetByProjectIncludingDomains(ctx, tx, projectUID)
	})
}

// TODO: Direct query
func (s *ProjectService) GetHostByProject(ctx context.Context, projectUID, hostUID string) (*model.Host, error) {
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
}

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
