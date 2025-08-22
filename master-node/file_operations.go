package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type FileOperations struct {
	workerManager *WorkerManager
	chunkManager  *ChunkManager
}

// extractChunkIndex extracts the chunk index from chunk ID for proper sorting
func extractChunkIndex(chunkID string) int {
	// ChunkID format: filename_chunk_########
	parts := strings.Split(chunkID, "_chunk_")
	if len(parts) != 2 {
		return 0 // fallback for malformed chunk IDs
	}

	index, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0 // fallback for invalid index
	}
	return index
}

func NewFileOperations(wm *WorkerManager) *FileOperations {
	cm := NewChunkManager(wm, MaxConcurrentUploads)
	return &FileOperations{
		workerManager: wm,
		chunkManager:  cm,
	}
}

// File splitting functionality (moved from file_handler.go)
func (fo *FileOperations) splitFile(file []byte, chunkSize int) [][]byte {
	if len(file) == 0 {
		return nil
	}

	numChunks := (len(file) + chunkSize - 1) / chunkSize
	chunks := make([][]byte, 0, numChunks)

	for i := 0; i < len(file); i += chunkSize {
		end := i + chunkSize
		if end > len(file) {
			end = len(file)
		}

		// Create a copy to avoid referencing the original slice
		chunk := make([]byte, end-i)
		copy(chunk, file[i:end])
		chunks = append(chunks, chunk)
	}
	return chunks
}

