package rules_test

import (
	"path/filepath"
	"testing"

	"github.com/sgoveia/cryptarium/internal/rules"
)

func TestLoadLibraryCatalog(t *testing.T) {
	path := filepath.Join("..", "..", "rules", "libraries", "catalog.yaml")
	libs, err := rules.LoadLibraryCatalog(path)
	if err != nil {
		t.Fatalf("LoadLibraryCatalog: %v", err)
	}
	if len(libs) == 0 {
		t.Fatal("expected at least one library")
	}
	found := false
	for _, lib := range libs {
		if lib.Name == "golang.org/x/crypto" {
			found = true
			if lib.Ecosystem != "go" {
				t.Fatalf("ecosystem=%q", lib.Ecosystem)
			}
			if len(lib.Primitives) == 0 {
				t.Fatal("missing primitives")
			}
		}
	}
	if !found {
		t.Fatal("golang.org/x/crypto not in catalog")
	}
}
