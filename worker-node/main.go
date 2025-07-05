package main

import (
	"log"
	"net/http"

	_ "net/http/pprof"
)

func main() {
	go func() {
		log.Println("pprof listening on :6060")
		log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
	}()

	server, err := NewWorkerServer("./chunks")
	if err != nil {
		log.Fatalf("Failed to create worker server: %v", err)
	}
	defer server.Close()

	if err := server.Start(DefaultWorkerPort); err != nil {
		log.Fatalf("Failed to start worker server: %v", err)
	}
}
