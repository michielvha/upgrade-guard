package analyst

import (
	"strings"
	"testing"
)

func TestBuildPrompt_Basic(t *testing.T) {
	changes := []ComponentChange{
		{
			Name:        "cert-manager",
			Type:        "helm-chart",
			FromVersion: "1.14.5",
			ToVersion:   "1.14.7",
			Files:       []string{"charts/cert-manager/Chart.yaml"},
		},
	}

	prompt := BuildPrompt(changes, nil, "basic")

	if !strings.Contains(prompt, "cert-manager") {
		t.Error("prompt should contain component name")
	}
	if !strings.Contains(prompt, "1.14.5") {
		t.Error("prompt should contain from version")
	}
	if !strings.Contains(prompt, "1.14.7") {
		t.Error("prompt should contain to version")
	}
	if !strings.Contains(prompt, "GENERAL compatibility matrix") {
		t.Error("basic level should request general compatibility matrix")
	}
	// Basic level should NOT include platform state section
	if strings.Contains(prompt, "Current Platform State") {
		t.Error("basic level should not include platform state")
	}
}

func TestBuildPrompt_Standard(t *testing.T) {
	changes := []ComponentChange{
		{
			Name:        "cluster-autoscaler",
			Type:        "container-image",
			FromVersion: "1.29.0",
			ToVersion:   "1.30.1",
		},
	}

	state := &PlatformState{
		KubernetesVersion: "1.29",
		Provider:          "eks",
		Components: map[string]string{
			"cert-manager": "1.14.5",
		},
		Context: "We use Cilium.",
	}

	prompt := BuildPrompt(changes, state, "standard")

	if !strings.Contains(prompt, "Current Platform State") {
		t.Error("standard level should include platform state")
	}
	if !strings.Contains(prompt, "1.29") {
		t.Error("prompt should contain k8s version")
	}
	if !strings.Contains(prompt, "eks") {
		t.Error("prompt should contain provider")
	}
	if !strings.Contains(prompt, "cert-manager") {
		t.Error("prompt should list installed components")
	}
	if !strings.Contains(prompt, "We use Cilium.") {
		t.Error("prompt should include context")
	}
	if !strings.Contains(prompt, "SPECIFIC advice") {
		t.Error("standard level should request specific advice")
	}
}

func TestBuildPrompt_MultipleChanges(t *testing.T) {
	changes := []ComponentChange{
		{Name: "cert-manager", Type: "helm-chart", FromVersion: "1.14.5", ToVersion: "1.14.7"},
		{Name: "argocd", Type: "helm-chart", FromVersion: "2.10.6", ToVersion: "2.11.0"},
	}

	prompt := BuildPrompt(changes, nil, "basic")

	if !strings.Contains(prompt, "cert-manager") {
		t.Error("prompt should contain first component")
	}
	if !strings.Contains(prompt, "argocd") {
		t.Error("prompt should contain second component")
	}
}
