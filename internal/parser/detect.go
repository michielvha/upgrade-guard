package parser

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vhco/upgrade-guard/internal/analyst"
)

var (
	// Helm Chart.yaml version patterns
	helmChartVersion = regexp.MustCompile(`(?m)^version:\s*["']?([^\s"']+)["']?`)
	helmAppVersion   = regexp.MustCompile(`(?m)^appVersion:\s*["']?([^\s"']+)["']?`)
	helmChartName    = regexp.MustCompile(`(?m)^name:\s*["']?([^\s"']+)["']?`)

	// Container image patterns
	imageTag     = regexp.MustCompile(`(?m)(?:image|tag):\s*["']?([^\s"']+):([^\s"']+)["']?`)
	imageTagOnly = regexp.MustCompile(`(?m)tag:\s*["']?v?([^\s"']+)["']?`)

	// Kustomize image patterns
	kustomizeImage = regexp.MustCompile(`(?m)newTag:\s*["']?v?([^\s"']+)["']?`)
	kustomizeName  = regexp.MustCompile(`(?m)newName:\s*["']?([^\s"']+)["']?`)
)

// DetectChanges analyzes file diffs to detect component version changes.
// It merges with any Renovate metadata when available.
func DetectChanges(diffs []FileDiff, renovateMeta []RenovateMetadata) []analyst.ComponentChange {
	var changes []analyst.ComponentChange
	seen := make(map[string]bool)

	// If we have Renovate metadata, use it as the primary source
	// and enrich with file info from the diff.
	if len(renovateMeta) > 0 {
		for _, meta := range renovateMeta {
			change := analyst.ComponentChange{
				Name:        meta.PackageName,
				FromVersion: meta.FromVersion,
				ToVersion:   meta.ToVersion,
				Source:      meta.SourceURL,
			}

			// Find related files
			for _, d := range diffs {
				if isRelatedFile(d.Path, meta.PackageName) {
					change.Files = append(change.Files, d.Path)
					if change.Type == "" {
						change.Type = detectComponentType(d.Path)
					}
				}
			}
			if change.Type == "" {
				change.Type = "unknown"
			}

			seen[meta.PackageName] = true
			changes = append(changes, change)
		}
	}

	// Fall back to diff-based detection for any files not covered by Renovate metadata.
	for _, d := range diffs {
		detectedChanges := detectFromDiff(d)
		for _, c := range detectedChanges {
			if seen[c.Name] {
				continue
			}
			seen[c.Name] = true
			changes = append(changes, c)
		}
	}

	return changes
}

func detectComponentType(path string) string {
	base := filepath.Base(path)
	dir := filepath.Dir(path)

	switch {
	case base == "Chart.yaml" || base == "Chart.lock":
		return "helm-chart"
	case base == "kustomization.yaml" || base == "kustomization.yml":
		return "kustomize-ref"
	case strings.HasSuffix(base, ".tf"):
		return "terraform-module"
	case strings.Contains(dir, "helm") || base == "values.yaml":
		return "helm-chart"
	default:
		return "container-image"
	}
}

func isRelatedFile(path, componentName string) bool {
	lower := strings.ToLower(path)
	nameNorm := strings.ToLower(strings.ReplaceAll(componentName, "/", "-"))

	// Direct name match in path
	if strings.Contains(lower, nameNorm) {
		return true
	}

	// Last segment of component name (e.g. "jetstack/cert-manager" → "cert-manager")
	parts := strings.Split(nameNorm, "-")
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		if len(last) > 3 && strings.Contains(lower, last) {
			return true
		}
	}

	return false
}

func detectFromDiff(d FileDiff) []analyst.ComponentChange {
	var changes []analyst.ComponentChange
	compType := detectComponentType(d.Path)

	switch compType {
	case "helm-chart":
		if c := detectHelmChange(d); c != nil {
			changes = append(changes, *c)
		}
	case "kustomize-ref":
		changes = append(changes, detectKustomizeChanges(d)...)
	default:
		if c := detectImageChange(d); c != nil {
			changes = append(changes, *c)
		}
	}

	return changes
}

