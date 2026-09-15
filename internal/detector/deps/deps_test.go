package deps_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sgoveia/cryptarium/internal/collector"
	"github.com/sgoveia/cryptarium/internal/detector/deps"
	"github.com/sgoveia/cryptarium/internal/model"
)

func TestHandles(t *testing.T) {
	d := deps.New()
	if !d.Handles(collector.FileRef{Path: "go.mod"}) {
		t.Fatal("should handle go.mod")
	}
	if d.Handles(collector.FileRef{Path: "package.json"}) {
		t.Fatal("should not handle package.json yet")
	}
}

func TestDetect_GoModPositive(t *testing.T) {
	dir := t.TempDir()
	mod := `module example.com/app

go 1.26

require (
	golang.org/x/crypto v0.31.0
	github.com/cloudflare/circl v1.5.0
	github.com/stretchr/testify v1.9.0
)
`
	path := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(path, []byte(mod), 0o644); err != nil {
		t.Fatal(err)
	}

	catalog := filepath.Join("..", "..", "..", "rules", "libraries", "catalog.yaml")
	d := deps.NewWithCatalog(catalog)
	findings, err := d.Detect(context.Background(), collector.FileRef{
		Path:    "go.mod",
		AbsPath: path,
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(findings) != 2 {
		t.Fatalf("got %d findings, want 2: %+v", len(findings), findings)
	}
	for _, f := range findings {
		if f.AssetType != model.AssetLibrary {
			t.Fatalf("assetType=%q", f.AssetType)
		}
		if f.Evidence.Confidence != model.ConfidenceMedium && f.Evidence.Confidence != model.ConfidenceLow {
			t.Fatalf("confidence=%q (deps must not claim high)", f.Evidence.Confidence)
		}
		if f.Evidence.Source != model.SourceDependency {
			t.Fatalf("source=%q", f.Evidence.Source)
		}
	}
}

func TestDetect_GoModNegativeNoCrypto(t *testing.T) {
	dir := t.TempDir()
	mod := `module example.com/app

go 1.26

require github.com/stretchr/testify v1.9.0
`
	path := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(path, []byte(mod), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog := filepath.Join("..", "..", "..", "rules", "libraries", "catalog.yaml")
	findings, err := deps.NewWithCatalog(catalog).Detect(context.Background(), collector.FileRef{
		Path:    "go.mod",
		AbsPath: path,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %+v", findings)
	}
}
