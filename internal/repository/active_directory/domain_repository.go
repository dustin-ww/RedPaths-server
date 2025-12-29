package active_directory

import (
	rperror "RedPaths-server/internal/error"
	"RedPaths-server/internal/repository/dgraphutil"
	"RedPaths-server/pkg/model/active_directory"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
)

type DomainRepository interface {
	//CRUD
	Create(ctx context.Context, tx *dgo.Txn, name string, actor string) (string, error) // Returns UID
	Get(ctx context.Context, tx *dgo.Txn, uid string) (*active_directory.Domain, error)
	GetByNameInProject(ctx context.Context, tx *dgo.Txn, projectUID, name string) (*active_directory.Domain, error)
	GetByUIDInProject(ctx context.Context, tx *dgo.Txn, projectUID, domainUID string) (*active_directory.Domain, error)

	UpdateDomain(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*active_directory.Domain, error)
	CreateWithObject(ctx context.Context, tx *dgo.Txn, model *active_directory.Domain, actor string) (string, error)

	//Relations
	AddHost(ctx context.Context, tx *dgo.Txn, domainUID, hostUID string) error
	AddUser(ctx context.Context, tx *dgo.Txn, domainUID, userUID string) error
	AddToProject(ctx context.Context, tx *dgo.Txn, domainUID string, projectUID string) error

	// Checker
	DomainExistsByName(ctx context.Context, tx *dgo.Txn, projectUID, name string) (bool, error)

	// Reverse Relations
	GetAllByProjectUID(ctx context.Context, tx *dgo.Txn, projectUID string) ([]*active_directory.Domain, error)
}

type DgraphDomainRepository struct {
	DB *dgo.Dgraph
}

func (r *DgraphDomainRepository) CreateWithObject(ctx context.Context, tx *dgo.Txn, domain *active_directory.Domain, actor string) (string, error) {
	domain.DiscoveredAt = time.Now().UTC()
	domain.LastSeenAt = time.Now().UTC()
	domain.DiscoveredBy = actor
	domain.LastSeenBy = actor
	domain.DType = []string{"Domain"}
	domain.UID = "_:blank-0"

	jsonData, err := json.Marshal(domain)
	if err != nil {
		return "", fmt.Errorf("marshal error: %w", err)
	}

	mu := &api.Mutation{
		SetJson: jsonData,
	}

	assigned, err := tx.Mutate(ctx, mu)
	if err != nil {
		return "", fmt.Errorf("mutation error: %w", err)
	}

	return assigned.Uids["blank-0"], nil
}

func (r *DgraphDomainRepository) GetByNameInProject(ctx context.Context, tx *dgo.Txn, projectUID, name string) (*active_directory.Domain, error) {
	fields := []string{"uid", "name"}

	domains, err := dgraphutil.GetEntityByFieldInProject[*active_directory.Domain](
		ctx,
		tx,
		projectUID,
		"Domain",
		"name",
		name,
		fields,
	)

	if err != nil {
		return nil, fmt.Errorf("get entity by name failed with message %w", err)
	}

	if len(domains) == 0 {
		return nil, rperror.ErrNotFound
	}

	return domains[0], nil
}

func (r *DgraphDomainRepository) GetByUIDInProject(ctx context.Context, tx *dgo.Txn, projectUID, domainUID string) (*active_directory.Domain, error) {
	fields := []string{"uid", "name"}

	domains, err := dgraphutil.GetEntityByFieldInProject[*active_directory.Domain](
		ctx,
		tx,
		projectUID,
		"Domain",
		"uid",
		domainUID,
		fields,
	)

	if err != nil {
		return nil, fmt.Errorf("get entity by uid failed with message %w", err)
	}

	if len(domains) == 0 {
		return nil, rperror.ErrNotFound
	}

	return domains[0], nil
}

func (r *DgraphDomainRepository) AddToProject(
	ctx context.Context,
	tx *dgo.Txn,
	domainUID string,
	projectUID string,
) error {
	relationName := "belongs_to_project"
	err := dgraphutil.AddRelation(ctx, tx, domainUID, projectUID, relationName)
	if err != nil {
		return fmt.Errorf("error while linking domain %s to project %s with relation %s", domainUID, projectUID, relationName)
	}
	return nil
}

func (r *DgraphDomainRepository) Get(ctx context.Context, tx *dgo.Txn, uid string) (*active_directory.Domain, error) {
	query := `
        query Domain($uid: string) {
            domain(func: uid($uid)) {
                uid
                name
                belongs_to_project
				discovered_by
				discovered_at
				last_seen_at
				last_seen_by
                has_hosts {
                    uid
				}
				has_user {
					uid
				}

            }
        }`
	return dgraphutil.GetEntityByUID[active_directory.Domain](ctx, tx, uid, "domain", query)

}

func (r *DgraphDomainRepository) GetAllByProjectUID(ctx context.Context, tx *dgo.Txn, projectUID string) ([]*active_directory.Domain, error) {
	fields := []string{
		"uid",
		"name",
		"dns_name",
		"net_bios_name",
		"domain_guid",
		"domain_sid",
		"domain_function_level",
		"forest_function_level",
		"fsmo_role_owners",
		"created",
		"last_modified",
		"linked_gpos",
		"default_containers",
		"belongs_to_project { uid }",
		"dgraph.type",
		"discovered_by",
		"discovered_at",
		"last_seen_at",
		"last_seen_by",
	}

	domains, err := dgraphutil.GetEntitiesByRelation[*active_directory.Domain](
		ctx,
		tx,
		"Domain",
		"belongs_to_project",
		projectUID,
		fields,
	)

	log.Printf("Fetched domain belongs to project uid: %s", domains[0].BelongsToProject.UID)
	if err != nil {
		return nil, err
	}

	log.Printf("Found %d domains for project %s\n", len(domains), projectUID)
	return domains, nil
}

func (r *DgraphDomainRepository) Create(ctx context.Context, tx *dgo.Txn, name string, actor string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (r *DgraphDomainRepository) UpdateDomain(ctx context.Context, tx *dgo.Txn, uid string, actor string, fields map[string]interface{}) (*active_directory.Domain, error) {
	return dgraphutil.UpdateAndGet(ctx, tx, uid, actor, fields, r.Get)
}

func (r *DgraphDomainRepository) AddHost(ctx context.Context, tx *dgo.Txn, domainUID, hostUID string) error {
	relationName := "has_host"
	err := dgraphutil.AddRelation(ctx, tx, domainUID, hostUID, relationName)
	if err != nil {
		return fmt.Errorf("error while linking host %s to domain %s with relation %s", hostUID, domainUID, relationName)
	}
	return nil
}

func (r *DgraphDomainRepository) DomainExistsByName(ctx context.Context, tx *dgo.Txn, projectUID, name string) (bool, error) {
	return dgraphutil.ExistsByFieldInProject(ctx, tx, projectUID, "Domain", "name", name)
}

func (r *DgraphDomainRepository) AddUser(ctx context.Context, tx *dgo.Txn, domainUID, userUID string) error {
	relationName := "has_user"
	err := dgraphutil.AddRelation(ctx, tx, domainUID, userUID, relationName)
	if err != nil {
		return fmt.Errorf("error while linking user %s to domain %s with relation %s", userUID, domainUID, relationName)
	}
	return nil
}

func NewDgraphDomainRepository(db *dgo.Dgraph) *DgraphDomainRepository {
	return &DgraphDomainRepository{DB: db}
}
