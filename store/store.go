package store

import (
	"encoding/json"
	"sync"
)

type Store struct {
	mu            sync.RWMutex
	data          map[string]Value
	lastSeenMsgID map[string]bool // for deduplication
}

type Value struct {
	Data      string `json:"data"`
	Timestamp int64  `json:"timestamp"`
	MsgID     string `json:"msg_id"`
}

type StoreSnapshot struct {
	Data map[string]Value `json:"data"`
}

func New() *Store {
	return &Store{
		data:          make(map[string]Value),
		lastSeenMsgID: make(map[string]bool),
	}
}

func (s *Store) Set(key, value string, timestamp int64, msgID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastSeenMsgID[msgID] {
		return
	}
	s.lastSeenMsgID[msgID] = true

	current, exists := s.data[key]
	if !exists || timestamp > current.Timestamp {
		s.data[key] = Value{
			Data:      value,
			Timestamp: timestamp,
			MsgID:     msgID,
		}
	}
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	return val.Data, ok
}

func (s *Store) Del(key string, timestamp int64, msgID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastSeenMsgID[msgID] {
		return
	}
	s.lastSeenMsgID[msgID] = true

	current, exists := s.data[key]
	if exists && timestamp > current.Timestamp {
		delete(s.data, key)
	}
}

// GetSnapshot returns a JSON snapshot of all data
func (s *Store) GetSnapshot() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := StoreSnapshot{
		Data: make(map[string]Value),
	}

	// Copy all data
	for k, v := range s.data {
		snapshot.Data[k] = v
	}

	return json.Marshal(snapshot)
}

// ApplySnapshot merges snapshot data with current data
func (s *Store) ApplySnapshot(data []byte) error {
	var snapshot StoreSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Merge data, keeping newer timestamps
	for key, incomingValue := range snapshot.Data {
		current, exists := s.data[key]
		if !exists || incomingValue.Timestamp > current.Timestamp {
			s.data[key] = incomingValue
			// Mark message as seen to prevent duplicates
			if incomingValue.MsgID != "" {
				s.lastSeenMsgID[incomingValue.MsgID] = true
			}
		}
	}

	return nil
}

// GetAllKeys returns all keys in the store
func (s *Store) GetAllKeys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	return keys
}

// GetStats returns statistics about the store
func (s *Store) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"total_keys":         len(s.data),
		"processed_messages": len(s.lastSeenMsgID),
	}
}
