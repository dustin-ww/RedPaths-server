package engine

import (
	"RedPaths-server/internal/repository/util/dgraph"
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/core/res"
	"context"
	"fmt"

	"github.com/dgraph-io/dgo/v210"
)

type CatalogRepository interface {
	// GetAssertionsForObject returns all catalog assertions where
	// subject=projectUID and object=objectUID.
	GetAssertionsForObject(
		ctx context.Context,
		tx *dgo.Txn,
		projectUID string,
		objectUID string,
	) ([]*core.Assertion, error)

	// ExistsInCatalog returns true if the entity has at least one
	// active (non-invalidated, non-expired) catalog assertion.
	ExistsInCatalog(
		ctx context.Context,
		tx *dgo.Txn,
		projectUID string,
		objectUID string,
	) (bool, error)

	// HasDiscoveredParent returns true if the entity has at least one
	// active catalog assertion that is NOT has_orphaned_entity —
	// meaning its parent in the AD hierarchy is already known.
	HasDiscoveredParent(
		ctx context.Context,
		tx *dgo.Txn,
		projectUID string,
		objectUID string,
	) (bool, error)
}

type DgraphCatalogRepository struct {
	DB *dgo.Dgraph
}

func NewCatalogRepository(db *dgo.Dgraph) *DgraphCatalogRepository {
	return &DgraphCatalogRepository{DB: db}
}

// GetByType returns all entities of objectType linked to the project
// via any assertion predicate (full catalog).
func GetByType[T any](
	ctx context.Context,
	tx *dgo.Txn,
	projectUID string,
	objectType string,
	fields []string,
) ([]*res.EntityResult[T], error) {
	if tx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}
	return dgraph.GetEntitiesWithAssertionsNHop[T](
		ctx, tx, projectUID,
		[]dgraph.HopConfig{
			{AnyPredicate: true, ObjectType: objectType},
		},
		fields,
		"catalogGetByType",
	)
}

// GetOrphanedByType returns only orphaned entities of objectType —
// those linked to the project via has_orphaned_entity (parent not yet known).
func GetOrphanedByType[T any](
	ctx context.Context,
	tx *dgo.Txn,
	projectUID string,
	objectType string,
	fields []string,
) ([]*res.EntityResult[T], error) {
	if tx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}
	return dgraph.GetEntitiesWithAssertions[T](
		ctx, tx,
		projectUID,
		core.PredicateHasOrphanedEntity,
		objectType,
		fields,
		"catalogGetOrphaned",
	)
}

// GetAssertionsForObject returns all catalog assertions where the given
// objectUID is the object and the subject is the project.
func (r *DgraphCatalogRepository) GetAssertionsForObject(
	ctx context.Context,
	tx *dgo.Txn,
	projectUID string,
	objectUID string,
) ([]*core.Assertion, error) {
	if tx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}

	query := `query getCatalogAssertions($projectUID: string, $objectUID: string) {
		assertions(func: type(Assertion)) @filter(
			uid_in(assertion.subject, $projectUID) AND
			uid_in(assertion.object, $objectUID)
		) {
			uid
			assertion.predicate
			assertion.method
			assertion.source
			assertion.confidence
			assertion.status
			assertion.timestamp
			assertion.note
			assertion.high_value_marked
			assertion.has_discovered_parent
			assertion.subject { uid }
			assertion.object { uid }
		}
	}`

	resp, err := tx.QueryWithVars(ctx, query, map[string]string{
		"$projectUID": projectUID,
		"$objectUID":  objectUID,
	})
	if err != nil {
		return nil, fmt.Errorf("getCatalogAssertions query failed: %w", err)
	}

	var rawResult struct {
		Assertions []*core.Assertion `json:"assertions"`
	}
	if err := dgraph.UnmarshalResponse(resp.Json, &rawResult); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}

	return rawResult.Assertions, nil
}

// ExistsInCatalog checks whether the given objectUID is already linked
// to the project via any active catalog assertion.
func (r *DgraphCatalogRepository) ExistsInCatalog(
	ctx context.Context,
	tx *dgo.Txn,
	projectUID string,
	objectUID string,
) (bool, error) {
	assertions, err := r.GetAssertionsForObject(ctx, tx, projectUID, objectUID)
	if err != nil {
		return false, err
	}
	for _, a := range assertions {
		if a.Status != core.StatusInvalidated && a.Status != core.StatusExpired {
			return true, nil
		}
	}
	return false, nil
}

// HasDiscoveredParent checks whether any active non-orphaned assertion
// exists for the given objectUID under this project — i.e. its parent is known.
func (r *DgraphCatalogRepository) HasDiscoveredParent(
	ctx context.Context,
	tx *dgo.Txn,
	projectUID string,
	objectUID string,
) (bool, error) {
	assertions, err := r.GetAssertionsForObject(ctx, tx, projectUID, objectUID)
	if err != nil {
		return false, err
	}
	for _, a := range assertions {
		if a.Predicate != core.PredicateHasOrphanedEntity &&
			a.Status != core.StatusInvalidated &&
			a.Status != core.StatusExpired {
			return true, nil
		}
	}
	return false, nil
}
