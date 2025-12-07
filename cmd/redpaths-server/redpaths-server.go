package main

import (
	"RedPaths-server/internal/db"
	"RedPaths-server/internal/rest"
	plugin "RedPaths-server/pkg/module_exec"
	"RedPaths-server/pkg/sse"
	"fmt"
	"log"
	// To load
	_ "RedPaths-server/modulelib/attacks"
	_ "RedPaths-server/modulelib/enumeration"
)

func main() {
	fmt.Printf("\n....WELCOME TO....\n\n\n")
	fmt.Println("RedPaths Server")
	fmt.Printf("Initializing Postgres Database Connection")

	postgres, err := db.GetPostgresDB()
	if err != nil {
		log.Fatalf("Failed to initialize Postgres: %v", err)
	}

	dgraph, err := db.GetDgraphDB()
	if err != nil {
		log.Fatalf("Failed to initialize Dgraph DB: %v", err)
	}

	go sse.StartServer("8082", postgres)
	err = plugin.InitializeRegistry(postgres, dgraph)
	if err != nil {
		log.Fatalf("Failed to initialize plugin registry: %v", err)
	}
	err = plugin.CompleteRegistration()
	if err != nil {
		log.Fatalf("Failed to complete plugin registration: %v", err)
	}
	rest.StartServer("8081", postgres, dgraph)
}
