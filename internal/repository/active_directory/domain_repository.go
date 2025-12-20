package active_directory

import (
	"RedPaths-server/internal/repository/dgraphutil"
	"RedPaths-server/pkg/model"
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
	Create(ctx context.Context, tx *dgo.Txn, name string) (string, error) // Returns UID
	Get(ctx context.Context, tx *dgo.Txn, uid string) (*model.Domain, error)
	GetByName(ctx context.Context, tx *dgo.Txn, projectUID, name string) (*model.Domain, error)
	UpdateFields(ctx context.Context, uid string, fields map[string]interface{}) error
	CreateWithObject(ctx context.Context, tx *dgo.Txn, model *model.Domain) (string, error)

	//Relations
	AddHost(ctx context.Context, tx *dgo.Txn, domainUID, hostUID string) error
	AddUser(ctx context.Context, domainUID, userUID string) error
	AddToProject(ctx context.Context, tx *dgo.Txn, domainUID string, projectUID string) error

	// Checker
	DomainExistsByName(ctx context.Context, tx *dgo.Txn, projectUID, name string) (bool, error)

	// Reverse Relations
	GetByProjectUID(ctx context.Context, tx *dgo.Txn, projectUID string) ([]*model.Domain, error)
}

type DgraphDomainRepository struct {
	DB *dgo.Dgraph
}

func (r *DgraphDomainRepository) CreateWithObject(ctx context.Context, tx *dgo.Txn, domain *model.Domain) (string, error) {
	domain.CreatedAt = time.Now().UTC()
	domain.DiscoveredAt = time.Now().UTC()
	domain.DiscoveredBy = "User"
	domain.LastModified = domain.CreatedAt
	domain.LastSeenAt = domain.CreatedAt
	domain.LastSeenBy = "User"
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

// In DomainRepository:
func (r *DgraphDomainRepository) GetByName(ctx context.Context, tx *dgo.Txn, projectUID, name string) (*model.Domain, error) {
	fields := []string{"uid", "name"}

	domains, err := dgraphutil.GetEntityByFieldInProject[*model.Domain](
		ctx,
		tx,
		projectUID,
		"Domain",
		"name",
		name,
		fields,
	)

	if err != nil {
		return nil, err
	}

	if len(domains) == 0 {
		return nil, fmt.Errorf("domain not found")
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

func (r *DgraphDomainRepository) Get(ctx context.Context, tx *dgo.Txn, uid string) (*model.Domain, error) {
	query := `
        query Domain($uid: string) {
            domain(func: uid($uid)) {
                uid
                name
                belongs_to_project
                has_hosts {
                    uid
				}
				has_user {
					uid
				}

            }
        }`
	return dgraphutil.GetEntityByUID[model.Domain](ctx, tx, uid, "domain", query)

}

func (r *DgraphDomainRepository) GetByProjectUID(ctx context.Context, tx *dgo.Txn, projectUID string) ([]*model.Domain, error) {
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
	}

	domains, err := dgraphutil.GetEntitiesByRelation[*model.Domain](
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

func (r *DgraphDomainRepository) Create(ctx context.Context, tx *dgo.Txn, name string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (r *DgraphDomainRepository) UpdateFields(ctx context.Context, uid string, fields map[string]interface{}) error {
	//TODO implement me
	panic("implement me")
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

func (r *DgraphDomainRepository) AddUser(ctx context.Context, domainUID, userUID string) error {
	//TODO implement me
	panic("implement me")
}

func NewDgraphDomainRepository(db *dgo.Dgraph) *DgraphDomainRepository {
	return &DgraphDomainRepository{DB: db}
}
