# FrostByte

**FrostByte** is a distributed file storage system built with Go and Docker. It automatically splits files into chunks and distributes them across multiple worker nodes for redundancy and fault tolerance.

## Architecture

### Web-UI
Interface for users to read/write files.

### Master Node (Metadata Server)
Manages file locations, metadata, serves client requests and access control.

### Worker Nodes (Storage Nodes)
Store file chunks, serve master node requests.
## Quick Start

1. **Start the system**:
   ```bash
   docker-compose up -d --build
   ```

2. **Open the interface**:
   ![FrostByte-Web-UI](./images/Web-UI-FrostByte.gif)



## Features

- **Automatic file chunking**
- **Distributed storage** across 5 worker nodes by default
- **MongoDB metadata storage**
- **Docker containerized** deployment
- **REST API** for file operations
- **Web-UI** for easy usage



## API Endpoints (internally used)

- **Upload File (binary)**  
  `POST http://localhost:8080/upload?filename=<filename>`  
  Uploads a file. The file is split into chunks and distributed to worker nodes.

- **List Files**  
  `GET http://localhost:8080/files`  
  Returns a JSON array of all stored filenames.

- **Download File**  
  `GET http://localhost:8080/download/<filename>`  
  Downloads a file by streaming and reassembling its chunks.

- **Delete File**  
  `DELETE http://localhost:8080/delete?filename=<filename>`  
  Deletes a file and its metadata from the system.

---


## pprof commands for profiling

   ### expose ports in the docker-compose file

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
