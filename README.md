# FrostByte

**FrostByte** is a distributed file storage system built with Go and Docker. It automatically splits files into chunks and distributes them across multiple worker nodes for redundancy and fault tolerance.

## Architecture

### Client
Interface for users to read/write files.

### Master Node (Metadata Server)
Manages file locations, metadata, serves client requests and access control.

### Worker Nodes (Storage Nodes)
Store file chunks, serve master node requests.

## API Endpoints

- **Upload File**: `POST localhost:8080/upload?filename=test2.mp4`
- **List Files**: `GET localhost:8080/files`
- **Download File**: `POST localhost:8080/download?filename=test.pdf`
- **Delete File**: `GET localhost:8080/delete?filename=test.mp4`

## Quick Start

1. **Start the system**:
   ```bash
   docker-compose up --build
   ```

2. **Upload a file**:
   ```bash
   curl -X POST -T your_file.txt "localhost:8080/upload?filename=your_file.txt"
   ```

3. **List files**:
   ```bash
   curl localhost:8080/files
   ```

4. **Download a file**:
   ```bash
   curl -X POST "localhost:8080/download?filename=your_file.txt" -o downloaded_file.txt
   ```

## Features

- **Automatic file chunking** (10KB chunks)
- **Distributed storage** across 5 worker nodes
- **MongoDB metadata storage**
- **Docker containerized** deployment
- **REST API** for file operations


## pprof commands

   ### For simple profiling with web view
   ```bash
    go tool pprof -http=":6060" http://localhost:6060/debug/pprof/heap
   ```

      ```bash
    go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
   ```

   ### For tool call from commandline
         ```bash
    go tool pprof http://localhost:6060/debug/pprof/heap 
   ```

   ### For online visualizer:

   https://pprofweb.evanjones.ca