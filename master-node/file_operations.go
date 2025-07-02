package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
)

type FileOperations struct {
	workerManager *WorkerManager
	chunkManager  *ChunkManager
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

	fileData, err := io.ReadAll(r.Body)
	if err != nil {
		writeErrorResponse(w, "Error reading file data", http.StatusInternalServerError)
		return
	}

	if len(fileData) == 0 {
		writeErrorResponse(w, "Empty file not allowed", http.StatusBadRequest)
		return
	}

	chunks := fo.splitFile(fileData, DefaultChunkSize)

	if err := fo.chunkManager.UploadChunks(filename, chunks); err != nil {
		writeErrorResponse(w, fmt.Sprintf("Upload failed: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("File %s uploaded successfully with %d chunks", filename, len(chunks))
	writeSuccessResponse(w, fmt.Sprintf("File %s uploaded successfully with %d chunks", filename, len(chunks)))
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
	sort.Strings(chunkIDs)

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

	filenames, err := GetAllFilenames(ctx)
	if err != nil {
		log.Printf("Failed to retrieve file metadata from database: %v", err)
		writeErrorResponse(w, "Failed to retrieve file metadata", http.StatusInternalServerError)
		return
	}

	if err := writeJSONResponse(w, filenames); err != nil {
		log.Printf("Failed to encode filenames: %v", err)
		writeErrorResponse(w, "Failed to encode filenames", http.StatusInternalServerError)
		return
	}
	log.Println("File list successfully retrieved")
}
