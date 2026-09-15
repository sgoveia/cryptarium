package main

import (
	"bytes"
	"os"
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

func TestRun_ScanGitURLRejected(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"scan", "https://github.com/example/repo"}, &stdout, &stderr)
	if code != exitScanError {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stderr.String(), "not supported") {
		t.Fatalf("stderr=%q", stderr.String())
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
