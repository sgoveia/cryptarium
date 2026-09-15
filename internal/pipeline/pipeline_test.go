package pipeline_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/sgoveia/cryptarium/internal/pipeline"
)

func TestRun_FixtureRepo(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "fixture-repo")
	catalog := filepath.Join("..", "..", "rules", "libraries", "catalog.yaml")
	result, err := pipeline.Run(context.Background(), pipeline.Options{
		Root:        root,
		CatalogPath: catalog,
		Concurrency: 2,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(result.Findings) < 3 {
		t.Fatalf("expected >=3 findings (2 certs + x/crypto), got %d: %+v", len(result.Findings), result.Findings)
	}

	// Determinism: second run byte-identical IDs and order.
	again, err := pipeline.Run(context.Background(), pipeline.Options{
		Root:        root,
		CatalogPath: catalog,
		Concurrency: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Findings) != len(result.Findings) {
		t.Fatalf("count changed with concurrency: %d vs %d", len(result.Findings), len(again.Findings))
	}
	for i := range result.Findings {
		if result.Findings[i].ID != again.Findings[i].ID {
			t.Fatalf("ID order mismatch at %d: %q vs %q", i, result.Findings[i].ID, again.Findings[i].ID)
		}
		if result.Findings[i].Evidence.Path != again.Findings[i].Evidence.Path {
			t.Fatalf("path order mismatch at %d", i)
		}
	}
}
