package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/google/uuid"
)

func generateChunkUUID() string {
	return uuid.NewString()
}

func sendChunkToWorker(filename string, workerID string, chunkData []byte) error {
	chunkUUID := generateChunkUUID()
	resp, err := http.Post(fmt.Sprintf("http://%s:8081/store?chunkID=%s", workerID, chunkUUID), "application/octet-stream", bytes.NewReader(chunkData))
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
	err = storeChunkInDB(filename, chunkUUID, workerID)
	if err != nil {
		log.Printf("Failed to store chunk in database: %v", err)
		return fmt.Errorf("failed to store chunk in database: %v", err)
	}

	log.Printf("Chunk %s sent to worker %s", chunkUUID, workerID)
	return nil
}

func fetchChunkFromWorker(workerID, chunkID string) ([]byte, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:8081/get?chunkID=%s", workerID, chunkID))
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

func deleteChunkFromWorker(workerID, chunkID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://%s:8081/delete?chunkID=%s", workerID, chunkID), nil)
	if err != nil {
		log.Printf("Failed to create DELETE request for chunk %s: %v", chunkID, err)
		return err
	}

	resp, err := http.DefaultClient.Do(req)
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
