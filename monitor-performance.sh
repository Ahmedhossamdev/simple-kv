#!/bin/bash

# Performance monitoring script for Simple KV Store
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "⚡ Simple KV Store - Performance Monitor"
echo "========================================"

# Configuration
MONITOR_DURATION=${1:-60}  # Default 60 seconds
SAMPLE_INTERVAL=${2:-5}    # Default 5 seconds
OUTPUT_FILE="performance_report_$(date +%Y%m%d_%H%M%S).txt"

echo "📊 Monitoring cluster performance for ${MONITOR_DURATION} seconds..."
echo "📝 Results will be saved to: $OUTPUT_FILE"
echo ""

# Start monitoring
{
    echo "Simple KV Store Performance Report"
    echo "Generated: $(date)"
    echo "Duration: ${MONITOR_DURATION} seconds"
    echo "Sample Interval: ${SAMPLE_INTERVAL} seconds"
    echo "=========================================="
    echo ""

    # Check if cluster is running
    if ! docker-compose ps | grep -q "Up"; then
        echo "❌ Cluster is not running. Please start with 'make up'"
        exit 1
    fi

    echo "🔍 Initial Cluster Status:"
    docker-compose ps
    echo ""

    # Start performance monitoring
    START_TIME=$(date +%s)
    SAMPLE_COUNT=0

    while [ $(($(date +%s) - START_TIME)) -lt $MONITOR_DURATION ]; do
        CURRENT_TIME=$(date +%s)
        ELAPSED=$((CURRENT_TIME - START_TIME))
        SAMPLE_COUNT=$((SAMPLE_COUNT + 1))

        echo "📈 Sample #$SAMPLE_COUNT (${ELAPSED}s elapsed)"
        echo "Time: $(date)"
        echo ""

        # Docker stats
        echo "🐳 Container Resources:"
        docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}" $(docker-compose ps -q)
        echo ""

        # Connection tests
        echo "🔗 Connection Tests:"
        for port in 8081 8082 8083; do
            if timeout 3 nc -z localhost $port; then
                echo "  ✅ Node $(($port - 8080)): OK"
            else
                echo "  ❌ Node $(($port - 8080)): FAILED"
            fi
        done
        echo ""

        # Performance tests
        echo "⚡ Quick Performance Test:"

        # Write test
        WRITE_START=$(date +%s%N)
        echo "SET perf_test_$SAMPLE_COUNT value_$SAMPLE_COUNT" | timeout 3 nc localhost 8081 > /dev/null 2>&1
        WRITE_END=$(date +%s%N)
        WRITE_TIME=$(( (WRITE_END - WRITE_START) / 1000000 ))
        echo "  Write latency: ${WRITE_TIME}ms"

        # Read test
        READ_START=$(date +%s%N)
        echo "GET perf_test_$SAMPLE_COUNT" | timeout 3 nc localhost 8081 > /dev/null 2>&1
        READ_END=$(date +%s%N)
        READ_TIME=$(( (READ_END - READ_START) / 1000000 ))
        echo "  Read latency: ${READ_TIME}ms"

        # Stats test
        STATS_START=$(date +%s%N)
        echo "STATS" | timeout 3 nc localhost 8081 > /dev/null 2>&1
        STATS_END=$(date +%s%N)
        STATS_TIME=$(( (STATS_END - STATS_START) / 1000000 ))
        echo "  Stats latency: ${STATS_TIME}ms"
        echo ""

        # Memory usage (if available)
        if command -v free > /dev/null; then
            echo "💾 System Memory:"
            free -h
            echo ""
        fi

        # Load average (if available)
        if [ -f /proc/loadavg ]; then
            echo "📊 System Load:"
            cat /proc/loadavg
            echo ""
        fi

        echo "----------------------------------------"
        echo ""

        sleep $SAMPLE_INTERVAL
    done

    echo "🏁 Performance Monitoring Completed"
    echo ""

    # Final cluster stats
    echo "📋 Final Cluster Status:"
    echo "STATS" | nc localhost 8081 || echo "Failed to get stats"
    echo ""

    # Summary
    echo "📊 Summary:"
    echo "Total samples: $SAMPLE_COUNT"
    echo "Total duration: ${MONITOR_DURATION}s"
    echo "Average sample interval: $(($MONITOR_DURATION / $SAMPLE_COUNT))s"
    echo ""

    echo "🎯 Recommendations:"
    echo "• Monitor CPU usage - should stay below 80%"
    echo "• Monitor memory usage - watch for memory leaks"
    echo "• Check network I/O for bottlenecks"
    echo "• Latency should be consistent (< 100ms for local)"
    echo "• All nodes should remain accessible"

} | tee "$OUTPUT_FILE"

echo ""
echo -e "${GREEN}✅ Performance monitoring completed!${NC}"
echo -e "${BLUE}📄 Report saved to: $OUTPUT_FILE${NC}"
echo ""
echo -e "${YELLOW}💡 Quick analysis commands:${NC}"
echo -e "${YELLOW}  grep 'Write latency' $OUTPUT_FILE | awk '{print \$3}' | sort -n${NC}"
echo -e "${YELLOW}  grep 'Read latency' $OUTPUT_FILE | awk '{print \$3}' | sort -n${NC}"
echo -e "${YELLOW}  grep 'CPUPerc' $OUTPUT_FILE${NC}"
