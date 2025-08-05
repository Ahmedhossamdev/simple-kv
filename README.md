# Simple-KV - Distributed Key-Value Store

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)](LICENSE)
[![Learning](https://img.shields.io/badge/Purpose-Learning-brightgreen?style=for-the-badge)](#learning-objectives)

A simple, lightweight distributed key-value store written in Go with peer-to-peer replication.

> 🎓 **Learning Project**: This project is designed for educational purposes to demonstrate distributed systems concepts including networking, concurrency, and peer-to-peer replication in Go.

## Features

- **TCP-based API** - Connect using telnet or any TCP client
- **Simple Commands** - SET, GET, DEL operations
- **Peer-to-Peer Replication** - Automatic data synchronization across nodes
- **Concurrent Access** - Thread-safe operations with mutex locks
- **Lightweight** - No external dependencies, pure Go implementation

## Learning Objectives

This project demonstrates key concepts in distributed systems:

- 🌐 **Network Programming** - TCP server implementation
- 🔄 **Peer-to-Peer Communication** - Node discovery and message broadcasting
- 🔒 **Concurrency Control** - Thread-safe data structures with mutexes
- 📡 **Data Replication** - Eventual consistency across distributed nodes
- 🏗️ **System Architecture** - Modular design with separation of concerns
- 🛠️ **Go Best Practices** - Goroutines, channels, and error handling

## Quick Start

### Installation

```bash
git clone https://github.com/Ahmedhossamdev/simple-kv.git
cd simple-kv
go mod tidy
```

### Running a Single Node

```bash
go run main.go 8080
```

The server will start on port 8080.

### Running Multiple Nodes (Cluster)

Start the first node:
```bash
go run main.go 8080
```

Start the second node with peer configuration:
```bash
go run main.go 8081 localhost:8080
```

Start the third node:
```bash
go run main.go 8082 localhost:8080,localhost:8081
```

## Usage

### Connecting to the Server

Use telnet or nc to connect:
```bash
telnet localhost 8080
```

### Commands

#### SET - Store a key-value pair
```
SET mykey myvalue
# Response: OK
```

#### GET - Retrieve a value by key
```
GET mykey
# Response: myvalue
```

#### DEL - Delete a key
```
DEL mykey
# Response: OK
```

### Example Session

```
$ telnet localhost 8080
Connected to localhost.
SET name Alice
OK
SET age 25
OK
GET name
Alice
GET age
25
DEL age
OK
GET age
Key not found
```

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Node 1        │    │   Node 2        │    │   Node 3        │
│   Port: 8080    │◄──►│   Port: 8081    │◄──►│   Port: 8082    │
│                 │    │                 │    │                 │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │   Store     │ │    │ │   Store     │ │    │ │   Store     │ │
│ │   (Memory)  │ │    │ │   (Memory)  │ │    │ │   (Memory)  │ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Components

- **Server** (`server/server.go`) - Handles TCP connections and command processing
- **Store** (`store/store.go`) - Thread-safe in-memory storage with mutex locks
- **Peer** (`peer/peer.go`) - Manages peer-to-peer replication
- **Main** (`main.go`) - Entry point and configuration

## API Reference

### Command Format
Commands are text-based and case-insensitive:
```
COMMAND [arguments...]
```

### Response Format
- **Success**: `OK` or the requested value
- **Error**: Error message (e.g., "Key not found", "Usage: GET key")

### Replication
When a SET or DEL operation is performed:
1. The operation is applied locally
2. The change is broadcast to all configured peers
3. Peers apply the change to maintain consistency

## Configuration

### Command Line Arguments

```bash
go run main.go [port] [peers]
```

- **port** (optional) - Port number to listen on (default: 8080)
- **peers** (optional) - Comma-separated list of peer addresses (e.g., "localhost:8081,localhost:8082")

### Examples

Single node:
```bash
go run main.go 8080
```

Node with peers:
```bash
go run main.go 8081 localhost:8080,localhost:8082
```

## Development

### Project Structure

```
simple-kv/
├── main.go           # Entry point
├── go.mod           # Go module definition
├── server/
│   └── server.go    # TCP server and command handling
├── store/
│   └── store.go     # In-memory storage with thread safety
└── peer/
    └── peer.go      # Peer-to-peer replication logic
```

### Building

```bash
go build -o simple-kv main.go
```

```bash
go build -o kvstore main.go
```

### Testing

Start multiple nodes and test replication:

1. Start three nodes:
   ```bash
   # Terminal 1
   go run main.go 8080

   # Terminal 2
   go run main.go 8081 localhost:8080

   # Terminal 3
   go run main.go 8082 localhost:8080,localhost:8081
   ```

2. Connect to any node and set a value:
   ```bash
   telnet localhost 8080
   SET test hello
   ```

3. Connect to another node and verify replication:
   ```bash
   telnet localhost 8081
   GET test
   # Should return: hello
   ```

## Limitations

- **In-Memory Only** - Data is not persisted to disk
- **No Authentication** - Anyone can connect and modify data
- **Basic Replication** - No conflict resolution or leader election
- **No Data Validation** - Keys and values are stored as-is
- **TCP Only** - No HTTP/REST API

## Contributing

This is a learning project and contributions are welcome! Whether you're a beginner or experienced developer:

**For Beginners:**
- 🐛 Report bugs or suggest improvements
- 📝 Improve documentation or add examples
- ❓ Ask questions in Issues - learning discussions are encouraged!

**For Contributors:**
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

**Learning Ideas:**
- Add HTTP REST API alongside TCP
- Implement persistent storage
- Add configuration file support
- Create client libraries in different languages

## Future Improvements

- [ ] Persistent storage (disk-based)
- [ ] HTTP/REST API
- [ ] Authentication and authorization
- [ ] Conflict resolution algorithms
- [ ] Health checks and failure detection
- [ ] Configuration file support
- [ ] Logging and metrics
- [ ] Docker support
- [ ] Clustering improvements (leader election, consensus)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by Redis and other distributed key-value stores
- Built with Go's excellent networking and concurrency primitives

---

**Made with ❤️ in Go**
