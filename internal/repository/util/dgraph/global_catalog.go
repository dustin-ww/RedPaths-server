package dgraph

import (
	"RedPaths-server/pkg/model/core/res"
	"context"

	"github.com/dgraph-io/dgo/v210"
)

func GetCatalogEntitiesAllPredicates[T any](
	ctx context.Context,
	tx *dgo.Txn,
	projectUID string,
	objectType string,
	objectFields []string,
) ([]*res.EntityResult[T], error) {
	return GetEntitiesWithAssertionsNHop[T](
		ctx, tx, projectUID,
		[]HopConfig{
			{AnyPredicate: true, ObjectType: objectType},
		},
		objectFields,
		"getCatalogAllPredicates",
	)
}
