package classify_test

import (
	"testing"

	"github.com/sgoveia/cryptarium/internal/classify"
	"github.com/sgoveia/cryptarium/internal/model"
)

func TestClassify_RSABroken(t *testing.T) {
	a := classify.Classify(model.CryptoFinding{Primitive: "RSA", Parameters: map[string]any{"keySize": 2048}})
	if a.QuantumClass != model.ClassBroken {
		t.Fatalf("class=%s", a.QuantumClass)
	}
	if a.Recommendation.Standard == "" {
		t.Fatal("missing standard citation path")
	}
}

func TestClassify_AESKeySize(t *testing.T) {
	w := classify.Classify(model.CryptoFinding{Primitive: "AES", Parameters: map[string]any{"keySize": 128}})
	if w.QuantumClass != model.ClassWeakened {
		t.Fatalf("AES-128 class=%s", w.QuantumClass)
	}
	s := classify.Classify(model.CryptoFinding{Primitive: "AES", Parameters: map[string]any{"keySize": 256}})
	if s.QuantumClass != model.ClassSafe {
		t.Fatalf("AES-256 class=%s", s.QuantumClass)
	}
}

func TestClassify_Unknown(t *testing.T) {
	a := classify.Classify(model.CryptoFinding{Primitive: "NOT-A-REAL-PRIMITIVE"})
	if a.QuantumClass != model.ClassUnknown {
		t.Fatalf("class=%s", a.QuantumClass)
	}
}

func TestCanonical(t *testing.T) {
	if classify.Canonical("sha256") != "SHA-256" && classify.Canonical("SHA256") != "SHA-256" {
		t.Fatalf("got %q", classify.Canonical("SHA256"))
	}
}
