package collector_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sgoveia/cryptarium/internal/collector"
)

func TestIsRemoteURL(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"https://github.com/o/r", true},
		{"http://example.com/r.git", true},
		{"file:///tmp/repo", true},
		{"./local", false},
		{"/abs/path", false},
		{"git@github.com:o/r.git", false},
		{"ssh://git@github.com/o/r.git", false},
	}
	for _, tc := range cases {
		if got := collector.IsRemoteURL(tc.in); got != tc.want {
			t.Errorf("IsRemoteURL(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestIsSSHURL(t *testing.T) {
	if !collector.IsSSHURL("git@github.com:o/r.git") {
		t.Fatal("expected git@ to be SSH")
	}
	if !collector.IsSSHURL("ssh://git@github.com/o/r.git") {
		t.Fatal("expected ssh:// to be SSH")
	}
	if collector.IsSSHURL("https://github.com/o/r") {
		t.Fatal("https must not be SSH")
	}
}

func TestResolve_LocalPath(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.txt"), "a")

	resolved, err := collector.Resolve(context.Background(), root)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	defer resolved.Cleanup()

	if resolved.Root != root && resolved.Root != mustAbs(t, root) {
		// Resolve returns Abs path
		abs, _ := filepath.Abs(root)
		if resolved.Root != abs {
			t.Fatalf("Root = %q, want abs of %q", resolved.Root, root)
		}
	}
	if resolved.Display != root {
		t.Fatalf("Display = %q, want %q", resolved.Display, root)
	}
	resolved.Cleanup() // must be safe to call twice / no-op for local
}

func TestResolve_SSHRejected(t *testing.T) {
	_, err := collector.Resolve(context.Background(), "git@github.com:o/r.git")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "SSH") {
		t.Fatalf("err = %v", err)
	}
}

func TestResolve_CloneLocalBare(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	src := t.TempDir()
	mustWrite(t, filepath.Join(src, "token.go"), "package main\n")
	runGit(t, src, "init")
	runGit(t, src, "config", "user.email", "test@example.com")
	runGit(t, src, "config", "user.name", "test")
	runGit(t, src, "add", ".")
	runGit(t, src, "commit", "-m", "init")

	url := "file://" + filepath.ToSlash(src)
	resolved, err := collector.Resolve(context.Background(), url)
	if err != nil {
		t.Fatalf("Resolve clone: %v", err)
	}
	defer resolved.Cleanup()

	if resolved.Display != url {
		t.Fatalf("Display = %q, want %q", resolved.Display, url)
	}
	cloned := filepath.Join(resolved.Root, "token.go")
	if _, err := os.Stat(cloned); err != nil {
		t.Fatalf("expected cloned file: %v", err)
	}

	root := resolved.Root
	resolved.Cleanup()
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("cleanup should remove temp dir; stat err = %v", err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func mustAbs(t *testing.T, p string) string {
	t.Helper()
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}
