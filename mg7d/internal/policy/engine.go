package policy

import (
	"fmt"
	"sync"
	"time"

	"github.com/mg7d/mg7d/internal/actions"
	"github.com/mg7d/mg7d/internal/config"
	"github.com/mg7d/mg7d/internal/state"
)

// Engine evaluates policies on each snapshot and emits actions.
type Engine struct {
	instanceName string
	cfg          config.Instance
	fpsGuard     *FPSGuard
	mu           sync.Mutex
}

// NewEngine creates a policy engine for one instance.
func NewEngine(instanceName string, cfg config.Instance) *Engine {
	e := &Engine{
		instanceName: instanceName,
		cfg:          cfg,
	}
	if cfg.Policy.FPSGuard != nil && cfg.Policy.FPSGuard.Enabled {
		e.fpsGuard = NewFPSGuard(instanceName, cfg.Policy.FPSGuard, cfg.Actions.ThrottleProfiles)
	}
	return e
}

// Evaluate runs policies on the snapshot and returns actions to apply.
// Only emits actions on state transitions (no repeated identical actions).
func (e *Engine) Evaluate(snap state.Snapshot) []actions.Action {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []actions.Action
	if e.fpsGuard != nil {
		if a := e.fpsGuard.Evaluate(snap); a != nil {
			out = append(out, a)
		}
	}
	return out
}

var actionIDCounter int
var actionIDMu sync.Mutex

func newActionID() string {
	actionIDMu.Lock()
	actionIDCounter++
	id := actionIDCounter
	actionIDMu.Unlock()
	return fmt.Sprintf("act-%d", id)
}
