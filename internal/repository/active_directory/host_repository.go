package active_directory

import (
	"RedPaths-server/internal/repository/dgraphutil"
	"RedPaths-server/pkg/model"
	"context"
	"fmt"
	"log"

	"github.com/dgraph-io/dgo/v210"
)

type HostRepository interface {
	Create(ctx context.Context, tx *dgo.Txn, host *model.Host) (string, error)
	SetDomainController(ctx context.Context, hostUID string, isDC bool) error
	AddService(ctx context.Context, tx *dgo.Txn, hostUID, serviceUID string) error
	AddToDomain(ctx context.Context, tx *dgo.Txn, hostUID string, domainUID string) error
	HostExistsByIP(ctx context.Context, tx *dgo.Txn, domainUID string, ip string) (bool, error)

	// HOSTS WITH UNDEFINED DOMAIN
	GetByProjectUID(ctx context.Context, tx *dgo.Txn, projectUID string) ([]*model.Host, error)
	// HOSTS WITH KNOWN/DISCOVERED DOMAIN
	GetByDomainUID(ctx context.Context, tx *dgo.Txn, domainUID string) ([]*model.Host, error)
}

type DraphHostRepository struct {
	DB *dgo.Dgraph
}

func (r *DraphHostRepository) HostExistsByIP(ctx context.Context, tx *dgo.Txn, domainUID string, ip string) (bool, error) {
	return dgraphutil.ExistsByFieldInDomain(ctx, tx, domainUID, "Host", "ip", ip)
}

func NewDgraphHostRepository(db *dgo.Dgraph) *DraphHostRepository {
	return &DraphHostRepository{DB: db}
}

func (r *DraphHostRepository) Create(ctx context.Context, tx *dgo.Txn, host *model.Host) (string, error) {

	hostToCreate := &model.Host{
		IP:   host.IP,
		Name: host.Name,
	}
	return dgraphutil.CreateEntity(ctx, tx, "Host", hostToCreate)
}

// WITH DOMAIN
func (r *DraphHostRepository) GetByDomainUID(ctx context.Context, tx *dgo.Txn, domainUID string) ([]*model.Host, error) {
	fields := []string{
		"uid",
		"ip",
		"name",
		"net_bios_name",
		"belongs_to_domain { uid }",
		"dgraph.type",
	}

	hosts, err := dgraphutil.GetEntitiesByRelation[*model.Host](
		ctx,
		tx,
		"host",
		"belongs_to_domain",
		domainUID,
		fields,
	)
	if err != nil {
		return nil,
			fmt.Errorf("failed to get entities by relation: %w", err)
	}

	log.Printf("Found %d hosts for domain %s\n", len(hosts), domainUID)
	return hosts, nil
}

// WITHOUT DOMAIN
func (r *DraphHostRepository) GetByProjectUID(ctx context.Context, tx *dgo.Txn, projectUID string) ([]*model.Host, error) {
	fields := []string{
		"uid",
		"name",
		"port",
		"dgraph.type",
	}

	services, err := dgraphutil.GetEntitiesByRelation[*model.Host](
		ctx,
		tx,
		"host",
		"has_host",
		projectUID,
		fields,
	)
	if err != nil {
		return nil, err
	}

	log.Printf("Found %d hosts without a domain for project %s\n", len(projectUID), "")
	return services, nil
}

// TODO
func (r *DraphHostRepository) SetDomainController(ctx context.Context, hostUID string, isDC bool) error {
	return nil
}

func (r *DraphHostRepository) AddService(ctx context.Context, tx *dgo.Txn, hostUID, serviceUID string) error {
	relationName := "has_service"
	err := dgraphutil.AddRelation(ctx, tx, hostUID, serviceUID, relationName)
	if err != nil {
		return fmt.Errorf("error while linking service %s to host %s with relation %s", serviceUID, hostUID, relationName)
	}
	return nil
}

/*func (r *DgraphDomainRepository) DomainExistsByName(ctx context.Context, tx *dgo.Txn, name string) (bool, error) {
	return dgraphutil.ExistsByField(ctx, tx, "Domain", "name", name)
}*/

func (r *DraphHostRepository) AddToDomain(
	ctx context.Context,
	tx *dgo.Txn,
	hostUID string,
	domainUID string,
) error {
	relationName := "belongs_to_domain"
	err := dgraphutil.AddRelation(ctx, tx, hostUID, domainUID, relationName)
	if err != nil {
		return fmt.Errorf("error while linking reverse relation from host %s to domain %s with relation %s", hostUID, domainUID, relationName)
	} else {
		log.Printf("linked host %s to domain %s with relation %s", hostUID, domainUID, relationName)
	}
	return nil
}
