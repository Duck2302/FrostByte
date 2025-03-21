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
- [X] add store endpoint to worker nodes (store in database)
- [ ] add upload endpoint to master node (distribute to worker nodes and keep ids of which blocks are where)
- [ ] add endpoint for retrieving files to master node (load chunks from all the nodes and put the original file back together)

## Later:

- [ ] proper logging