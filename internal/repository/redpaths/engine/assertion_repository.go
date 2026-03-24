package engine

import (
	"RedPaths-server/internal/repository/util/dgraph"
	"RedPaths-server/pkg/model/utils"
	"context"
	"fmt"
	"time"

	"RedPaths-server/pkg/model/core"

	"github.com/dgraph-io/dgo/v210"
)

type AssertionRepository interface {
	// CRUD
	Create(ctx context.Context, tx *dgo.Txn, assertion *core.Assertion) (*core.Assertion, error)
	Get(ctx context.Context, tx *dgo.Txn, assertionUID string) (*core.Assertion, error)
	Update(ctx context.Context, tx *dgo.Txn, uid string, fields map[string]interface{}) (*core.Assertion, error)

	// Relations / Queries
	Link(ctx context.Context, tx *dgo.Txn, subjectUID, objectUID string, predicate core.Predicate, method core.Method, actor string, confidence float64) (*core.Assertion, error)
	// Alle Assertions für eine Entity (als Subject oder Object)
	GetAssertionsForEntity(ctx context.Context, tx *dgo.Txn, entityUID string) ([]*core.Assertion, error)

	// Nur wo Entity Subject ist (ausgehende Beziehungen)
	GetAssertionsWhereSubject(ctx context.Context, tx *dgo.Txn, entityUID string) ([]*core.Assertion, error)

	// Nur wo Entity Object ist (eingehende Beziehungen)
	GetAssertionsWhereObject(ctx context.Context, tx *dgo.Txn, entityUID string) ([]*core.Assertion, error)

	// Filtern nach Predicate
	GetAssertionsByPredicate(ctx context.Context, tx *dgo.Txn, entityUID string, predicate core.Predicate) ([]*core.Assertion, error)

	// Filtern nach Status
	GetAssertionsByStatus(ctx context.Context, tx *dgo.Txn, entityUID string, status string) ([]*core.Assertion, error)

	// Filtern nach Source
	GetAssertionsBySource(ctx context.Context, tx *dgo.Txn, entityUID string, source string) ([]*core.Assertion, error)
}

type DgraphAssertionRepository struct {
	DB *dgo.Dgraph
}

func (r *DgraphAssertionRepository) GetAssertionsForEntity(ctx context.Context, tx *dgo.Txn, entityUID string) ([]*core.Assertion, error) {
	//TODO implement me
	panic("implement me")
}

func (r *DgraphAssertionRepository) GetAssertionsWhereSubject(ctx context.Context, tx *dgo.Txn, entityUID string) ([]*core.Assertion, error) {
	//TODO implement me
	panic("implement me")
}

func (r *DgraphAssertionRepository) GetAssertionsWhereObject(ctx context.Context, tx *dgo.Txn, entityUID string) ([]*core.Assertion, error) {
	//TODO implement me
	panic("implement me")
}

func (r *DgraphAssertionRepository) GetAssertionsByPredicate(ctx context.Context, tx *dgo.Txn, entityUID string, predicate core.Predicate) ([]*core.Assertion, error) {
	//TODO implement me
	panic("implement me")
}

func (r *DgraphAssertionRepository) GetAssertionsByStatus(ctx context.Context, tx *dgo.Txn, entityUID string, status string) ([]*core.Assertion, error) {
	//TODO implement me
	panic("implement me")
}

func (r *DgraphAssertionRepository) GetAssertionsBySource(ctx context.Context, tx *dgo.Txn, entityUID string, source string) ([]*core.Assertion, error) {
	//TODO implement me
	panic("implement me")
}

func NewDgraphAssertionRepository(db *dgo.Dgraph) *DgraphAssertionRepository {
	return &DgraphAssertionRepository{DB: db}
}

func (r *DgraphAssertionRepository) Create(ctx context.Context, tx *dgo.Txn, assertion *core.Assertion) (*core.Assertion, error) {
	uid, err := dgraph.OldCreateEntity(ctx, tx, "Assertion", assertion)
	if err != nil {
		return nil, fmt.Errorf("failed to create assertion: %w", err)
	}
	assertion.UID = uid
	return assertion, nil
}

func (r *DgraphAssertionRepository) Get(ctx context.Context, tx *dgo.Txn, assertionUID string) (*core.Assertion, error) {
	query := `
		query Assertion($uid: string) {
			Assertion(func: uid($uid)) {
				uid
				assertion.predicate
				assertion.method
				assertion.source
				assertion.confidence
				assertion.status
				assertion.timestamp
				assertion.subject { uid }
				assertion.object { uid }
				assertion.note
				assertion.high_value_marked
			}
		}`
	return dgraph.GetEntityByUID[core.Assertion](ctx, tx, assertionUID, "assertion", query)
}

func (r *DgraphAssertionRepository) Update(ctx context.Context, tx *dgo.Txn, uid string, fields map[string]interface{}) (*core.Assertion, error) {
	return dgraph.UpdateAndGet(ctx, tx, uid, "", fields, r.Get)
}

func (r *DgraphAssertionRepository) Link(ctx context.Context, tx *dgo.Txn, subjectUID, objectUID string, predicate core.Predicate, method core.Method, actor string, confidence float64) (*core.Assertion, error) {
	assertion := &core.Assertion{
		Subject:    &utils.UIDRef{UID: subjectUID},
		Object:     &utils.UIDRef{UID: objectUID},
		Predicate:  predicate,
		Method:     method,
		Source:     actor,
		Confidence: confidence,
		Status:     "validated",
		Timestamp:  time.Now(),
	}

	return r.Create(ctx, tx, assertion)
}

func (r *DgraphAssertionRepository) GetBySubjectUID(ctx context.Context, tx *dgo.Txn, subjectUID string) ([]*core.Assertion, error) {
	fields := []string{
		"uid",
		"assertion.predicate",
		"assertion.method",
		"assertion.source",
		"assertion.confidence",
		"assertion.status",
		"assertion.timestamp",
		"assertion.object { uid }",
		"assertion.note",
		"assertion.high_value_marked",
	}
	return dgraph.GetEntitiesByRelation[*core.Assertion](ctx, tx, "Assertion", "assertion.subject", subjectUID, fields)
}

func (r *DgraphAssertionRepository) GetByObjectUID(ctx context.Context, tx *dgo.Txn, objectUID string) ([]*core.Assertion, error) {
	fields := []string{
		"uid",
		"assertion.predicate",
		"assertion.method",
		"assertion.source",
		"assertion.confidence",
		"assertion.status",
		"assertion.timestamp",
		"assertion.subject { uid }",
		"assertion.note",
		"assertion.high_value_marked",
	}
	return dgraph.GetEntitiesByRelation[*core.Assertion](ctx, tx, "Assertion", "assertion.object", objectUID, fields)
}
