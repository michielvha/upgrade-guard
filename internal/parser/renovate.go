package parser

import (
	"regexp"
	"strings"
)

// RenovateMetadata holds version info extracted from a Renovate PR body.
type RenovateMetadata struct {
	PackageName string
	FromVersion string
	ToVersion   string
	SourceURL   string
}

var (
	// Renovate PR body patterns
	renovateUpdateLine = regexp.MustCompile(`(?i)update\s+\x60(.+?)\x60\s+from\s+v?(\S+)\s+to\s+v?(\S+)`)
	renovateTable      = regexp.MustCompile(`\|\s*(.+?)\s*\|\s*\x60?v?([^\s\x60|]+)\x60?\s*\|\s*\x60?v?([^\s\x60|]+)\x60?\s*\|`)
	changelogURL       = regexp.MustCompile(`https?://\S+`)
)

// ParseRenovateBody extracts version change metadata from a Renovate PR body.
func ParseRenovateBody(body string) []RenovateMetadata {
	var results []RenovateMetadata
	seen := make(map[string]bool)

	// Try to match the tabular format Renovate often uses
	for _, match := range renovateTable.FindAllStringSubmatch(body, -1) {
		pkg := strings.TrimSpace(match[1])
		// Skip table headers
		if strings.Contains(strings.ToLower(pkg), "package") || strings.Contains(pkg, "---") {
			continue
		}
		if seen[pkg] {
			continue
		}
		seen[pkg] = true

		meta := RenovateMetadata{
			PackageName: pkg,
			FromVersion: cleanVersion(match[2]),
			ToVersion:   cleanVersion(match[3]),
		}
		results = append(results, meta)
	}

	// Try the "Update X from A to B" format
	if len(results) == 0 {
		for _, match := range renovateUpdateLine.FindAllStringSubmatch(body, -1) {
			pkg := match[1]
			if seen[pkg] {
				continue
			}
			seen[pkg] = true

			meta := RenovateMetadata{
				PackageName: pkg,
				FromVersion: cleanVersion(match[2]),
				ToVersion:   cleanVersion(match[3]),
			}
			results = append(results, meta)
		}
	}

	// Try to find changelog URLs and attach to results
	urls := changelogURL.FindAllString(body, -1)
	for i := range results {
		for _, u := range urls {
			if strings.Contains(strings.ToLower(u), strings.ToLower(results[i].PackageName)) {
				results[i].SourceURL = u
				break
			}
		}
	}

	return results
}

func cleanVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	return v
}
