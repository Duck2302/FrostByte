package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type Worker struct {
	ID string
}

var (
	workers = make(map[string]Worker) // Store workers by ID
	mu      sync.Mutex                // muxtex to protect workers map
)

// Register worker nodes
func registerWorker(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id") // Worker ID (docker id)
	addr := r.RemoteAddr          // Worker Address (inside of docker network)

	mu.Lock()
	workers[id] = Worker{ID: id}
	mu.Unlock()

	fmt.Fprintf(w, "Worker %s registered from %s\n", id, addr)
}

func listWorkers(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workers)
}

func testWorker(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	mu.Lock()
	worker, exists := workers[id]
	mu.Unlock()

	if !exists {
		http.Error(w, "Worker not found", http.StatusNotFound)
		return
	}

	resp, err := http.Get(fmt.Sprintf("http://%s:8081/worker-test", worker.ID))
	if err != nil {
		http.Error(w, "Failed to reach worker", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	log.Printf("Response from worker %s: %s", id, resp.Status)
	fmt.Fprintf(w, "Response from worker %s: %s", id, resp.Status)
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	fileData, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read file data", http.StatusInternalServerError)
		return
	}

	chunks := splitFile(fileData, 64*1024*1024) // Split file into MiB sized chunks
	for _, chunk := range chunks {

		workerID := selectWorker(workers)

		err := sendChunkToWorker(filename, workerID, chunk)
		if err != nil {
			http.Error(w, "Failed to send chunk to worker", http.StatusInternalServerError)
			return
		}

	}
}

func downloadFile(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Retrieve the file metadata using the database handler
	fileChunks, err := GetFileMetadata(ctx, filename)
	if err != nil {
		http.Error(w, "Failed to retrieve file metadata", http.StatusInternalServerError)
		return
	}

	// Set headers for file download
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Type", "application/octet-stream")

	// Stream the file chunks to the response
	for chunkID, workerIDs := range fileChunks {
		var chunkData []byte
		for _, workerID := range workerIDs {
			chunkData, err = fetchChunkFromWorker(workerID, chunkID)
			if err == nil {
				break // Successfully fetched the chunk
			}
			log.Printf("Failed to fetch chunk %s from worker %s: %v", chunkID, workerID, err)
		}

		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to fetch chunk %s from all workers", chunkID), http.StatusInternalServerError)
			return
		}

		// Write the chunk data directly to the response
		_, err = w.Write(chunkData)
		if err != nil {
			log.Printf("Failed to write chunk %s to response: %v", chunkID, err)
			return
		}
	}
}

func deleteFile(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Retrieve the file metadata using the database handler
	fileChunks, err := GetFileMetadata(ctx, filename)
	if err != nil {
		http.Error(w, "Failed to retrieve file metadata", http.StatusInternalServerError)
		return
	}

	// Delete the file chunks from workers
	for chunkID, workerIDs := range fileChunks {
		for _, workerID := range workerIDs {
			err = deleteChunkFromWorker(workerID, chunkID)
			if err == nil {
				break // Successfully deleted the chunk
			}
			log.Printf("Failed to delete chunk %s from worker %s: %v", chunkID, workerID, err)
		}

		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete chunk %s from all workers", chunkID), http.StatusInternalServerError)
			return
		}
		log.Printf("Deleted all chunks of the file: %s", filename)
	}

	// Delete the file metadata using the database handler
	err = DeleteFileMetadata(ctx, filename)
	if err != nil {
		http.Error(w, "Failed to delete file metadata", http.StatusInternalServerError)
		return
	}
}

func listFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	filenames, err := GetAllFilenames(ctx)
	if err != nil {
		log.Printf("Failed to retrieve file metadata from database: %v\n", err)
		http.Error(w, "Failed to retrieve file metadata", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(filenames)
}

func main() {
	http.HandleFunc("/register", registerWorker)
	http.HandleFunc("/workers", listWorkers)
	http.HandleFunc("/test", testWorker)
	http.HandleFunc("/upload", uploadFile)
	http.HandleFunc("/download", downloadFile)
	http.HandleFunc("/delete", deleteFile)
	http.HandleFunc("/files", listFiles)

	fmt.Println("Master node listening on :8080")
	http.ListenAndServe(":8080", nil)
}
