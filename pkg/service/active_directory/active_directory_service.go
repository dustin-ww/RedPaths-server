package active_directory

import (
	"RedPaths-server/internal/db"
	"RedPaths-server/internal/repository/active_directory"
	"RedPaths-server/internal/utils"
	rp_ad_model "RedPaths-server/pkg/model/active_directory"
	"context"
	"fmt"
	"log"

	"github.com/dgraph-io/dgo/v210"
)

type ActiveDirectoryService struct {
	domainRepo          active_directory.DomainRepository
	hostRepo            active_directory.HostRepository
	activeDirectoryRepo active_directory.ActiveDirectoryRepository
	db                  *dgo.Dgraph
}

func NewActiveDirectoryService(dgraphCon *dgo.Dgraph) (*ActiveDirectoryService, error) {
	domainRepo := active_directory.NewDgraphDomainRepository(dgraphCon)
	hostRepo := active_directory.NewDgraphHostRepository(dgraphCon)
	activeDirectoryRepo := active_directory.NewDgraphActiveDirectoryRepository(dgraphCon)

	return &ActiveDirectoryService{
		db:                  dgraphCon,
		domainRepo:          domainRepo,
		hostRepo:            hostRepo,
		activeDirectoryRepo: activeDirectoryRepo,
	}, nil
}

func (s *ActiveDirectoryService) AddDomain(ctx context.Context, activeDirectoryUID string, incomingDomain *rp_ad_model.Domain, actor string) (*rp_ad_model.Domain, error) {

	log.Println("[ADD DOMAIN to Active Directory]")

	var domain *rp_ad_model.Domain

	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {

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

		existingDomain, err := s.domainRepo.FindByNameInActiveDirectory(ctx, tx, activeDirectoryUID, incomingDomain.Name)
		if err != nil {
			return fmt.Errorf("error while checking if active directory exists: %v", err)
		}

		if existingDomain != nil {
			log.Println("[Active Directory already exists]")
			domain = existingDomain
		}

		domain, err = s.domainRepo.Create(ctx, tx, incomingDomain, actor)
		if err != nil {
			return fmt.Errorf("failed to create domain: %w", err)
		}

		log.Printf("Created domain with name %s receiving uid %s\n", domain.Name, domain.UID)

		// connect domain with host
		if err := s.activeDirectoryRepo.AddDomain(ctx, tx, activeDirectoryUID, domain.UID); err != nil {
			return fmt.Errorf("failed to link domain to active directory forest: %w", err)
		}

		/*// reverse link from host to domain
		if err := s.domainRepo.AddToProject().AddToDomain(ctx, tx, hostUID, domainUID); err != nil {
			return fmt.Errorf("failed to reverse link domain to host: %w", err)
		}*/

		return nil
	})

	return domain, err
}

func (s *ActiveDirectoryService) UpdateActiveDirectory(ctx context.Context, uid, actor string, fields map[string]interface{}) (*rp_ad_model.ActiveDirectory, error) {
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

	return db.ExecuteInTransactionWithResult[*rp_ad_model.ActiveDirectory](ctx, s.db, func(tx *dgo.Txn) (*rp_ad_model.ActiveDirectory, error) {
		return s.activeDirectoryRepo.UpdateActiveDirectory(ctx, tx, uid, actor, fields)
	})
}

func (s *ActiveDirectoryService) Get(ctx context.Context, activeDirectoryUID string) (*rp_ad_model.ActiveDirectory, error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) (*rp_ad_model.ActiveDirectory, error) {
		return s.activeDirectoryRepo.Get(ctx, tx, activeDirectoryUID)
	})
}

func (s *ActiveDirectoryService) GetDomainsByActiveDirectory(ctx context.Context, activeDirectoryUID string) ([]*rp_ad_model.Domain, error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*rp_ad_model.Domain, error) {
		return s.domainRepo.GetAllByActiveDirectoryUID(ctx, tx, activeDirectoryUID)
	})
}
