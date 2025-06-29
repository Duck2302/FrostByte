package main

import (
	"log"
)

func main() {
	server := NewMasterServer()
	if err := server.Start(DefaultMasterPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
