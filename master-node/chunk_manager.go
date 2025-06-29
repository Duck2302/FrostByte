package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// HTTP client with connection pooling for better performance
var httpClient = &http.Client{
	Timeout: NetworkTimeout,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     30 * time.Second,
	},
}

// ChunkManager handles all chunk-related operations
type ChunkManager struct {
	workerManager *WorkerManager
	maxConcurrent int
}

func NewChunkManager(wm *WorkerManager, maxConcurrent int) *ChunkManager {
	return &ChunkManager{
		workerManager: wm,
		maxConcurrent: maxConcurrent,
	}
}

// Chunk ID generation with improved sanitization
func (cm *ChunkManager) generateChunkID(filename string, chunkIndex int) string {
	safeFilename := cm.sanitizeFilename(filename)
	return fmt.Sprintf("%s_chunk_%08d", safeFilename, chunkIndex) // 8 digits for better sorting
}

func (cm *ChunkManager) sanitizeFilename(filename string) string {
	replacements := map[string]string{
		".": "_", " ": "_", "/": "_", "\\": "_", ":": "_",
		"<": "_", ">": "_", "\"": "_", "|": "_", "?": "_", "*": "_",
	}

	result := filename
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}
	return result
}

// Upload chunks in parallel with improved error handling
func (cm *ChunkManager) UploadChunks(filename string, chunks [][]byte) error {
	var wg sync.WaitGroup
	errors := make(chan error, len(chunks))
	semaphore := make(chan struct{}, cm.maxConcurrent)

	for i, chunk := range chunks {
		wg.Add(1)
		go func(chunkData []byte, chunkIndex int) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			workerID := cm.workerManager.SelectWorker()
			if workerID == "" {
				errors <- fmt.Errorf("no available workers")
				return
			}

			err := cm.sendChunkToWorker(filename, workerID, chunkData, chunkIndex)
			if err != nil {
				errors <- fmt.Errorf("failed to send chunk to worker %s: %v", workerID, err)
				return
			}
		}(chunk, i)
	}

	wg.Wait()
	close(errors)

	// Collect and return errors
	var uploadErrors []string
	for err := range errors {
		uploadErrors = append(uploadErrors, err.Error())
	}

	if len(uploadErrors) > 0 {
		return fmt.Errorf("upload failed: %v", uploadErrors)
	}

	return nil
}

func (cm *ChunkManager) sendChunkToWorker(filename string, workerID string, chunkData []byte, chunkIndex int) error {
	chunkID := cm.generateChunkID(filename, chunkIndex)
	resp, err := httpClient.Post(
		fmt.Sprintf("http://%s:%s/store?chunkID=%s", workerID, DefaultWorkerPort, chunkID),
		ContentTypeOctetStream,
		bytes.NewReader(chunkData),
	)
	if err != nil {
		log.Printf("Failed to send chunk to worker %s: %v", workerID, err)
		return fmt.Errorf("failed to send chunk to worker %s: %v", workerID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Worker %s returned error: %s", workerID, resp.Status)
		return fmt.Errorf("worker %s returned error: %s", workerID, resp.Status)
	}

	// Store the chunk information in the database
	err = storeChunkInDB(filename, chunkID, workerID)
	if err != nil {
		log.Printf("Failed to store chunk in database: %v", err)
		return fmt.Errorf("failed to store chunk in database: %v", err)
	}

	log.Printf("Chunk %s sent to worker %s", chunkID, workerID)
	return nil
}

func (cm *ChunkManager) fetchChunkFromWorker(workerID, chunkID string) ([]byte, error) {
	resp, err := httpClient.Get(fmt.Sprintf("http://%s:%s/get?chunkID=%s", workerID, DefaultWorkerPort, chunkID))
	if err != nil {
		log.Printf("Failed to fetch chunk %s from worker %s: %v", chunkID, workerID, err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Worker %s returned error for chunk %s: %s", workerID, chunkID, resp.Status)
		return nil, fmt.Errorf("failed to fetch chunk: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read chunk data from worker %s: %v", workerID, err)
		return nil, err
	}

	log.Printf("Chunk %s fetched from worker %s", chunkID, workerID)
	return data, nil
}

func (cm *ChunkManager) deleteChunkFromWorker(workerID, chunkID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://%s:%s/delete?chunkID=%s", workerID, DefaultWorkerPort, chunkID), nil)
	if err != nil {
		log.Printf("Failed to create DELETE request for chunk %s: %v", chunkID, err)
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to send DELETE request for chunk %s to worker %s: %v", chunkID, workerID, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Worker %s returned error for DELETE chunk %s: %s", workerID, chunkID, resp.Status)
		return fmt.Errorf("failed to delete chunk: %s", resp.Status)
	}

	log.Printf("Chunk %s deleted from worker %s", chunkID, workerID)
	return nil
}
