package report

import (
	"fmt"
	"io"

	"github.com/sgoveia/cryptarium/internal/classify"
	"github.com/sgoveia/cryptarium/internal/model"
	"github.com/sgoveia/cryptarium/internal/pipeline"
)

// Format is a supported reporter name.
type Format string

// Known report formats.
const (
	FormatJSON     Format = "json"
	FormatMarkdown Format = "markdown"
	FormatCBOM     Format = "cbom"
	FormatSARIF    Format = "sarif"
)

// Write emits result in the named format to w.
func Write(w io.Writer, format Format, result *pipeline.Result, meta Meta) error {
	switch format {
	case FormatJSON:
		return WriteJSON(w, result, meta)
	case FormatMarkdown:
		return WriteMarkdown(w, result, meta)
	case FormatCBOM:
		assets := result.Assets
		if len(assets) == 0 && result != nil {
			assets = classifyFindings(result.Findings)
		}
		return WriteCBOM(w, assets, meta)
	case FormatSARIF:
		return WriteSARIF(w, result, meta)
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}

// Meta is scan metadata. Timestamps are omitted when Deterministic is true.
type Meta struct {
	ToolVersion   string
	Root          string
	Deterministic bool
}

// FindingCount is a helper for summaries.
func FindingCount(findings []model.CryptoFinding) int {
	return len(findings)
}

func classifyFindings(findings []model.CryptoFinding) []model.CryptoAsset {
	return classify.All(findings)
}
