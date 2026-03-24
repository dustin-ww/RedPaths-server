package active_directory

import (
	dgraphutil2 "RedPaths-server/internal/repository/util/dgraph"
	"RedPaths-server/pkg/model"
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/core/res"
	"context"
	"fmt"
	"log"

	"github.com/dgraph-io/dgo/v210"
)

type HostRepository interface {
	Get(ctx context.Context, tx *dgo.Txn, uid string) (*model.Host, error)
	Create(ctx context.Context, tx *dgo.Txn, host *model.Host, actor string) (*model.Host, error)
	SetDomainController(ctx context.Context, hostUID string, isDC bool) error
	AddService(ctx context.Context, tx *dgo.Txn, hostUID, serviceUID string) error
	AddToDomain(ctx context.Context, tx *dgo.Txn, hostUID string, domainUID string) error
	FindByIPInDomain(ctx context.Context, tx *dgo.Txn, domainUID string, ip string) (*model.Host, error)

	GetByProjectIncludingDomains(ctx context.Context, tx *dgo.Txn, projectUID string) ([]*res.EntityResult[*model.Host], error)

	// HOSTS WITH UNDEFINED DOMAIN
	GetAllByProjectUID(ctx context.Context, tx *dgo.Txn, projectUID string) ([]*model.Host, error)

	// HOSTS WITH KNOWN/DISCOVERED DOMAIN
	GetAllByDomainUID(ctx context.Context, tx *dgo.Txn, domainUID string) ([]*res.EntityResult[*model.Host], error)

	UpdateHost(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*model.Host, error)
	FindExisting(ctx context.Context, tx *dgo.Txn, projectUID string, host *model.Host) (*dgraphutil2.ExistenceResult[*model.Host], error)
}

var hostHierarchyHops = []dgraphutil2.HopConfig{
	{Predicate: core.PredicateHasActiveDirectory},
	{Predicate: core.PredicateHasDomain, ObjectType: "Domain"},
	{Predicate: core.PredicateHasHost, ObjectType: "Host"},
}

var hostFields = []string{
	"uid",
	"host.name",
	"host.ip",
	"host.hostname",
	"host.dns_host_name",
	"host.is_domain_controller",
	"host.distinguished_name",
	"host.operating_system",
	"host.operating_system_version",
	"created_at",
	"modified_at",
	"dgraph.type",
}

type DraphHostRepository struct {
	DB *dgo.Dgraph
}

func (r *DraphHostRepository) Get(ctx context.Context, tx *dgo.Txn, uid string) (*model.Host, error) {
	query := `
        query Host($uid: string) {
            host(func: uid($uid)) {
                uid
                host.name
				host.ip
				host.hostname
				host.dns_host_name
				host.description
				host.is_domain_controller
				host.distinguished_name
				host.operating_system
				host.operating_system_version
            }
        }
    `
	return dgraphutil2.GetEntityByUID[model.Host](ctx, tx, uid, "host", query)
}

func (r *DraphHostRepository) UpdateHost(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*model.Host, error) {
	return dgraphutil2.UpdateAndGet(ctx, tx, uid, actor, fields, r.Get)
}

func (r *DraphHostRepository) FindByIPInDomain(ctx context.Context, tx *dgo.Txn, domainUID string, ip string) (*model.Host, error) {
	return dgraphutil2.GetEntityByFieldInDomain[model.Host](ctx, tx, domainUID, "Host", "ip", ip)
}

func NewDgraphHostRepository(db *dgo.Dgraph) *DraphHostRepository {
	return &DraphHostRepository{DB: db}
}

func (r *DraphHostRepository) GetByProjectIncludingDomains(ctx context.Context,
	tx *dgo.Txn,
	projectUID string,
) ([]*res.EntityResult[*model.Host], error) {

	fields := []string{
		"uid",
		"host.name",
		"host.ip",
		"host.is_domain_controller",
		"host.distinguished_name",
		"host.dns_host_name",
		"host.operating_system",
		"host.operating_system_version",
		"host.last_logon_timestamp",
		"host.user_account_control",
		"created_at",
		"modified_at",
		"dgraph.type",
	}

	return dgraphutil2.GetEntitiesWithAssertionsNHop[*model.Host](
		ctx, tx, projectUID,
		hostHierarchyHops,
		fields, "getDomainHosts",
	)

}

