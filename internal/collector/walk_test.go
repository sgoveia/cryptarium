package collector_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sgoveia/cryptarium/internal/collector"
)

func TestWalk_ListsRegularFiles(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.txt"), "a")
	mustWrite(t, filepath.Join(root, "sub", "b.go"), "package b")
	mustMkdir(t, filepath.Join(root, ".git", "objects"))
	mustWrite(t, filepath.Join(root, ".git", "objects", "pack"), "nope")
	mustMkdir(t, filepath.Join(root, "node_modules", "x"))
	mustWrite(t, filepath.Join(root, "node_modules", "x", "index.js"), "nope")

	refs, err := collector.Walk(context.Background(), collector.Target{Root: root})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("got %d files, want 2: %+v", len(refs), refs)
	}
	if refs[0].Path != "a.txt" || refs[1].Path != "sub/b.go" {
		t.Fatalf("paths = %q, %q", refs[0].Path, refs[1].Path)
	}
	if refs[0].AbsPath == "" || refs[0].Size == 0 {
		t.Fatalf("missing abs/size on %+v", refs[0])
	}
}

func TestWalk_Cancelable(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.txt"), "a")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := collector.Walk(ctx, collector.Target{Root: root})
	if err == nil {
		t.Fatal("expected canceled context error")
	}
}

func TestWalk_RejectsFileRoot(t *testing.T) {
	f := filepath.Join(t.TempDir(), "only.txt")
	mustWrite(t, f, "x")
	_, err := collector.Walk(context.Background(), collector.Target{Root: f})
	if err == nil {
		t.Fatal("expected error for file root")
	}
}

func TestExt(t *testing.T) {
	got := collector.Ext(collector.FileRef{Path: "certs/API.PEM"})
	if got != ".pem" {
		t.Fatalf("Ext = %q, want .pem", got)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	mustMkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}
