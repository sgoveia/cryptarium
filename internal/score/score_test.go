package score_test

import (
	"testing"

	"github.com/sgoveia/cryptarium/internal/model"
	"github.com/sgoveia/cryptarium/internal/score"
)

func TestScore_BrokenSourceIsHighOrCritical(t *testing.T) {
	a := model.CryptoAsset{
		CryptoFinding: model.CryptoFinding{
			Primitive: "RSA",
			Functions: []model.CryptoFunction{model.FunctionKeygen},
			Evidence:  model.Evidence{Path: "services/auth/token.go", Source: model.SourceCode},
		},
		QuantumClass: model.ClassBroken,
	}
	got := score.Score(a)
	if got.Risk.Vulnerability != 100 {
		t.Fatalf("vulnerability=%d", got.Risk.Vulnerability)
	}
	if got.Risk.Score < 60 {
		t.Fatalf("score=%d, want >=60; %s", got.Risk.Score, got.Risk.Explanation)
	}
	if got.Risk.Priority != model.PriorityHigh && got.Risk.Priority != model.PriorityCritical {
		t.Fatalf("priority=%s", got.Risk.Priority)
	}
}

func TestScore_TestdataLowExposure(t *testing.T) {
	a := model.CryptoAsset{
		CryptoFinding: model.CryptoFinding{
			Primitive: "RSA",
			Evidence:  model.Evidence{Path: "testdata/certs/rsa2048.pem", Source: model.SourceCertificate},
		},
		QuantumClass: model.ClassBroken,
	}
	got := score.Score(a)
	if got.Risk.Exposure != 10 {
		t.Fatalf("exposure=%d", got.Risk.Exposure)
	}
	if got.Risk.Priority == model.PriorityCritical {
		t.Fatalf("testdata should not be critical: %s", got.Risk.Explanation)
	}
}

func TestScore_SafeIsLow(t *testing.T) {
	a := model.CryptoAsset{
		CryptoFinding: model.CryptoFinding{
			Primitive:  "AES",
			Parameters: map[string]any{"keySize": 256},
			Evidence:   model.Evidence{Path: "internal/x.go", Source: model.SourceCode},
		},
		QuantumClass: model.ClassSafe,
	}
	got := score.Score(a)
	if got.Risk.Vulnerability != 0 {
		t.Fatalf("vulnerability=%d", got.Risk.Vulnerability)
	}
	if got.Risk.Priority == model.PriorityCritical || got.Risk.Priority == model.PriorityHigh {
		t.Fatalf("safe should not be high/critical: %s", got.Risk.Explanation)
	}
}

func TestScore_Deterministic(t *testing.T) {
	a := model.CryptoAsset{
		CryptoFinding: model.CryptoFinding{
			Primitive: "ECDSA",
			Evidence:  model.Evidence{Path: "api.pem", Source: model.SourceCertificate},
		},
		QuantumClass: model.ClassBroken,
	}
	x := score.Score(a)
	y := score.Score(a)
	if x.Risk.Score != y.Risk.Score || x.Risk.Explanation != y.Risk.Explanation {
		t.Fatalf("nondeterministic: %+v vs %+v", x.Risk, y.Risk)
	}
}

func TestMeetsFailOn(t *testing.T) {
	if score.MeetsFailOn(model.PriorityHigh, "none") {
		t.Fatal("none should never fail")
	}
	if !score.MeetsFailOn(model.PriorityHigh, "high") {
		t.Fatal("high meets high")
	}
	if !score.MeetsFailOn(model.PriorityCritical, "high") {
		t.Fatal("critical meets high")
	}
	if score.MeetsFailOn(model.PriorityMedium, "high") {
		t.Fatal("medium should not meet high")
	}
}
