package model_test

import (
	"encoding/json"
	"testing"

	"github.com/sgoveia/cryptarium/internal/model"
)

func TestFindingID_Deterministic(t *testing.T) {
	in := model.FindingIDInput{
		Detector:  "certs",
		RuleID:    "certs.x509.rsa",
		RelPath:   "certs/api.pem",
		Line:      0,
		Primitive: "RSA",
		Parameters: map[string]any{
			"keySize": 2048,
		},
	}

	a := model.FindingID(in)
	b := model.FindingID(in)
	if a != b {
		t.Fatalf("same input produced different IDs: %q vs %q", a, b)
	}
	if len(a) != 16 {
		t.Fatalf("ID length = %d, want 16", len(a))
	}
}

func TestFindingID_ParameterKeyOrderIndependent(t *testing.T) {
	a := model.FindingID(model.FindingIDInput{
		Detector:  "source",
		RuleID:    "go.crypto.rsa.generatekey",
		RelPath:   "token.go",
		Line:      88,
		Primitive: "RSA",
		Parameters: map[string]any{
			"keySize": 2048,
			"mode":    "OAEP",
		},
	})
	b := model.FindingID(model.FindingIDInput{
		Detector:  "source",
		RuleID:    "go.crypto.rsa.generatekey",
		RelPath:   "token.go",
		Line:      88,
		Primitive: "RSA",
		Parameters: map[string]any{
			"mode":    "OAEP",
			"keySize": 2048,
		},
	})
	if a != b {
		t.Fatalf("parameter insertion order affected ID: %q vs %q", a, b)
	}
}

func TestFindingID_NormalizesPathSeparators(t *testing.T) {
	a := model.FindingID(model.FindingIDInput{
		Detector:  "certs",
		RuleID:    "certs.x509",
		RelPath:   `deploy\tls\server.pem`,
		Line:      0,
		Primitive: "ECDSA",
	})
	b := model.FindingID(model.FindingIDInput{
		Detector:  "certs",
		RuleID:    "certs.x509",
		RelPath:   "deploy/tls/server.pem",
		Line:      0,
		Primitive: "ECDSA",
	})
	if a != b {
		t.Fatalf("path separator normalization failed: %q vs %q", a, b)
	}
}

func TestFindingID_DiffersWhenInputsDiffer(t *testing.T) {
	base := model.FindingIDInput{
		Detector:  "deps",
		RuleID:    "deps.go.crypto-rsa",
		RelPath:   "go.mod",
		Line:      12,
		Primitive: "RSA",
	}
	a := model.FindingID(base)
	base.Line = 13
	b := model.FindingID(base)
	if a == b {
		t.Fatalf("different line produced same ID %q", a)
	}
}

func TestCryptoFinding_JSONRoundTrip(t *testing.T) {
	f := model.CryptoFinding{
		ID:        "0123456789abcdef",
		AssetType: model.AssetAlgorithm,
		Primitive: "RSA",
		Parameters: map[string]any{
			"keySize": float64(2048), // JSON numbers are float64
		},
		Functions: []model.CryptoFunction{model.FunctionKeygen},
		Evidence: model.Evidence{
			Source:     model.SourceCode,
			Path:       "token.go",
			Line:       88,
			RuleID:     "go.crypto.rsa.generatekey",
			Confidence: model.ConfidenceHigh,
		},
	}

	raw, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got model.CryptoFinding
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ID != f.ID || got.Primitive != f.Primitive || got.Evidence.Line != f.Evidence.Line {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}
