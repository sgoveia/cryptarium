package report_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sgoveia/cryptarium/internal/model"
	"github.com/sgoveia/cryptarium/internal/pipeline"
	"github.com/sgoveia/cryptarium/internal/report"
)

func TestWriteSARIF(t *testing.T) {
	result := &pipeline.Result{
		Scored: []model.ScoredAsset{{
			CryptoAsset: model.CryptoAsset{
				CryptoFinding: model.CryptoFinding{
					ID:        "deadbeefcafebabe",
					Primitive: "RSA",
					Evidence: model.Evidence{
						Path:       "token.go",
						Line:       9,
						RuleID:     "go.crypto.rsa.generatekey",
						Confidence: model.ConfidenceHigh,
					},
				},
				QuantumClass:   model.ClassBroken,
				Rationale:      "integer factorization; broken by Shor's algorithm",
				Recommendation: model.Migration{Target: "ML-DSA-65", Standard: "FIPS 204"},
			},
			Risk: model.RiskScore{Score: 82, Priority: model.PriorityCritical, Explanation: "test"},
		}},
	}
	var buf bytes.Buffer
	if err := report.WriteSARIF(&buf, result, report.Meta{ToolVersion: "0.0.0-dev", Deterministic: true}); err != nil {
		t.Fatal(err)
	}
	raw := buf.String()
	if !strings.Contains(raw, `"version": "2.1.0"`) {
		t.Fatalf("missing sarif version: %s", raw)
	}
	if !strings.Contains(raw, `"ruleId": "go.crypto.rsa.generatekey"`) {
		t.Fatal("missing ruleId")
	}
	if !strings.Contains(raw, `"level": "error"`) {
		t.Fatal("critical should map to error")
	}
	if !strings.Contains(raw, `"confidence"`) {
		t.Fatal("confidence property bag missing")
	}
	var doc map[string]any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
}
