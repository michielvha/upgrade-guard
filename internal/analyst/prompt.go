package analyst

import (
	"fmt"
	"strings"
)

// BuildPrompt constructs the analysis prompt based on guard level and context.
func BuildPrompt(changes []ComponentChange, state *PlatformState, guardLevel string) string {
	var b strings.Builder

	b.WriteString(systemPrompt)
	b.WriteString("\n\n## What Changed\n\n")

	for _, c := range changes {
		fmt.Fprintf(&b, "- **%s** (%s): `%s` → `%s`\n", c.Name, c.Type, c.FromVersion, c.ToVersion)
		if c.Source != "" {
			fmt.Fprintf(&b, "  Source: %s\n", c.Source)
		}
		if len(c.Files) > 0 {
			fmt.Fprintf(&b, "  Files: %s\n", strings.Join(c.Files, ", "))
		}
	}

	if state != nil && guardLevel != "basic" {
		b.WriteString("\n## Current Platform State\n\n")

		if state.KubernetesVersion != "" {
			fmt.Fprintf(&b, "- **Kubernetes version:** %s\n", state.KubernetesVersion)
		}
		if state.Provider != "" {
			fmt.Fprintf(&b, "- **Cloud provider:** %s\n", state.Provider)
		}
		if len(state.Components) > 0 {
			b.WriteString("- **Installed components:**\n")
			for name, ver := range state.Components {
				fmt.Fprintf(&b, "  - %s: %s\n", name, ver)
			}
		}
		if state.Context != "" {
			fmt.Fprintf(&b, "\n## Additional Context\n\n%s\n", state.Context)
		}
	}

	b.WriteString("\n## Your Task\n\n")
	b.WriteString(taskPrompt(guardLevel))

	return b.String()
}

const systemPrompt = `You are a Kubernetes platform engineering expert reviewing a dependency update PR. Your job is to identify risks, breaking changes, and compatibility issues that the team should be aware of before merging.

Use web search to look up official changelogs, release notes, and compatibility matrices for the specific version transitions below. Do not rely solely on your training data — always verify with current sources.`

func taskPrompt(level string) string {
	base := `For each changed component:
1. Search for the official changelog/release notes between the from and to versions
2. Check if there are version compatibility requirements with Kubernetes or other components
3. Identify breaking changes, deprecations, or behavioral changes
4. Flag any upgrade ordering requirements
5. Note any required manual steps (CRD updates, migration scripts, etc.)

Rate each change:
- 🟢 LOW RISK — routine patch, no compatibility concerns
- 🟡 MEDIUM RISK — minor version bump with notable changes, review recommended
- 🔴 HIGH RISK — breaking changes, version skew risk, or manual steps required

`
	switch level {
	case "standard":
		return base + `You have the current platform state above. Use it to give SPECIFIC advice:
- If a component requires a minimum Kubernetes version, check against the actual cluster version
- Cross-reference with other installed components for known incompatibilities
- Consider the additional context provided about the platform setup

Output your analysis as structured markdown suitable for a GitHub PR comment. For each component, include sections for Compatibility, Breaking Changes, and Action Required (if any).`
	case "full":
		return base + `You have both the declared platform state AND live cluster state above. Give the most thorough analysis possible:
- Check for drift between declared and actual versions
- Validate CRD compatibility
- Check node-level requirements
- Cross-reference all component interdependencies

Output your analysis as structured markdown suitable for a GitHub PR comment. For each component, include sections for Compatibility, Breaking Changes, Drift Detection, and Action Required (if any).`
	default: // basic
		return base + `You do not have specific platform state, so provide a GENERAL compatibility matrix:
- List which Kubernetes versions are supported by the new version
- Note any version requirements for common companion components
- The team will use this to check against their own setup

Output your analysis as structured markdown suitable for a GitHub PR comment. For each component, include sections for Compatibility, Breaking Changes, and Action Required (if any).`
	}
}
