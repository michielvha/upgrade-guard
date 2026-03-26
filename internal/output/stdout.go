package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/vhco/upgrade-guard/internal/analyst"
)

// WriteStdout writes the analysis result to the given writer as formatted text.
func WriteStdout(w io.Writer, result *analyst.AnalysisResult) error {
	if result.RawMarkdown != "" {
		_, err := fmt.Fprintln(w, result.RawMarkdown)
		return err
	}

	fmt.Fprintf(w, "🛡️ Upgrade Guard — Risk Assessment (level: %s)\n", result.GuardLevel)
	fmt.Fprintln(w, strings.Repeat("=", 60))

	for _, a := range result.Assessments {
		riskIcon := riskIcon(a.Risk)
		fmt.Fprintf(w, "\n%s %s  %s → %s  %s %s\n",
			riskIcon, a.Component.Name,
			a.Component.FromVersion, a.Component.ToVersion,
			strings.ToUpper(string(a.Risk)), "RISK")
		fmt.Fprintln(w, strings.Repeat("-", 40))
		if a.Markdown != "" {
			fmt.Fprintln(w, a.Markdown)
		} else {
			fmt.Fprintln(w, a.Summary)
		}
	}

	return nil
}

func riskIcon(r analyst.RiskLevel) string {
	switch r {
	case analyst.RiskHigh:
		return "🔴"
	case analyst.RiskMedium:
		return "🟡"
	default:
		return "🟢"
	}
}
