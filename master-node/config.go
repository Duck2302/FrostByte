package main

import "time"

// Worker represents a worker node in the cluster
type Worker struct {
	ID string
}

const (
	// Server configuration
	DefaultMasterPort = "8080"
	DefaultWorkerPort = "8081"

	// File processing configuration
	DefaultChunkSize     = 64 * 1024 * 1024 // 64MB
	MaxConcurrentUploads = 5

	// Network configuration
	NetworkTimeout  = 10 * time.Second
	DatabaseTimeout = 10 * time.Second

	// HTTP configuration
	ContentTypeJSON        = "application/json"
	ContentTypeOctetStream = "application/octet-stream"

	// Database configuration
	DatabaseName     = "frostbyte"
	FilesCollection  = "files"
	ChunksCollection = "chunks"
)
