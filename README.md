# RaftKV

RaftKV is a distributed key-value store that uses the Raft consensus algorithm for replication and fault tolerance. It provides a simple HTTP API for key-value operations and supports automatic leader election and cluster membership management.

## Features

- Distributed key-value store with Raft consensus
- HTTP API for key-value operations
- Automatic leader election
- Cluster membership management
- Fault tolerance and data replication
- Persistent storage using BoltDB
- Snapshot support for log compaction

## Prerequisites

- Go 1.24 or later
- Make (for using the Makefile)

## Installation

1. Clone the repository:
```bash
git clone https://github.com/yourusername/raftkv.git
cd raftkv
```

2. Install dependencies:
```bash
go mod download
```

## Building

Build the project using make:
```bash
make build
```

This will create the `raftkv` binary in the `bin` directory.

## Running the Cluster

The project includes a Makefile with targets to run a 3-node cluster locally. Each node runs on different ports:

- Node 1: HTTP API on 8080, Raft on 8001
- Node 2: HTTP API on 8081, Raft on 8002
- Node 3: HTTP API on 8082, Raft on 8003

### Option 1: Start Individual Nodes

You can start each node separately in different terminal windows:

1. Start the first node (leader):
```bash
make run-node1
```

2. Start the second node:
```bash
make run-node2
```

3. Start the third node:
```bash
make run-node3
```

### Option 2: Start Entire Cluster

Alternatively, you can start all nodes with a single command:
```bash
make run-cluster
```

This will start all three nodes in the background. You can check their status using:
```bash
make status
```

### Managing the Cluster

To stop all nodes:
```bash
make stop-cluster
```

To clean up all node data:
```bash
make clean-data
```

## API Usage

### Key-Value Operations

1. Set a value:
```bash
curl -X PUT -d "value1" http://localhost:8080/v1/keys/key1
```

2. Get a value:
```bash
curl http://localhost:8080/v1/keys/key1
```

3. Delete a key:
```bash
curl -X DELETE http://localhost:8080/v1/keys/key1
```

### Cluster Status

Check the status of any node:
```bash
curl http://localhost:8080/v1/status
```

The response will include:
- `leader`: The address of the current leader
- `state`: The current state of the node (Leader/Follower)

## Configuration

Each node is configured using a YAML file in the `configs` directory. The configuration includes:

- Node ID
- Bind address for Raft communication
- Advertise address for Raft communication
- Data directory for persistent storage
- HTTP API address

Example configuration (node1.yaml):
```yaml
node_id: "node1"
bind_addr: "127.0.0.1:8001"
advertise_addr: "127.0.0.1:8001"
data_dir: "data/node1"
http_addr: "127.0.0.1:8080"
```

## Project Structure

```
.
├── bin/                    # Compiled binaries
├── cmd/                    # Command-line applications
│   └── raftkv/            # Main application
├── configs/               # Configuration files
├── data/                  # Data directory (created at runtime)
├── pkg/                   # Core packages
│   ├── api/              # HTTP API implementation
│   ├── raft/             # Raft consensus implementation
│   └── storage/          # Storage implementation
├── Makefile              # Build and run targets
└── README.md            # This file
```

## Development

### Building

```bash
make build
```

### Running Tests

```bash
make test
```

### Cleaning

```bash
make clean
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
