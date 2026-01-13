package active_directory

import (
	"RedPaths-server/internal/repository/dgraphutil"
	"RedPaths-server/pkg/model"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/dgraph-io/dgo/v210"
)

type HostRepository interface {
	Get(ctx context.Context, tx *dgo.Txn, uid string) (*model.Host, error)
	Create(ctx context.Context, tx *dgo.Txn, host *model.Host, actor string) (string, error)
	SetDomainController(ctx context.Context, hostUID string, isDC bool) error
	AddService(ctx context.Context, tx *dgo.Txn, hostUID, serviceUID string) error
	AddToDomain(ctx context.Context, tx *dgo.Txn, hostUID string, domainUID string) error
	FindByIPInDomain(ctx context.Context, tx *dgo.Txn, domainUID string, ip string) (*model.Host, error)

	GetByProjectIncludingDomains(ctx context.Context, tx *dgo.Txn, projectUID string) ([]*model.Host, error)

	// HOSTS WITH UNDEFINED DOMAIN
	GetAllByProjectUID(ctx context.Context, tx *dgo.Txn, projectUID string) ([]*model.Host, error)

	// HOSTS WITH KNOWN/DISCOVERED DOMAIN
	GetAllByDomainUID(ctx context.Context, tx *dgo.Txn, domainUID string) ([]*model.Host, error)

	UpdateHost(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*model.Host, error)
}

type DraphHostRepository struct {
	DB *dgo.Dgraph
}

func (r *DraphHostRepository) Get(ctx context.Context, tx *dgo.Txn, uid string) (*model.Host, error) {
	panic("implement me")
}

func (r *DraphHostRepository) UpdateHost(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*model.Host, error) {
	return dgraphutil.UpdateAndGet(ctx, tx, uid, actor, fields, r.Get)
}

func (r *DraphHostRepository) FindByIPInDomain(ctx context.Context, tx *dgo.Txn, domainUID string, ip string) (*model.Host, error) {
	return dgraphutil.GetEntityByFieldInDomain[model.Host](ctx, tx, domainUID, "Host", "ip", ip)
}

func NewDgraphHostRepository(db *dgo.Dgraph) *DraphHostRepository {
	return &DraphHostRepository{DB: db}
}

func (r *DraphHostRepository) GetByProjectIncludingDomains(ctx context.Context,
	tx *dgo.Txn,
	projectUID string,
) ([]*model.Host, error) {

	if tx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}

	query := `
	query HostsByProject($pid: string) {
		project(func: uid($pid)) {
			has_host {
				uid
				ip
				name
				net_bios_name
				belongs_to_domain { uid }
				dgraph.type
			}
			has_domain {
				has_host {
					uid
					ip
					name
					net_bios_name
					belongs_to_domain { uid }
					dgraph.type
				}
			}
		}
	}`

	resp, err := tx.QueryWithVars(ctx, query, map[string]string{
		"$pid": projectUID,
	})
	if err != nil {
		return nil, fmt.Errorf("query hosts by project failed: %w", err)
	}

	// Response Mapping
	type domainWrapper struct {
		HasHost []*model.Host `json:"has_host"`
	}

	type projectWrapper struct {
		HasHost   []*model.Host   `json:"has_host"`
		HasDomain []domainWrapper `json:"has_domain"`
	}

	var result struct {
		Project []projectWrapper `json:"project"`
	}

	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("unmarshal project hosts failed: %w", err)
	}

	// --- Deduplicate Hosts by UID ---
	hostMap := make(map[string]*model.Host)

	for _, project := range result.Project {

		// hosts directly linked to project
		for _, host := range project.HasHost {
			if host.UID != "" {
				hostMap[host.UID] = host
			}
		}

		// hosts linked via domains
		for _, domain := range project.HasDomain {
			for _, host := range domain.HasHost {
				if host.UID != "" {
					hostMap[host.UID] = host
				}
			}
		}
	}

	hosts := make([]*model.Host, 0, len(hostMap))
	for _, host := range hostMap {
		hosts = append(hosts, host)
	}

	return hosts, nil
}

func (r *DraphHostRepository) Create(ctx context.Context, tx *dgo.Txn, host *model.Host, actor string) (string, error) {

	hostToCreate := &model.Host{
		IP:   host.IP,
		Name: host.Name,
	}
	return dgraphutil.OldCreateEntity(ctx, tx, "Host", hostToCreate)
}

// WITH DOMAIN
func (r *DraphHostRepository) GetAllByDomainUID(ctx context.Context, tx *dgo.Txn, domainUID string) ([]*model.Host, error) {
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
func (r *DraphHostRepository) GetAllByProjectUID(ctx context.Context, tx *dgo.Txn, projectUID string) ([]*model.Host, error) {
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
	}

	log.Printf("linked host %s to domain %s with relation %s", hostUID, domainUID, relationName)
	return nil
}