func (r *DraphHostRepository) Create(ctx context.Context, tx *dgo.Txn, host *model.Host, actor string) (*model.Host, error) {

	hostToCreate := &model.Host{
		IP:                     host.IP,
		Name:                   host.Name,
		Hostname:               host.Hostname,
		DNSHostName:            host.DNSHostName,
		OperatingSystem:        host.OperatingSystem,
		OperatingSystemVersion: host.OperatingSystemVersion,
		IsDomainController:     host.IsDomainController,
		DistinguishedName:      host.DistinguishedName,
		Description:            host.Description,
	}
	dgraphutil2.InitCreateMetadata(&hostToCreate.RedPathsMetadata, actor)

	return dgraphutil2.CreateEntity(ctx, tx, "Host", hostToCreate)
}

// WITH DOMAIN
func (r *DraphHostRepository) GetAllByDomainUID(ctx context.Context, tx *dgo.Txn, domainUID string) ([]*res.EntityResult[*model.Host], error) {
	fields := []string{
		"uid",
		"host.name",
		"host.ip",
		"host.is_domain_controller",
		"host.distinguished_name",
		"host.dns_host_name",
		"host.operating_system",
		"host.operating_system_version",
		"host.last_logon_timestamp",
		"host.user_account_control",
		"created_at",
		"modified_at",
		"dgraph.type",
	}

	return dgraphutil2.GetEntitiesWithAssertions[*model.Host](
		ctx,
		tx,
		domainUID,
		core.PredicateHasHost,
		"Host",
		fields,
		"getDomainHosts",
	)
}

// WITHOUT DOMAIN
func (r *DraphHostRepository) GetAllByProjectUID(ctx context.Context, tx *dgo.Txn, projectUID string) ([]*model.Host, error) {
	fields := []string{
		"uid",
		"name",
		"port",
		"dgraph.type",
	}

	services, err := dgraphutil2.GetEntitiesByRelation[*model.Host](
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
	err := dgraphutil2.AddRelation(ctx, tx, hostUID, serviceUID, relationName)
	if err != nil {
		return fmt.Errorf("error while linking service %s to host %s with relation %s", serviceUID, hostUID, relationName)
	}
	return nil
}

/*func (r *DgraphDomainRepository) DomainExistsByName(ctx context.Context, tx *dgo.Txn, name string) (bool, error) {
	return dgraph.ExistsByField(ctx, tx, "Domain", "name", name)
}*/

func (r *DraphHostRepository) AddToDomain(
	ctx context.Context,
	tx *dgo.Txn,
	hostUID string,
	domainUID string,
) error {
	relationName := "belongs_to_domain"
	err := dgraphutil2.AddRelation(ctx, tx, hostUID, domainUID, relationName)
	if err != nil {
		return fmt.Errorf("error while linking reverse relation from host %s to domain %s with relation %s", hostUID, domainUID, relationName)
	}

	log.Printf("linked host %s to domain %s with relation %s", hostUID, domainUID, relationName)
	return nil
}

// FindExisting performs a two-phase existence check for a Host.
//
// Phase 1: Searches via the full AD hierarchy (Project → AD → Domain → Host).
// Phase 2: Falls back to direct project-level search for orphaned hosts.
//
// Uses OR mode so that a match on any unique field (ip OR dns_host_name) is
// enough to find a candidate. The caller (service) decides the confidence level.
func (r *DraphHostRepository) FindExisting(
	ctx context.Context,
	tx *dgo.Txn,
	projectUID string,
	host *model.Host,
) (*dgraphutil2.ExistenceResult[*model.Host], error) {

	filters := BuildHostFilter(host)

	return dgraphutil2.CheckEntityExists[*model.Host](
		ctx, tx,
		projectUID,
		"Host",
		filters,
		dgraphutil2.FilterModeOR, // OR: any unique field match returns candidates
		hostFields,
		hostHierarchyHops,
	)
}

// buildHostFilters constructs the unique field filters for a Host.
// Empty values are automatically skipped inside CheckEntityExists.
func BuildHostFilter(host *model.Host) []dgraphutil2.UniqueFieldFilter {
	return []dgraphutil2.UniqueFieldFilter{
		{Field: "host.ip", Value: host.IP},
		{Field: "host.dns_host_name", Value: host.DNSHostName},
		{Field: "host.distinguished_name", Value: host.DistinguishedName},
	}
}
