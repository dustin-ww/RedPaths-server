package active_directory

import (
	"RedPaths-server/internal/repository/dgraphutil"
	"RedPaths-server/pkg/model/active_directory"
	"RedPaths-server/pkg/model/core"
	"context"
	"fmt"

	"github.com/dgraph-io/dgo/v210"
)

type DomainRepository interface {
	//CRUD
	Create(ctx context.Context, tx *dgo.Txn, domain *active_directory.Domain, actor string) (*active_directory.Domain, error)
	Get(ctx context.Context, tx *dgo.Txn, uid string) (*active_directory.Domain, error)
	Update(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*active_directory.Domain, error)

	// Find
	FindByNameInActiveDirectory(ctx context.Context, tx *dgo.Txn, activeDirectoryUID, domainName string) (*active_directory.Domain, error)

	GetAllByActiveDirectoryUID(ctx context.Context, tx *dgo.Txn, activeDirectoryUID string) ([]*core.EntityResult[*active_directory.Domain], error)
}

type DgraphDomainRepository struct {
	DB *dgo.Dgraph
}

func NewDgraphDomainRepository(db *dgo.Dgraph) *DgraphDomainRepository {
	return &DgraphDomainRepository{DB: db}
}

func (r *DgraphDomainRepository) Create(ctx context.Context, tx *dgo.Txn, incomingDomain *active_directory.Domain, actor string) (*active_directory.Domain, error) {
	return dgraphutil.CreateEntity(ctx, tx, "Domain", incomingDomain)
}

func (r *DgraphDomainRepository) AddAssertion(ctx context.Context, tx *dgo.Txn, domainUID, assertionUID string) error {
	relationName := "has_assertion"
	err := dgraphutil.AddRelation(ctx, tx, domainUID, assertionUID, relationName)
	if err != nil {
		return fmt.Errorf("error while linking red paths assertion %s to domain %s with relation %s", assertionUID, domainUID, relationName)
	}
	return nil
}

func (r *DgraphDomainRepository) FindByNameInActiveDirectory(ctx context.Context, tx *dgo.Txn, activeDirectoryUID, domainName string) (*active_directory.Domain, error) {
	fields := []string{
		"uid",
		"domain.name",
		"domain.description",
		"domain.dns_name",
		"domain.netbios_name",
		"domain.domain_guid",
		"domain.domain_sid",
		"domain.forest_functional_level",
		"domain.fsmo_role_owners",
		"domain.linked_gpos",
		"domain.default_containers",
		"created_at",
		"modified_at",
		"dgraph.type",
	}

	return dgraphutil.FindEntityByFieldViaAssertion[active_directory.Domain](
		ctx,
		tx,
		activeDirectoryUID,
		core.PredicateHasDomain,
		"Domain",
		"domain.name",
		domainName,
		fields,
	)
}

func (r *DgraphDomainRepository) Get(ctx context.Context, tx *dgo.Txn, uid string) (*active_directory.Domain, error) {
	query := `
        query Domain($uid: string) {
            domain(func: uid($uid)) {
				uid,
				domain.name
				domain.description
				domain.dns_name
				domain.netbios_name
				domain.domain_guid
				domain.domain_sid
				domain.forest_functional_level
				domain.fsmo_role_owners
				domain.linked_gpos
				domain.default_containers
				discovered_by
				discovered_at
				last_seen_at
				last_seen_by
            }
        }`
	return dgraphutil.GetEntityByUID[active_directory.Domain](ctx, tx, uid, "domain", query)

}

func (r *DgraphDomainRepository) GetAllByActiveDirectoryUID(ctx context.Context, tx *dgo.Txn, activeDirectoryUID string) ([]*core.EntityResult[*active_directory.Domain], error) {
	fields := []string{
		"uid",
		"domain.name",
		"domain.description",
		"domain.dns_name",
		"domain.netbios_name",
		"domain.domain_guid",
		"domain.domain_sid",
		"domain.forest_functional_level",
		"domain.fsmo_role_owners",
		"domain.linked_gpos",
		"domain.default_containers",
		"dgraph.type",
		"discovered_by",
		"discovered_at",
		"last_seen_at",
		"last_seen_by",
	}

	return dgraphutil.GetEntitiesWithAssertions[*active_directory.Domain](
		ctx,
		tx,
		activeDirectoryUID,
		core.PredicateHasDomain,
		"Domain",
		fields,
		"getADDomains",
	)
}

func (r *DgraphDomainRepository) Update(ctx context.Context, tx *dgo.Txn, uid string, actor string, fields map[string]interface{}) (*active_directory.Domain, error) {
	return dgraphutil.UpdateAndGet(ctx, tx, uid, actor, fields, r.Get)
}
