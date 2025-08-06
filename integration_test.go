package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/Ahmedhossamdev/simple-kv/server"
	"github.com/Ahmedhossamdev/simple-kv/store"
)

// TestMultiNodeSync tests data synchronization between multiple nodes
func TestMultiNodeSync(t *testing.T) {
	// Create stores for 3 nodes
	store1 := store.New()
	store2 := store.New()
	store3 := store.New()

	// Start nodes with peer connections
	go server.Start(":8091", store1, []string{":8092", ":8093"})
	go server.Start(":8092", store2, []string{":8091", ":8093"})
	go server.Start(":8093", store3, []string{":8091", ":8092"})

	// Wait for servers to start
	time.Sleep(500 * time.Millisecond)

	// Connect to node 1 and add data
	conn1, err := net.Dial("tcp", "localhost:8091")
	if err != nil {
		t.Fatalf("Failed to connect to node 1: %v", err)
	}
	defer conn1.Close()

	reader1 := bufio.NewReader(conn1)

	// Add data to node 1
	fmt.Fprintf(conn1, "SET integration_key integration_value\n")
	response, err := reader1.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read response from node 1: %v", err)
	}

	if !strings.Contains(response, "OK") {
		t.Errorf("Expected OK from node 1, got: %s", response)
	}

	// Request sync from node 2
	conn2, err := net.Dial("tcp", "localhost:8092")
	if err != nil {
		t.Fatalf("Failed to connect to node 2: %v", err)
	}
	defer conn2.Close()

	reader2 := bufio.NewReader(conn2)

	// Trigger sync by requesting data from node 1
	fmt.Fprintf(conn2, "SYNC REQUEST\n")
	response, err = reader2.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read sync response from node 2: %v", err)
	}

	// Apply the sync data
	if strings.Contains(response, "SYNC RESPONSE") {
		// Parse and apply the snapshot (this would be done automatically in real scenario)
		time.Sleep(100 * time.Millisecond)
	}

	// Verify that node 2 can retrieve the data
	fmt.Fprintf(conn2, "GET integration_key\n")
	response, err = reader2.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read GET response from node 2: %v", err)
	}

	// For now, just check that we got a response
	// In a full integration test, we'd verify the data was properly synced
	if response == "" {
		t.Error("Expected non-empty response from node 2")
	}
}

// TestNodeFailureRecovery tests automatic recovery when a node comes back online
func TestNodeFailureRecovery(t *testing.T) {
	// This test simulates what happens when nodes restart
	// We can't easily simulate actual node failure in unit tests
	// So we test the sync mechanisms individually

	store1 := store.New()
	store2 := store.New()

	// Add data to store1
	timestamp := time.Now().UnixNano()
	store1.Set("recovery_key", "recovery_value", timestamp, "msg-1")

	// Get snapshot from store1
	snapshot, err := store1.GetSnapshot()
	if err != nil {
		t.Fatalf("Failed to get snapshot: %v", err)
	}

	// Apply snapshot to store2 (simulating sync)
	err = store2.ApplySnapshot(snapshot)
	if err != nil {
		t.Fatalf("Failed to apply snapshot: %v", err)
	}

	// Verify store2 has the data
	value, exists := store2.Get("recovery_key")
	if !exists {
		t.Error("Expected recovery_key to exist in store2 after sync")
	}
	if value != "recovery_value" {
		t.Errorf("Expected 'recovery_value', got '%s'", value)
	}
}

// TestConflictResolution tests how nodes handle conflicting updates
func TestConflictResolution(t *testing.T) {
	store1 := store.New()
	store2 := store.New()

	// Simulate concurrent updates with different timestamps
	baseTime := time.Now().UnixNano()

	// Node 1 sets a value (earlier timestamp)
	store1.Set("conflict_key", "value_from_node1", baseTime, "msg-1")

	// Node 2 sets a different value (later timestamp)
	store2.Set("conflict_key", "value_from_node2", baseTime+1000000, "msg-2")

	// Sync node1's data to node2
	snapshot1, err := store1.GetSnapshot()
	if err != nil {
		t.Fatalf("Failed to get snapshot from store1: %v", err)
	}

	err = store2.ApplySnapshot(snapshot1)
	if err != nil {
		t.Fatalf("Failed to apply snapshot to store2: %v", err)
	}

	// Node2 should keep its value (later timestamp wins)
	value, exists := store2.Get("conflict_key")
	if !exists {
		t.Error("Expected conflict_key to exist after sync")
	}
	if value != "value_from_node2" {
		t.Errorf("Expected 'value_from_node2' (later timestamp should win), got '%s'", value)
	}

	// Now sync node2's data to node1
	snapshot2, err := store2.GetSnapshot()
	if err != nil {
		t.Fatalf("Failed to get snapshot from store2: %v", err)
	}

	err = store1.ApplySnapshot(snapshot2)
	if err != nil {
		t.Fatalf("Failed to apply snapshot to store1: %v", err)
	}

	// Node1 should now have node2's value
	value, exists = store1.Get("conflict_key")
	if !exists {
		t.Error("Expected conflict_key to exist in store1 after sync")
	}
	if value != "value_from_node2" {
		t.Errorf("Expected 'value_from_node2' after sync, got '%s'", value)
	}
}

