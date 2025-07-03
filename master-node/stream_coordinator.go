package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

type StreamCoordinator struct {
	workerManager *WorkerManager
	chunkManager  *ChunkManager
}

func NewStreamCoordinator(wm *WorkerManager, cm *ChunkManager) *StreamCoordinator {
	return &StreamCoordinator{
		workerManager: wm,
		chunkManager:  cm,
	}
}

func (sc *StreamCoordinator) StreamUpload(filename string, reader io.Reader) error {
	buffer := make([]byte, StreamBufferSize)
	chunkIndex := 0
	var currentStream *StreamConnection

	for {
		bytesRead, err := reader.Read(buffer)
		if err == io.EOF {
			if currentStream != nil {
				sc.closeCurrentStream(filename, currentStream)
			}
			break
		}
		if err != nil {
			if currentStream != nil {
				currentStream.Stream.Close()
			}
			return fmt.Errorf("error reading from client stream: %v", err)
		}

		data := buffer[:bytesRead]

		// Process the data in chunks
		for len(data) > 0 {
			// Start new chunk if needed
			if currentStream == nil || currentStream.BytesWritten >= DefaultChunkSize {
				if currentStream != nil {
					err := sc.closeCurrentStream(filename, currentStream)
					if err != nil {
						return err
					}
				}

				currentStream, err = sc.startNewChunk(filename, chunkIndex)
				if err != nil {
					return err
				}
				chunkIndex++
			}

			// Write as much as possible to current chunk
			remainingInChunk := DefaultChunkSize - currentStream.BytesWritten
			writeSize := int64(len(data))
			if writeSize > remainingInChunk {
				writeSize = remainingInChunk
			}

			written, err := currentStream.Stream.Write(data[:writeSize])
			if err != nil {
				currentStream.Stream.Close()
				return fmt.Errorf("error writing to worker stream: %v", err)
			}

			currentStream.BytesWritten += int64(written)
			data = data[written:]
		}
	}

	log.Printf("Successfully streamed file %s in %d chunks", filename, chunkIndex)
	return nil
}

func (sc *StreamCoordinator) startNewChunk(filename string, chunkIndex int) (*StreamConnection, error) {
	workerID := sc.workerManager.SelectWorker()
	if workerID == "" {
		return nil, fmt.Errorf("no available workers")
	}

	chunkID := sc.chunkManager.generateChunkID(filename, chunkIndex)

	// Create streaming HTTP request to worker
	url := fmt.Sprintf("http://%s:%s/stream-store?chunkID=%s", workerID, DefaultWorkerPort, chunkID)

	// Use a pipe to create streaming connection
	pr, pw := io.Pipe()

	// Start HTTP request in goroutine
	go func() {
		defer pr.Close()
		resp, err := http.Post(url, ContentTypeOctetStream, pr)
		if err != nil {
			log.Printf("Failed to stream to worker %s: %v", workerID, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Worker %s returned error: %s", workerID, resp.Status)
			return
		}

		// Store chunk metadata after successful upload
		err = storeChunkInDB(filename, chunkID, workerID)
		if err != nil {
			log.Printf("Failed to store chunk metadata: %v", err)
		}
	}()

	return &StreamConnection{
		WorkerID:     workerID,
		ChunkID:      chunkID,
		ChunkIndex:   chunkIndex,
		BytesWritten: 0,
		Stream:       pw,
	}, nil
}

func (sc *StreamCoordinator) closeCurrentStream(filename string, stream *StreamConnection) error {
	err := stream.Stream.Close()
	if err != nil {
		log.Printf("Error closing stream to worker %s: %v", stream.WorkerID, err)
		return err
	}

	log.Printf("Completed chunk %s to worker %s (%d bytes)",
		stream.ChunkID, stream.WorkerID, stream.BytesWritten)
	return nil
}
