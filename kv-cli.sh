#!/bin/bash

# Simple KV Cluster CLI Tool
# Usage: ./kv-cli.sh [node] [command] [args...]

set -e

# Default configuration
DEFAULT_NODE="node1"
DEFAULT_PORT="8081"

# Node port mapping
declare -A NODE_PORTS
NODE_PORTS["node1"]="8081"
NODE_PORTS["node2"]="8082"
NODE_PORTS["node3"]="8083"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Help function
show_help() {
    echo -e "${BLUE}Simple KV Cluster CLI Tool${NC}"
    echo ""
    echo "Usage: $0 [OPTIONS] COMMAND [ARGS...]"
    echo ""
    echo "Options:"
    echo "  -n, --node NODE     Target node (node1, node2, node3) [default: node1]"
    echo "  -h, --help          Show this help message"
    echo ""
    echo "Commands:"
    echo "  set KEY VALUE       Set a key-value pair"
    echo "  get KEY            Get value for a key"
    echo "  del KEY            Delete a key"
    echo "  sync NODE          Request sync from all peers for a specific node"
    echo "  stats NODE         Get statistics from a node"
    echo "  status             Check cluster status"
    echo "  test               Run a simple test suite"
    echo "  demo               Run failure recovery demo"
    echo ""
    echo "Examples:"
    echo "  $0 set mykey myvalue"
    echo "  $0 -n node2 get mykey"
    echo "  $0 del mykey"
    echo "  $0 sync node2"
    echo "  $0 stats node1"
    echo "  $0 status"
    echo "  $0 test"
    echo "  $0 demo"
}

# Function to send command to node
send_command() {
    local node=$1
    local command=$2
    local port=${NODE_PORTS[$node]}

    if [ -z "$port" ]; then
        echo -e "${RED}Error: Invalid node '$node'${NC}" >&2
        echo "Valid nodes: ${!NODE_PORTS[@]}" >&2
        exit 1
    fi

    # Check if node is reachable
    if ! nc -z localhost "$port" 2>/dev/null; then
        echo -e "${RED}Error: Node '$node' is not reachable on port $port${NC}" >&2
        echo "Make sure the cluster is running: docker-compose up -d" >&2
        exit 1
    fi

    # Send command using netcat
    echo "$command" | nc localhost "$port"
}

# Function to check cluster status
check_status() {
    echo -e "${BLUE}Checking cluster status...${NC}"
    echo ""

    for node in "${!NODE_PORTS[@]}"; do
        port=${NODE_PORTS[$node]}
        echo -n "Node $node (port $port): "

        if nc -z localhost "$port" 2>/dev/null; then
            echo -e "${GREEN}UP${NC}"
        else
            echo -e "${RED}DOWN${NC}"
        fi
    done
}

# Function to run failure recovery demo
run_demo() {
    echo -e "${BLUE}🚀 Failure Recovery Demo${NC}"
    echo ""

    echo -e "${YELLOW}Step 1: Set initial data${NC}"
    send_command "node1" "SET demo_key initial_value"
    echo "✓ Set demo_key=initial_value on node1"

    echo ""
    echo -e "${YELLOW}Step 2: Verify replication to all nodes${NC}"
    for node in "node1" "node2" "node3"; do
        result=$(send_command "$node" "GET demo_key")
        echo "  $node: $result"
    done

    echo ""
    echo -e "${YELLOW}Step 3: Simulate node2 failure${NC}"
    echo "Stopping node2..."
    docker-compose stop kv-node2
    sleep 2

    echo ""
    echo -e "${YELLOW}Step 4: Add data while node2 is down${NC}"
    send_command "node1" "SET missed_data important_value"
    send_command "node1" "SET another_key another_value"
    echo "✓ Added data while node2 was down"

    echo ""
    echo -e "${YELLOW}Step 5: Check remaining nodes have the data${NC}"
    for node in "node1" "node3"; do
        result1=$(send_command "$node" "GET missed_data")
        result2=$(send_command "$node" "GET another_key")
        echo "  $node: missed_data=$result1, another_key=$result2"
    done

    echo ""
    echo -e "${YELLOW}Step 6: Restart node2${NC}"
    echo "Starting node2..."
    docker-compose start kv-node2
    sleep 3

    echo ""
    echo -e "${YELLOW}Step 7: Check if node2 missed the data${NC}"
    result1=$(send_command "node2" "GET missed_data")
    result2=$(send_command "node2" "GET another_key")
    echo "  node2: missed_data=$result1, another_key=$result2"

    if [ "$result1" = "Key not found" ] || [ "$result2" = "Key not found" ]; then
        echo -e "${RED}❌ Node2 is missing data (as expected with current implementation)${NC}"
    else
        echo -e "${GREEN}✓ Node2 has all data${NC}"
    fi

    echo ""
    echo -e "${YELLOW}Step 8: Manual recovery using SYNC${NC}"
    send_command "node2" "SYNC REQUEST"
    sleep 2

    echo ""
    echo -e "${YELLOW}Step 9: Verify recovery${NC}"
    result1=$(send_command "node2" "GET missed_data")
    result2=$(send_command "node2" "GET another_key")
    echo "  node2 after sync: missed_data=$result1, another_key=$result2"

    if [ "$result1" = "important_value" ] && [ "$result2" = "another_value" ]; then
        echo -e "${GREEN}✅ Recovery successful! Node2 is now up-to-date${NC}"
    else
        echo -e "${RED}❌ Recovery failed${NC}"
    fi

    echo ""
    echo -e "${BLUE}Demo completed!${NC}"
}

