package report

import (
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"

	"github.com/sgoveia/cryptarium/internal/model"
	"github.com/sgoveia/cryptarium/internal/pipeline"
)

// WriteSARIF emits SARIF 2.1.0 results for scored assets (DESIGN.md §10).
// Confidence is carried in result properties (DESIGN.md §17: property bag).
func WriteSARIF(w io.Writer, result *pipeline.Result, meta Meta) error {
	if result == nil {
		result = &pipeline.Result{}
	}
	scored := result.Scored
	rules := collectSARIFRules(scored)
	sort.Slice(rules, func(i, j int) bool { return rules[i].ID < rules[j].ID })

	runResults := make([]sarifResult, 0, len(scored))
	for _, s := range scored {
		runResults = append(runResults, scoredToSARIFResult(s))
	}

	doc := sarifLog{
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{{
			Tool: sarifTool{Driver: sarifDriver{
				Name:           "cryptarium",
				InformationURI: "https://github.com/sgoveia/cryptarium",
				Version:        meta.ToolVersion,
				Rules:          rules,
			}},
			Results: runResults,
		}},
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(doc)
}

func collectSARIFRules(scored []model.ScoredAsset) []sarifReportingDescriptor {
	seen := map[string]sarifReportingDescriptor{}
	for _, s := range scored {
		id := s.Evidence.RuleID
		if id == "" {
			id = "cryptarium.unspecified"
		}
		if _, ok := seen[id]; ok {
			continue
		}
		help := s.Recommendation.Target
		if s.Recommendation.Standard != "" {
			help = fmt.Sprintf("%s (%s)", s.Recommendation.Target, s.Recommendation.Standard)
		}
		seen[id] = sarifReportingDescriptor{
			ID:   id,
			Name: id,
			ShortDescription: sarifMessage{
				Text: fmt.Sprintf("%s (%s)", s.Primitive, s.QuantumClass),
			},
			FullDescription: sarifMessage{
				Text: s.Rationale,
			},
			Help: sarifMessage{
				Text: help,
			},
			HelpURI: "https://github.com/sgoveia/cryptarium",
			DefaultConfiguration: &sarifDefaultConfig{
				Level: priorityToLevel(s.Risk.Priority),
			},
		}
	}
	out := make([]sarifReportingDescriptor, 0, len(seen))
	for _, r := range seen {
		out = append(out, r)
	}
	return out
}

func scoredToSARIFResult(s model.ScoredAsset) sarifResult {
	ruleID := s.Evidence.RuleID
	if ruleID == "" {
		ruleID = "cryptarium.unspecified"
	}
	msg := fmt.Sprintf("%s at %s; %s; score %d (%s)",
		s.Primitive, locationString(s), s.Rationale, s.Risk.Score, s.Risk.Priority)
	if s.Recommendation.Target != "" {
		msg += "; migrate to " + s.Recommendation.Target
		if s.Recommendation.Standard != "" {
			msg += " (" + s.Recommendation.Standard + ")"
		}
	}
	res := sarifResult{
		RuleID: ruleID,
		Level:  priorityToLevel(s.Risk.Priority),
		Message: sarifMessage{
			Text: msg,
		},
		// Omit partialFingerprints: GitHub's upload-sarif action computes
		// primaryLocationLineHash from file content. Putting the finding ID
		// there caused inconsistent-fingerprint warnings and unstable alerts.
		Properties: map[string]any{
			"findingId":       s.ID,
			"confidence":      string(s.Evidence.Confidence),
			"quantumClass":    string(s.QuantumClass),
			"riskScore":       s.Risk.Score,
			"riskPriority":    string(s.Risk.Priority),
			"riskExplanation": s.Risk.Explanation,
		},
	}
	if s.Evidence.Path != "" {
		loc := sarifLocation{
			PhysicalLocation: sarifPhysicalLocation{
				ArtifactLocation: sarifArtifactLocation{
					URI: path.Clean(s.Evidence.Path),
				},
			},
		}
		if s.Evidence.Line > 0 {
			col := s.Evidence.Column
			if col < 1 {
				col = 1
			}
			loc.PhysicalLocation.Region = &sarifRegion{
				StartLine:   s.Evidence.Line,
				StartColumn: col,
			}
		}
		res.Locations = []sarifLocation{loc}
	}
	return res
}

func priorityToLevel(p model.Priority) string {
	switch p {
	case model.PriorityCritical, model.PriorityHigh:
		return "error"
	case model.PriorityMedium:
		return "warning"
	default:
		return "note"
	}
}

func locationString(s model.ScoredAsset) string {
	if s.Evidence.Line > 0 {
		return fmt.Sprintf("%s:%d", s.Evidence.Path, s.Evidence.Line)
	}
	return s.Evidence.Path
}

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string                     `json:"name"`
	InformationURI string                     `json:"informationUri,omitempty"`
	Version        string                     `json:"version,omitempty"`
	Rules          []sarifReportingDescriptor `json:"rules,omitempty"`
}

type sarifReportingDescriptor struct {
	ID                   string              `json:"id"`
	Name                 string              `json:"name,omitempty"`
	ShortDescription     sarifMessage        `json:"shortDescription,omitempty"`
	FullDescription      sarifMessage        `json:"fullDescription,omitempty"`
	Help                 sarifMessage        `json:"help,omitempty"`
	HelpURI              string              `json:"helpUri,omitempty"`
	DefaultConfiguration *sarifDefaultConfig `json:"defaultConfiguration,omitempty"`
}

type sarifDefaultConfig struct {
	Level string `json:"level"`
}

type sarifResult struct {
	RuleID              string            `json:"ruleId"`
	Level               string            `json:"level,omitempty"`
	Message             sarifMessage      `json:"message"`
	Locations           []sarifLocation   `json:"locations,omitempty"`
	PartialFingerprints map[string]string `json:"partialFingerprints,omitempty"`
	Properties          map[string]any    `json:"properties,omitempty"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           *sarifRegion          `json:"region,omitempty"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn,omitempty"`
}
