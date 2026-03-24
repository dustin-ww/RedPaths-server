package active_directory

import (
	dgraphutil2 "RedPaths-server/internal/repository/util/dgraph"
	"RedPaths-server/pkg/model/active_directory/priv"
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/core/res"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/dgo/v210"
)

type ACLRepository interface {
	// CRUD
	// ACL
	CreateACL(ctx context.Context, tx *dgo.Txn, acl *priv.ACL, actor string) (*priv.ACL, error)
	GetACL(ctx context.Context, tx *dgo.Txn, uid string) (*priv.ACL, error)
	DeleteACL(ctx context.Context, tx *dgo.Txn, aclUID string) error
	UpdateACL(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*priv.ACL, error)

	// ACE
	CreateACE(ctx context.Context, tx *dgo.Txn, ace *priv.ACE, actor string) (*priv.ACE, error)
	GetACE(ctx context.Context, tx *dgo.Txn, uid string) (*priv.ACE, error)
	DeleteACE(ctx context.Context, tx *dgo.Txn, aceUID string) error
	UpdateACE(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*priv.ACE, error)

	GetAllACEByACL(ctx context.Context, tx *dgo.Txn, aclUID string) ([]*res.EntityResult[*priv.ACE], error)

	// ADRight
	CreateADRight(ctx context.Context, tx *dgo.Txn, ace *priv.ADRight, actor string) (*priv.ADRight, error)
	GetADRight(ctx context.Context, tx *dgo.Txn, uid string) (*priv.ADRight, error)
	DeleteADRight(ctx context.Context, tx *dgo.Txn, adRightUID string) error
	UpdateADRight(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*priv.ADRight, error)

	GetAllRightsByACE(ctx context.Context, tx *dgo.Txn, aceUID string) ([]*res.EntityResult[*priv.ADRight], error)
	// Finds
	//FindByDistinguishedNameInDomain(ctx context.Context, tx *dgo.Txn, domainUID string, dsName string) (*active_directory.DirectoryNode, error)

	GetByDomainUID(ctx context.Context, tx *dgo.Txn, domainUID string) (*priv.ACL, error)
	GetByDirectoryNodeUID(ctx context.Context, tx *dgo.Txn, domainUID string) (*priv.ACL, error)
	GetByDirectoryGPOUID(ctx context.Context, tx *dgo.Txn, domainUID string) (*priv.ACL, error)
	LinkACLToEntity(ctx context.Context, tx *dgo.Txn, aclUID, entityUID string) error
}

type DgraphACLRepository struct {
	DB *dgo.Dgraph
}

func (r *DgraphACLRepository) DeleteACL(ctx context.Context, tx *dgo.Txn, aclUID string) error {
	//TODO implement me
	panic("implement me")
}

func (r *DgraphACLRepository) DeleteACE(ctx context.Context, tx *dgo.Txn, aceUID string) error {
	//TODO implement me
	panic("implement me")
}

func (r *DgraphACLRepository) DeleteADRight(ctx context.Context, tx *dgo.Txn, adRightUID string) error {
	//TODO implement me
	panic("implement me")
}

func (r *DgraphACLRepository) GetByDomainUID(ctx context.Context, tx *dgo.Txn, domainUID string) (*priv.ACL, error) {
	//TODO implement me
	panic("implement me")
}

func (r *DgraphACLRepository) GetByDirectoryNodeUID(ctx context.Context, tx *dgo.Txn, domainUID string) (*priv.ACL, error) {
	//TODO implement me
	panic("implement me")
}

func (r *DgraphACLRepository) GetByDirectoryGPOUID(ctx context.Context, tx *dgo.Txn, domainUID string) (*priv.ACL, error) {
	//TODO implement me
	panic("implement me")
}

func NewDgraphDgraphACLRepository(db *dgo.Dgraph) *DgraphACLRepository {
	return &DgraphACLRepository{DB: db}
}

// ACL
func (r *DgraphACLRepository) CreateACL(ctx context.Context, tx *dgo.Txn, acl *priv.ACL, actor string) (*priv.ACL, error) {
	dgraphutil2.InitCreateMetadata(&acl.RedPathsMetadata, actor)
	log.Println("Create ACL")
	return dgraphutil2.CreateEntity(ctx, tx, "ACL", acl)
}

func (r *DgraphACLRepository) GetACL(ctx context.Context, tx *dgo.Txn, uid string) (*priv.ACL, error) {
	query := `
        query ACL($uid: string) {
            acl(func: uid($uid)) {
                uid
                acl.name
				acl.security_principal
            }
        }
    `
	return dgraphutil2.GetEntityByUID[priv.ACL](ctx, tx, uid, "acl", query)
}

