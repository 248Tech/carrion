package state

import (
	"sync/atomic"
	"time"
)

// Snapshot is one parsed "Time:" line from the game log.
type Snapshot struct {
	ParsedAt       time.Time
	Timestamp      time.Time
	FPS            float64
	HeapMB         float64
	RSSMB          float64
	Chunks         int
	CGo            int
	CGoMissing     bool
	Players        int
	Zombies        int
	EntitiesTotal  int
	EntitiesActive int // -1 if not present
	CO             int // connections
}

// SnapshotStore holds the current snapshot atomically.
type SnapshotStore struct {
	v atomic.Value
}

// NewSnapshotStore creates a store with a zero snapshot.
func NewSnapshotStore() *SnapshotStore {
	s := &SnapshotStore{}
	s.v.Store(Snapshot{})
	return s
}

// Update sets the current snapshot.
func (s *SnapshotStore) Update(snap Snapshot) {
	s.v.Store(snap)
}

// Current returns the current snapshot (copy).
func (s *SnapshotStore) Current() Snapshot {
	return s.v.Load().(Snapshot)
}
