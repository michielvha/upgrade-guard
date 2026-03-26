package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Level != LevelBasic {
		t.Errorf("expected level 'basic', got %q", cfg.Level)
	}
	if cfg.Settings.LLMModel != "auto" {
		t.Errorf("expected llm_model 'auto', got %q", cfg.Settings.LLMModel)
	}
	if cfg.Settings.OutputFormat != "stdout" {
		t.Errorf("expected output_format 'stdout', got %q", cfg.Settings.OutputFormat)
	}
	if cfg.Settings.RiskThreshold != "low" {
		t.Errorf("expected risk_threshold 'low', got %q", cfg.Settings.RiskThreshold)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/upgrade-guard.yaml")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if cfg.Level != LevelBasic {
		t.Errorf("expected default level 'basic', got %q", cfg.Level)
	}
}

func TestLoad_ValidFile(t *testing.T) {
	content := `
level: standard
platform:
  kubernetes_version: "1.29"
  provider: eks
components:
  cert-manager: "1.14.5"
settings:
  llm_model: claude-opus-4-6
  risk_threshold: medium
`
	dir := t.TempDir()
	path := filepath.Join(dir, "upgrade-guard.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Level != LevelStandard {
		t.Errorf("expected level 'standard', got %q", cfg.Level)
	}
	if cfg.Platform.KubernetesVersion != "1.29" {
		t.Errorf("expected k8s version '1.29', got %q", cfg.Platform.KubernetesVersion)
	}
	if cfg.Platform.Provider != "eks" {
		t.Errorf("expected provider 'eks', got %q", cfg.Platform.Provider)
	}
	if cfg.Components["cert-manager"] != "1.14.5" {
		t.Errorf("expected cert-manager '1.14.5', got %q", cfg.Components["cert-manager"])
	}
	if cfg.Settings.LLMModel != "claude-opus-4-6" {
		t.Errorf("expected llm_model 'claude-opus-4-6', got %q", cfg.Settings.LLMModel)
	}
	if cfg.Settings.RiskThreshold != "medium" {
		t.Errorf("expected risk_threshold 'medium', got %q", cfg.Settings.RiskThreshold)
	}
	// OutputFormat should get default since not specified
	if cfg.Settings.OutputFormat != "stdout" {
		t.Errorf("expected default output_format 'stdout', got %q", cfg.Settings.OutputFormat)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "upgrade-guard.yaml")
	if err := os.WriteFile(path, []byte("{{invalid yaml"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestModelForLevel(t *testing.T) {
	tests := []struct {
		level    GuardLevel
		model    string
		expected string
	}{
		{LevelBasic, "auto", "claude-sonnet-4-6"},
		{LevelStandard, "auto", "claude-sonnet-4-6"},
		{LevelFull, "auto", "claude-opus-4-6"},
		{LevelBasic, "claude-opus-4-6", "claude-opus-4-6"},
		{LevelFull, "claude-sonnet-4-6", "claude-sonnet-4-6"},
	}

	for _, tt := range tests {
		cfg := &Config{
			Level:    tt.level,
			Settings: Settings{LLMModel: tt.model},
		}
		got := cfg.ModelForLevel()
		if got != tt.expected {
			t.Errorf("ModelForLevel(level=%q, model=%q) = %q, want %q",
				tt.level, tt.model, got, tt.expected)
		}
	}
}

func TestIsComponentIgnored(t *testing.T) {
	cfg := &Config{
		Settings: Settings{
			IgnoreComponents: []string{"kube-proxy", "coredns"},
		},
	}

	if !cfg.IsComponentIgnored("kube-proxy") {
		t.Error("expected kube-proxy to be ignored")
	}
	if !cfg.IsComponentIgnored("coredns") {
		t.Error("expected coredns to be ignored")
	}
	if cfg.IsComponentIgnored("cert-manager") {
		t.Error("expected cert-manager to NOT be ignored")
	}
}
