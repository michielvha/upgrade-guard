package state

import (
	"testing"

	"github.com/vhco/upgrade-guard/internal/analyst"
	"github.com/vhco/upgrade-guard/internal/config"
)

func TestFromConfig_MergesOnInferred(t *testing.T) {
	inferred := &analyst.PlatformState{
		KubernetesVersion: "1.28",
		Provider:          "gke",
		Components: map[string]string{
			"cert-manager": "1.13.0",
			"nginx":        "1.25.0",
		},
	}

	cfg := &config.Config{
		Platform: config.Platform{
			KubernetesVersion: "1.29",
			// Provider left empty — should keep inferred
		},
		Components: map[string]string{
			"cert-manager": "1.14.5", // overrides inferred
			"argocd":       "2.10.6", // new
		},
		Context: "We use Cilium.",
	}

	state := FromConfig(cfg, inferred)

	if state.KubernetesVersion != "1.29" {
		t.Errorf("expected k8s version '1.29' (from config), got %q", state.KubernetesVersion)
	}
	if state.Provider != "gke" {
		t.Errorf("expected provider 'gke' (from inferred), got %q", state.Provider)
	}
	if state.Components["cert-manager"] != "1.14.5" {
		t.Errorf("expected cert-manager '1.14.5' (config override), got %q", state.Components["cert-manager"])
	}
	if state.Components["nginx"] != "1.25.0" {
		t.Errorf("expected nginx '1.25.0' (from inferred), got %q", state.Components["nginx"])
	}
	if state.Components["argocd"] != "2.10.6" {
		t.Errorf("expected argocd '2.10.6' (from config), got %q", state.Components["argocd"])
	}
	if state.Context != "We use Cilium." {
		t.Errorf("expected context to be set, got %q", state.Context)
	}
}

func TestFromConfig_NilInferred(t *testing.T) {
	cfg := &config.Config{
		Platform: config.Platform{
			KubernetesVersion: "1.30",
			Provider:          "eks",
		},
		Components: map[string]string{
			"cert-manager": "1.14.5",
		},
	}

	state := FromConfig(cfg, nil)

	if state.KubernetesVersion != "1.30" {
		t.Errorf("expected '1.30', got %q", state.KubernetesVersion)
	}
	if state.Provider != "eks" {
		t.Errorf("expected 'eks', got %q", state.Provider)
	}
	if state.Components["cert-manager"] != "1.14.5" {
		t.Errorf("expected '1.14.5', got %q", state.Components["cert-manager"])
	}
}
