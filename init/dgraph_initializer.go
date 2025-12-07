package init

import (
	_ "embed"
	"log"
	"time"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"golang.org/x/net/context"
)

//go:embed redpaths.schema
var dgraphSchema string

func InitializeDgraphSchema(db *dgo.Dgraph) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("Dgraph: Using embedded schema")

	op := &api.Operation{
		Schema: dgraphSchema,
	}

	err := db.Alter(context.Background(), op)
	if err != nil {
		log.Fatalf("Dgraph: Failed to alter schema: %v", err)
	}

	txn := db.NewTxn()
	defer txn.Discard(ctx)
}
