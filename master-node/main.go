package main

import (
	"log"
	"net/http"
	_ "net/http/pprof" // Add this import
)

func main() {
	go func() {
		log.Println("pprof listening on :6060")
		log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
	}()
	server := NewMasterServer()
	if err := server.Start(DefaultMasterPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
