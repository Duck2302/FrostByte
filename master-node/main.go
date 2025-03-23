package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
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

func main() {
	http.HandleFunc("/register", registerWorker)
	http.HandleFunc("/workers", listWorkers)
	http.HandleFunc("/test", testWorker)
	http.HandleFunc("/upload", uploadFile)

	fmt.Println("Master node listening on :8080")
	http.ListenAndServe(":8080", nil)
}
