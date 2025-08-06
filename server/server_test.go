package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Ahmedhossamdev/simple-kv/store"
)

func TestServerBasicCommands(t *testing.T) {
	s := store.New()

	// Start server in background
	go Start(":9001", s, []string{})

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Connect to server
	conn, err := net.Dial("tcp", "localhost:9001")
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Test SET command
	fmt.Fprintf(conn, "SET test_key test_value\n")
	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read SET response: %v", err)
	}

	if !strings.Contains(response, "OK") {
		t.Errorf("Expected OK response for SET, got: %s", response)
	}

	// Test GET command
	fmt.Fprintf(conn, "GET test_key\n")
	response, err = reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read GET response: %v", err)
	}

	if !strings.Contains(response, "test_value") {
		t.Errorf("Expected test_value in GET response, got: %s", response)
	}
}

func TestServerDELCommand(t *testing.T) {
	s := store.New()

	go Start(":9004", s, []string{})

	time.Sleep(200 * time.Millisecond)

	conn, err := net.Dial("tcp", "localhost:9004")
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Set a key first
	fmt.Fprintf(conn, "SET del_key del_value\n")
	reader.ReadString('\n')

	// Delete the key
	fmt.Fprintf(conn, "DEL del_key\n")
	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read DEL response: %v", err)
	}

	if !strings.Contains(response, "DELETED") {
		t.Errorf("Expected OK response for DEL, got: %s", response)
	}

	// Verify key is deleted
	fmt.Fprintf(conn, "GET del_key\n")
	response, err = reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read GET response: %v", err)
	}

	if !strings.Contains(response, "Key not found") {
		t.Errorf("Expected NOT_FOUND after deletion, got: %s", response)
	}
}

func TestServerSYNCCommand(t *testing.T) {
	s := store.New()

	go Start(":9007", s, []string{})

	time.Sleep(200 * time.Millisecond)

	conn, err := net.Dial("tcp", "localhost:9007")
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Test SYNC REQUEST
	fmt.Fprintf(conn, "SYNC REQUEST\n")
	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read SYNC REQUEST response: %v", err)
	}

	if !strings.Contains(response, "SYNC requested from all peers") {
		t.Errorf("Expected SYNC RESPONSE, got: %s", response)
	}
}

func TestServerSTATSCommand(t *testing.T) {
	s := store.New()

	go Start(":9010", s, []string{})

	time.Sleep(200 * time.Millisecond)

	conn, err := net.Dial("tcp", "localhost:9010")
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Add some data first
	fmt.Fprintf(conn, "SET stats_key stats_value\n")
	reader.ReadString('\n')

	// Test STATS command
	fmt.Fprintf(conn, "STATS\n")
	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read STATS response: %v", err)
	}

	if !strings.Contains(response, "total_keys") {
		t.Errorf("Expected stats to contain total_keys, got: %s", response)
	}
}

func TestServerConcurrentConnections(t *testing.T) {
	s := store.New()

	go Start(":9013", s, []string{})

	time.Sleep(200 * time.Millisecond)

	// Test multiple concurrent connections
	var wg sync.WaitGroup
	numConnections := 10

	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			conn, err := net.Dial("tcp", "localhost:9013")
			if err != nil {
				t.Errorf("Client %d failed to connect: %v", clientID, err)
				return
			}
			defer conn.Close()

			reader := bufio.NewReader(conn)

			// Each client sets and gets a unique key
			key := fmt.Sprintf("client_%d_key", clientID)
			value := fmt.Sprintf("client_%d_value", clientID)

			fmt.Fprintf(conn, "SET %s %s\n", key, value)
			response, err := reader.ReadString('\n')
			if err != nil {
				t.Errorf("Client %d failed to read SET response: %v", clientID, err)
				return
			}

			if !strings.Contains(response, "OK") {
				t.Errorf("Client %d expected OK, got: %s", clientID, response)
			}

			fmt.Fprintf(conn, "GET %s\n", key)
			response, err = reader.ReadString('\n')
			if err != nil {
				t.Errorf("Client %d failed to read GET response: %v", clientID, err)
				return
			}

			if !strings.Contains(response, value) {
				t.Errorf("Client %d expected %s, got: %s", clientID, value, response)
			}
		}(i)
	}

	wg.Wait()
}

func TestServerInvalidCommands(t *testing.T) {
	s := store.New()

	go Start(":9016", s, []string{})

	time.Sleep(200 * time.Millisecond)

	conn, err := net.Dial("tcp", "localhost:9016")
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Test invalid command
	fmt.Fprintf(conn, "INVALID_COMMAND\n")
	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if !strings.Contains(response, "INVALID_COMMAND") {
		t.Errorf("Expected ERROR for invalid command, got: %s", response)
	}

	// Test SET with insufficient arguments
	fmt.Fprintf(conn, "SET only_one_arg\n")
	response, err = reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if !strings.Contains(response, "SET key value") {
		t.Errorf("Expected ERROR for insufficient args, got: %s", response)
	}
}

// Benchmark tests
func BenchmarkServerSETCommand(b *testing.B) {
	s := store.New()

	go Start(":9019", s, []string{})

	time.Sleep(200 * time.Millisecond)

	conn, err := net.Dial("tcp", "localhost:9019")
	if err != nil {
		b.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fmt.Fprintf(conn, "SET bench_key_%d bench_value_%d\n", i, i)
		reader.ReadString('\n')
	}
}

func BenchmarkServerGETCommand(b *testing.B) {
	s := store.New()

	go Start(":9022", s, []string{})

	time.Sleep(200 * time.Millisecond)

	conn, err := net.Dial("tcp", "localhost:9022")
	if err != nil {
		b.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Set up test data
	fmt.Fprintf(conn, "SET bench_key bench_value\n")
	reader.ReadString('\n')

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fmt.Fprintf(conn, "GET bench_key\n")
		reader.ReadString('\n')
	}
}
