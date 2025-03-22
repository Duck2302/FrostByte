package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func registerWithMaster() {
	hostname, _ := os.Hostname() // Get container name
	masterURL := "http://master:8080/register?id=" + hostname

	resp, err := http.Get(masterURL)
	if err != nil {
		log.Fatal("Failed to register with master:", err)
		return
	}
	defer resp.Body.Close()

	log.Println("Registered with master:", hostname)
}

func storeChunk(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "store endpoint only accepts POST requests", http.StatusMethodNotAllowed)
		return
	}
	chunkID := r.URL.Query().Get("chunkID")
	chunkData, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read chunk data", http.StatusInternalServerError)
		return
	}

	err = storeChunkInDB(chunkID, chunkData)
	if err != nil {
		http.Error(w, "Failed to store chunk in database", http.StatusInternalServerError)
		return
	}

	log.Printf("Chunk %s stored successfully", chunkID)
	fmt.Fprintf(w, "Chunk %s stored successfully", chunkID)
}

func main() {
	registerWithMaster()

	http.HandleFunc("/worker-test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Worker %s responding", os.Getenv("HOSTNAME"))
	})

	http.HandleFunc("/store", storeChunk)

	http.ListenAndServe(":8081", nil)
}
