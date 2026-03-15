package active_directory

import (
	"RedPaths-server/internal/repository/active_directory"
	"RedPaths-server/internal/repository/redpaths"
	"RedPaths-server/pkg/model/active_directory/gpo"
	"context"

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

func (s *GPOService) CreateAndLinkGPO(ctx context.Context, sourceObjectUID string, incomingGPOLink gpo.Link, actor string) (*gpo.Link, error) {
	panic("implement me")
}
