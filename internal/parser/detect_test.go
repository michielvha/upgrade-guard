package parser

import (
	"testing"
)

func TestDetectChanges_WithRenovateMeta(t *testing.T) {
	diffs := []FileDiff{
		{
			Path:         "charts/cert-manager/Chart.yaml",
			RemovedLines: []string{"version: 1.14.5"},
			AddedLines:   []string{"version: 1.14.7"},
		},
	}

	meta := []RenovateMetadata{
		{
			PackageName: "cert-manager",
			FromVersion: "1.14.5",
			ToVersion:   "1.14.7",
		},
	}

	changes := DetectChanges(diffs, meta)

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	c := changes[0]
	if c.Name != "cert-manager" {
		t.Errorf("expected name 'cert-manager', got %q", c.Name)
	}
	if c.FromVersion != "1.14.5" {
		t.Errorf("expected from '1.14.5', got %q", c.FromVersion)
	}
	if c.ToVersion != "1.14.7" {
		t.Errorf("expected to '1.14.7', got %q", c.ToVersion)
	}
	if c.Type != "helm-chart" {
		t.Errorf("expected type 'helm-chart', got %q", c.Type)
	}
	if len(c.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(c.Files))
	}
}

func TestDetectChanges_DiffOnly_HelmChart(t *testing.T) {
	diffs := []FileDiff{
		{
			Path:         "charts/argocd/Chart.yaml",
			RemovedLines: []string{"version: 2.10.6", "name: argocd"},
			AddedLines:   []string{"version: 2.11.0", "name: argocd"},
		},
	}

	changes := DetectChanges(diffs, nil)

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	c := changes[0]
	if c.Name != "argocd" {
		t.Errorf("expected name 'argocd', got %q", c.Name)
	}
	if c.Type != "helm-chart" {
		t.Errorf("expected type 'helm-chart', got %q", c.Type)
	}
	if c.FromVersion != "2.10.6" {
		t.Errorf("expected from '2.10.6', got %q", c.FromVersion)
	}
	if c.ToVersion != "2.11.0" {
		t.Errorf("expected to '2.11.0', got %q", c.ToVersion)
	}
}

func TestDetectChanges_DiffOnly_ImageTag(t *testing.T) {
	diffs := []FileDiff{
		{
			Path:         "apps/nginx/values.yaml",
			RemovedLines: []string{"  tag: 1.25.0"},
			AddedLines:   []string{"  tag: 1.26.0"},
		},
	}

	changes := DetectChanges(diffs, nil)

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].FromVersion != "1.25.0" {
		t.Errorf("expected from '1.25.0', got %q", changes[0].FromVersion)
	}
	if changes[0].ToVersion != "1.26.0" {
		t.Errorf("expected to '1.26.0', got %q", changes[0].ToVersion)
	}
}

func TestDetectComponentType(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"charts/cert-manager/Chart.yaml", "helm-chart"},
		{"apps/nginx/values.yaml", "helm-chart"},
		{"overlays/prod/kustomization.yaml", "kustomize-ref"},
		{"infra/main.tf", "terraform-module"},
		{"apps/nginx/deployment.yaml", "container-image"},
	}

	for _, tt := range tests {
		got := detectComponentType(tt.path)
		if got != tt.want {
			t.Errorf("detectComponentType(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestDetectChanges_NoChanges(t *testing.T) {
	diffs := []FileDiff{
		{
			Path:         "README.md",
			RemovedLines: []string{"old text"},
			AddedLines:   []string{"new text"},
		},
	}

	changes := DetectChanges(diffs, nil)
	if len(changes) != 0 {
		t.Errorf("expected 0 changes for non-version files, got %d", len(changes))
	}
}