# Function to run tests
run_tests() {
    echo -e "${BLUE}Running Simple KV Cluster Tests...${NC}"
    echo ""

    # Test 1: Basic SET/GET
    echo -e "${YELLOW}Test 1: Basic SET/GET${NC}"
    send_command "node1" "SET test_key test_value"
    result=$(send_command "node1" "GET test_key")
    if [ "$result" = "test_value" ]; then
        echo -e "${GREEN}✓ Basic SET/GET working${NC}"
    else
        echo -e "${RED}✗ Basic SET/GET failed${NC}"
    fi

    # Test 2: Cross-node replication (give some time for replication)
    echo -e "${YELLOW}Test 2: Cross-node replication${NC}"
    send_command "node1" "SET repl_key repl_value" > /dev/null
    sleep 1  # Wait for replication
    result=$(send_command "node2" "GET repl_key")
    if [ "$result" = "repl_value" ]; then
        echo -e "${GREEN}✓ Cross-node replication working${NC}"
    else
        echo -e "${RED}✗ Cross-node replication failed${NC}"
    fi

    # Test 3: DELETE operation
    echo -e "${YELLOW}Test 3: DELETE operation${NC}"
    send_command "node1" "SET del_key del_value" > /dev/null
    send_command "node2" "DEL del_key" > /dev/null
    sleep 1  # Wait for replication
    result=$(send_command "node3" "GET del_key")
    if [ "$result" = "Key not found" ]; then
        echo -e "${GREEN}✓ DELETE operation working${NC}"
    else
        echo -e "${RED}✗ DELETE operation failed${NC}"
    fi

    echo ""
    echo -e "${BLUE}Tests completed!${NC}"
}

# Parse arguments
NODE="$DEFAULT_NODE"
COMMAND=""
ARGS=()

while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--node)
            NODE="$2"
            shift 2
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            if [ -z "$COMMAND" ]; then
                COMMAND="$1"
            else
                ARGS+=("$1")
            fi
            shift
            ;;
    esac
done

# Check if command is provided
if [ -z "$COMMAND" ]; then
    echo -e "${RED}Error: No command provided${NC}" >&2
    show_help
    exit 1
fi

# Execute command
case "$COMMAND" in
    set)
        if [ ${#ARGS[@]} -ne 2 ]; then
            echo -e "${RED}Error: SET requires key and value${NC}" >&2
            echo "Usage: $0 set KEY VALUE" >&2
            exit 1
        fi
        send_command "$NODE" "SET ${ARGS[0]} ${ARGS[1]}"
        ;;
    get)
        if [ ${#ARGS[@]} -ne 1 ]; then
            echo -e "${RED}Error: GET requires a key${NC}" >&2
            echo "Usage: $0 get KEY" >&2
            exit 1
        fi
        send_command "$NODE" "GET ${ARGS[0]}"
        ;;
    del|delete)
        if [ ${#ARGS[@]} -ne 1 ]; then
            echo -e "${RED}Error: DEL requires a key${NC}" >&2
            echo "Usage: $0 del KEY" >&2
            exit 1
        fi
        send_command "$NODE" "DEL ${ARGS[0]}"
        ;;
    sync)
        if [ ${#ARGS[@]} -ne 1 ]; then
            echo -e "${RED}Error: SYNC requires a target node${NC}" >&2
            echo "Usage: $0 sync TARGET_NODE" >&2
            exit 1
        fi
        target_node=${ARGS[0]}
        send_command "$target_node" "SYNC REQUEST"
        ;;
    stats)
        if [ ${#ARGS[@]} -ne 1 ]; then
            echo -e "${RED}Error: STATS requires a target node${NC}" >&2
            echo "Usage: $0 stats NODE" >&2
            exit 1
        fi
        target_node=${ARGS[0]}
        send_command "$target_node" "STATS"
        ;;
    status)
        check_status
        ;;
    test)
        run_tests
        ;;
    demo)
        run_demo
        ;;
    *)
        echo -e "${RED}Error: Unknown command '$COMMAND'${NC}" >&2
        show_help
        exit 1
        ;;
esac
