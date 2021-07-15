package guac

import (
	"sync"
)

// SessionDataStore is a store for arbitrary session data
var SessionDataStore *MemorySessionDataStore

func init() {
	SessionDataStore = NewMemorySessionDataStore()
}

// MemorySessionDataStore is a generic in-memory data store
type MemorySessionDataStore struct {
	sync.RWMutex
	Data map[string]interface{}
}

// NewMemorySessionDataStore creates a new store
func NewMemorySessionDataStore() *MemorySessionDataStore {
	return &MemorySessionDataStore{
		Data: make(map[string]interface{}),
	}
}

// Get returns session data by id
func (s *MemorySessionDataStore) Get(id string) interface{} {
	s.RLock()
	defer s.RUnlock()
	return s.Data[id]
}

// Set insert session data to data store
func (s *MemorySessionDataStore) Set(id string, data interface{}) {
	s.Lock()
	defer s.Unlock()
	s.Data[id] = data
}

// Delete removes session data by id
func (s *MemorySessionDataStore) Delete(id string) {
	s.Lock()
	defer s.Unlock()
	delete(s.Data, id)
}
