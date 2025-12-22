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

type UserRepository interface {
	Create(ctx context.Context, tx *dgo.Txn, incomingUser *model.ADUser, actor string) (*model.ADUser, error)
	AddToDomain(ctx context.Context, tx *dgo.Txn, userID string, domainUID string) error
	UserExistsByName(ctx context.Context, tx *dgo.Txn, projectUID string, name string) (bool, error)

	// Users WITH KNOWN/DISCOVERED DOMAIN
	GetByDomainUID(ctx context.Context, tx *dgo.Txn, domainUID string) ([]*model.ADUser, error)
	GetByProjectUID(ctx context.Context, tx *dgo.Txn, projectUID string) ([]*model.ADUser, error)

	UpdateFields(ctx context.Context, tx *dgo.Txn, uid string, fields map[string]interface{}) error
	GetByProjectIncludingDomains(ctx context.Context, tx *dgo.Txn, projectUID string) ([]*model.ADUser, error)
}

type DraphUserRepository struct {
	DB *dgo.Dgraph
}

func (r *DraphUserRepository) AddToDomain(ctx context.Context, tx *dgo.Txn, userID string, domainUID string) error {
	//TODO implement me
	panic("implement me")
}

func NewDgraphUserRepository(db *dgo.Dgraph) *DraphUserRepository {
	return &DraphUserRepository{DB: db}
}

func (r *DraphUserRepository) UserExistsByName(ctx context.Context, tx *dgo.Txn, projectUID string, name string) (bool, error) {
	return dgraphutil.ExistsByFieldInProject(ctx, tx, projectUID, "User", "Name", name)
}

func (r *DraphUserRepository) Create(ctx context.Context, tx *dgo.Txn, incomingUser *model.ADUser, actor string) (*model.ADUser, error) {
	incomingUser.DiscoveredAt = time.Now().UTC()
	incomingUser.DiscoveredBy = actor
	incomingUser.LastSeenAt = time.Now().UTC()
	incomingUser.LastSeenBy = actor
	createdUser, err := dgraphutil.CreateEntity(ctx, tx, "User", incomingUser)
	if err != nil {
		return nil, err
	}
	return createdUser, nil
}

func (r *DraphUserRepository) GetByDomainUID(ctx context.Context, tx *dgo.Txn, domainUID string) ([]*model.ADUser, error) {
	fields := []string{
		"uid",
		"name",
		"dgraph.type",
	}

	users, err := dgraphutil.GetEntitiesByRelation[*model.ADUser](
		ctx,
		tx,
		"User",
		"belongs_to_domain",
		domainUID,
		fields,
	)
	if err != nil {
		return nil,
			fmt.Errorf("failed to get users by relation: %w", err)
	}

	log.Printf("Found %d users for domain %s\n", len(users), domainUID)
	return users, nil
}

// TODO: Fix Relation
func (r *DraphUserRepository) GetByProjectUID(ctx context.Context, tx *dgo.Txn, projectUID string) ([]*model.ADUser, error) {
	fields := []string{
		"uid",
		"name",
		"dgraph.type",
	}

	users, err := dgraphutil.GetEntitiesByRelation[*model.ADUser](
		ctx,
		tx,
		"User",
		"has_user",
		projectUID,
		fields,
	)
	if err != nil {
		return nil,
			fmt.Errorf("failed to get users by relation: %w", err)
	}

	log.Printf("Found %d users for domain %s\n", len(users), projectUID)
	return users, nil
}

func (r *DraphUserRepository) GetByProjectIncludingDomains(
	ctx context.Context,
	tx *dgo.Txn,
	projectUID string,
) ([]*model.ADUser, error) {

	if tx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}

	query := `
	query UsersByProject($pid: string) {
		project(func: uid($pid)) {

			# Users directly linked to project
			has_user {
				uid
				name
				sam_account_name
				upn
				sid
				account_type

				is_admin
				is_domain_admin
				member_of { uid }

				spns
				kerberoastable
				asrep_roastable

				trusted_for_delegation
				unconstrained_delegation

				last_logon
				workstations

				risk_score
				risk_reasons

				discovered_at
				discovered_by
				last_seen_at
				last_seen_by

				dgraph.type
			}

			# Users via domains
			has_domain {
				has_user {
					uid
					name
					sam_account_name
					upn
					sid
					account_type

					is_admin
					is_domain_admin
					member_of { uid }

					spns
					kerberoastable
					asrep_roastable

					trusted_for_delegation
					unconstrained_delegation

					last_logon
					workstations

					risk_score
					risk_reasons

					discovered_at
					discovered_by
					last_seen_at
					last_seen_by

					dgraph.type
				}
			}
		}
	}`

	resp, err := tx.QueryWithVars(ctx, query, map[string]string{
		"$pid": projectUID,
	})
	if err != nil {
		return nil, fmt.Errorf("query users by project failed: %w", err)
	}

	// --- Response Mapping ---

	type domainWrapper struct {
		HasUser []*model.ADUser `json:"has_user"`
	}

	type projectWrapper struct {
		HasUser   []*model.ADUser `json:"has_user"`
		HasDomain []domainWrapper `json:"has_domain"`
	}

	var result struct {
		Project []projectWrapper `json:"project"`
	}

	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("unmarshal project users failed: %w", err)
	}

	// --- Deduplicate Users by UID ---

	userMap := make(map[string]*model.ADUser)

	for _, project := range result.Project {

		// direct users
		for _, user := range project.HasUser {
			if user.UID != "" {
				userMap[user.UID] = user
			}
		}

		// users via domains
		for _, domain := range project.HasDomain {
			for _, user := range domain.HasUser {
				if user.UID != "" {
					userMap[user.UID] = user
				}
			}
		}
	}

	users := make([]*model.ADUser, 0, len(userMap))
	for _, user := range userMap {
		users = append(users, user)
	}

	return users, nil
}

func (r *DraphUserRepository) UpdateFields(ctx context.Context, tx *dgo.Txn, uid string, fields map[string]interface{}) error {
	fields["uid"] = uid

	updateJSON, err := json.Marshal(fields)
	if err != nil {
		return fmt.Errorf("error while updating user with json %s and error: %w", updateJSON, err)
	}

	mu := &api.Mutation{
		SetJson: updateJSON,
	}

	_, err = tx.Mutate(ctx, mu)
	if err != nil {
		return fmt.Errorf("mutation error: %w", err)
	}
	return nil
}
