package main

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/Ahmedhossamdev/simple-kv/server"
	"github.com/Ahmedhossamdev/simple-kv/store"
)

// TestHighConcurrency tests the system under high concurrent load
func TestHighConcurrency(t *testing.T) {
	s := store.New()
	go server.Start(":7001", s, []string{})
	time.Sleep(200 * time.Millisecond)

	const numClients = 100
	const operationsPerClient = 50

	var wg sync.WaitGroup
	errors := make(chan error, numClients)

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			conn, err := net.Dial("tcp", "localhost:7001")
			if err != nil {
				errors <- fmt.Errorf("client %d failed to connect: %v", clientID, err)
				return
			}
			defer conn.Close()

			reader := bufio.NewReader(conn)

			for j := 0; j < operationsPerClient; j++ {
				key := fmt.Sprintf("client_%d_key_%d", clientID, j)
				value := fmt.Sprintf("client_%d_value_%d", clientID, j)

				// SET operation
				fmt.Fprintf(conn, "SET %s %s\n", key, value)
				_, err := reader.ReadString('\n')
				if err != nil {
					errors <- fmt.Errorf("client %d SET failed: %v", clientID, err)
					return
				}

				// GET operation
				fmt.Fprintf(conn, "GET %s\n", key)
				_, err = reader.ReadString('\n')
				if err != nil {
					errors <- fmt.Errorf("client %d GET failed: %v", clientID, err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Errorf("Concurrent operation error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("High concurrency test failed with %d errors", errorCount)
	}

	t.Logf("✅ High concurrency test completed: %d clients × %d operations = %d total operations",
		numClients, operationsPerClient, numClients*operationsPerClient)
}

// TestMemoryUsage tests memory usage under load
func TestMemoryUsage(t *testing.T) {
	s := store.New()
	go server.Start(":7002", s, []string{})
	time.Sleep(200 * time.Millisecond)

	conn, err := net.Dial("tcp", "localhost:7002")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Add a large number of keys
	const numKeys = 10000
	const keySize = 100   // Average key size
	const valueSize = 500 // Average value size

	start := time.Now()
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("memory_test_key_%0*d", keySize-20, i)
		value := fmt.Sprintf("memory_test_value_%0*d", valueSize-25, i)

		fmt.Fprintf(conn, "SET %s %s\n", key, value)
		_, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("Failed to set key %d: %v", i, err)
		}

		// Print progress every 1000 keys
		if (i+1)%1000 == 0 {
			t.Logf("Progress: %d/%d keys added", i+1, numKeys)
		}
	}
	elapsed := time.Since(start)

	// Get stats
	fmt.Fprintf(conn, "STATS\n")
	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	t.Logf("✅ Memory test completed:")
	t.Logf("   Added %d keys in %v", numKeys, elapsed)
	t.Logf("   Average: %.2f keys/sec", float64(numKeys)/elapsed.Seconds())
	t.Logf("   Estimated memory: ~%.2f MB", float64(numKeys*(keySize+valueSize))/1024/1024)
	t.Logf("   Stats: %s", response)
}

// TestThroughput measures operations per second
func TestThroughput(t *testing.T) {
	s := store.New()
	go server.Start(":7003", s, []string{})
	time.Sleep(200 * time.Millisecond)

	const duration = 5 * time.Second
	const numWorkers = 10

	var wg sync.WaitGroup
	var totalOps int64
	var mu sync.Mutex

	start := time.Now()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			conn, err := net.Dial("tcp", "localhost:7003")
			if err != nil {
				t.Errorf("Worker %d failed to connect: %v", workerID, err)
				return
			}
			defer conn.Close()

			reader := bufio.NewReader(conn)
			ops := 0

			for time.Since(start) < duration {
				key := fmt.Sprintf("throughput_key_%d_%d", workerID, ops)
				value := fmt.Sprintf("throughput_value_%d_%d", workerID, ops)

				// SET operation
				fmt.Fprintf(conn, "SET %s %s\n", key, value)
				_, err := reader.ReadString('\n')
				if err != nil {
					continue
				}

				// GET operation
				fmt.Fprintf(conn, "GET %s\n", key)
				_, err = reader.ReadString('\n')
				if err != nil {
					continue
				}

				ops += 2 // SET + GET = 2 operations
			}

			mu.Lock()
			totalOps += int64(ops)
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	throughput := float64(totalOps) / elapsed.Seconds()
	t.Logf("✅ Throughput test completed:")
	t.Logf("   Duration: %v", elapsed)
	t.Logf("   Total operations: %d", totalOps)
	t.Logf("   Throughput: %.2f ops/sec", throughput)
	t.Logf("   Workers: %d", numWorkers)
}

// TestLatency measures response times
func TestLatency(t *testing.T) {
	s := store.New()
	go server.Start(":7004", s, []string{})
	time.Sleep(200 * time.Millisecond)

	conn, err := net.Dial("tcp", "localhost:7004")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	const numOps = 1000
	setLatencies := make([]time.Duration, numOps)
	getLatencies := make([]time.Duration, numOps)

	// Measure SET latencies
	for i := 0; i < numOps; i++ {
		key := fmt.Sprintf("latency_key_%d", i)
		value := fmt.Sprintf("latency_value_%d", i)

		start := time.Now()
		fmt.Fprintf(conn, "SET %s %s\n", key, value)
		_, err := reader.ReadString('\n')
		setLatencies[i] = time.Since(start)

		if err != nil {
			t.Fatalf("SET operation %d failed: %v", i, err)
		}
	}

	// Measure GET latencies
	for i := 0; i < numOps; i++ {
		key := fmt.Sprintf("latency_key_%d", i)

		start := time.Now()
		fmt.Fprintf(conn, "GET %s\n", key)
		_, err := reader.ReadString('\n')
		getLatencies[i] = time.Since(start)

		if err != nil {
			t.Fatalf("GET operation %d failed: %v", i, err)
		}
	}

	// Calculate statistics
	setAvg := calculateAverage(setLatencies)
	setP95 := calculatePercentile(setLatencies, 95)
	setMax := calculateMax(setLatencies)

	getAvg := calculateAverage(getLatencies)
	getP95 := calculatePercentile(getLatencies, 95)
	getMax := calculateMax(getLatencies)

	t.Logf("✅ Latency test completed (%d operations):", numOps)
	t.Logf("   SET - Avg: %v, P95: %v, Max: %v", setAvg, setP95, setMax)
	t.Logf("   GET - Avg: %v, P95: %v, Max: %v", getAvg, getP95, getMax)
}

// TestStressTest runs multiple load scenarios simultaneously
func TestStressTest(t *testing.T) {
	s := store.New()
	go server.Start(":7005", s, []string{})
	time.Sleep(200 * time.Millisecond)

	const duration = 10 * time.Second
	var wg sync.WaitGroup

	// Start multiple stress scenarios
	scenarios := []struct {
		name    string
		workers int
		fn      func(int, time.Duration, *testing.T)
	}{
		{"Heavy SET operations", 5, heavySetWorker},
		{"Heavy GET operations", 5, heavyGetWorker},
		{"Mixed operations", 5, mixedWorker},
		{"STATS operations", 2, statsWorker},
	}

	t.Logf("🔥 Starting stress test for %v...", duration)

	for _, scenario := range scenarios {
		for i := 0; i < scenario.workers; i++ {
			wg.Add(1)
			go func(name string, workerID int, fn func(int, time.Duration, *testing.T)) {
				defer wg.Done()
				fn(workerID, duration, t)
			}(scenario.name, i, scenario.fn)
		}
	}

	wg.Wait()
	t.Logf("✅ Stress test completed successfully!")
}

// Helper functions
func heavySetWorker(workerID int, duration time.Duration, t *testing.T) {
	conn, err := net.Dial("tcp", "localhost:7005")
	if err != nil {
		t.Errorf("SET worker %d failed to connect: %v", workerID, err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	start := time.Now()
	ops := 0

	for time.Since(start) < duration {
		key := fmt.Sprintf("stress_set_%d_%d", workerID, ops)
		value := fmt.Sprintf("stress_value_%d_%d", workerID, ops)

		fmt.Fprintf(conn, "SET %s %s\n", key, value)
		_, err := reader.ReadString('\n')
		if err == nil {
			ops++
		}
	}

	t.Logf("SET worker %d completed %d operations", workerID, ops)
}

func heavyGetWorker(workerID int, duration time.Duration, t *testing.T) {
	conn, err := net.Dial("tcp", "localhost:7005")
	if err != nil {
		t.Errorf("GET worker %d failed to connect: %v", workerID, err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	start := time.Now()
	ops := 0

	// First, set a key to read
	fmt.Fprintf(conn, "SET stress_get_key_%d stress_get_value_%d\n", workerID, workerID)
	reader.ReadString('\n')

	for time.Since(start) < duration {
		fmt.Fprintf(conn, "GET stress_get_key_%d\n", workerID)
		_, err := reader.ReadString('\n')
		if err == nil {
			ops++
		}
	}

	t.Logf("GET worker %d completed %d operations", workerID, ops)
}

func mixedWorker(workerID int, duration time.Duration, t *testing.T) {
	conn, err := net.Dial("tcp", "localhost:7005")
	if err != nil {
		t.Errorf("Mixed worker %d failed to connect: %v", workerID, err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	start := time.Now()
	ops := 0

	for time.Since(start) < duration {
		if ops%3 == 0 {
			key := fmt.Sprintf("stress_mixed_%d_%d", workerID, ops)
			value := fmt.Sprintf("stress_mixed_value_%d_%d", workerID, ops)
			fmt.Fprintf(conn, "SET %s %s\n", key, value)
		} else if ops%3 == 1 {
			key := fmt.Sprintf("stress_mixed_%d_%d", workerID, ops-1)
			fmt.Fprintf(conn, "GET %s\n", key)
		} else {
			key := fmt.Sprintf("stress_mixed_%d_%d", workerID, ops-2)
			fmt.Fprintf(conn, "DEL %s\n", key)
		}

		_, err := reader.ReadString('\n')
		if err == nil {
			ops++
		}
	}

	t.Logf("Mixed worker %d completed %d operations", workerID, ops)
}

func statsWorker(workerID int, duration time.Duration, t *testing.T) {
	conn, err := net.Dial("tcp", "localhost:7005")
	if err != nil {
		t.Errorf("Stats worker %d failed to connect: %v", workerID, err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	start := time.Now()
	ops := 0

	for time.Since(start) < duration {
		fmt.Fprintf(conn, "STATS\n")
		_, err := reader.ReadString('\n')
		if err == nil {
			ops++
		}
		time.Sleep(500 * time.Millisecond) // Don't spam stats too much
	}

	t.Logf("Stats worker %d completed %d operations", workerID, ops)
}

// Statistics helper functions
func calculateAverage(durations []time.Duration) time.Duration {
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

func calculatePercentile(durations []time.Duration, percentile float64) time.Duration {
	// Simple percentile calculation (not optimized)
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)

	// Simple bubble sort (good enough for tests)
	for i := 0; i < len(sorted); i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	index := int(float64(len(sorted)) * percentile / 100.0)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func calculateMax(durations []time.Duration) time.Duration {
	max := durations[0]
	for _, d := range durations {
		if d > max {
			max = d
		}
	}
	return max
}
