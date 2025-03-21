# dfs-go


# Architecture:
## Client: 
### Interface for users to read/write files.

## Metadata Server (Master Node):
Manages file locations, metadata, and access control.

## Storage Nodes (Worker Nodes):
Store file chunks and serve client requests.


# Roadmap:

- [X] worker registration with master node
- [ ] add store endpoint to worker nodes (link to docker volume)
- [ ] add upload endpoint to master node (distribute to worker nodes and keep ids of which blocks are where)
- [ ] add endpoint for retrieving files