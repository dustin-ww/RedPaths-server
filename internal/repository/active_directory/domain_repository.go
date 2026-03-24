package active_directory

import (
	dgraphutil2 "RedPaths-server/internal/repository/util/dgraph"
	"RedPaths-server/pkg/model/active_directory"
	"RedPaths-server/pkg/model/active_directory/gpo"
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/core/res"
	"context"
	"fmt"
	"log"

	"github.com/dgraph-io/dgo/v210"
)

type DomainRepository interface {
	//CRUD
	Create(ctx context.Context, tx *dgo.Txn, domain *active_directory.Domain, actor string) (*active_directory.Domain, error)
	Get(ctx context.Context, tx *dgo.Txn, uid string) (*active_directory.Domain, error)
	Update(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*active_directory.Domain, error)

	// Find
	FindByNameInActiveDirectory(ctx context.Context, tx *dgo.Txn, activeDirectoryUID, domainName string) (*active_directory.Domain, error)

	GetAllByActiveDirectoryUID(ctx context.Context, tx *dgo.Txn, activeDirectoryUID string) ([]*res.EntityResult[*active_directory.Domain], error)

	GetAllByProjectUID(ctx context.Context, tx *dgo.Txn, projectUID string) ([]*res.EntityResult[*active_directory.Domain], error)

	GetGPOIfKnown(ctx context.Context, tx *dgo.Txn, gpoName string) ([]*gpo.GPO, error)
	FindExisting(ctx context.Context, tx *dgo.Txn, projectUID string, domain *active_directory.Domain) (*dgraphutil2.ExistenceResult[*active_directory.Domain], error)
}

type DgraphDomainRepository struct {
	DB *dgo.Dgraph
}

func NewDgraphDomainRepository(db *dgo.Dgraph) *DgraphDomainRepository {
	return &DgraphDomainRepository{DB: db}
}

func (r *DgraphDomainRepository) Create(ctx context.Context, tx *dgo.Txn, incomingDomain *active_directory.Domain, actor string) (*active_directory.Domain, error) {
	dgraphutil2.InitCreateMetadata(&incomingDomain.RedPathsMetadata, actor)
	return dgraphutil2.CreateEntity(ctx, tx, "Domain", incomingDomain)
}

func (r *DgraphDomainRepository) AddAssertion(ctx context.Context, tx *dgo.Txn, domainUID, assertionUID string) error {
	relationName := "has_assertion"
	err := dgraphutil2.AddRelation(ctx, tx, domainUID, assertionUID, relationName)
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

	return dgraphutil2.FindEntityByFieldViaAssertion[active_directory.Domain](
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

func (r *DgraphDomainRepository) GetAllByProjectUID(ctx context.Context, tx *dgo.Txn, projectUID string) ([]*res.EntityResult[*active_directory.Domain], error) {
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

	return dgraphutil2.GetEntitiesWithAssertionsNHop[*active_directory.Domain](
		ctx, tx, projectUID,
		[]dgraphutil2.HopConfig{
			{Predicate: core.PredicateHasActiveDirectory},
			{Predicate: core.PredicateHasDomain, ObjectType: "Domain"},
		},

		fields, "getProjectDomains",
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
	return dgraphutil2.GetEntityByUID[active_directory.Domain](ctx, tx, uid, "domain", query)

}

func (r *DgraphDomainRepository) GetAllByActiveDirectoryUID(ctx context.Context, tx *dgo.Txn, activeDirectoryUID string) ([]*res.EntityResult[*active_directory.Domain], error) {
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

	return dgraphutil2.GetEntitiesWithAssertions[*active_directory.Domain](
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
	return dgraphutil2.UpdateAndGet(ctx, tx, uid, actor, fields, r.Get)
}

var domainHierarchyHops = []dgraphutil2.HopConfig{
	{Predicate: core.PredicateHasActiveDirectory},
	{Predicate: core.PredicateHasDomain, ObjectType: "Domain"},
}

var domainFields = []string{
	"uid",
	"domain.name",
	"domain.dns_name",
	"domain.netbios_name",
	"domain.domain_guid",
	"domain.domain_sid",
	"domain.functional_level",
	"domain.forest_functional_level",
	"dgraph.type",
}

func BuildDomainFilter(domain *active_directory.Domain) []dgraphutil2.UniqueFieldFilter {
	return []dgraphutil2.UniqueFieldFilter{
		{Field: "domain.domain_guid", Value: domain.DomainGUID},
		{Field: "domain.domain_sid", Value: domain.DomainSID},
		{Field: "domain.dns_name", Value: domain.DNSName},
		{Field: "domain.name", Value: domain.Name},
	}
}

// FindExisting performs a two-phase existence check for a Domain.
//
// Phase 1: Searches via the project hierarchy (Project → AD → Domain).
// Phase 2: Falls back to direct project-level search for orphaned domains.
func (r *DgraphDomainRepository) FindExisting(
	ctx context.Context,
	tx *dgo.Txn,
	projectUID string,
	domain *active_directory.Domain,
) (*dgraphutil2.ExistenceResult[*active_directory.Domain], error) {

	filters := BuildDomainFilter(domain)

	return dgraphutil2.CheckEntityExists[*active_directory.Domain](
		ctx, tx,
		projectUID,
		"Domain",
		filters,
		dgraphutil2.FilterModeOR,
		domainFields,
		domainHierarchyHops,
	)
}

func (r *DgraphDomainRepository) AddGPOUtilityReference(ctx context.Context, tx *dgo.Txn, domainUID, gpoUID string) error {
	relationName := "has_gpo"
	err := dgraphutil2.AddRelation(ctx, tx, domainUID, gpoUID, relationName)
	if err != nil {
		return fmt.Errorf("error while linking gpo %s to domain %s with relation %s", gpoUID, domainUID, relationName)
	}
	log.Printf("Created relation %s for gpo %s and domain %s", relationName, gpoUID, domainUID)
	return nil
}

func (r *DgraphDomainRepository) GetGPOIfKnown(ctx context.Context, tx *dgo.Txn, gpoName string) ([]*gpo.GPO, error) {
	//exists, err := dgraph.ExistsByField(ctx, tx, "GPO", "gpo.name", gpoName)
	fetchedGPO, err := dgraphutil2.GetEntityByField[*gpo.GPO](
		ctx, tx,
		"GPO",
		"gpo.name",
		gpoName,
		[]string{"uid", "gpo.name"},
	)

	if err != nil {
		return nil, fmt.Errorf("error while searching for gpo with name %s in domain lib for seen gpos", gpoName)
	}
	return fetchedGPO, nil
}

func (r *DgraphDomainRepository) GetKnownGPOs(ctx context.Context, tx *dgo.Txn, domainUID string) ([]*res.EntityResult[*gpo.GPO], error) {
	fields := []string{
		"uid",
		"gpo.name",
		"created_at",
		"modified_at",
		"dgraph.type",
	}

	return dgraphutil2.GetEntitiesWithAssertionsNHop[*gpo.GPO](
		ctx, tx, domainUID,
		[]dgraphutil2.HopConfig{
			{Predicate: core.PredicateHasGPOLink},
			{Predicate: core.PredicateLinksTo, ObjectType: "Domain"},
			{Predicate: core.PredicateContains, ObjectType: "DirectoryNode"},
		},

		fields, "getProjectDirectorNodes",
	)
}
