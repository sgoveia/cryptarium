package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/sgoveia/cryptarium/internal/pipeline"
)

// WriteMarkdown emits a human-readable migration-oriented finding list.
func WriteMarkdown(w io.Writer, result *pipeline.Result, meta Meta) error {
	if result == nil {
		result = &pipeline.Result{}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# cryptarium report\n\n")
	fmt.Fprintf(&b, "Tool: cryptarium %s\n\n", meta.ToolVersion)
	if !meta.Deterministic && meta.Root != "" {
		fmt.Fprintf(&b, "Target: `%s`\n\n", meta.Root)
	}
	fmt.Fprintf(&b, "Findings: **%d**", len(result.Findings))
	if len(result.Warnings) > 0 {
		fmt.Fprintf(&b, " · Warnings: **%d**", len(result.Warnings))
	}
	b.WriteString("\n\n")

	if len(result.Findings) == 0 {
		b.WriteString("No cryptographic assets detected in scanned files.\n")
	} else {
		b.WriteString("| Class | Location | Primitive | Evidence |\n")
		b.WriteString("|---|---|---|---|\n")
		for i, f := range result.Findings {
			loc := f.Evidence.Path
			if f.Evidence.Line > 0 {
				loc = fmt.Sprintf("%s:%d", f.Evidence.Path, f.Evidence.Line)
			}
			snippet := strings.ReplaceAll(f.Evidence.Snippet, "|", "\\|")
			class := string(f.Evidence.Confidence) + " · " + string(f.Evidence.Source)
			if result != nil && i < len(result.Assets) {
				class = string(result.Assets[i].QuantumClass)
			}
			fmt.Fprintf(&b, "| %s | `%s` | %s | %s |\n",
				class,
				loc,
				f.Primitive,
				snippet,
			)
		}
		b.WriteString("\n")
		b.WriteString("Confidence reflects detection strength only. Classification and risk scoring land in later phases.\n")
	}

	if len(result.Warnings) > 0 {
		b.WriteString("\n## Warnings\n\n")
		for _, wmsg := range result.Warnings {
			fmt.Fprintf(&b, "- %s\n", wmsg)
		}
	}

	_, err := io.WriteString(w, b.String())
	return err
}
