package active_directory

import (
	"RedPaths-server/internal/db"
	"RedPaths-server/internal/repository/active_directory"
	"RedPaths-server/internal/utils"
	rpad "RedPaths-server/pkg/model/active_directory"
	"context"
	"fmt"
	"log"

	"github.com/dgraph-io/dgo/v210"
)

type DirectoryNodeService struct {
	domainRepo          active_directory.DomainRepository
	hostRepo            active_directory.HostRepository
	userRepo            active_directory.UserRepository
	activeDirectoryRepo active_directory.ActiveDirectoryRepository
	directoryNodeRepo   active_directory.DirectoryNodeRepository
	db                  *dgo.Dgraph
}

func NewDirectoryNodeService(dgraphCon *dgo.Dgraph) (*DirectoryNodeService, error) {
	domainRepo := active_directory.NewDgraphDomainRepository(dgraphCon)
	hostRepo := active_directory.NewDgraphHostRepository(dgraphCon)
	activeDirectoryRepo := active_directory.NewDgraphActiveDirectoryRepository(dgraphCon)
	userRepo := active_directory.NewDgraphUserRepository(dgraphCon)
	directoryNodeRepo := active_directory.NewDgraphDirectoryNodeRepository(dgraphCon)

	return &DirectoryNodeService{
		db:                  dgraphCon,
		domainRepo:          domainRepo,
		hostRepo:            hostRepo,
		activeDirectoryRepo: activeDirectoryRepo,
		userRepo:            userRepo,
		directoryNodeRepo:   directoryNodeRepo,
	}, nil
}

func (s *DirectoryNodeService) AddSecurityPrincipal(ctx context.Context, directoryNodeUID string, incomingSecurityPrincipal rpad.SecurityPrincipal, actor string) (*rpad.SecurityPrincipal, error) {

	log.Println("[ADD Security Principal to Directory Node]")

	var createdSecurityPrincipal rpad.SecurityPrincipal

	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		var err error

		switch p := incomingSecurityPrincipal.(type) {

		case *rpad.User:

			createdSecurityPrincipal, err = s.userRepo.Create(ctx, tx, p, "UserInput")
			if err != nil {
				return fmt.Errorf("error while creating user: %v", err)
			}

			err := s.directoryNodeRepo.AddSecurityPrincipal(ctx, tx, directoryNodeUID, createdSecurityPrincipal.GetUID())
			if err != nil {
				return fmt.Errorf("error while adding user to directory node: %v", err)
			}

		/*case *active_directory.Group:
			return s.addGroup(ctx, tx, directoryNodeUID, p, actor)

		case *active_directory.Computer:
			return s.addComputer(ctx, tx, directoryNodeUID, p, actor)

		case *active_directory.ServiceAccount:
			return s.addServiceAccount(ctx, tx, directoryNodeUID, p, actor)*/

		default:
			return fmt.Errorf("unsupported security principal type: %T", p.PrincipalType())
		}

		// check if host already exists in domain
		/*ex
		existingHost, err := s.hostRepo.FindByIPInDomain(ctx, tx, domainUID, host.IP)
		if err != nil {
			return fmt.Errorf("host existence check failed: %w", err)
		}

		if existingHost != nil {
			log.Printf("[ADD HOST]: Host already exists with ip %s in domain %s\n", host.IP, domainUID)
			hostUID = existingHost.UID
			return nil
		}
		*/

		return nil
	})

	return &createdSecurityPrincipal, err

}

func (s *DirectoryNodeService) GetDirectoryNodeSecurityPrincipals(ctx context.Context, directoryNodeUID string) ([]*rpad.SecurityPrincipal, error) {
	panic("implement me")
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
		return s.directoryNodeRepo.UpdateDirectoryNode(ctx, tx, uid, actor, fields)
	})
}
