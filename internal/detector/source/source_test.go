package source_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sgoveia/cryptarium/internal/collector"
	"github.com/sgoveia/cryptarium/internal/detector/source"
)

func TestDetect_GoRSAGenerateKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.go")
	code := `package token

import (
	"crypto/rand"
	"crypto/rsa"
)

func New() {
	rsa.GenerateKey(rand.Reader, 2048)
}

// rsa.GenerateKey should not match in comments
`
	if err := os.WriteFile(path, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	rulesDir := filepath.Join("..", "..", "..", "rules")
	d := source.NewWithRulesDir(rulesDir)
	findings, err := d.Detect(context.Background(), collector.FileRef{Path: "token.go", AbsPath: path})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1: %+v", len(findings), findings)
	}
	f := findings[0]
	if f.Primitive != "RSA" || f.Evidence.RuleID != "go.crypto.rsa.generatekey" {
		t.Fatalf("unexpected: %+v", f)
	}
	if f.Parameters["keySize"] != 2048 {
		t.Fatalf("keySize=%v", f.Parameters["keySize"])
	}
	if f.Evidence.Line < 1 {
		t.Fatal("expected line")
	}
}

func TestDetect_GoNegativeCommentOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.go")
	code := `package note
// This mentions rsa.GenerateKey but does not call it.
func F() {}
`
	if err := os.WriteFile(path, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	d := source.NewWithRulesDir(filepath.Join("..", "..", "..", "rules"))
	findings, err := d.Detect(context.Background(), collector.FileRef{Path: "note.go", AbsPath: path})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %+v", findings)
	}
}

func TestHandles(t *testing.T) {
	d := source.New()
	if !d.Handles(collector.FileRef{Path: "a.go"}) {
		t.Fatal("go")
	}
	if !d.Handles(collector.FileRef{Path: "a.py"}) {
		t.Fatal("py")
	}
	if d.Handles(collector.FileRef{Path: "a.txt"}) {
		t.Fatal("txt")
	}
}
