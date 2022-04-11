# GO-DAG-FSD2L
This repository is for the implementation for the FS2DL system, a route verification framework for the BGP protocol. Look at our paper for more details - Coming soon!


## Architecture of the system

### Types of nodes 
- Blockchain Node
- Discovery Node
- simulation Node


## Usage

### Prerequisites

- golang (version > 1.08) installed
- Discovery node up and running
- provide the address of the discovery node in bootstrapNodes.txt file

### Building the Source 

You can build the blockchain node by running :
> go build main.go 

A sample DockerFile is provided in the repository to run the node in a docker container.

### Running Discovery Node
The code for discovery node is provided in the DiscoveryNode directory in the repo. Build the discoveryService.go file by running
> go build discoveryService.go

To run the discovery node we need to provide port number, max nodes in a shard as command line arguments
> ./discoveryService port_number max_nodes

### Running a simNode 

simulation Node generates BGP routing messages like pathAnnounceMessage, prefixAllocateMessage at random to simulate a real world BGP network. The sim Node is coded in python and can be run by
> python3 simNode.py
