package rules_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/sgoveia/cryptarium/internal/rules"
)

func TestLoadEmbeddedCatalog(t *testing.T) {
	libs, err := rules.LoadEmbeddedCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(libs) == 0 {
		t.Fatal("expected embedded catalog entries")
	}
}

func TestLoadEmbeddedRulePacks(t *testing.T) {
	packs, err := rules.LoadEmbeddedRulePacks()
	if err != nil {
		t.Fatal(err)
	}
	if len(packs) < 5 {
		t.Fatalf("expected embedded packs for go/python/js/java/c, got %d", len(packs))
	}
}

func TestLoadDefaultFallsBackOutsideTree(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	if _, err := rules.FindDefaultCatalog(); !errors.Is(err, rules.ErrNotFound) {
		t.Fatalf("FindDefaultCatalog: want ErrNotFound, got %v", err)
	}
	if _, err := rules.FindDefaultRulesDir(); !errors.Is(err, rules.ErrNotFound) {
		t.Fatalf("FindDefaultRulesDir: want ErrNotFound, got %v", err)
	}

	libs, err := rules.LoadDefaultCatalog()
	if err != nil {
		t.Fatalf("LoadDefaultCatalog: %v", err)
	}
	if len(libs) == 0 {
		t.Fatal("expected embedded catalog via LoadDefaultCatalog")
	}
	packs, err := rules.LoadDefaultRulePacks()
	if err != nil {
		t.Fatalf("LoadDefaultRulePacks: %v", err)
	}
	if len(packs) == 0 {
		t.Fatal("expected embedded packs via LoadDefaultRulePacks")
	}
}

func TestLoadDefaultPrefersOnDisk(t *testing.T) {
	// Running under the module, walking up should find repo-root rules/.
	path, err := rules.FindDefaultCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "catalog.yaml" {
		t.Fatalf("unexpected catalog path %q", path)
	}
	dir, err := rules.FindDefaultRulesDir()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(dir) != "rules" {
		t.Fatalf("unexpected rules dir %q", dir)
	}
}
