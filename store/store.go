package store

import (
	"sync"
)

type Store struct {
	mu            sync.RWMutex
	data          map[string]Value
	lastSeenMsgID map[string]bool // for deduplication

}

type Value struct {
	Data      string
	Timestamp int64
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
		s.data[key] = Value{Data: value, Timestamp: timestamp}
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
