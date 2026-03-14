package active_directory

import (
	"RedPaths-server/internal/repository/dgraphutil"
	"RedPaths-server/pkg/model/active_directory/gpo"
	"RedPaths-server/pkg/model/core"
	"context"
	"fmt"

	"github.com/dgraph-io/dgo/v210"
)

type GPORepository interface {

	//GPO
	//CRUD
	CreateGPO(ctx context.Context, tx *dgo.Txn, gpo *gpo.GPO, actor string) (*gpo.GPO, error)
	CreateLink(ctx context.Context, tx *dgo.Txn, gpoLink *gpo.Link, actor string) (*gpo.Link, error)
	GetGPO(ctx context.Context, tx *dgo.Txn, uid string) (*gpo.GPO, error)
	GetLink(ctx context.Context, tx *dgo.Txn, uid string) (*gpo.Link, error)
	UpdateGPO(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*gpo.GPO, error)
	UpdateLink(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*gpo.Link, error)

	AddGPOToLink(ctx context.Context, tx *dgo.Txn, linkUID, gpoUID string) error

	FindGPOByNameInContainer(ctx context.Context, tx *dgo.Txn, domainUID, gpoName string) (*gpo.GPO, error)

	ExistsGPOByNameInContainer(ctx context.Context, tx *dgo.Txn, domainUID, gpoName string) (bool, string, error)

	GetGPOLinksWithGPO(ctx context.Context, tx *dgo.Txn, domainUID string) ([]*core.EntityResult[*gpo.Link], error)
	FindGPOLinkByGPOName(ctx context.Context, tx *dgo.Txn, domainUID, gpoName string) (*core.EntityResult[*gpo.Link], error)
}

type DgraphGPORepository struct {
	DB *dgo.Dgraph
}

func NewDgraphGPORepository(db *dgo.Dgraph) *DgraphGPORepository {
	return &DgraphGPORepository{DB: db}
}

// GetGPOLinksWithGPO lädt alle GPOLinks einer Domain mit eingebetteten GPOs
func (d *DgraphGPORepository) GetGPOLinksWithGPO(
	ctx context.Context,
	tx *dgo.Txn,
	domainUID string,
) ([]*core.EntityResult[*gpo.Link], error) {

	// Felder für den GPOLink
	linkFields := []string{
		"uid",
		"dgraph.type",
		"gpo.link_order",
		"gpo.is_enforced",
		"gpo.is_enabled",
		"created_at",
		"modified_at",
		"discovered_at",
		"discovered_by",
		"last_seen_at",
		"last_seen_by",
	}

	// Felder für die eingebettete GPO
	gpoFields := []string{
		"uid",
		"dgraph.type",
		"gpo.name",
		"gpo.description",
		"created_at",
		"modified_at",
		"discovered_at",
		"discovered_by",
		"last_seen_at",
		"last_seen_by",
	}

	results, err := dgraphutil.GetEntitiesWithAssertionsAndEmbeddedRelation[*gpo.Link](
		ctx,
		tx,
		domainUID,
		"has_gpo_link", // Assertion Predicate
		"GPOLink",      // Object Type
		linkFields,     // Object Fields
		"gpo.links_to", // Embedded Relation Name
		"GPO",          // Embedded Type
		gpoFields,      // Embedded Fields
		"getGPOLinksWithGPO",
	)

	if err != nil {
		return nil, fmt.Errorf("error getting GPO links with GPO: %w", err)
	}

	return results, nil
}

// FindGPOLinkByGPOName findet einen GPOLink anhand des GPO-Namens in einer Domain
func (d *DgraphGPORepository) FindGPOLinkByGPOName(
	ctx context.Context,
	tx *dgo.Txn,
	domainUID string,
	gpoName string,
) (*core.EntityResult[*gpo.Link], error) {

	// Alle GPOLinks mit eingebetteten GPOs laden
	allLinks, err := d.GetGPOLinksWithGPO(ctx, tx, domainUID)
	if err != nil {
		return nil, err
	}

	// Nach GPO-Namen filtern
	for _, linkResult := range allLinks {
		if linkResult.Entity.LinksTo.Name == gpoName {
			return linkResult, nil
		}
	}

	return nil, nil
}

