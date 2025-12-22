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

func (s *DomainService) AddHost(ctx context.Context, domainUID string, host *model.Host, actor string) (string, error) {

	log.Println("[ADD HOST]")

	var hostUID string

	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {

		// check if host already exists in domain
		existingHost, err := s.hostRepo.FindByIPInDomain(ctx, tx, domainUID, host.IP)
		if err != nil {
			return fmt.Errorf("host existence check failed: %w", err)
		}

		if existingHost != nil {
			log.Printf("[ADD HOST]: Host already exists with ip %s in domain %s\n", host.IP, domainUID)
			hostUID = existingHost.UID
			return nil
		}

		// otherwise create new host in domain
		hostUID, err = s.hostRepo.Create(ctx, tx, host, actor)
		if err != nil {
			return fmt.Errorf("failed to create host: %w", err)
		}

		log.Printf("Created host with ip %s, name %s receiving uid %s\n", host.IP, host.Name, hostUID)

		// connect domain with host
		if err := s.domainRepo.AddHost(ctx, tx, domainUID, hostUID); err != nil {
			return fmt.Errorf("failed to link host to domain: %w", err)
		}

		// reverse link from host to domain
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
