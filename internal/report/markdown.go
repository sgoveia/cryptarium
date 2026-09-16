package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/sgoveia/cryptarium/internal/model"
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

	counts := priorityCounts(result.Scored)
	reported := len(result.Findings)
	if len(result.Scored) > 0 {
		reported = len(result.Scored)
	}
	fmt.Fprintf(&b, "Findings: **%d**", reported)
	if len(result.Scored) > 0 {
		fmt.Fprintf(&b, " · %d critical · %d high · %d medium · %d low",
			counts[model.PriorityCritical], counts[model.PriorityHigh],
			counts[model.PriorityMedium], counts[model.PriorityLow])
	}
	if len(result.Warnings) > 0 {
		fmt.Fprintf(&b, " · Warnings: **%d**", len(result.Warnings))
	}
	b.WriteString("\n\n")

	if len(result.Findings) == 0 {
		b.WriteString("No cryptographic assets detected in scanned files.\n")
	} else if len(result.Scored) > 0 {
		b.WriteString("| Priority | Class | Location | Primitive | Recommendation |\n")
		b.WriteString("|---|---|---|---|---|\n")
		for _, s := range result.Scored {
			loc := s.Evidence.Path
			if s.Evidence.Line > 0 {
				loc = fmt.Sprintf("%s:%d", s.Evidence.Path, s.Evidence.Line)
			}
			rec := s.Recommendation.Target
			if s.Recommendation.Standard != "" {
				rec = fmt.Sprintf("%s (%s)", s.Recommendation.Target, s.Recommendation.Standard)
			}
			rec = strings.ReplaceAll(rec, "|", "\\|")
			fmt.Fprintf(&b, "| %s (%d) | %s | `%s` | %s | %s |\n",
				s.Risk.Priority, s.Risk.Score, s.QuantumClass, loc, s.Primitive, rec)
			if len(s.RelatedIDs) > 0 {
				fmt.Fprintf(&b, "| | | | | correlated with %d related finding(s) |\n", len(s.RelatedIDs))
			}
		}
		b.WriteString("\n")
		b.WriteString("Scores use vulnerability, longevity, exposure, and agility heuristics (DESIGN.md §7).\n")
	} else {
		b.WriteString("| Class | Location | Primitive | Evidence |\n")
		b.WriteString("|---|---|---|---|\n")
		for i, f := range result.Findings {
			loc := f.Evidence.Path
			if f.Evidence.Line > 0 {
				loc = fmt.Sprintf("%s:%d", f.Evidence.Path, f.Evidence.Line)
			}
			snippet := strings.ReplaceAll(f.Evidence.Snippet, "|", "\\|")
			class := string(f.Evidence.Confidence)
			if i < len(result.Assets) {
				class = string(result.Assets[i].QuantumClass)
			}
			fmt.Fprintf(&b, "| %s | `%s` | %s | %s |\n", class, loc, f.Primitive, snippet)
		}
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

func priorityCounts(scored []model.ScoredAsset) map[model.Priority]int {
	m := map[model.Priority]int{}
	for _, s := range scored {
		m[s.Risk.Priority]++
	}
	return m
}
