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

// ProjectRepository defines operations for project data access
type ProjectRepository interface {
	// CRUD
	Create(ctx context.Context, tx *dgo.Txn, name string) (string, error)
	Get(ctx context.Context, tx *dgo.Txn, uid string) (*model.Project, error)
	GetAll(ctx context.Context, tx *dgo.Txn) ([]*model.Project, error)
	// TODO: Move into target repo
	GetTargets(ctx context.Context, tx *dgo.Txn, uid string) ([]*model.Target, error)
	Delete(ctx context.Context, tx *dgo.Txn, uid string) error
	UpdateProject(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*model.Project, error)

	// Relations
	AddActiveDirectory(ctx context.Context, tx *dgo.Txn, projectUID, activeDirectoryUID string) error

	AddDomain(ctx context.Context, tx *dgo.Txn, projectUID, domainUID string) error
	AddTarget(ctx context.Context, tx *dgo.Txn, projectUID, targetUID string) error
	AddHostWithUnknownDomain(ctx context.Context, tx *dgo.Txn, projectUID, hostUID string) error
	AddUser(ctx context.Context, tx *dgo.Txn, projectUID, userUID string) error
}

// DgraphProjectRepository implements ProjectRepository using Dgraph
type DgraphProjectRepository struct {
	DB *dgo.Dgraph
}

// NewDgraphProjectRepository creates a new Dgraph project repository
func NewDgraphProjectRepository(db *dgo.Dgraph) *DgraphProjectRepository {
	return &DgraphProjectRepository{DB: db}
}

// AddDomain connects a domain to a project
func (r *DgraphProjectRepository) AddActiveDirectory(ctx context.Context, tx *dgo.Txn, projectUID, activeDirectoryUID string) error {
	relationName := "has_ad"
	err := dgraphutil.AddRelation(ctx, tx, projectUID, activeDirectoryUID, relationName)
	if err != nil {
		return fmt.Errorf("error while linking domain %s to project %s with relation %s", domainUID, projectUID, relationName)
	}
	return nil
}

// Create adds a new project to the database
func (r *DgraphProjectRepository) Create(ctx context.Context, tx *dgo.Txn, name string) (string, error) {
	projectData := map[string]interface{}{
		"name":        name,
		"dgraph.type": "Project",
		"created_at":  time.Now(),
	}

	jsonData, err := json.Marshal(projectData)
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

// Get retrieves a project by UID
func (r *DgraphProjectRepository) Get(ctx context.Context, tx *dgo.Txn, uid string) (*model.Project, error) {
	fmt.Println("REPO")
	query := `
        query Project($uid: string) {
            project(func: uid($uid)) {
                uid
                name
                tags
                created_at
                modified_at
                description
                has_domain {
                    uid
                }
                has_target {
                    uid
                    ip_range
                    name
                }
            }
        }
    `
	return dgraphutil.GetEntityByUID[model.Project](ctx, tx, uid, "project", query)
}

// GetTargets retrieves all targets for a project
func (r *DgraphProjectRepository) GetTargets(ctx context.Context, tx *dgo.Txn, uid string) ([]*model.Target, error) {
	query := `
        query Project($uid: string) {
            project(func: uid($uid)) @filter(eq(dgraph.type, "Project")) {
                uid
                has_target {
                    uid
                    ip
					cidr
					note
                    name
                    dgraph.type
                }
            }
        }
    `

	vars := map[string]string{"$uid": uid}
	res, err := tx.QueryWithVars(ctx, query, vars)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}

	var result struct {
		Project []struct {
			UID       string          `json:"uid"`
			HasTarget []*model.Target `json:"has_target"`
		} `json:"project"`
	}

	if err := json.Unmarshal(res.Json, &result); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	if len(result.Project) == 0 {
		return nil, fmt.Errorf("project not found: %s", uid)
	}

	targets := result.Project[0].HasTarget

	if targets == nil {
		return []*model.Target{}, nil
	}

	return targets, nil
}

// GetAll retrieves all projects with full details
func (r *DgraphProjectRepository) GetAll(ctx context.Context, tx *dgo.Txn) ([]*model.Project, error) {
	fields := []string{
		"uid",
		"name",
		"created_at",
		"description",
		"dgraph.type",
	}
	return dgraphutil.GetAllEntities[*model.Project](ctx, tx, "Project", fields, 0, 0)
}

// UpdateProject updates specified fields on a project
func (r *DgraphProjectRepository) UpdateProject(ctx context.Context, tx *dgo.Txn, uid, actor string, fields map[string]interface{}) (*model.Project, error) {
	// legacy
	fields["updated_at"] = time.Now().Format(time.RFC3339)
	return dgraphutil.UpdateAndGet(ctx, tx, uid, actor, fields, r.Get)
}

// Delete removes a project by UID
func (r *DgraphProjectRepository) Delete(ctx context.Context, tx *dgo.Txn, uid string) error {
	projectMap := map[string][]string{
		"Project": {
			"has_domain",
			"has_target",
			"has_redpaths_modules",
			"has_unknown_domain_host",
		},
		"Domain": {
			"has_host",
			"has_user",
			"security_policies",
			"trust_relationships",
		},
		"Host": {
			"has_service",
		},
		"Target":         {},
		"User":           {},
		"Service":        {},
		"RedPathModule":  {},
		"SecurityPolicy": {},
		"Trust":          {},
	}

	return dgraphutil.DeleteEntityCascadeByTypeMap(ctx, tx, uid, projectMap)
}

// AddDomain connects a domain to a project
func (r *DgraphProjectRepository) AddDomain(ctx context.Context, tx *dgo.Txn, projectUID, domainUID string) error {
	relationName := "has_domain"
	err := dgraphutil.AddRelation(ctx, tx, projectUID, domainUID, relationName)
	if err != nil {
		return fmt.Errorf("error while linking domain %s to project %s with relation %s", domainUID, projectUID, relationName)
	}
	return nil
}

// AddTarget connects a target to a project
func (r *DgraphProjectRepository) AddTarget(ctx context.Context, tx *dgo.Txn, projectUID, targetUID string) error {
	relationName := "has_target"
	err := dgraphutil.AddRelation(ctx, tx, projectUID, targetUID, relationName)
	if err != nil {
		return fmt.Errorf("error while linking target %s to project %s with relation %s", targetUID, projectUID, relationName)
	}
	return nil
}

func (r *DgraphProjectRepository) AddHostWithUnknownDomain(ctx context.Context, tx *dgo.Txn, projectUID, hostUID string) error {
	relationName := "has_host"
	err := dgraphutil.AddRelation(ctx, tx, projectUID, hostUID, relationName)
	if err != nil {
		return fmt.Errorf("error while linking unknown domain host %s to project %s with relation %s", hostUID, projectUID, relationName)
	}
	log.Printf("Created relation %s for host %s and project %s", relationName, hostUID, projectUID)
	return nil
}

func (r *DgraphProjectRepository) AddUser(ctx context.Context, tx *dgo.Txn, projectUID, userUID string) error {
	relationName := "has_user"
	err := dgraphutil.AddRelation(ctx, tx, projectUID, userUID, relationName)
	if err != nil {
		return fmt.Errorf("error while linking unknown domain user %s to project %s with relation %s", userUID, projectUID, relationName)
	}
	log.Printf("Created relation %s for user %s and project %s", relationName, userUID, projectUID)
	return nil
}
