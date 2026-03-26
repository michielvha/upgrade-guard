package parser

import (
	"regexp"
	"strings"
)

// FileDiff represents the changes to a single file in a unified diff.
type FileDiff struct {
	Path       string
	AddedLines []string
	RemovedLines []string
}

var diffFileHeader = regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)
var diffHunkHeader = regexp.MustCompile(`^@@ .+ @@`)

// ParseUnifiedDiff parses a unified diff string into per-file diffs.
func ParseUnifiedDiff(diff string) []FileDiff {
	var diffs []FileDiff
	var current *FileDiff

	for _, line := range strings.Split(diff, "\n") {
		if matches := diffFileHeader.FindStringSubmatch(line); matches != nil {
			if current != nil {
				diffs = append(diffs, *current)
			}
			current = &FileDiff{Path: matches[2]}
			continue
		}

		if current == nil {
			continue
		}

		// Skip diff metadata lines
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") ||
			strings.HasPrefix(line, "index ") || diffHunkHeader.MatchString(line) {
			continue
		}

		if strings.HasPrefix(line, "+") {
			current.AddedLines = append(current.AddedLines, strings.TrimPrefix(line, "+"))
		} else if strings.HasPrefix(line, "-") {
			current.RemovedLines = append(current.RemovedLines, strings.TrimPrefix(line, "-"))
		}
	}

	if current != nil {
		diffs = append(diffs, *current)
	}

	return diffs
}

// ChangedFiles returns a list of file paths from the diff.
func ChangedFiles(diffs []FileDiff) []string {
	files := make([]string, len(diffs))
	for i, d := range diffs {
		files[i] = d.Path
	}
	return files
}
