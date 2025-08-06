package store

import (
	"encoding/json"
	"testing"
	"time"
)

func TestStoreBasicOperations(t *testing.T) {
	s := New()

	// Test Set and Get
	timestamp := time.Now().UnixNano()
	msgID := "test-msg-1"

	s.Set("key1", "value1", timestamp, msgID)

	value, exists := s.Get("key1")
	if !exists {
		t.Error("Expected key1 to exist")
	}
	if value != "value1" {
		t.Errorf("Expected 'value1', got '%s'", value)
	}
}

func TestStoreConcurrentOperations(t *testing.T) {
	s := New()

	// Test concurrent writes
	done := make(chan bool)

	for i := 0; i < 100; i++ {
		go func(i int) {
			timestamp := time.Now().UnixNano()
			msgID := "msg-" + string(rune(i))
			s.Set("concurrent-key", "value", timestamp, msgID)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	_, exists := s.Get("concurrent-key")
	if !exists {
		t.Error("Expected concurrent-key to exist after concurrent operations")
	}
}

func TestStoreConflictResolution(t *testing.T) {
	s := New()

	// Earlier timestamp
	earlierTime := time.Now().UnixNano()
	s.Set("conflict-key", "old-value", earlierTime, "msg-1")

	// Later timestamp should win
	laterTime := earlierTime + 1000000
	s.Set("conflict-key", "new-value", laterTime, "msg-2")

	value, exists := s.Get("conflict-key")
	if !exists {
		t.Error("Expected conflict-key to exist")
	}
	if value != "new-value" {
		t.Errorf("Expected 'new-value' (later timestamp should win), got '%s'", value)
	}

	// Earlier timestamp should not overwrite
	evenEarlierTime := earlierTime - 1000000
	s.Set("conflict-key", "very-old-value", evenEarlierTime, "msg-3")

	if value != "new-value" {
		t.Errorf("Earlier timestamp should not overwrite, expected 'new-value', got '%s'", value)
	}
}

func TestStoreDeduplication(t *testing.T) {
	s := New()

	timestamp := time.Now().UnixNano()
	msgID := "duplicate-msg"

	// First time should work
	s.Set("dedup-key", "value1", timestamp, msgID)

	// Same message ID should be ignored
	s.Set("dedup-key", "value2", timestamp+1000000, msgID)

	value, exists := s.Get("dedup-key")
	if !exists {
		t.Error("Expected dedup-key to exist")
	}
	if value != "value1" {
		t.Errorf("Duplicate message should be ignored, expected 'value1', got '%s'", value)
	}
}

func TestStoreDeletion(t *testing.T) {
	s := New()

	// Set a key
	timestamp := time.Now().UnixNano()
	s.Set("delete-key", "delete-value", timestamp, "msg-1")

	// Verify it exists
	_, exists := s.Get("delete-key")
	if !exists {
		t.Error("Expected delete-key to exist before deletion")
	}

	// Delete it
	s.Del("delete-key", timestamp+1000000, "msg-2")

	// Verify it's gone
	_, exists = s.Get("delete-key")
	if exists {
		t.Error("Expected delete-key to not exist after deletion")
	}
}

func TestStoreSnapshot(t *testing.T) {
	s := New()

	// Add some data
	timestamp := time.Now().UnixNano()
	s.Set("snap-key1", "snap-value1", timestamp, "msg-1")
	s.Set("snap-key2", "snap-value2", timestamp+1000, "msg-2")

	// Get snapshot
	snapshotData, err := s.GetSnapshot()
	if err != nil {
		t.Fatalf("Failed to get snapshot: %v", err)
	}

	// Verify snapshot is valid JSON
	var snapshot StoreSnapshot
	err = json.Unmarshal(snapshotData, &snapshot)
	if err != nil {
		t.Fatalf("Failed to unmarshal snapshot: %v", err)
	}

	// Verify data in snapshot
	if len(snapshot.Data) != 2 {
		t.Errorf("Expected 2 items in snapshot, got %d", len(snapshot.Data))
	}

	if snapshot.Data["snap-key1"].Data != "snap-value1" {
		t.Error("Snapshot data mismatch for snap-key1")
	}
}

func TestStoreApplySnapshot(t *testing.T) {
	s1 := New()
	s2 := New()

	// Add data to s1
	timestamp := time.Now().UnixNano()
	s1.Set("apply-key1", "apply-value1", timestamp, "msg-1")
	s1.Set("apply-key2", "apply-value2", timestamp+1000, "msg-2")

	// Get snapshot from s1
	snapshotData, err := s1.GetSnapshot()
	if err != nil {
		t.Fatalf("Failed to get snapshot: %v", err)
	}

	// Apply snapshot to s2
	err = s2.ApplySnapshot(snapshotData)
	if err != nil {
		t.Fatalf("Failed to apply snapshot: %v", err)
	}

	// Verify s2 has the data
	value, exists := s2.Get("apply-key1")
	if !exists || value != "apply-value1" {
		t.Error("Snapshot application failed for apply-key1")
	}

	value, exists = s2.Get("apply-key2")
	if !exists || value != "apply-value2" {
		t.Error("Snapshot application failed for apply-key2")
	}
}

func TestStoreStats(t *testing.T) {
	s := New()

	// Add some data
	timestamp := time.Now().UnixNano()
	s.Set("stats-key1", "stats-value1", timestamp, "msg-1")
	s.Set("stats-key2", "stats-value2", timestamp+1000, "msg-2")

	// Get stats
	stats := s.GetStats()

	totalKeys, exists := stats["total_keys"]
	if !exists {
		t.Error("Expected total_keys in stats")
	}

	if totalKeys != 2 {
		t.Errorf("Expected 2 total keys, got %v", totalKeys)
	}

	processedMessages, exists := stats["processed_messages"]
	if !exists {
		t.Error("Expected processed_messages in stats")
	}

	if processedMessages != 2 {
		t.Errorf("Expected 2 processed messages, got %v", processedMessages)
	}
}

// Benchmark tests
func BenchmarkStoreSet(b *testing.B) {
	s := New()
	timestamp := time.Now().UnixNano()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msgID := "bench-msg-" + string(rune(i))
		s.Set("bench-key", "bench-value", timestamp+int64(i), msgID)
	}
}

func BenchmarkStoreGet(b *testing.B) {
	s := New()
	s.Set("bench-key", "bench-value", time.Now().UnixNano(), "bench-msg")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Get("bench-key")
	}
}

func BenchmarkStoreConcurrentOperations(b *testing.B) {
	s := New()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			timestamp := time.Now().UnixNano()
			msgID := "concurrent-bench-" + string(rune(i))
			s.Set("concurrent-key", "concurrent-value", timestamp, msgID)
			s.Get("concurrent-key")
			i++
		}
	})
}
