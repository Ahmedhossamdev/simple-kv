# Simple KV Store - Docker Setup

This directory contains Docker configuration for running a distributed simple-kv cluster.

## Quick Start

1. **Build and start the cluster:**
   ```bash
   docker-compose up -d
   ```

2. **Check cluster status:**
   ```bash
   ./kv-cli.sh status
   ```

3. **Run tests to verify everything works:**
   ```bash
   ./kv-cli.sh test
   ```

## Cluster Architecture

The setup creates a 3-node cluster:

- **Node 1**: `localhost:8081` (Container: `simple-kv-node1`)
- **Node 2**: `localhost:8082` (Container: `simple-kv-node2`)
- **Node 3**: `localhost:8083` (Container: `simple-kv-node3`)

All nodes are connected to each other as peers and will replicate data automatically.

## Using the CLI Tool

### Basic Operations

```bash
# Set a key-value pair (on node1 by default)
./kv-cli.sh set mykey myvalue

# Get a value (from any node)
./kv-cli.sh get mykey
./kv-cli.sh -n node2 get mykey

# Delete a key
./kv-cli.sh del mykey
```

### Advanced Usage

```bash
# Target specific nodes
./kv-cli.sh -n node2 set key2 value2
./kv-cli.sh -n node3 get key2

# Check cluster health
./kv-cli.sh status

# Run comprehensive tests
./kv-cli.sh test
```

## Docker Commands

### Cluster Management

```bash
# Start cluster
docker-compose up -d

# Stop cluster
docker-compose down

# View logs from all nodes
docker-compose logs -f

# View logs from specific node
docker-compose logs -f kv-node1

# Restart cluster
docker-compose restart

# Remove everything (including volumes)
docker-compose down -v
```

### Individual Node Management

```bash
# Stop a single node (simulate failure)
docker-compose stop kv-node2

# Start it back
docker-compose start kv-node2

# Restart a single node
docker-compose restart kv-node1
```

### Debugging

```bash
# Execute shell in a container
docker-compose exec kv-node1 sh

# Check container status
docker-compose ps

# View resource usage
docker stats
```

## Testing Scenarios

### 1. Basic Functionality
```bash
./kv-cli.sh test
```

### 2. Node Failure Simulation
```bash
# Set some data
./kv-cli.sh set test_key test_value

# Stop one node
docker-compose stop kv-node2

# Verify data is still accessible from other nodes
./kv-cli.sh -n node1 get test_key
./kv-cli.sh -n node3 get test_key

# Start the node back
docker-compose start kv-node2

# Verify the node rejoins the cluster
./kv-cli.sh -n node2 get test_key
```

### 3. Network Partition Testing
```bash
# Create network isolation (advanced)
docker network disconnect simple-kv_kv-cluster simple-kv-node3
# ... test split-brain scenarios ...
docker network connect simple-kv_kv-cluster simple-kv-node3
```

## Data Persistence

Each node has its own named volume:
- `node1-data` → `/data` in `kv-node1`
- `node2-data` → `/data` in `kv-node2`
- `node3-data` → `/data` in `kv-node3`

*Note: Current implementation stores data in memory. You can extend the store to persist to these volumes.*

## Network Configuration

- **Network**: `simple-kv_kv-cluster` (bridge network)
- **Subnet**: `172.20.0.0/16`
- **Internal Communication**: Nodes communicate via container names (`kv-node1:8080`, etc.)
- **External Access**: Via mapped ports (`8081`, `8082`, `8083`)

## Health Checks

Each container includes health checks that:
- Test if the service is listening on port 8080
- Run every 10 seconds
- Timeout after 5 seconds
- Retry 3 times before marking as unhealthy

## Troubleshooting

### Common Issues

1. **Port already in use:**
   ```bash
   # Check what's using the ports
   lsof -i :8081
   lsof -i :8082
   lsof -i :8083
   ```

2. **Containers not starting:**
   ```bash
   # Check build logs
   docker-compose build --no-cache

   # Check container logs
   docker-compose logs
   ```

3. **Peer connection issues:**
   ```bash
   # Test connectivity between containers
   docker-compose exec kv-node1 nc -z kv-node2 8080
   ```

### Manual Testing

You can also test manually using netcat:

```bash
# Connect to node1
echo "SET manual_key manual_value" | nc localhost 8081

# Check if it replicated to node2
echo "GET manual_key" | nc localhost 8082
```

## Next Steps

Consider these enhancements:
1. **Persistence**: Implement file-based storage to the mounted volumes
2. **Monitoring**: Add Prometheus metrics and Grafana dashboards
3. **Load Balancer**: Add nginx/haproxy for client load balancing
4. **Service Discovery**: Use Consul or etcd for dynamic peer discovery
5. **Security**: Add TLS encryption and authentication
6. **Backup**: Implement periodic data backup strategies
