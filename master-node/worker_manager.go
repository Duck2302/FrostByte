package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type WorkerManager struct {
	workers      map[string]Worker
	mu           sync.RWMutex
	lastSelected int // For round-robin selection
}

func NewWorkerManager() *WorkerManager {
	return &WorkerManager{
		workers: make(map[string]Worker),
	}
}

func (wm *WorkerManager) AddWorker(id string, worker Worker) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.workers[id] = worker
	log.Printf("Worker %s registered", id)
}

func (wm *WorkerManager) GetWorkers() map[string]Worker {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	// Return a copy to avoid race conditions
	workersCopy := make(map[string]Worker, len(wm.workers))
	for k, v := range wm.workers {
		workersCopy[k] = v
	}
	return workersCopy
}

func (wm *WorkerManager) GetWorker(id string) (Worker, bool) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	worker, exists := wm.workers[id]
	return worker, exists
}

// Improved worker selection with round-robin
func (wm *WorkerManager) SelectWorker() string {
	wm.mu.Lock() // Use write lock since we're updating lastSelected
	defer wm.mu.Unlock()

	if len(wm.workers) == 0 {
		return ""
	}

	// Convert to slice for round-robin selection
	workerIDs := make([]string, 0, len(wm.workers))
	for id := range wm.workers {
		workerIDs = append(workerIDs, id)
	}

	// Round-robin selection
	selectedWorker := workerIDs[wm.lastSelected%len(workerIDs)]
	wm.lastSelected++

	return selectedWorker
}

// SelectWorkerRandom provides random worker selection for load balancing
func (wm *WorkerManager) SelectWorkerRandom() string {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	if len(wm.workers) == 0 {
		return ""
	}

	// Convert to slice for random selection
	workerIDs := make([]string, 0, len(wm.workers))
	for id := range wm.workers {
		workerIDs = append(workerIDs, id)
	}
	// Random selection with crypto/rand for better distribution
	rand.Seed(time.Now().UnixNano())
	return workerIDs[rand.Intn(len(workerIDs))]
}

// Register worker nodes
func (wm *WorkerManager) registerWorker(w http.ResponseWriter, r *http.Request) {
	id, err := getRequiredParam(r, "id")
	if err != nil {
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	addr := r.RemoteAddr
	worker := Worker{ID: id}
	wm.AddWorker(id, worker)

	log.Printf("Worker %s registered from %s", id, addr)
	writeSuccessResponse(w, fmt.Sprintf("Worker %s registered from %s\n", id, addr))
}

func (wm *WorkerManager) listWorkers(w http.ResponseWriter, r *http.Request) {
	workers := wm.GetWorkers()
	if err := writeJSONResponse(w, workers); err != nil {
		log.Printf("Failed to encode workers list: %v", err)
		writeErrorResponse(w, "Failed to encode workers list", http.StatusInternalServerError)
	}
}

func (wm *WorkerManager) testWorker(w http.ResponseWriter, r *http.Request) {
	id, err := getRequiredParam(r, "id")
	if err != nil {
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	worker, exists := wm.GetWorker(id)
	if !exists {
		log.Printf("Worker %s not found", id)
		writeErrorResponse(w, "Worker not found", http.StatusNotFound)
		return
	}

	resp, err := http.Get(fmt.Sprintf("http://%s:%s/worker-test", worker.ID, DefaultWorkerPort))
	if err != nil {
		log.Printf("Failed to reach worker %s: %v", id, err)
		writeErrorResponse(w, "Failed to reach worker", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	log.Printf("Response from worker %s: %s", id, resp.Status)
	writeSuccessResponse(w, fmt.Sprintf("Response from worker %s: %s", id, resp.Status))
}
