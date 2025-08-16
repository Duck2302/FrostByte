package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
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

		// Handle EOF - process any remaining data first
		if err == io.EOF {
			// Process any remaining data in the buffer
			if bytesRead > 0 {
				data := buffer[:bytesRead]
				// Process the remaining data in chunks
				for len(data) > 0 {
					// Start new chunk if needed
					if currentStream == nil || currentStream.BytesWritten >= DefaultChunkSize {
						if currentStream != nil {
							closeErr := sc.closeCurrentStream(filename, currentStream)
							if closeErr != nil {
								return closeErr
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

			// Close the current stream if any
			if currentStream != nil {
				err := sc.closeCurrentStream(filename, currentStream)
				if err != nil {
					return err
				}
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

	// Create a channel to track the upload result
	resultChan := make(chan error, 1)

	// Start HTTP request in goroutine
	go func() {
		defer pr.Close()
		resp, err := http.Post(url, ContentTypeOctetStream, pr)
		if err != nil {
			log.Printf("Failed to stream to worker %s: %v", workerID, err)
			resultChan <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			err := fmt.Errorf("worker %s returned error: %s", workerID, resp.Status)
			log.Printf("%v", err)
			resultChan <- err
			return
		}

		// Store chunk metadata after successful upload
		err = storeChunkInDB(filename, chunkID, workerID)
		if err != nil {
			log.Printf("Failed to store chunk metadata: %v", err)
			resultChan <- err
			return
		}

		resultChan <- nil
	}()

	return &StreamConnection{
		WorkerID:     workerID,
		ChunkID:      chunkID,
		ChunkIndex:   chunkIndex,
		BytesWritten: 0,
		Stream:       pw,
		ResultChan:   resultChan,
	}, nil
}

func (sc *StreamCoordinator) closeCurrentStream(filename string, stream *StreamConnection) error {
	err := stream.Stream.Close()
	if err != nil {
		log.Printf("Error closing stream to worker %s: %v", stream.WorkerID, err)
		return err
	}

	// Wait for the upload goroutine to complete and check for errors
	if stream.ResultChan != nil {
		select {
		case uploadErr := <-stream.ResultChan:
			if uploadErr != nil {
				log.Printf("Upload failed for chunk %s to worker %s: %v",
					stream.ChunkID, stream.WorkerID, uploadErr)
				return uploadErr
			}
		case <-time.After(30 * time.Second): // Add timeout
			return fmt.Errorf("timeout waiting for chunk %s upload to complete", stream.ChunkID)
		}
	}

	log.Printf("Completed chunk %s to worker %s (%d bytes)",
		stream.ChunkID, stream.WorkerID, stream.BytesWritten)
	return nil
}
