package active_directory

import (
	"RedPaths-server/internal/db"
	"RedPaths-server/internal/repository/active_directory"
	"RedPaths-server/pkg/model"
	"context"
	"fmt"
	"log"

	"github.com/dgraph-io/dgo/v210"
)

type DomainService struct {
	domainRepo active_directory.DomainRepository
	hostRepo   active_directory.HostRepository
	db         *dgo.Dgraph
}

func NewDomainService(dgraphCon *dgo.Dgraph) (*DomainService, error) {
	domainRepo := active_directory.NewDgraphDomainRepository(dgraphCon)
	hostRepo := active_directory.NewDgraphHostRepository(dgraphCon)

	return &DomainService{
		db:         dgraphCon,
		domainRepo: domainRepo,
		hostRepo:   hostRepo,
	}, nil
}

func (s *DomainService) AddHost(ctx context.Context, domainUID string, projectUID string, host *model.Host) (string, error) {
	var hostUID string
	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {

		hostsExists, err := s.hostRepo.HostExistsByIP(ctx, tx, projectUID, host.IP)
		if err != nil {
			return fmt.Errorf("check if hosts already exists in a domain failed with: %w", err)
		}

		if hostsExists {
			log.Printf("host with name %s and ip %s walready exists in a domain", host.Name, host.IP)
			return nil
		}

		hostUID, err = s.hostRepo.Create(ctx, tx, host)
		if err != nil {
			return fmt.Errorf("failed to create host: %w", err)
		}
		log.Printf("Created host with ip %s, name %s receiving uid %s\n", host.IP, host.Name, host.UID)

		if err := s.domainRepo.AddHost(ctx, tx, domainUID, hostUID); err != nil {
			return fmt.Errorf("failed to link host to domain: %w", err)
		}

		if err := s.hostRepo.AddToDomain(ctx, tx, hostUID, domainUID); err != nil {
			return fmt.Errorf("failed to reverse link domain to host: %w", err)
		}
		return nil
	})
	return hostUID, err
}

func (s *DomainService) GetDomainHosts(ctx context.Context, domainUID string) ([]*model.Host, error) {
	return db.ExecuteRead(ctx, s.db, func(tx *dgo.Txn) ([]*model.Host, error) {
		return s.hostRepo.GetByDomainUID(ctx, tx, domainUID)
	})
}

/*func (s *ProjectService) AddDomain(ctx context.Context, projectUID string, domain *model.Domain) (string, error) {
	var domainUID string

	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		// check if domain already exists
		existingDomain, err := s.getDomainIfExists(ctx, tx, domain.Name)
		if err != nil {
			return fmt.Errorf("domain existence check failed: %w", err)
		}

		if existingDomain != nil {
			domainUID = existingDomain.UID
			return nil
		}
		return s.createAndLinkDomain(ctx, tx, domain, projectUID, &domainUID)
	})

	return domainUID, err
}
*/
