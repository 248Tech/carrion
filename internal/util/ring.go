package util

import "sync"

// Ring is a fixed-size ring buffer. Safe for concurrent read; single writer for Append.
type Ring[T any] struct {
	mu     sync.RWMutex
	slice  []T
	head   int
	full   bool
	maxLen int
}

// NewRing creates a ring buffer with maxLen capacity.
func NewRing[T any](maxLen int) *Ring[T] {
	if maxLen <= 0 {
		maxLen = 64
	}
	return &Ring[T]{
		slice:  make([]T, maxLen),
		maxLen: maxLen,
	}
}

// Append adds an item; oldest is dropped when full.
func (r *Ring[T]) Append(v T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.slice[r.head] = v
	r.head = (r.head + 1) % r.maxLen
	if r.head == 0 {
		r.full = true
	}
}

// CopyOut copies up to maxLen items, oldest first. Returns number copied.
func (r *Ring[T]) CopyOut(dst []T) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n := r.Len()
	if n > len(dst) {
		n = len(dst)
	}
	if n == 0 {
		return 0
	}
	start := 0
	if r.full {
		start = r.head
	}
	for i := 0; i < n; i++ {
		idx := (start + i) % r.maxLen
		dst[i] = r.slice[idx]
	}
	return n
}

// Len returns current number of elements.
func (r *Ring[T]) Len() int {
	if r.full {
		return r.maxLen
	}
	return r.head
}

// Last returns the most recently appended element and true, or zero value and false if empty.
func (r *Ring[T]) Last() (T, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.head == 0 && !r.full {
		var z T
		return z, false
	}
	idx := r.head - 1
	if idx < 0 {
		idx = r.maxLen - 1
	}
	return r.slice[idx], true
}
