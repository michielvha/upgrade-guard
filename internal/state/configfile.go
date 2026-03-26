package state

import (
	"github.com/vhco/upgrade-guard/internal/analyst"
	"github.com/vhco/upgrade-guard/internal/config"
)

// FromConfig builds PlatformState from the upgrade-guard.yaml config.
// It merges config values on top of any inferred state.
func FromConfig(cfg *config.Config, inferred *analyst.PlatformState) *analyst.PlatformState {
	state := inferred
	if state == nil {
		state = &analyst.PlatformState{
			Components: make(map[string]string),
		}
	}

	// Config values override inferred values
	if cfg.Platform.KubernetesVersion != "" {
		state.KubernetesVersion = cfg.Platform.KubernetesVersion
	}
	if cfg.Platform.Provider != "" {
		state.Provider = cfg.Platform.Provider
	}

	// Merge component versions (config wins)
	for name, version := range cfg.Components {
		state.Components[name] = version
	}

	// Add free-text context
	if cfg.Context != "" {
		state.Context = cfg.Context
	}

	return state
}