func detectHelmChange(d FileDiff) *analyst.ComponentChange {
	base := filepath.Base(d.Path)

	if base == "Chart.yaml" {
		var oldVer, newVer, name string

		for _, line := range d.RemovedLines {
			if m := helmChartVersion.FindStringSubmatch(line); m != nil {
				oldVer = m[1]
			}
			if m := helmAppVersion.FindStringSubmatch(line); m != nil {
				oldVer = m[1]
			}
		}
		for _, line := range d.AddedLines {
			if m := helmChartVersion.FindStringSubmatch(line); m != nil {
				newVer = m[1]
			}
			if m := helmAppVersion.FindStringSubmatch(line); m != nil {
				newVer = m[1]
			}
			if m := helmChartName.FindStringSubmatch(line); m != nil {
				name = m[1]
			}
		}

		if oldVer != "" && newVer != "" && oldVer != newVer {
			if name == "" {
				name = inferNameFromPath(d.Path)
			}
			return &analyst.ComponentChange{
				Name:        name,
				Type:        "helm-chart",
				FromVersion: cleanVersion(oldVer),
				ToVersion:   cleanVersion(newVer),
				Files:       []string{d.Path},
			}
		}
	}

	// values.yaml — look for tag changes
	if base == "values.yaml" {
		return detectImageTagInValues(d)
	}

	return nil
}

func detectImageTagInValues(d FileDiff) *analyst.ComponentChange {
	var oldTag, newTag string

	for _, line := range d.RemovedLines {
		if m := imageTagOnly.FindStringSubmatch(line); m != nil {
			oldTag = m[1]
		}
	}
	for _, line := range d.AddedLines {
		if m := imageTagOnly.FindStringSubmatch(line); m != nil {
			newTag = m[1]
		}
	}

	if oldTag != "" && newTag != "" && oldTag != newTag {
		return &analyst.ComponentChange{
			Name:        inferNameFromPath(d.Path),
			Type:        "container-image",
			FromVersion: cleanVersion(oldTag),
			ToVersion:   cleanVersion(newTag),
			Files:       []string{d.Path},
		}
	}
	return nil
}

func detectKustomizeChanges(d FileDiff) []analyst.ComponentChange {
	var changes []analyst.ComponentChange
	var oldTag, newTag, name string

	for _, line := range d.RemovedLines {
		if m := kustomizeImage.FindStringSubmatch(line); m != nil {
			oldTag = m[1]
		}
		if m := kustomizeName.FindStringSubmatch(line); m != nil {
			name = m[1]
		}
	}
	for _, line := range d.AddedLines {
		if m := kustomizeImage.FindStringSubmatch(line); m != nil {
			newTag = m[1]
		}
		if m := kustomizeName.FindStringSubmatch(line); m != nil {
			name = m[1]
		}
	}

	if oldTag != "" && newTag != "" && oldTag != newTag {
		if name == "" {
			name = inferNameFromPath(d.Path)
		}
		changes = append(changes, analyst.ComponentChange{
			Name:        name,
			Type:        "kustomize-ref",
			FromVersion: cleanVersion(oldTag),
			ToVersion:   cleanVersion(newTag),
			Files:       []string{d.Path},
		})
	}

	return changes
}

func detectImageChange(d FileDiff) *analyst.ComponentChange {
	var oldImage, oldTag, newImage, newTag string

	for _, line := range d.RemovedLines {
		if m := imageTag.FindStringSubmatch(line); m != nil {
			oldImage = m[1]
			oldTag = m[2]
		}
	}
	for _, line := range d.AddedLines {
		if m := imageTag.FindStringSubmatch(line); m != nil {
			newImage = m[1]
			newTag = m[2]
		}
	}

	if oldTag != "" && newTag != "" && oldTag != newTag {
		name := newImage
		if name == "" {
			name = oldImage
		}
		if name == "" {
			name = inferNameFromPath(d.Path)
		}
		return &analyst.ComponentChange{
			Name:        name,
			Type:        "container-image",
			FromVersion: cleanVersion(oldTag),
			ToVersion:   cleanVersion(newTag),
			Files:       []string{d.Path},
		}
	}
	return nil
}

func inferNameFromPath(path string) string {
	dir := filepath.Dir(path)
	parts := strings.Split(dir, string(filepath.Separator))
	for i := len(parts) - 1; i >= 0; i-- {
		p := parts[i]
		if p != "" && p != "." && p != "charts" && p != "helm" && p != "manifests" &&
			p != "base" && p != "overlays" && p != "templates" {
			return p
		}
	}
	return filepath.Base(dir)
}
