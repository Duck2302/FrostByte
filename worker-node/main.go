package main

import (
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	server, err := NewWorkerServer("./dfs.db")
	if err != nil {
		log.Fatalf("Failed to create worker server: %v", err)
	}
	defer server.Close()

	if err := server.Start(DefaultWorkerPort); err != nil {
		log.Fatalf("Failed to start worker server: %v", err)
	}
}
