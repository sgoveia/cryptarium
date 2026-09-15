package main

import (
	"bytes"
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

func TestRun_ScanNotImplemented(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"scan", "."}, &stdout, &stderr)
	if code != exitScanError {
		t.Fatalf("exit = %d, want %d", code, exitScanError)
	}
	if !strings.Contains(stderr.String(), "not implemented") {
		t.Fatalf("stderr = %q, want not-implemented message", stderr.String())
	}
}

func TestRun_ScanMissingTarget(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"scan"}, &stdout, &stderr)
	if code != exitScanError {
		t.Fatalf("exit = %d, want %d", code, exitScanError)
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
