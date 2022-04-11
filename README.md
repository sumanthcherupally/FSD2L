# GO-DAG-FSD2L
This repository is for the implementation for the FS2DL system, a route verification framework for the BGP protocol. Look at our paper for more details - Coming soon!


## Architecture of the system

### Types of nodes 
- Router Node
- Discovery Node
- simulation Node


### Router Node
The node has an RESTFUL API that accepts pathAnnounce, prefixAllocate ... etc BGP routing protocol messages and route them to other nodes in the network.
It also generates a hash(sha256) of the BGP routing protocol message and stores it in the DAG based Blockchain as a transaction, such as other router nodes can verify the integrity of the BGP routing messages. The router nodes also run a DAG based blockchain node inspired from LSDI https://ieeexplore.ieee.org/abstract/document/9334000/.

### Discovery Node
This Node is helpful for creating the network of blockchain nodes and intial connectivity.


### simulation Node
simulation Node generates BGP routing messages like pathAnnounceMessage, prefixAllocateMessage at random to simulate a real world BGP network. These messsages are routed across all the router nodes in the network.


## Usage

### Prerequisites

- golang (version > 1.08) installed
- Discovery node up and running
- provide the address of the discovery node in bootstrapNodes.txt file

### Building the Source 

You can build the node by running :
> go build main.go 

A sample DockerFile is provided in the repository to run the node in a docker container.

### Running Discovery Node
The code for discovery node is provided in the DiscoveryNode directory in the repo. Build the discoveryService.go file by running
> go build discoveryService.go

To run the discovery node we need to provide port number, max nodes in a shard as command line arguments
> ./discoveryService port_number max_nodes

### Running a simNode 
The sim Node is coded in python and can be run by
> python3 simNode.py