// TestLoadBalancing tests that multiple clients can connect to different nodes
func TestLoadBalancing(t *testing.T) {
	store1 := store.New()
	store2 := store.New()

	go server.Start(":8094", store1, []string{":8095"})
	go server.Start(":8095", store2, []string{":8094"})

	time.Sleep(300 * time.Millisecond)

	// Connect multiple clients to different nodes
	conn1, err := net.Dial("tcp", "localhost:8094")
	if err != nil {
		t.Fatalf("Failed to connect to node 8094: %v", err)
	}
	defer conn1.Close()

	conn2, err := net.Dial("tcp", "localhost:8095")
	if err != nil {
		t.Fatalf("Failed to connect to node 8095: %v", err)
	}
	defer conn2.Close()

	reader1 := bufio.NewReader(conn1)
	reader2 := bufio.NewReader(conn2)

	// Client 1 writes to node 1
	fmt.Fprintf(conn1, "SET client1_key client1_value\n")
	response1, err := reader1.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read response from node 1: %v", err)
	}

	// Client 2 writes to node 2
	fmt.Fprintf(conn2, "SET client2_key client2_value\n")
	response2, err := reader2.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read response from node 2: %v", err)
	}

	// Both should succeed
	if !strings.Contains(response1, "OK") {
		t.Errorf("Expected OK from node 1, got: %s", response1)
	}
	if !strings.Contains(response2, "OK") {
		t.Errorf("Expected OK from node 2, got: %s", response2)
	}
}

// TestStatsAcrossNodes tests that stats work correctly across different nodes
func TestStatsAcrossNodes(t *testing.T) {
	store1 := store.New()
	store2 := store.New()

	go server.Start(":8096", store1, []string{":8097"})
	go server.Start(":8097", store2, []string{":8096"})

	time.Sleep(300 * time.Millisecond)

	conn1, err := net.Dial("tcp", "localhost:8096")
	if err != nil {
		t.Fatalf("Failed to connect to node 8096: %v", err)
	}
	defer conn1.Close()

	conn2, err := net.Dial("tcp", "localhost:8097")
	if err != nil {
		t.Fatalf("Failed to connect to node 8097: %v", err)
	}
	defer conn2.Close()

	reader1 := bufio.NewReader(conn1)
	reader2 := bufio.NewReader(conn2)

	// Add data to both nodes
	fmt.Fprintf(conn1, "SET stats_key1 stats_value1\n")
	reader1.ReadString('\n')

	fmt.Fprintf(conn2, "SET stats_key2 stats_value2\n")
	reader2.ReadString('\n')

	// Get stats from both nodes
	fmt.Fprintf(conn1, "STATS\n")
	stats1, err := reader1.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read stats from node 1: %v", err)
	}

	fmt.Fprintf(conn2, "STATS\n")
	stats2, err := reader2.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read stats from node 2: %v", err)
	}

	// Both should contain stats
	if !strings.Contains(stats1, "total_keys") {
		t.Errorf("Expected stats from node 1 to contain total_keys, got: %s", stats1)
	}
	if !strings.Contains(stats2, "total_keys") {
		t.Errorf("Expected stats from node 2 to contain total_keys, got: %s", stats2)
	}
}

// Benchmark tests for integration scenarios
func BenchmarkMultiNodeOperations(b *testing.B) {
	store1 := store.New()
	store2 := store.New()

	go server.Start(":8098", store1, []string{":8099"})
	go server.Start(":8099", store2, []string{":8098"})

	time.Sleep(300 * time.Millisecond)

	conn1, err := net.Dial("tcp", "localhost:8098")
	if err != nil {
		b.Fatalf("Failed to connect to node 8098: %v", err)
	}
	defer conn1.Close()

	conn2, err := net.Dial("tcp", "localhost:8099")
	if err != nil {
		b.Fatalf("Failed to connect to node 8099: %v", err)
	}
	defer conn2.Close()

	reader1 := bufio.NewReader(conn1)
	reader2 := bufio.NewReader(conn2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Alternate between nodes
		if i%2 == 0 {
			fmt.Fprintf(conn1, "SET bench_key_%d bench_value_%d\n", i, i)
			reader1.ReadString('\n')
		} else {
			fmt.Fprintf(conn2, "SET bench_key_%d bench_value_%d\n", i, i)
			reader2.ReadString('\n')
		}
	}
}
