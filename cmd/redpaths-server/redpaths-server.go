package main

import (
	"RedPaths-server/internal/db"
	rplog "RedPaths-server/internal/log"
	"RedPaths-server/internal/rest"
	plugin "RedPaths-server/pkg/module_exec"
	"RedPaths-server/pkg/sse"
	"fmt"
	"log"

	// Load modules
	_ "RedPaths-server/modulelib/attacks"
	_ "RedPaths-server/modulelib/enumeration"
)

func main() {

	fmt.Printf("\n....WELCOME TO....\n\n\n")
	fmt.Println("Starting to initialize system logger")

	err := rplog.InitLogger()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	log.Println("Starting redpaths server...")
	log.Println("Initialized logger. Starting to write logs...")

	log.Println("Starting to initializing Postgres Database Connection")
	log.Println("Initializing Postgres Database")

	postgres, err := db.GetPostgresDB()
	if err != nil {
		log.Fatalf("Failed to initialize Postgres database: %v", err)
	}

	dgraph, err := db.GetDgraphDB()
	if err != nil {
		log.Fatalf("Failed to initialize Dgraph database: %v", err)
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
