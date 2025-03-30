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

func getChunk(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "store endpoint only accepts GET requests", http.StatusMethodNotAllowed)
		return
	}
	chunkID := r.URL.Query().Get("chunkID")

	chunkData, err := getChunkFromDB(chunkID)
	if err != nil {
		http.Error(w, "Failed to store chunk in database", http.StatusInternalServerError)
		return
	}

	log.Printf("Chunk %s retrieved successfully", chunkID)
	w.Header().Set("Content-Type", "application/octet-stream")
	_, err = w.Write(chunkData)
	if err != nil {
		log.Println("Failed to write chunk data")
	}
	log.Printf("Chunk %s successfully retrieved from database", chunkID)
	fmt.Fprintf(w, "Chunk %s successfully retrieved from database", chunkID)
}

func deleteFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		http.Error(w, "store endpoint only accepts DELETE requests", http.StatusMethodNotAllowed)
		return
	}
	chunkID := r.URL.Query().Get("chunkID")

	err := deleteChunkFromDB(chunkID)
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to delete chunk from database", http.StatusInternalServerError)
		return
	}
	log.Printf("Chunk %s deleted successfully from database", chunkID)
	fmt.Fprintf(w, "Chunk %s successfully deleted from database", chunkID)
}

func main() {
	registerWithMaster()

	http.HandleFunc("/worker-test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Worker %s responding", os.Getenv("HOSTNAME"))
	})

	http.HandleFunc("/store", storeChunk)
	http.HandleFunc("/get", getChunk)
	http.HandleFunc("/delete", deleteFile)
	http.ListenAndServe(":8081", nil)
}