func (fo *FileOperations) uploadFile(w http.ResponseWriter, r *http.Request) {
	filename, err := getRequiredParam(r, "filename")
	if err != nil {
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get file size parameter
	sizeParam, err := getRequiredParam(r, "size")
	if err != nil {
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	fileSize, err := strconv.ParseInt(sizeParam, 10, 64)
	if err != nil {
		writeErrorResponse(w, "Invalid file size parameter", http.StatusBadRequest)
		return
	}

	log.Printf("Request Header: %s", r.Header)
	log.Printf("Uploading file %s with size %d bytes", filename, fileSize)

	// Check Content-Type to determine if it's multipart form data or raw file
	contentType := r.Header.Get("Content-Type")
	var fileReader io.Reader

	if strings.Contains(contentType, "multipart/form-data") {
		// Parse multipart form data
		err := r.ParseMultipartForm(32 << 20) // 32 MB max memory
		if err != nil {
			writeErrorResponse(w, fmt.Sprintf("Failed to parse multipart form: %v", err), http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			writeErrorResponse(w, fmt.Sprintf("Failed to get file from form: %v", err), http.StatusBadRequest)
			return
		}
		defer file.Close()
		fileReader = file
	} else {
		// Raw file data in body
		fileReader = r.Body
	}

	// Use streaming coordinator
	streamCoordinator := NewStreamCoordinator(fo.workerManager, fo.chunkManager)
	err = streamCoordinator.StreamUpload(filename, fileReader, fileSize)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("Streaming upload failed: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("File %s uploaded successfully via streaming", filename)
	writeSuccessResponse(w, fmt.Sprintf("File %s uploaded successfully via streaming", filename))
}

// New method to handle streaming file chunking
func (fo *FileOperations) streamAndChunkFile(filename string, reader io.Reader) error {
	chunkBuffer := make([]byte, DefaultChunkSize)
	chunkIndex := 0
	var wg sync.WaitGroup
	errors := make(chan error, 100) // Buffer for errors
	semaphore := make(chan struct{}, fo.chunkManager.maxConcurrent)

	for {
		// Read chunk from the stream
		bytesRead, err := io.ReadFull(reader, chunkBuffer)

		// Handle end of file or partial read
		if err == io.EOF {
			break
		}
		if err == io.ErrUnexpectedEOF {
			// Last chunk is smaller than buffer size
			chunk := make([]byte, bytesRead)
			copy(chunk, chunkBuffer[:bytesRead])

			wg.Add(1)
			go fo.uploadChunkAsync(filename, chunk, chunkIndex, &wg, errors, semaphore)
			chunkIndex++
			break
		}
		if err != nil {
			return fmt.Errorf("error reading from stream: %v", err)
		}

		// Create a copy of the chunk data
		chunk := make([]byte, bytesRead)
		copy(chunk, chunkBuffer[:bytesRead])

		// Upload chunk asynchronously
		wg.Add(1)
		go fo.uploadChunkAsync(filename, chunk, chunkIndex, &wg, errors, semaphore)
		chunkIndex++
	}

	// Wait for all uploads to complete
	wg.Wait()
	close(errors)

	// Check for errors
	var uploadErrors []string
	for err := range errors {
		uploadErrors = append(uploadErrors, err.Error())
	}

	if len(uploadErrors) > 0 {
		return fmt.Errorf("upload failed: %v", uploadErrors)
	}

	log.Printf("Successfully uploaded %d chunks for file %s", chunkIndex, filename)
	return nil
}

// Helper method for async chunk upload
func (fo *FileOperations) uploadChunkAsync(filename string, chunk []byte, chunkIndex int, wg *sync.WaitGroup, errors chan<- error, semaphore chan struct{}) {
	defer wg.Done()

	// Acquire semaphore
	semaphore <- struct{}{}
	defer func() { <-semaphore }()

	workerID := fo.workerManager.SelectWorker()
	if workerID == "" {
		errors <- fmt.Errorf("no available workers for chunk %d", chunkIndex)
		return
	}

	err := fo.chunkManager.sendChunkToWorker(filename, workerID, chunk, chunkIndex)
	if err != nil {
		errors <- fmt.Errorf("failed to send chunk %d to worker %s: %v", chunkIndex, workerID, err)
		return
	}

	log.Printf("Chunk %d uploaded successfully to worker %s", chunkIndex, workerID)
}

func (fo *FileOperations) downloadFile(w http.ResponseWriter, r *http.Request) {
	if !validateHTTPMethod(w, r, http.MethodGet) {
		return
	}

	filename, err := getDownloadPathParameter(r)
	if err != nil {
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), DatabaseTimeout)
	defer cancel()

	fileChunks, err := GetFileMetadata(ctx, filename)
	if err != nil {
		log.Printf("Failed to retrieve file metadata for %s: %v", filename, err)
		writeErrorResponse(w, "Failed to retrieve file metadata", http.StatusInternalServerError)
		return
	}

	// Sort chunk IDs to ensure correct order
	var chunkIDs []string
	for chunkID := range fileChunks {
		chunkIDs = append(chunkIDs, chunkID)
	}

	// Sort by extracting chunk index for proper numerical ordering
	sort.Slice(chunkIDs, func(i, j int) bool {
		// Extract chunk indices from chunk IDs (format: filename_chunk_########)
		iIndex := extractChunkIndex(chunkIDs[i])
		jIndex := extractChunkIndex(chunkIDs[j])
		return iIndex < jIndex
	})

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Type", ContentTypeOctetStream)

	for _, chunkID := range chunkIDs {
		workerIDs := fileChunks[chunkID]
		var chunkData []byte
		for _, workerID := range workerIDs {
			chunkData, err = fo.chunkManager.fetchChunkFromWorker(workerID, chunkID)
			if err == nil {
				break
			}
			log.Printf("Failed to fetch chunk %s from worker %s: %v", chunkID, workerID, err)
		}

		if err != nil {
			log.Printf("Failed to fetch chunk %s from all workers: %v", chunkID, err)
			writeErrorResponse(w, fmt.Sprintf("Failed to fetch chunk %s from all workers", chunkID), http.StatusInternalServerError)
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

func (fo *FileOperations) deleteFile(w http.ResponseWriter, r *http.Request) {
	filename, err := getRequiredParam(r, "filename")
	if err != nil {
		writeErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), DatabaseTimeout)
	defer cancel()

	fileChunks, err := GetFileMetadata(ctx, filename)
	if err != nil {
		log.Printf("Failed to retrieve file metadata for %s: %v", filename, err)
		writeErrorResponse(w, "Failed to retrieve file metadata", http.StatusInternalServerError)
		return
	}

	for chunkID, workerIDs := range fileChunks {
		for _, workerID := range workerIDs {
			err = fo.chunkManager.deleteChunkFromWorker(workerID, chunkID)
			if err == nil {
				break
			}
			log.Printf("Failed to delete chunk %s from worker %s: %v", chunkID, workerID, err)
		}

		if err != nil {
			log.Printf("Failed to delete chunk %s from all workers: %v", chunkID, err)
			writeErrorResponse(w, fmt.Sprintf("Failed to delete chunk %s from all workers", chunkID), http.StatusInternalServerError)
			return
		}
		log.Printf("Deleted chunk %s for file %s", chunkID, filename)
	}

	err = DeleteFileMetadata(ctx, filename)
	if err != nil {
		log.Printf("Failed to delete file metadata for %s: %v", filename, err)
		writeErrorResponse(w, "Failed to delete file metadata", http.StatusInternalServerError)
		return
	}
	log.Printf("Deleted file metadata for %s", filename)
	writeSuccessResponse(w, fmt.Sprintf("File %s deleted successfully", filename))
}

func (fo *FileOperations) listFiles(w http.ResponseWriter, r *http.Request) {
	if !validateHTTPMethod(w, r, http.MethodGet) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), DatabaseTimeout)
	defer cancel()

	files, err := GetAllFilenames(ctx)
	if err != nil {
		log.Printf("Failed to retrieve file metadata from database: %v", err)
		writeErrorResponse(w, "Failed to retrieve file metadata", http.StatusInternalServerError)
		return
	}

	if err := writeJSONResponse(w, files); err != nil {
		log.Printf("Failed to encode file metadata: %v", err)
		writeErrorResponse(w, "Failed to encode file metadata", http.StatusInternalServerError)
		return
	}
	log.Println("File list with metadata successfully retrieved")
}
