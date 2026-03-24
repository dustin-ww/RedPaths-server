package catalog

import (
	"RedPaths-server/internal/db"
	"RedPaths-server/internal/repository/redpaths/engine"
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/core/res"
	"RedPaths-server/pkg/model/utils"
	"RedPaths-server/pkg/schema"
	"context"
	"fmt"
	"time"

	"github.com/dgraph-io/dgo/v210"
)

// -----------------------------------------------------------------------------
// CatalogService
// -----------------------------------------------------------------------------

type CatalogService struct {
	db            *dgo.Dgraph
	catalogRepo   engine.CatalogRepository
	assertionRepo engine.AssertionRepository
}

func NewCatalogService(
	db *dgo.Dgraph,
) *CatalogService {

	catalogRepo := engine.NewCatalogRepository(db)
	assertionRepo := engine.NewDgraphAssertionRepository(db)

	return &CatalogService{
		db:            db,
		catalogRepo:   catalogRepo,
		assertionRepo: assertionRepo,
	}
}

// -----------------------------------------------------------------------------
// Add
// -----------------------------------------------------------------------------

// AddToCatalog links an entity to the project-level global catalog.
//
// If incomingAssertion.HasDiscoveredParent is false, the entity is added
// as orphaned (predicate: has_orphaned_entity, status: orphaned).
// Otherwise it is added with the predicate from the incoming assertion.
//
// Returns early without creating a duplicate if the entity is already
// present in the catalog with an active assertion.
func AddToCatalog(
	ctx context.Context,
	s *CatalogService,
	projectUID string,
	objectUID string,
	objectType string,
	incomingAssertion *core.Assertion,
	actor string,
) (*core.Assertion, error) {

	var created *core.Assertion

	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		// Idempotency check — skip if already in catalog
		exists, err := s.catalogRepo.ExistsInCatalog(ctx, tx, projectUID, objectUID)
		if err != nil {
			return fmt.Errorf("catalog existence check failed: %w", err)
		}
		if exists {
			return nil
		}

		predicate := incomingAssertion.Predicate
		status := core.StatusValidated

		if !incomingAssertion.HasDiscoveredParent {
			predicate = core.PredicateHasOrphanedEntity
			status = core.StatusOrphaned
		}

		catalogAssertion := &core.Assertion{
			Predicate:           predicate,
			Method:              incomingAssertion.Method,
			Source:              actor,
			Confidence:          incomingAssertion.Confidence,
			Status:              status,
			Timestamp:           time.Now(),
			HasDiscoveredParent: incomingAssertion.HasDiscoveredParent,
			MarkedAsHighValue:   incomingAssertion.MarkedAsHighValue,
			Subject:             &utils.UIDRef{UID: projectUID, Type: "Project"},
			Object:              &utils.UIDRef{UID: objectUID, Type: objectType},
		}

		created, err = s.assertionRepo.Create(ctx, tx, catalogAssertion)
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("AddToCatalog failed: %w", err)
	}

	return created, nil
}

// -----------------------------------------------------------------------------
// Promote
// -----------------------------------------------------------------------------

// PromoteInCatalog transitions an orphaned entity to a placed entity once its
// parent (e.g. Domain) has been discovered.
//
// Steps:
//  1. Invalidate the existing has_orphaned_entity assertion.
//  2. Create a new catalog assertion with the correct predicate and
//     HasDiscoveredParent=true.
func PromoteInCatalog(
	ctx context.Context,
	s *CatalogService,
	projectUID string,
	objectUID string,
	objectType string,
	realPredicate core.Predicate,
	actor string,
) error {

	return db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		// Check if parent is already known — skip if already promoted
		hasParent, err := s.catalogRepo.HasDiscoveredParent(ctx, tx, projectUID, objectUID)
		if err != nil {
			return fmt.Errorf("HasDiscoveredParent check failed: %w", err)
		}
		if hasParent {
			return nil
		}

		// Step 1: Invalidate orphaned assertions for this object
		assertions, err := s.catalogRepo.GetAssertionsForObject(ctx, tx, projectUID, objectUID)
		if err != nil {
			return fmt.Errorf("fetching catalog assertions failed: %w", err)
		}

		for _, a := range assertions {
			if a.Predicate == core.PredicateHasOrphanedEntity &&
				a.Status != core.StatusInvalidated {
				_, err := s.assertionRepo.Update(ctx, tx, a.UID, map[string]interface{}{
					"assertion.status": string(core.StatusInvalidated),
				})
				if err != nil {
					return fmt.Errorf("invalidating orphaned assertion %s failed: %w", a.UID, err)
				}
			}
		}

		// Step 2: Create new placed catalog assertion
		_, err = s.assertionRepo.Create(ctx, tx, &core.Assertion{
			Predicate:           realPredicate,
			Method:              core.MethodPromotion,
			Source:              actor,
			Confidence:          1.0,
			Status:              core.StatusValidated,
			Timestamp:           time.Now(),
			HasDiscoveredParent: true,
			MarkedAsHighValue:   false,
			Subject:             &utils.UIDRef{UID: projectUID, Type: "Project"},
			Object:              &utils.UIDRef{UID: objectUID, Type: objectType},
		})
		return err
	})
}

