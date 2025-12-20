package rpsdk

import (
	"RedPaths-server/pkg/service/active_directory"

	"github.com/dgraph-io/dgo/v210"
	"gorm.io/gorm"

	"log"
)

type Services struct {
	ProjectService active_directory.ProjectService
	DomainService  active_directory.DomainService
	HostService    active_directory.HostService
}

func NewServicesContainer(dgraphCon *dgo.Dgraph, postgresCon *gorm.DB) *Services {
	projectService, err := active_directory.NewProjectService(dgraphCon, postgresCon)
	if err != nil {
		log.Fatalf("Failed to initialize ProjectService for redpaths sdk: %v", err)
	}

	domainService, err := active_directory.NewDomainService(dgraphCon)
	if err != nil {
		log.Fatalf("Failed to initialize DomainService for redpaths sdk: %v", err)
	}

	hostService, err := active_directory.NewHostService(dgraphCon)
	if err != nil {
		log.Fatalf("Failed to initialize HostService for redpaths sdk: %v", err)
	}

	return &Services{
		ProjectService: *projectService,
		DomainService:  *domainService,
		HostService:    *hostService,
	}
}
