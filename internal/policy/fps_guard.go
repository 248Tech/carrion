package policy

import (
	"sync"
	"time"

	"github.com/mg7d/mg7d/internal/actions"
	"github.com/mg7d/mg7d/internal/config"
	"github.com/mg7d/mg7d/internal/state"
	"github.com/mg7d/mg7d/internal/util"
)

// FPSGuard implements the FPS guardrail policy with hysteresis and cooldown.
type FPSGuard struct {
	instanceName string
	cfg          *config.FPSGuardPolicy
	profiles     map[string]config.ThrottleProfile

	fpsRing    *util.Ring[float64]
	lastAction time.Time
	lastStep   int
	throttled  bool
	restoreAt  time.Time
	lowSince   time.Time
	mu         sync.Mutex
}

// NewFPSGuard creates an FPS guard policy.
func NewFPSGuard(instanceName string, cfg *config.FPSGuardPolicy, profiles map[string]config.ThrottleProfile) *FPSGuard {
	if cfg.SampleWindowSamples <= 0 {
		cfg.SampleWindowSamples = 60
	}
	if cfg.RequireLowSamples <= 0 {
		cfg.RequireLowSamples = 3
	}
	return &FPSGuard{
		instanceName: instanceName,
		cfg:          cfg,
		profiles:     profiles,
		fpsRing:      util.NewRing[float64](cfg.SampleWindowSamples),
	}
}

// Evaluate returns one action or nil. Uses hysteresis and cooldown.
func (g *FPSGuard) Evaluate(snap state.Snapshot) actions.Action {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()
	g.fpsRing.Append(snap.FPS)

	samples := g.fpsRing.Len()
	if samples < g.cfg.RequireLowSamples {
		return nil
	}

	// Count low FPS in window
	buf := make([]float64, g.fpsRing.Len())
	n := g.fpsRing.CopyOut(buf)
	lowCount := 0
	for i := 0; i < n; i++ {
		if buf[i] < g.cfg.ThresholdLow && buf[i] >= 0 {
			lowCount++
		}
	}

	profile, ok := g.profiles[g.cfg.ThrottleProfile]
	if !ok || len(profile.Steps) == 0 {
		return nil
	}

	// Cooldown: don't step again within cooldown_seconds
	cooldown := time.Duration(g.cfg.CooldownSeconds * float64(time.Second))
	if now.Sub(g.lastAction) < cooldown && g.throttled {
		return nil
	}

	// Trigger throttle if low for require_low_samples
	if lowCount >= g.cfg.RequireLowSamples {
		if !g.throttled {
			g.throttled = true
			g.lowSince = now
			g.lastAction = now
			g.lastStep = 0
			step := profile.Steps[0]
			return actions.NewSetGamePref(
				newActionID(), g.instanceName,
				"fps_guardrail: FPS below threshold",
				step.Pref, step.Value,
			)
		}
		// Already throttled: consider next step
		if g.lastStep+1 < len(profile.Steps) && now.Sub(g.lastAction) >= cooldown {
			g.lastStep++
			g.lastAction = now
			step := profile.Steps[g.lastStep]
			return actions.NewSetGamePref(
				newActionID(), g.instanceName,
				"fps_guardrail: stepping throttle",
				step.Pref, step.Value,
			)
		}
		return nil
	}

	// Restore: sustained FPS above threshold_restore
	if g.throttled {
		if snap.FPS >= g.cfg.ThresholdRestore {
			if g.restoreAt.IsZero() {
				g.restoreAt = now
			}
			stableDur := time.Duration(g.cfg.RestoreStableSeconds * float64(time.Second))
			if now.Sub(g.restoreAt) >= stableDur {
				g.throttled = false
				g.restoreAt = time.Time{}
				g.lastAction = now
				return actions.NewRestoreBaseline(
					newActionID(), g.instanceName,
					"fps_guardrail: FPS stable, restore baseline",
				)
			}
		} else {
			g.restoreAt = time.Time{}
		}
	}

	return nil
}
