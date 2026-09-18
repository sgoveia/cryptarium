package rules_test

import (
	"os"
	"path/filepath"
	"strings"
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
	// Flagship Go rule (DESIGN §5).
	base := filepath.Join("..", "..", "testdata", "rules", "go.crypto.rsa.generatekey")
	for _, name := range []string{"positive.go", "negative.go"} {
		if _, err := os.Stat(filepath.Join(base, name)); err != nil {
			t.Fatalf("missing fixture %s: %v", name, err)
		}
	}

	// All C and C++ rules must ship positive+negative fixtures.
	dir := filepath.Join("..", "..", "rules")
	packs, err := rules.LoadRulePacksDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join("..", "..", "testdata", "rules")
	for _, p := range packs {
		if p.Language != "c" && p.Language != "cpp" {
			continue
		}
		ext := ".c"
		if p.Language == "cpp" {
			ext = ".cpp"
		}
		for _, r := range p.Rules {
			if !strings.HasPrefix(r.ID, "c.") && !strings.HasPrefix(r.ID, "cpp.") {
				t.Fatalf("unexpected rule id %q in %s pack", r.ID, p.Language)
			}
			dir := filepath.Join(root, r.ID)
			for _, name := range []string{"positive" + ext, "negative" + ext} {
				if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
					t.Fatalf("missing fixture for %s (%s): %v", r.ID, name, err)
				}
			}
		}
	}
}
