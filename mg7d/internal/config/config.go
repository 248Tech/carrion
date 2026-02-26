package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the root agent configuration.
type Config struct {
	Instances []Instance `yaml:"instances"`
	API       API        `yaml:"api"`
	Metrics   Metrics    `yaml:"metrics"`
}

// Instance is a single 7DTD server instance.
type Instance struct {
	Name    string       `yaml:"name"`
	LogPath string       `yaml:"log_path"`
	Telnet  Telnet       `yaml:"telnet"`
	Policy  Policy       `yaml:"policy"`
	Actions ActionsCfg   `yaml:"actions"`
}

// Telnet holds telnet connection and safety settings.
type Telnet struct {
	Host            string  `yaml:"host"`
	Port            int     `yaml:"port"`
	Password        string  `yaml:"password"`
	RateLimitPerSec float64 `yaml:"rate_limit_per_sec"`
}

// Policy holds policy-specific config (e.g. fps_guard).
type Policy struct {
	FPSGuard *FPSGuardPolicy `yaml:"fps_guard"`
}

// FPSGuardPolicy config for FPS guardrail.
type FPSGuardPolicy struct {
	Enabled              bool    `yaml:"enabled"`
	ThresholdLow         float64 `yaml:"threshold_low"`
	ThresholdRestore     float64 `yaml:"threshold_restore"`
	RequireLowSamples    int     `yaml:"require_low_samples"`
	SampleWindowSamples  int     `yaml:"sample_window_samples"`
	RestoreStableSeconds float64 `yaml:"restore_stable_seconds"`
	CooldownSeconds      float64 `yaml:"cooldown_seconds"`
	DeltaSpikeThreshold  float64 `yaml:"delta_spike_threshold"`
	SpikeWindowSeconds   float64 `yaml:"spike_window_seconds"`
	ThrottleProfile      string  `yaml:"throttle_profile"`
}

// ActionsCfg holds action-related config (e.g. throttle profiles and baseline).
type ActionsCfg struct {
	ThrottleProfiles map[string]ThrottleProfile `yaml:"throttle_profiles"`
	Baseline         map[string]string         `yaml:"baseline"` // pref -> value for RestoreBaseline
}

// ThrottleProfile is a named list of steps (game pref sets).
type ThrottleProfile struct {
	Steps []ThrottleStep `yaml:"steps"`
}

// ThrottleStep is one step in a throttle profile.
type ThrottleStep struct {
	Pref  string `yaml:"pref"`
	Value string `yaml:"value"`
}

// API holds HTTP API settings.
type API struct {
	Listen    string `yaml:"listen"`
	AuthToken string `yaml:"auth_token"`
}

// Metrics holds metrics exposition settings.
type Metrics struct {
	Enable bool   `yaml:"enable"`
	Path   string `yaml:"path"`
}

// Load reads and validates config from path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := Validate(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Validate checks config and fails fast on invalid values.
func Validate(c *Config) error {
	if len(c.Instances) == 0 {
		return fmt.Errorf("config: at least one instance required")
	}
	for i, inst := range c.Instances {
		if inst.Name == "" {
			return fmt.Errorf("config: instances[%d].name required", i)
		}
		if inst.LogPath == "" {
			return fmt.Errorf("config: instances[%d].log_path required", i)
		}
		if inst.Telnet.RateLimitPerSec <= 0 {
			inst.Telnet.RateLimitPerSec = 2.0
		}
	}
	if c.API.Listen == "" {
		c.API.Listen = "127.0.0.1:9090"
	}
	if c.Metrics.Path == "" {
		c.Metrics.Path = "/metrics"
	}
	return nil
}
