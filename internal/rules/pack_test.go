package rules_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sgoveia/cryptarium/internal/rules"
)

func TestLoadAllRulePacks(t *testing.T) {
	dir := filepath.Join("..", "..", "rules")
	packs, err := rules.LoadRulePacksDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(packs) < 5 {
		t.Fatalf("expected packs for go/python/js/java/c, got %d", len(packs))
	}
	seen := map[string]bool{}
	for _, p := range packs {
		for _, r := range p.Rules {
			if seen[r.ID] {
				t.Fatalf("duplicate rule id %q", r.ID)
			}
			seen[r.ID] = true
			if r.Primitive == "" {
				t.Fatalf("rule %s missing primitive", r.ID)
			}
		}
	}
}

func TestRuleFixturesExist(t *testing.T) {
	// Positive+negative fixtures for the flagship Go rule (DESIGN §5 invariant).
	base := filepath.Join("..", "..", "testdata", "rules", "go.crypto.rsa.generatekey")
	for _, name := range []string{"positive.go", "negative.go"} {
		if _, err := os.Stat(filepath.Join(base, name)); err != nil {
			t.Fatalf("missing fixture %s: %v", name, err)
		}
	}
}
