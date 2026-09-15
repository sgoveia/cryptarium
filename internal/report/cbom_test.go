package report_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sgoveia/cryptarium/internal/model"
	"github.com/sgoveia/cryptarium/internal/report"
)

func TestWriteCBOM_Deterministic(t *testing.T) {
	assets := []model.CryptoAsset{{
		CryptoFinding: model.CryptoFinding{
			ID:         "abc123",
			AssetType:  model.AssetAlgorithm,
			Primitive:  "RSA",
			Parameters: map[string]any{"keySize": 2048},
			Functions:  []model.CryptoFunction{model.FunctionKeygen},
			Evidence:   model.Evidence{Path: "token.go", Line: 8, RuleID: "go.crypto.rsa.generatekey"},
		},
		QuantumClass: model.ClassBroken,
		OID:          "1.2.840.113549.1.1.1",
	}}
	var a, b bytes.Buffer
	meta := report.Meta{ToolVersion: "0.0.0-dev", Deterministic: true}
	if err := report.WriteCBOM(&a, assets, meta); err != nil {
		t.Fatal(err)
	}
	if err := report.WriteCBOM(&b, assets, meta); err != nil {
		t.Fatal(err)
	}
	if a.String() != b.String() {
		t.Fatal("CBOM not deterministic")
	}
	if !strings.Contains(a.String(), `"bomFormat": "CycloneDX"`) {
		t.Fatalf("missing bomFormat: %s", a.String())
	}
	if !strings.Contains(a.String(), `"cryptographic-asset"`) {
		t.Fatal("missing cryptographic-asset")
	}
	if !strings.Contains(a.String(), `"nistQuantumSecurityLevel": 0`) {
		t.Fatal("expected level 0 for broken")
	}
}
