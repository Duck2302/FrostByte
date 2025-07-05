package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type WorkerServer struct {
	storage  ChunkStorage
	hostname string
}

func NewWorkerServer(baseDir string) (*WorkerServer, error) {
	storage, err := NewFileChunkStorage(baseDir)
	if err != nil {
		return nil, err
	}

	hostname, _ := os.Hostname()
	return &WorkerServer{
		storage:  storage,
		hostname: hostname,
	}, nil
}

func (ws *WorkerServer) registerWithMaster() {
	masterURL := fmt.Sprintf("http://master:%s/register?id=%s", DefaultMasterPort, ws.hostname)

	maxRetries := 10
	retryDelay := 2 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("Attempting to register with master (attempt %d/%d)...", attempt, maxRetries)

		resp, err := http.Get(masterURL)
		if err != nil {
			log.Printf("Failed to register with master (attempt %d): %v", attempt, err)
			if attempt < maxRetries {
				log.Printf("Retrying in %v...", retryDelay)
				time.Sleep(retryDelay)
				continue
			}
			log.Fatalf("Failed to register with master after %d attempts: %v", maxRetries, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Printf("Successfully registered with master: %s", ws.hostname)
			return
		} else {
			log.Printf("Master returned error status: %s (attempt %d)", resp.Status, attempt)
			if attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			log.Fatalf("Failed to register with master after %d attempts: status %s", maxRetries, resp.Status)
		}
	}
}

func (ws *WorkerServer) handleStoreChunk(w http.ResponseWriter, r *http.Request) {
	if !validateHTTPMethod(w, r, http.MethodPost) {
		return
	}

	chunkID, err := getRequiredParam(r, "chunkID")
	if err != nil {
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = ws.storage.StoreStream(chunkID, r.Body)
	if err != nil {
		writeErrorResponse(w, "Failed to store chunk in file", http.StatusInternalServerError)
		return
	}

	log.Printf("Chunk %s stored successfully", chunkID)
	writeSuccessResponse(w, fmt.Sprintf("Chunk %s stored successfully", chunkID))
}

func (ws *WorkerServer) handleGetChunk(w http.ResponseWriter, r *http.Request) {
	if !validateHTTPMethod(w, r, http.MethodGet) {
		return
	}

	chunkID, err := getRequiredParam(r, "chunkID")
	if err != nil {
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	chunkData, err := ws.storage.Retrieve(chunkID)
	if err != nil {
		writeErrorResponse(w, "Failed to retrieve chunk from database", http.StatusInternalServerError)
		return
	}

	if chunkData == nil {
		writeErrorResponse(w, "Chunk not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", ContentTypeOctetStream)
	_, err = w.Write(chunkData)
	if err != nil {
		log.Printf("Failed to write chunk %s data: %v", chunkID, err)
		return
	}

	log.Printf("Chunk %s retrieved successfully", chunkID)
}

func (ws *WorkerServer) handleDeleteChunk(w http.ResponseWriter, r *http.Request) {
	if !validateHTTPMethod(w, r, http.MethodDelete) {
		return
	}

	chunkID, err := getRequiredParam(r, "chunkID")
	if err != nil {
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if chunk exists first
	exists, err := ws.storage.Exists(chunkID)
	if err != nil {
		writeErrorResponse(w, "Failed to check chunk existence", http.StatusInternalServerError)
		return
	}

	if !exists {
		writeErrorResponse(w, "Chunk not found", http.StatusNotFound)
		return
	}

	err = ws.storage.Delete(chunkID)
	if err != nil {
		writeErrorResponse(w, "Failed to delete chunk from database", http.StatusInternalServerError)
		return
	}

	log.Printf("Chunk %s deleted successfully", chunkID)
	writeSuccessResponse(w, fmt.Sprintf("Chunk %s successfully deleted from database", chunkID))
}

func (ws *WorkerServer) handleWorkerTest(w http.ResponseWriter, r *http.Request) {
	writeSuccessResponse(w, fmt.Sprintf("Worker %s responding", ws.hostname))
}

func (ws *WorkerServer) handleStreamStore(w http.ResponseWriter, r *http.Request) {
	if !validateHTTPMethod(w, r, http.MethodPost) {
		return
	}

	chunkID, err := getRequiredParam(r, "chunkID")
	if err != nil {
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = ws.storage.StoreStream(chunkID, r.Body)
	if err != nil {
		writeErrorResponse(w, "Failed to store streamed chunk in file", http.StatusInternalServerError)
		log.Printf("Error: %s", err)
		return
	}

	log.Printf("Streamed chunk %s stored successfully", chunkID)
	writeSuccessResponse(w, fmt.Sprintf("Chunk %s stored successfully", chunkID))
}

func (ws *WorkerServer) setupRoutes() {
	http.HandleFunc("/worker-test", ws.handleWorkerTest)
	http.HandleFunc("/store", ws.handleStoreChunk)
	http.HandleFunc("/stream-store", ws.handleStreamStore) // New streaming endpoint
	http.HandleFunc("/get", ws.handleGetChunk)
	http.HandleFunc("/delete", ws.handleDeleteChunk)
}

func (ws *WorkerServer) Start(port string) error {
	ws.registerWithMaster()
	ws.setupRoutes()

	log.Printf("Worker %s listening on :%s", ws.hostname, port)
	return http.ListenAndServe(":"+port, nil)
}

func (ws *WorkerServer) Close() error {
	return ws.storage.Close()
}
