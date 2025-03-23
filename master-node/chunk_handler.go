package main

import (
	"bytes"
	"fmt"
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
