package state

import (
	"sync"
	"time"

	"github.com/mg7d/mg7d/internal/util"
)

// AuditEvent records one action lifecycle (queued/sent/success/failure).
type AuditEvent struct {
	ActionID   string
	ActionType string
	Status     string // queued, sent, success, failure
	Error      string
	QueuedAt   time.Time
	SentAt     time.Time
	DoneAt     time.Time
}

// AuditRing is a fixed-size ring buffer of audit events.
type AuditRing struct {
	ring *util.Ring[AuditEvent]
}

// NewAuditRing creates an audit ring with maxLen capacity.
func NewAuditRing(maxLen int) *AuditRing {
	return &AuditRing{ring: util.NewRing[AuditEvent](maxLen)}
}

// Append adds an event to the ring.
func (a *AuditRing) Append(ev AuditEvent) {
	a.ring.Append(ev)
}

// CopyOut copies up to len(dst) recent events into dst, oldest first. Returns count.
func (a *AuditRing) CopyOut(dst []AuditEvent) int {
	return a.ring.CopyOut(dst)
}

// Len returns the number of events in the ring.
func (a *AuditRing) Len() int {
	return a.ring.Len()
}
