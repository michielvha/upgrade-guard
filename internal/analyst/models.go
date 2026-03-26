package analyst

// RiskLevel represents the severity of an upgrade risk.
type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

// ComponentChange represents a single component version change detected in a PR.
type ComponentChange struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // helm-chart, container-image, kustomize-ref, terraform-module
	FromVersion string   `json:"from_version"`
	ToVersion   string   `json:"to_version"`
	Source      string   `json:"source,omitempty"`
	Files       []string `json:"files_changed"`
}

// ComponentAssessment is the risk assessment for a single component change.
type ComponentAssessment struct {
	Component ComponentChange `json:"component"`
	Risk      RiskLevel       `json:"risk"`
	Summary   string          `json:"summary"`
	Markdown  string          `json:"markdown"`
}

// AnalysisResult is the complete output of an upgrade analysis.
type AnalysisResult struct {
	Assessments []ComponentAssessment `json:"assessments"`
	RawMarkdown string                `json:"raw_markdown"`
	GuardLevel  string                `json:"guard_level"`
}

// PlatformState represents the current state of the platform.
type PlatformState struct {
	KubernetesVersion string            `json:"kubernetes_version,omitempty"`
	Provider          string            `json:"provider,omitempty"`
	Components        map[string]string `json:"components,omitempty"`
	Context           string            `json:"context,omitempty"`
}
