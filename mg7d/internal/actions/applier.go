package actions

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mg7d/mg7d/internal/state"
	"github.com/mg7d/mg7d/internal/telnet"
)

// Applier applies actions via the telnet client. Bounded queue; drops on overload with audit.
type Applier struct {
	client    *telnet.Client
	audit     *state.AuditRing
	baseline  map[string]string
	baselineMu sync.RWMutex
	queue     chan Action
	queueSize int
	mu        sync.Mutex
	running   bool
	cancel    context.CancelFunc
}

// NewApplier creates an applier with a bounded queue.
func NewApplier(client *telnet.Client, audit *state.AuditRing, queueSize int) *Applier {
	if queueSize <= 0 {
		queueSize = 32
	}
	return &Applier{
		client:    client,
		audit:     audit,
		baseline:  make(map[string]string),
		queue:     make(chan Action, queueSize),
		queueSize: queueSize,
	}
}

// SetBaseline sets the baseline prefs for RestoreBaseline actions.
func (a *Applier) SetBaseline(m map[string]string) {
	a.baselineMu.Lock()
	defer a.baselineMu.Unlock()
	a.baseline = make(map[string]string)
	for k, v := range m {
		a.baseline[k] = v
	}
}

// Enqueue adds an action. If queue is full, records audit and returns error.
func (a *Applier) Enqueue(ctx context.Context, action Action) error {
	ev := state.AuditEvent{
		ActionID:   action.ID(),
		ActionType: action.Type(),
		Status:     "queued",
		QueuedAt:   time.Now(),
	}
	a.audit.Append(ev)
	select {
	case a.queue <- action:
		return nil
	default:
		ev.Status = "dropped"
		ev.Error = "queue full"
		a.audit.Append(ev)
		return fmt.Errorf("applier: queue full")
	}
}

// Run processes the queue until ctx is cancelled.
func (a *Applier) Run(ctx context.Context) {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return
	}
	ctx, a.cancel = context.WithCancel(ctx)
	a.running = true
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.running = false
		a.mu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case action, ok := <-a.queue:
			if !ok {
				return
			}
			a.applyOne(ctx, action)
		}
	}
}

func (a *Applier) applyOne(ctx context.Context, action Action) {
	sentAt := time.Now()
	ev := state.AuditEvent{
		ActionID:   action.ID(),
		ActionType: action.Type(),
		Status:     "sent",
		SentAt:     sentAt,
	}
	var err error
	switch act := action.(type) {
	case *SetGamePref:
		err = a.client.Send(ctx, telnet.SetGamePref(act.Pref, act.Value))
	case *Say:
		err = a.client.Send(ctx, telnet.Say(act.Message))
	case *RestoreBaseline:
		err = a.applyRestoreBaseline(ctx, act)
	case *Noop:
		err = nil
	default:
		err = fmt.Errorf("unknown action type: %T", action)
	}
	doneAt := time.Now()
	ev.DoneAt = doneAt
	if err != nil {
		ev.Status = "failure"
		ev.Error = err.Error()
	} else {
		ev.Status = "success"
	}
	a.audit.Append(ev)
}

// applyRestoreBaseline sends setpref for each baseline pref.
func (a *Applier) applyRestoreBaseline(ctx context.Context, _ *RestoreBaseline) error {
	a.baselineMu.RLock()
	baseline := a.baseline
	a.baselineMu.RUnlock()
	for pref, val := range baseline {
		if err := a.client.Send(ctx, telnet.SetGamePref(pref, val)); err != nil {
			return err
		}
	}
	return nil
}
