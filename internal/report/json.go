package report

import (
	"encoding/json"
	"io"

	"github.com/sgoveia/cryptarium/internal/model"
	"github.com/sgoveia/cryptarium/internal/pipeline"
)

type jsonDocument struct {
	ToolVersion string                `json:"toolVersion"`
	Root        string                `json:"root,omitempty"`
	Findings    []model.CryptoFinding `json:"findings"`
	Warnings    []string              `json:"warnings,omitempty"`
}

// WriteJSON emits a deterministic JSON document of findings.
func WriteJSON(w io.Writer, result *pipeline.Result, meta Meta) error {
	if result == nil {
		result = &pipeline.Result{}
	}
	doc := jsonDocument{
		ToolVersion: meta.ToolVersion,
		Findings:    result.Findings,
		Warnings:    result.Warnings,
	}
	if !meta.Deterministic {
		doc.Root = meta.Root
	}
	if doc.Findings == nil {
		doc.Findings = []model.CryptoFinding{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(doc)
}
