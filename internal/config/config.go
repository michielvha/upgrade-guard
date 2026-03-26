package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// GuardLevel controls the depth of analysis.
type GuardLevel string

const (
	LevelBasic    GuardLevel = "basic"
	LevelStandard GuardLevel = "standard"
	LevelFull     GuardLevel = "full"
)

// Config is the top-level configuration for upgrade-guard.
type Config struct {
	Level      GuardLevel        `yaml:"level"`
	Platform   Platform          `yaml:"platform"`
	Components map[string]string `yaml:"components"`
	Context    string            `yaml:"context"`
	Cluster    Cluster           `yaml:"cluster"`
	Settings   Settings          `yaml:"settings"`
}

// Platform describes the target Kubernetes platform.
type Platform struct {
	KubernetesVersion string `yaml:"kubernetes_version"`
	Provider          string `yaml:"provider"`
}

// Cluster holds connection details for live cluster queries (Full level).
type Cluster struct {
	KubeconfigPath string `yaml:"kubeconfig_path"`
	Context        string `yaml:"context"`
}

// Settings holds operational settings.
type Settings struct {
	LLMModel         string   `yaml:"llm_model"`
	OutputFormat     string   `yaml:"output_format"`
	IgnoreComponents []string `yaml:"ignore_components"`
	RiskThreshold    string   `yaml:"risk_threshold"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Level: LevelBasic,
		Settings: Settings{
			LLMModel:      "auto",
			OutputFormat:  "stdout",
			RiskThreshold: "low",
		},
	}
}

// Load reads the config file from the given path, falling back to defaults.
// If the file doesn't exist, it returns the default config without error.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		path = "upgrade-guard.yaml"
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return cfg, nil
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	cfg.applyDefaults()
	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Level == "" {
		c.Level = LevelBasic
	}
	if c.Settings.LLMModel == "" {
		c.Settings.LLMModel = "auto"
	}
	if c.Settings.OutputFormat == "" {
		c.Settings.OutputFormat = "stdout"
	}
	if c.Settings.RiskThreshold == "" {
		c.Settings.RiskThreshold = "low"
	}
}

// ModelForLevel returns the appropriate Claude model based on guard level and config.
func (c *Config) ModelForLevel() string {
	if c.Settings.LLMModel != "auto" {
		return c.Settings.LLMModel
	}
	switch c.Level {
	case LevelFull:
		return "claude-opus-4-6"
	default:
		return "claude-sonnet-4-6"
	}
}

// IsComponentIgnored checks if a component should be skipped.
func (c *Config) IsComponentIgnored(name string) bool {
	for _, ignored := range c.Settings.IgnoreComponents {
		if ignored == name {
			return true
		}
	}
	return false
}
