# dfs-go


# Architecture:
## Client: 
### Interface for users to read/write files.

## Master Node (Metadata Server):
Manages file locations, metadata, serve client requests and access control.

## Worker Nodes (Storage Nodes):
Store file chunks, serve master node requests.
