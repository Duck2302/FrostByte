# dfs-go


# Architecture:
## Client: 
### Interface for users to read/write files.

## Master Node (Metadata Server):
Manages file locations, metadata, serve client requests and access control.

## Worker Nodes (Storage Nodes):
Store file chunks, serve master node requests.


# Roadmap:

- [X] worker registration with master node
- [X] add store endpoint to worker nodes (store in database)
- [X] change storage location to sqlite databases
- [X] add upload endpoint to master node (distribute to worker nodes and keep ids of which blocks are where)
- [ ] more efficient and easier to use database schema
- [ ] add a config file for parameters such as chunk size and how much storage worker nodes have
- [ ] maybe make master node keep track of worker nodes storage capacity
- [ ] implement check for available space in workers before distributing chunks (do workers or master keep track?)
- [ ] add endpoint for retrieving files to master node (load chunks from all the nodes and put the original file back together)

## Later:

- [ ] come up with a better name for the project
- [ ] proper logging
- [ ] testing
- [ ] maybe automated testing of the database?
- [ ] documentation
- [ ] better strategy for distributing chunks (total pool and distributed by percentage of free storage)
- [ ] sqlite config to set maximum possible database size (and check if other limits could be reached)
- [ ] add health check to worker nodes (so workers get deleted from master node dict if they are down)
- [ ] implement recovery strategy
- [ ] add limit on storage capacity of worker nodes