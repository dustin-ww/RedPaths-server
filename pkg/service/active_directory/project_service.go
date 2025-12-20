package active_directory

import (
	"RedPaths-server/internal/db"
	"RedPaths-server/internal/repository/active_directory"
	"RedPaths-server/internal/repository/changes"
	"RedPaths-server/internal/utils"
	"RedPaths-server/pkg/model"
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
	projectRepo active_directory.ProjectRepository
	domainRepo  active_directory.DomainRepository
	hostRepo    active_directory.HostRepository
	targetRepo  active_directory.TargetRepository

	changeRepo changes.RedPathsChangeRepository
	db         *dgo.Dgraph
	pdb        *gorm.DB
}

// NewProjectService creates a new ProjectService instance
func NewProjectService(dgraphCon *dgo.Dgraph, postgresCon *gorm.DB) (*ProjectService, error) {

	return &ProjectService{
		db:          dgraphCon,
		pdb:         postgresCon,
		projectRepo: active_directory.NewDgraphProjectRepository(dgraphCon),
		domainRepo:  active_directory.NewDgraphDomainRepository(dgraphCon),
		hostRepo:    active_directory.NewDgraphHostRepository(dgraphCon),
		targetRepo:  active_directory.NewDgraphTargetRepository(dgraphCon),
		changeRepo:  changes.NewPostgresRedPathsChangesRepository(),
	}, nil
}

// AddDomainWithHosts adds a domain with associated hosts to a project.
//func (s *ProjectService) AddDomainWithHosts(ctx context.Context, projectUID, domainName string, hosts []string) error {
//	return db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
//		domainUID, err := s.domainRepo.Create(ctx, domainName)
//		if err != nil {
//			return fmt.Errorf("failed to create domain: %w", err)
//		}
//
//		if err := s.projectRepo.AddDomain(ctx, tx, projectUID, domainUID); err != nil {
//			return fmt.Errorf("failed to link domain: %w", err)
//		}
//
//		for _, ip := range hosts {
//			hostUID, err := s.hostRepo.Create(ctx, ip)
//			if err != nil {
//				return fmt.Errorf("failed to create host %s: %w", ip, err)
//			}
//			if err := s.domainRepo.AddHost(ctx, domainUID, hostUID); err != nil {
//				return fmt.Errorf("failed to link host: %w", err)
//			}
//		}
//		return nil
//	})
//}

func (s *ProjectService) AddDomain(ctx context.Context, projectUID string, incomingDomain *model.Domain) (string, error) {
	var domainUID string

	log.Printf("[AddDomain] incomingDomain.Name=%s, projectUID=%s", incomingDomain.Name, projectUID)
	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		// check if incomingDomain already exists
		existingDomain, err := s.getDomainIfExists(ctx, tx, projectUID, incomingDomain.Name)
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
		return s.createAndLinkDomain(ctx, tx, incomingDomain, projectUID, &domainUID)
	})

	return domainUID, err
}

func (s *ProjectService) getDomainIfExists(ctx context.Context, tx *dgo.Txn, projectUID, domainName string) (*model.Domain, error) {
	isExisting, err := s.domainRepo.DomainExistsByName(ctx, tx, projectUID, domainName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if domain exists: %w", err)
	}

	if !isExisting {
		return nil, nil
	}

	log.Println("domain with name " + domainName + " already exists. Skipping!")
	domain, err := s.domainRepo.GetByName(ctx, tx, projectUID, domainName)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain with name %s: %w", domainName, err)
	}

	return domain, nil
}

func (s *ProjectService) createAndLinkDomain(
	ctx context.Context,
	tx *dgo.Txn,
	domain *model.Domain,
	projectUID string,
	domainUIDOut *string,
) error {
	var err error

	*domainUIDOut, err = s.domainRepo.CreateWithObject(ctx, tx, domain)
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

func (s *ProjectService) GetProjectDomains(ctx context.Context, projectUID string) ([]*model.Domain, error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*model.Domain, error) {
		return s.domainRepo.GetByProjectUID(ctx, tx, projectUID)
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

// UpdateFields updates specified fields of a project.
func (s *ProjectService) UpdateFields(ctx context.Context, uid string, fields map[string]interface{}) error {
	if uid == "" {
		return utils.ErrUIDRequired
	}

	allowed := map[string]bool{"name": true, "description": true}
	protected := map[string]bool{"uid": true, "created_at": true, "updated_at": true, "type": true}

	for field := range fields {
		if protected[field] {
			return fmt.Errorf("%w: %s", utils.ErrFieldProtected, field)
		}
		if !allowed[field] {
			return fmt.Errorf("%w: %s", utils.ErrFieldNotAllowed, field)
		}
	}

	return db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		return s.projectRepo.UpdateFields(ctx, tx, uid, fields)
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
		var hosts []*model.Host
		domains, err := s.domainRepo.GetByProjectUID(ctx, tx, projectUID)
		if err != nil {
			return nil, err
		}

		// collect hosts with domain
		for _, domain := range domains {
			domainHosts, err := s.hostRepo.GetByDomainUID(ctx, tx, domain.UID)
			if err != nil {
				return nil, fmt.Errorf("failed to get domain hosts in building a list with all hosts in project: %w", err)
			}
			hosts = append(hosts, domainHosts...)
		}
		// collect hosts without domain
		withoutDomainHosts, err := s.hostRepo.GetByProjectUID(ctx, tx, projectUID)

		if err != nil {
			return nil, fmt.Errorf("failed to get hosts by project: %w", err)
		}
		hosts = append(hosts, withoutDomainHosts...)

		return hosts, nil
	})
}
