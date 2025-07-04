package main

import (
	"log"
	"net/http"

	_ "net/http/pprof" // Add this import

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Start pprof server in a goroutine
	go func() {
		log.Println("pprof listening on :6060")
		log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
	}()

	server, err := NewWorkerServer("./dfs.db")
	if err != nil {
		log.Fatalf("Failed to create worker server: %v", err)
	}
	defer server.Close()

	if err := server.Start(DefaultWorkerPort); err != nil {
		log.Fatalf("Failed to start worker server: %v", err)
	}
}
