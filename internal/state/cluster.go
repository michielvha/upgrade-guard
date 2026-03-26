package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/vhco/upgrade-guard/internal/analyst"
	"github.com/vhco/upgrade-guard/internal/config"
)

var k8sServerVersion = regexp.MustCompile(`Server Version:\s*v?(\S+)`)

// FromCluster gathers live cluster state by running kubectl and helm commands.
// It merges the results on top of any existing state from config/inference.
func FromCluster(ctx context.Context, cfg *config.Config, base *analyst.PlatformState) (*analyst.PlatformState, error) {
	if base == nil {
		base = &analyst.PlatformState{
			Components: make(map[string]string),
		}
	}

	kubectlArgs := kubectlBaseArgs(cfg)

	// Get K8s server version
	if ver, err := getK8sVersion(ctx, kubectlArgs); err == nil && ver != "" {
		base.KubernetesVersion = ver
	}

	// Get Helm releases
	if releases, err := getHelmReleases(ctx, kubectlArgs); err == nil {
		for name, ver := range releases {
			base.Components[name] = ver
		}
	}

	return base, nil
}

func kubectlBaseArgs(cfg *config.Config) []string {
	var args []string
	if cfg.Cluster.KubeconfigPath != "" {
		args = append(args, "--kubeconfig", cfg.Cluster.KubeconfigPath)
	}
	if cfg.Cluster.Context != "" {
		args = append(args, "--context", cfg.Cluster.Context)
	}
	return args
}

func getK8sVersion(ctx context.Context, baseArgs []string) (string, error) {
	args := append(baseArgs, "version", "--short")
	out, err := exec.CommandContext(ctx, "kubectl", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("kubectl version: %w", err)
	}

	if m := k8sServerVersion.FindStringSubmatch(string(out)); m != nil {
		return m[1], nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Server Version:") {
			ver := strings.TrimPrefix(line, "Server Version:")
			ver = strings.TrimSpace(ver)
			ver = strings.TrimPrefix(ver, "v")
			return ver, nil
		}
	}

	return "", fmt.Errorf("could not parse K8s version from output: %s", string(out))
}

type helmRelease struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Chart      string `json:"chart"`
	AppVersion string `json:"app_version"`
}

func getHelmReleases(ctx context.Context, kubectlArgs []string) (map[string]string, error) {
	args := []string{"list", "-A", "-o", "json"}
	for i := 0; i < len(kubectlArgs)-1; i += 2 {
		args = append(args, kubectlArgs[i], kubectlArgs[i+1])
	}

	out, err := exec.CommandContext(ctx, "helm", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("helm list: %w", err)
	}

	var releases []helmRelease
	if err := json.Unmarshal(out, &releases); err != nil {
		return nil, fmt.Errorf("parsing helm output: %w", err)
	}

	result := make(map[string]string)
	for _, r := range releases {
		ver := r.AppVersion
		if ver == "" {
			ver = extractChartVersion(r.Chart)
		}
		if ver != "" {
			result[r.Name] = strings.TrimPrefix(ver, "v")
		}
	}

	return result, nil
}

var chartVersionPattern = regexp.MustCompile(`-v?(\d+\.\d+\.\d+.*)$`)

func extractChartVersion(chart string) string {
	if m := chartVersionPattern.FindStringSubmatch(chart); m != nil {
		return m[1]
	}
	return ""
}
