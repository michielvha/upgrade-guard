package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/vhco/upgrade-guard/internal/analyst"
)

func TestWriteStdout_RawMarkdown(t *testing.T) {
	result := &analyst.AnalysisResult{
		RawMarkdown: "## Risk Assessment\nAll good.",
		GuardLevel:  "basic",
	}

	var buf bytes.Buffer
	if err := WriteStdout(&buf, result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Risk Assessment") {
		t.Error("expected raw markdown in output")
	}
}

func TestWriteStdout_Assessments(t *testing.T) {
	result := &analyst.AnalysisResult{
		GuardLevel: "standard",
		Assessments: []analyst.ComponentAssessment{
			{
				Component: analyst.ComponentChange{
					Name:        "cert-manager",
					FromVersion: "1.14.5",
					ToVersion:   "1.14.7",
				},
				Risk:    analyst.RiskLow,
				Summary: "Patch release, no issues.",
			},
			{
				Component: analyst.ComponentChange{
					Name:        "cluster-autoscaler",
					FromVersion: "1.29.0",
					ToVersion:   "1.30.1",
				},
				Risk:    analyst.RiskHigh,
				Summary: "Requires K8s >= 1.30",
			},
		},
	}

	var buf bytes.Buffer
	if err := WriteStdout(&buf, result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "cert-manager") {
		t.Error("expected cert-manager in output")
	}
	if !strings.Contains(output, "cluster-autoscaler") {
		t.Error("expected cluster-autoscaler in output")
	}
	if !strings.Contains(output, "HIGH") {
		t.Error("expected HIGH risk label in output")
	}
}

func TestWriteJSON(t *testing.T) {
	result := &analyst.AnalysisResult{
		GuardLevel: "basic",
		Assessments: []analyst.ComponentAssessment{
			{
				Component: analyst.ComponentChange{
					Name:        "cert-manager",
					FromVersion: "1.14.5",
					ToVersion:   "1.14.7",
				},
				Risk:    analyst.RiskLow,
				Summary: "Patch release.",
			},
		},
	}

	var buf bytes.Buffer
	if err := WriteJSON(&buf, result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify valid JSON
	var parsed analyst.AnalysisResult
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if parsed.GuardLevel != "basic" {
		t.Errorf("expected guard_level 'basic', got %q", parsed.GuardLevel)
	}
	if len(parsed.Assessments) != 1 {
		t.Fatalf("expected 1 assessment, got %d", len(parsed.Assessments))
	}
	if parsed.Assessments[0].Component.Name != "cert-manager" {
		t.Errorf("expected component 'cert-manager', got %q", parsed.Assessments[0].Component.Name)
	}
}

func TestFormatPRComment(t *testing.T) {
	result := &analyst.AnalysisResult{
		GuardLevel:  "standard",
		RawMarkdown: "### cert-manager\nAll good.",
	}

	comment := formatPRComment(result)

	if !strings.Contains(comment, commentMarker) {
		t.Error("comment should contain marker for update detection")
	}
	if !strings.Contains(comment, "Upgrade Guard") {
		t.Error("comment should contain tool name")
	}
	if !strings.Contains(comment, "cert-manager") {
		t.Error("comment should contain analysis content")
	}
	if !strings.Contains(comment, "standard") {
		t.Error("comment should contain guard level")
	}
}