func (r *DgraphACLRepository) UpdateACL(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*priv.ACL, error) {
	// legacy
	fields["updated_at"] = time.Now().Format(time.RFC3339)
	return dgraphutil2.UpdateAndGet(ctx, tx, uid, actor, fields, r.GetACL)
}

func (r *DgraphDirectoryNodeRepository) DeleteACL(ctx context.Context, tx *dgo.Txn, aclUID string) error {
	//TODO implement me
	panic("implement me")
}

// ACE

func (r *DgraphACLRepository) CreateACE(ctx context.Context, tx *dgo.Txn, ace *priv.ACE, actor string) (*priv.ACE, error) {
	dgraphutil2.InitCreateMetadata(&ace.RedPathsMetadata, actor)
	return dgraphutil2.CreateEntity(ctx, tx, "ACE", ace)
}

func (r *DgraphACLRepository) GetACE(ctx context.Context, tx *dgo.Txn, uid string) (*priv.ACE, error) {
	query := `
        query ACE($uid: string) {
            ace(func: uid($uid)) {
                uid
                ace.name
				ace.access_type
				ace.inherited
				ace.applies_to
            }
        }
    `
	return dgraphutil2.GetEntityByUID[priv.ACE](ctx, tx, uid, "ace", query)
}

func (r *DgraphACLRepository) UpdateACE(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*priv.ACE, error) {
	// legacy
	fields["updated_at"] = time.Now().Format(time.RFC3339)
	return dgraphutil2.UpdateAndGet(ctx, tx, uid, actor, fields, r.GetACE)
}

func (r *DgraphDirectoryNodeRepository) DeleteACE(ctx context.Context, tx *dgo.Txn, aceUID string) error {
	//TODO implement me
	panic("implement me")
}

//ADRights

func (r *DgraphACLRepository) CreateADRight(ctx context.Context, tx *dgo.Txn, adRight *priv.ADRight, actor string) (*priv.ADRight, error) {
	dgraphutil2.InitCreateMetadata(&adRight.RedPathsMetadata, actor)
	return dgraphutil2.CreateEntity(ctx, tx, "ad_right", adRight)
}

func (r *DgraphACLRepository) GetADRight(ctx context.Context, tx *dgo.Txn, uid string) (*priv.ADRight, error) {
	query := `
        query ADRight($uid: string) {
            ace(func: uid($uid)) {
                uid
                ad_right.name
				ad_right.category
				ad_right.risk_level
            }
        }
    `
	return dgraphutil2.GetEntityByUID[priv.ADRight](ctx, tx, uid, "ad_right", query)
}

func (r *DgraphACLRepository) UpdateADRight(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*priv.ADRight, error) {
	// legacy
	fields["updated_at"] = time.Now().Format(time.RFC3339)
	return dgraphutil2.UpdateAndGet(ctx, tx, uid, actor, fields, r.GetADRight)
}

func (r *DgraphDirectoryNodeRepository) DeleteADRight(ctx context.Context, tx *dgo.Txn, aceUID string) error {
	//TODO implement me
	panic("implement me")
}

func (r *DgraphACLRepository) GetAllACEByACL(ctx context.Context, tx *dgo.Txn, aclUID string) ([]*res.EntityResult[*priv.ACE], error) {
	fields := []string{
		"uid",
		"ace.name",
		"ace.access_type",
		"ace.inherited",
		"ace.applies_to",
		"dgraph.type",
		"discovered_by",
		"discovered_at",
		"last_seen_at",
		"last_seen_by",
	}

	return dgraphutil2.GetEntitiesWithAssertions[*priv.ACE](
		ctx,
		tx,
		aclUID,
		core.PredicateContains,
		"ACE",
		fields,
		"getACEs",
	)
}

func (r *DgraphACLRepository) GetAllRightsByACE(ctx context.Context, tx *dgo.Txn, aceUID string) ([]*res.EntityResult[*priv.ADRight], error) {
	fields := []string{
		"uid",
		"ad_right.name",
		"ad_right.category",
		"ad_right.risk_level",
		"dgraph.type",
		"discovered_by",
		"discovered_at",
		"last_seen_at",
		"last_seen_by",
	}

	return dgraphutil2.GetEntitiesWithAssertions[*priv.ADRight](
		ctx,
		tx,
		aceUID,
		core.PredicateContains,
		"ad_right",
		fields,
		"getADRights",
	)
}

func (r *DgraphACLRepository) LinkACLToEntity(ctx context.Context, tx *dgo.Txn, aclUID, entityUID string) error {
	relationName := "has_acl"
	log.Printf("AUTO LINKING: acl %s and entity %s", aclUID, entityUID)
	err := dgraphutil2.AddRelation(ctx, tx, aclUID, entityUID, relationName)
	if err != nil {
		return fmt.Errorf("error while linking service %s to host %s with relation name %s", aclUID, entityUID, relationName)
	}
	return nil
}
