package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_Version(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"version"}, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit = %d, want %d; stderr=%q", code, exitOK, stderr.String())
	}
	got := strings.TrimSpace(stdout.String())
	want := "cryptarium 0.0.0-dev"
	if got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestRun_ScanFixture(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "fixture-repo")
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("fixture-repo missing: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"scan", "--format", "json", "--deterministic", root}, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit = %d; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"findings"`) {
		t.Fatalf("expected JSON findings, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "RSA") && !strings.Contains(stdout.String(), "ECDSA") {
		t.Fatalf("expected cert primitive in output: %s", stdout.String())
	}
}

func TestRun_ScanMissingTarget(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"scan"}, &stdout, &stderr)
	if code != exitScanError {
		t.Fatalf("exit = %d, want %d", code, exitScanError)
	}
}

func TestRun_ScanSSHURLRejected(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"scan", "git@github.com:example/repo.git"}, &stdout, &stderr)
	if code != exitScanError {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stderr.String(), "SSH") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestRun_ScanRemoteFileURL(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	src := t.TempDir()
	pem := filepath.Join(src, "rsa.pem")
	// Minimal PEM-looking content is enough for walk; cert detector may warn.
	if err := os.WriteFile(pem, []byte("not-a-cert\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGitCLI(t, src, "init")
	runGitCLI(t, src, "config", "user.email", "test@example.com")
	runGitCLI(t, src, "config", "user.name", "test")
	runGitCLI(t, src, "add", ".")
	runGitCLI(t, src, "commit", "-m", "init")

	url := "file://" + filepath.ToSlash(src)
	var stdout, stderr bytes.Buffer
	code := run([]string{"scan", "--format", "json", "--deterministic", url}, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit = %d; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"findings"`) {
		t.Fatalf("expected JSON findings, got %q", stdout.String())
	}
}

func runGitCLI(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"explode"}, &stdout, &stderr)
	if code != exitScanError {
		t.Fatalf("exit = %d, want %d", code, exitScanError)
	}
}

func TestScanFlags_Defaults(t *testing.T) {
	fs := newScanFlags()
	if err := fs.parse([]string{"."}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if fs.target != "." {
		t.Fatalf("target = %q", fs.target)
	}
	if len(fs.format) != 1 || fs.format[0] != "markdown" {
		t.Fatalf("format = %#v, want [markdown]", fs.format)
	}
	if fs.failOn != "none" {
		t.Fatalf("failOn = %q", fs.failOn)
	}
}

func TestScanFlags_RepeatableFormat(t *testing.T) {
	fs := newScanFlags()
	out := filepath.Join("out")
	if err := fs.parse([]string{"--format", "json", "--format", "cbom", "--output", out, "/tmp/repo"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := strings.Join(fs.format, ","); got != "json,cbom" {
		t.Fatalf("format = %q", got)
	}
	if fs.output != out {
		t.Fatalf("output = %q", fs.output)
	}
}
