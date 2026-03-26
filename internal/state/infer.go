package state

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vhco/upgrade-guard/internal/analyst"
)

var (
	// Kubernetes version patterns across different sources
	k8sVersionTF    = regexp.MustCompile(`(?m)(?:kubernetes_version|cluster_version)\s*=\s*"([^"]+)"`)
	helmChartVer    = regexp.MustCompile(`(?m)^(?:version|appVersion):\s*["']?([^\s"']+)["']?`)
	helmChartName   = regexp.MustCompile(`(?m)^name:\s*["']?([^\s"']+)["']?`)
	imageTagPattern = regexp.MustCompile(`(?m)tag:\s*["']?v?([^\s"']+)["']?`)
	kustomizeTag    = regexp.MustCompile(`(?m)newTag:\s*["']?v?([^\s"']+)["']?`)
	kustomizeName   = regexp.MustCompile(`(?m)newName:\s*["']?([^\s"']+)["']?`)
)

// InferFromRepo scans the repository at repoRoot to discover platform state.
func InferFromRepo(repoRoot string) (*analyst.PlatformState, error) {
	state := &analyst.PlatformState{
		Components: make(map[string]string),
	}

	err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable paths
		}

		// Skip hidden directories and common non-relevant dirs
		if info.IsDir() {
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") || base == "node_modules" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process relevant file types
		base := filepath.Base(path)
		switch {
		case base == "Chart.yaml":
			scanHelmChart(path, state)
		case base == "kustomization.yaml" || base == "kustomization.yml":
			scanKustomization(path, state)
		case strings.HasSuffix(base, ".tf"):
			scanTerraform(path, state)
		case base == "values.yaml":
			scanHelmValues(path, state)
		}

		return nil
	})

	if err != nil {
		return state, err
	}

	return state, nil
}

func scanHelmChart(path string, state *analyst.PlatformState) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	content := string(data)

	var name, version string
	if m := helmChartName.FindStringSubmatch(content); m != nil {
		name = m[1]
	}
	if m := helmChartVer.FindStringSubmatch(content); m != nil {
		version = strings.TrimPrefix(m[1], "v")
	}

	if name != "" && version != "" {
		state.Components[name] = version
	}
}

func scanHelmValues(path string, state *analyst.PlatformState) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	content := string(data)

	// Infer component name from directory
	dir := filepath.Dir(path)
	name := filepath.Base(dir)
	if name == "." || name == "helm" || name == "charts" {
		return
	}

	if m := imageTagPattern.FindStringSubmatch(content); m != nil {
		version := strings.TrimPrefix(m[1], "v")
		if _, exists := state.Components[name]; !exists {
			state.Components[name] = version
		}
	}
}

func scanKustomization(path string, state *analyst.PlatformState) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	content := string(data)

	names := kustomizeName.FindAllStringSubmatch(content, -1)
	tags := kustomizeTag.FindAllStringSubmatch(content, -1)

	for i := 0; i < len(names) && i < len(tags); i++ {
		imageName := names[i][1]
		// Use last segment of image name as component name
		parts := strings.Split(imageName, "/")
		shortName := parts[len(parts)-1]
		version := strings.TrimPrefix(tags[i][1], "v")
		state.Components[shortName] = version
	}
}

func scanTerraform(path string, state *analyst.PlatformState) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	content := string(data)

	// Look for Kubernetes version in EKS/AKS/GKE resources
	if m := k8sVersionTF.FindStringSubmatch(content); m != nil {
		state.KubernetesVersion = m[1]
	}

	// Try to detect cloud provider
	lower := strings.ToLower(content)
	switch {
	case strings.Contains(lower, "aws_eks_cluster"):
		state.Provider = "eks"
	case strings.Contains(lower, "azurerm_kubernetes_cluster"):
		state.Provider = "aks"
	case strings.Contains(lower, "google_container_cluster"):
		state.Provider = "gke"
	}
}
