package main

import (
	"io"
	"time"
)

// Worker represents a worker node in the cluster
type Worker struct {
	ID               string
	CurrentChunkSize int64             // Track current chunk being written
	StreamConn       *StreamConnection // Active connection for streaming
}

const (
	// Server configuration
	DefaultMasterPort = "8080"
	DefaultWorkerPort = "8081"

	// File processing configuration
	DefaultChunkSize     = 10 * 1024 * 1024 // 10MB
	MaxConcurrentUploads = 5
	StreamBufferSize     = 32 * 1024 // 32KB buffer for streaming

	// Network configuration
	NetworkTimeout  = 30 * time.Second
	DatabaseTimeout = 30 * time.Second

	// HTTP configuration
	ContentTypeJSON        = "application/json"
	ContentTypeOctetStream = "application/octet-stream"

	// Database configuration
	DatabaseName     = "frostbyte"
	FilesCollection  = "files"
	ChunksCollection = "chunks"
)

// StreamConnection manages active streaming to a worker
type StreamConnection struct {
	WorkerID     string
	ChunkID      string
	ChunkIndex   int
	BytesWritten int64
	Stream       io.WriteCloser
	ResultChan   chan error // Channel to track upload completion
}
