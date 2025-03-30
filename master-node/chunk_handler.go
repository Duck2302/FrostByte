package main

import (
	"bytes"
	"fmt"
	"io"
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
		return fmt.Errorf("failed to send chunk to worker %s: %v", workerID, err)
	}
	defer resp.Body.Close()

	// Store the chunk information in the database
	err = storeChunkInDB(filename, chunkUUID, workerID)
	if err != nil {
		return fmt.Errorf("failed to store chunk in database: %v", err)
	}

	fmt.Printf("Chunk %s sent to worker %s\n", chunkUUID, workerID)
	return nil
}

func fetchChunkFromWorker(workerID, chunkID string) ([]byte, error) {
	// Fetch the chunk from the worker node
	resp, err := http.Get(fmt.Sprintf("http://%s:8081/get?chunkID=%s", workerID, chunkID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch chunk: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func deleteChunkFromWorker(workerID, chunkID string) error {
	// Create a new DELETE request
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://%s:8081/delete?chunkID=%s", workerID, chunkID), nil)
	if err != nil {
		return err
	}

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete chunk: %s", resp.Status)
	}

	return nil
}