func (d *DgraphGPORepository) AddGPOToLink(ctx context.Context, tx *dgo.Txn, domainUID, assertionUID string) error {
	relationName := "has_assertion"
	err := dgraphutil.AddRelation(ctx, tx, domainUID, assertionUID, relationName)
	if err != nil {
		return fmt.Errorf("error while linking red paths assertion %s to domain %s with relation %s", assertionUID, domainUID, relationName)
	}
	return nil
}

func (r *DgraphGPORepository) GetAllByDomainUID(ctx context.Context, tx *dgo.Txn, activeDirectoryUID string) ([]*core.EntityResult[*gpo.Link], error) {
	fields := []string{
		"uid",
		"gpo_link.name",
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

	return dgraphutil.GetEntitiesWithAssertions[*gpo.Link](
		ctx,
		tx,
		activeDirectoryUID,
		core.PredicateHasGPOLink,
		"GPOLink",
		fields,
		"getDomainGPOs",
	)
}

func (d *DgraphGPORepository) FindGPOByNameInContainer(
	ctx context.Context,
	tx *dgo.Txn,
	containerUID,
	gpoName string,
) (*gpo.GPO, error) {

	fields := []string{
		"uid",
		"dgraph.type",
		"gpo.name",
		"gpo.display_name",
		"gpo.distinguished_name",
		"gpo.guid",
		"created_at",
		"modified_at",
		"discovered_at",
		"discovered_by",
		"last_seen_at",
		"last_seen_by",
	}

	result, err := dgraphutil.FindEntityViaAssertionAndRelation[gpo.GPO](
		ctx,
		tx,
		containerUID,
		"has_gpo_link", // Assertion Predicate
		"GPOLink",      // Intermediate Type
		"links_to",     // Direct Relation
		"GPO",          // Target Type
		"name",         // Filter Field
		gpoName,        // Filter Value
		fields,         // Target Fields
	)

	if err != nil {
		return nil, fmt.Errorf("error finding GPO by name in domain: %w", err)
	}

	return result, nil
}

func (d *DgraphGPORepository) ExistsGPOByNameInContainer(
	ctx context.Context,
	tx *dgo.Txn,
	containerUID,
	gpoName string,
) (bool, string, error) {

	exists, gpoUID, err := dgraphutil.ExistsEntityViaAssertionAndRelation(
		ctx,
		tx,
		containerUID,
		"has_gpo_link", // Assertion Predicate
		"GPOLink",      // Intermediate Type
		"links_to",     // Direct Relation
		"GPO",          // Target Type
		"name",         // Filter Field
		gpoName,        // Filter Value
	)

	if err != nil {
		return false, "", fmt.Errorf("error checking GPO existence in domain: %w", err)
	}

	return exists, gpoUID, nil
}

func (d *DgraphGPORepository) CreateGPO(ctx context.Context, tx *dgo.Txn, gpo *gpo.GPO, actor string) (*gpo.GPO, error) {
	return dgraphutil.CreateEntity(ctx, tx, "GPO", gpo)
}

func (d *DgraphGPORepository) CreateLink(ctx context.Context, tx *dgo.Txn, gpoLink *gpo.Link, actor string) (*gpo.Link, error) {
	return dgraphutil.CreateEntity(ctx, tx, "GPOLink", gpoLink)
}

func (r *DgraphDomainRepository) AddLinksToRelation(ctx context.Context, tx *dgo.Txn, gpoLinkUID, gpoUID string) error {
	relationName := "links_to"
	err := dgraphutil.AddRelation(ctx, tx, gpoLinkUID, gpoUID, relationName)
	if err != nil {
		return fmt.Errorf("error while linking gpo link %s to gpo %s with relation %s", gpoLinkUID, gpoUID, relationName)
	}
	return nil
}

func (d *DgraphGPORepository) GetGPO(ctx context.Context, tx *dgo.Txn, uid string) (*gpo.GPO, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DgraphGPORepository) GetLink(ctx context.Context, tx *dgo.Txn, uid string) (*gpo.Link, error) {
	//TODO implement me
	panic("implement me")
}

func (d *DgraphGPORepository) UpdateGPO(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*gpo.GPO, error) {
	return dgraphutil.UpdateAndGet(ctx, tx, uid, actor, fields, d.GetGPO)

}

func (d *DgraphGPORepository) UpdateLink(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*gpo.Link, error) {
	return dgraphutil.UpdateAndGet(ctx, tx, uid, actor, fields, d.GetLink)

}
