package main

import "time"

const (
	// Server configuration
	DefaultWorkerPort = "8081"
	DefaultMasterPort = "8080"

	// Network configuration
	NetworkTimeout  = 10 * time.Second
	DatabaseTimeout = 10 * time.Second

	// HTTP configuration
	ContentTypeJSON        = "application/json"
	ContentTypeOctetStream = "application/octet-stream"

	// Database configuration
	DatabaseName = "worker.db"
	ChunksTable  = "chunks"
)
