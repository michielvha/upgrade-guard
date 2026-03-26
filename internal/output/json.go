package output

import (
	"encoding/json"
	"io"

	"github.com/vhco/upgrade-guard/internal/analyst"
)

// WriteJSON writes the analysis result as JSON to the given writer.
func WriteJSON(w io.Writer, result *analyst.AnalysisResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
