package db

import (
	setup "RedPaths-server/init"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
)

var (
	db    *dgo.Dgraph
	once  sync.Once
	dbErr error
)

func GetDgraphDB() (*dgo.Dgraph, error) {
	once.Do(func() {

		// Connection String
		host := os.Getenv("DGRAPH_HOST")
		port := os.Getenv("DGRAPH_PORT")

		dialOpts := append([]grpc.DialOption{},
			grpc.WithInsecure(),
			grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)),
		)
		var err error

		if host == "" {
			host = "localhost"
			port = ":9080"
			log.Println("WARNING: Dgraph host not set, using localhost")
		}

		connectionString := fmt.Sprintf("%s:%s", host, port)

		conn, err := grpc.NewClient(connectionString, dialOpts...)
		if err != nil {
			dbErr = err
			log.Fatal(err)
		}

		db = dgo.NewDgraphClient(api.NewDgraphClient(conn))
		initialized, err := isDgraphInitialized(context.Background(), db)

		if err != nil || !initialized {
			log.Println("WARNING: Dgraph is not initialized yet! Initializing database...")
			setup.InitializeDgraphSchema(db)
		} else {
			log.Println("Dgraph seems to be initialized. Continuing...")
		}
	})

	if dbErr != nil {
		return nil, dbErr
	}

	return db, nil
}

func isDgraphInitialized(ctx context.Context, dg *dgo.Dgraph) (bool, error) {
	// Check for specific initialized predicate
	resp, err := dg.NewReadOnlyTxn().Query(ctx, `
        schema(pred: initialized) {
            type
        }
    `)
	if err != nil {
		return false, fmt.Errorf("schema query for 'initialized' predicate failed: %w", err)
	}

	var schemaResp struct {
		Schema []struct {
			Predicate string `json:"predicate"`
			Type      string `json:"type"`
		} `json:"schema"`
	}

	if err := json.Unmarshal(resp.Json, &schemaResp); err != nil {
		return false, fmt.Errorf("failed to unmarshal schema response: %w", err)
	}

	for _, s := range schemaResp.Schema {
		if s.Predicate == "initialized" {
			return true, nil
		}
	}
	return false, nil
}

func ExecuteInTransaction(ctx context.Context, db *dgo.Dgraph, op func(tx *dgo.Txn) error) error {
	tx := db.NewTxn()
	defer tx.Discard(ctx)

	if err := op(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}
	return nil
}

func ExecuteRead[T any](ctx context.Context, db *dgo.Dgraph, op func(tx *dgo.Txn) (T, error)) (T, error) {
	tx := db.NewReadOnlyTxn()
	defer tx.Discard(ctx)
	return op(tx)
}