// -----------------------------------------------------------------------------
// Remove
// -----------------------------------------------------------------------------

// RemoveFromCatalog invalidates all active catalog assertions for the given
// entity. Does not hard-delete — preserves audit trail.
func RemoveFromCatalog(
	ctx context.Context,
	s *CatalogService,
	projectUID string,
	objectUID string,
) error {

	return db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		assertions, err := s.catalogRepo.GetAssertionsForObject(ctx, tx, projectUID, objectUID)
		if err != nil {
			return fmt.Errorf("fetching catalog assertions failed: %w", err)
		}

		for _, a := range assertions {
			if a.Status == core.StatusInvalidated || a.Status == core.StatusExpired {
				continue
			}
			_, err := s.assertionRepo.Update(ctx, tx, a.UID, map[string]interface{}{
				"assertion.status": string(core.StatusInvalidated),
			})
			if err != nil {
				return fmt.Errorf("invalidating assertion %s failed: %w", a.UID, err)
			}
		}

		return nil
	})
}

// -----------------------------------------------------------------------------
// GetProjectActiveDirectory
// -----------------------------------------------------------------------------

// GetFromCatalog returns all entities of objectType linked to the project
// via any active assertion (full catalog, orphaned + placed).
func GetFromCatalog[T any](
	ctx context.Context,
	s *CatalogService,
	projectUID string,
	objectType string,
) ([]*res.EntityResult[T], error) {

	entitySchema, err := schema.Get(objectType)
	if err != nil {
		return nil, fmt.Errorf("GetFromCatalog: %w", err)
	}

	var result []*res.EntityResult[T]

	err = db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		var err error
		result, err = engine.GetByType[T](ctx, tx, projectUID, objectType, entitySchema.DefaultFields)
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("GetFromCatalog[%s] failed: %w", objectType, err)
	}

	return result, nil
}

// GetOrphanedFromCatalog returns only entities of objectType that are
// orphaned (no parent discovered yet).
// GetOrphanedFromCatalog
func GetOrphanedFromCatalog[T any](
	ctx context.Context,
	s *CatalogService,
	projectUID string,
	objectType string,
) ([]*res.EntityResult[T], error) {

	entitySchema, err := schema.Get(objectType)
	if err != nil {
		return nil, fmt.Errorf("GetOrphanedFromCatalog: %w", err)
	}

	var result []*res.EntityResult[T]

	err = db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		var err error
		result, err = engine.GetOrphanedByType[T](ctx, tx, projectUID, objectType, entitySchema.DefaultFields)
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("GetOrphanedFromCatalog[%s] failed: %w", objectType, err)
	}

	return result, nil
}

// -----------------------------------------------------------------------------
// Mark as High Value
// -----------------------------------------------------------------------------

// MarkAsHighValue sets high_value_marked=true on all active catalog assertions
// for the given entity.
func MarkAsHighValue(
	ctx context.Context,
	s *CatalogService,
	projectUID string,
	objectUID string,
) error {

	return db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		assertions, err := s.catalogRepo.GetAssertionsForObject(ctx, tx, projectUID, objectUID)
		if err != nil {
			return fmt.Errorf("fetching catalog assertions failed: %w", err)
		}

		for _, a := range assertions {
			if a.Status == core.StatusInvalidated || a.Status == core.StatusExpired {
				continue
			}
			_, err := s.assertionRepo.Update(ctx, tx, a.UID, map[string]interface{}{
				"assertion.high_value_marked": true,
			})
			if err != nil {
				return fmt.Errorf("marking assertion %s as HVT failed: %w", a.UID, err)
			}
		}

		return nil
	})
}
