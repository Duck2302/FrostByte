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

	log.Printf("Worker %s registered from %s", id, addr)
	fmt.Fprintf(w, "Worker %s registered from %s\n", id, addr)
}

func listWorkers(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(workers)
	if err != nil {
		log.Printf("Failed to encode workers list: %v", err)
		http.Error(w, "Failed to encode workers list", http.StatusInternalServerError)
	}
}

func testWorker(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	mu.Lock()
	worker, exists := workers[id]
	mu.Unlock()

	if !exists {
		log.Printf("Worker %s not found", id)
		http.Error(w, "Worker not found", http.StatusNotFound)
		return
	}

	resp, err := http.Get(fmt.Sprintf("http://%s:8081/worker-test", worker.ID))
	if err != nil {
		log.Printf("Failed to reach worker %s: %v", id, err)
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
		log.Println("Filename is required")
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	fileData, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read file data: %v", err)
		http.Error(w, "Failed to read file data", http.StatusInternalServerError)
		return
	}

	chunks := splitFile(fileData, 10*1024) // Split file into MiB sized chunks
	for _, chunk := range chunks {
		workerID := selectWorker(workers)

		err := sendChunkToWorker(filename, workerID, chunk)
		if err != nil {
			log.Printf("Failed to send chunk to worker %s: %v", workerID, err)
			http.Error(w, "Failed to send chunk to worker", http.StatusInternalServerError)
			return
		}
		log.Printf("Chunk sent to worker %s for file %s", workerID, filename)
	}
}

func downloadFile(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		log.Println("Filename is required")
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fileChunks, err := GetFileMetadata(ctx, filename)
	if err != nil {
		log.Printf("Failed to retrieve file metadata for %s: %v", filename, err)
		http.Error(w, "Failed to retrieve file metadata", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Type", "application/octet-stream")

	for chunkID, workerIDs := range fileChunks {
		var chunkData []byte
		for _, workerID := range workerIDs {
			chunkData, err = fetchChunkFromWorker(workerID, chunkID)
			if err == nil {
				break
			}
			log.Printf("Failed to fetch chunk %s from worker %s: %v", chunkID, workerID, err)
		}

		if err != nil {
			log.Printf("Failed to fetch chunk %s from all workers: %v", chunkID, err)
			http.Error(w, fmt.Sprintf("Failed to fetch chunk %s from all workers", chunkID), http.StatusInternalServerError)
			return
		}

		_, err = w.Write(chunkData)
		if err != nil {
			log.Printf("Failed to write chunk %s to response: %v", chunkID, err)
			return
		}
		log.Printf("Chunk %s written to response", chunkID)
	}
}

func deleteFile(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		log.Println("Filename is required")
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fileChunks, err := GetFileMetadata(ctx, filename)
	if err != nil {
		log.Printf("Failed to retrieve file metadata for %s: %v", filename, err)
		http.Error(w, "Failed to retrieve file metadata", http.StatusInternalServerError)
		return
	}

	for chunkID, workerIDs := range fileChunks {
		for _, workerID := range workerIDs {
			err = deleteChunkFromWorker(workerID, chunkID)
			if err == nil {
				break
			}
			log.Printf("Failed to delete chunk %s from worker %s: %v", chunkID, workerID, err)
		}

		if err != nil {
			log.Printf("Failed to delete chunk %s from all workers: %v", chunkID, err)
			http.Error(w, fmt.Sprintf("Failed to delete chunk %s from all workers", chunkID), http.StatusInternalServerError)
			return
		}
		log.Printf("Deleted chunk %s for file %s", chunkID, filename)
	}

	err = DeleteFileMetadata(ctx, filename)
	if err != nil {
		log.Printf("Failed to delete file metadata for %s: %v", filename, err)
		http.Error(w, "Failed to delete file metadata", http.StatusInternalServerError)
		return
	}
	log.Printf("Deleted file metadata for %s", filename)
}

func listFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Println("Method not allowed for listing files")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filenames, err := GetAllFilenames(ctx)
	if err != nil {
		log.Printf("Failed to retrieve file metadata from database: %v", err)
		http.Error(w, "Failed to retrieve file metadata", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(filenames)
	if err != nil {
		log.Printf("Failed to encode filenames: %v", err)
		http.Error(w, "Failed to encode filenames", http.StatusInternalServerError)
	}
	log.Println("File list successfully retrieved")
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
