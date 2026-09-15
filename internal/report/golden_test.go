package report_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/sgoveia/cryptarium/internal/pipeline"
	"github.com/sgoveia/cryptarium/internal/report"
)

var update = flag.Bool("update", false, "update golden files")

func TestGolden_JSON(t *testing.T) {
	result := scanFixture(t)
	var buf bytes.Buffer
	err := report.WriteJSON(&buf, result, report.Meta{
		ToolVersion:   "0.0.0-dev",
		Root:          "testdata/fixture-repo",
		Deterministic: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	compareGolden(t, "fixture-repo.json", buf.Bytes())
}

func TestGolden_Markdown(t *testing.T) {
	result := scanFixture(t)
	var buf bytes.Buffer
	err := report.WriteMarkdown(&buf, result, report.Meta{
		ToolVersion:   "0.0.0-dev",
		Deterministic: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	compareGolden(t, "fixture-repo.md", buf.Bytes())
}

func scanFixture(t *testing.T) *pipeline.Result {
	t.Helper()
	root := filepath.Join("..", "..", "testdata", "fixture-repo")
	catalog := filepath.Join("..", "..", "rules", "libraries", "catalog.yaml")
	result, err := pipeline.Run(t.Context(), pipeline.Options{
		Root:        root,
		CatalogPath: catalog,
		RulesDir:    filepath.Join("..", "..", "rules"),
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("pipeline: %v", err)
	}
	return result
}

func compareGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "golden", name)
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("updated %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s (run go test -update): %v", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}
