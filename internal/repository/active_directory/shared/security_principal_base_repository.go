package shared

import (
	"RedPaths-server/internal/repository/util/dgraph"
	"context"

	"github.com/dgraph-io/dgo/v210"
)

type SecurityPrincipalRepository[T any] interface {
	Get(ctx context.Context, tx *dgo.Txn, uid string) (*T, error)
	Create(ctx context.Context, tx *dgo.Txn, incomingSecurityPrincipal *T, actor string) (*T, error)

	FindByUID(ctx context.Context, uid string) (*T, error)
	FindBySID(ctx context.Context, sid string) (*T, error)
	FindByDomain(ctx context.Context, domainUID string) (*dgraph.ExistenceResult[*T], error)
	Delete(ctx context.Context, uid string) error
}
