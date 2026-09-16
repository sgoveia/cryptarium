package config_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sgoveia/cryptarium/internal/collector"
	"github.com/sgoveia/cryptarium/internal/detector/config"
)

func TestHandles(t *testing.T) {
	d := config.New()
	cases := []struct {
		path string
		want bool
	}{
		{"nginx.conf", true},
		{"deploy/nginx.conf", true},
		{"sshd_config", true},
		{"jwt-auth.json", true},
		{"README.md", false},
		{"main.go", false},
	}
	for _, tc := range cases {
		got := d.Handles(collector.FileRef{Path: tc.path})
		if got != tc.want {
			t.Fatalf("Handles(%q)=%v want %v", tc.path, got, tc.want)
		}
	}
}

func TestDetect_NginxCipherSuites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nginx.conf")
	content := `
# ssl_ciphers TLS_RSA_WITH_AES_128_CBC_SHA; comment only — must not match
ssl_ciphers TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256:TLS_AES_256_GCM_SHA384;
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	f := collector.FileRef{Path: "nginx.conf", AbsPath: path}
	got, err := config.New().Detect(context.Background(), f)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < 3 {
		t.Fatalf("expected multiple primitives from cipher suites, got %d: %+v", len(got), got)
	}
	prims := map[string]bool{}
	for _, g := range got {
		prims[g.Primitive] = true
		if g.Evidence.Source != "configuration" {
			t.Fatalf("source=%s", g.Evidence.Source)
		}
	}
	for _, want := range []string{"ECDH", "RSA", "AES"} {
		if !prims[want] {
			t.Fatalf("missing primitive %s in %+v", want, prims)
		}
	}
}

func TestDetect_NegativeCommentOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nginx.conf")
	content := "# ssl_ciphers TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256;\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := config.New().Detect(context.Background(), collector.FileRef{Path: "nginx.conf", AbsPath: path})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("comment-only should not match, got %+v", got)
	}
}

func TestDetect_SSHD(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sshd_config")
	content := "KexAlgorithms curve25519-sha256,diffie-hellman-group14-sha1\nHostKeyAlgorithms rsa-sha2-512\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := config.New().Detect(context.Background(), collector.FileRef{Path: "sshd_config", AbsPath: path})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < 2 {
		t.Fatalf("expected ssh findings, got %d", len(got))
	}
}

func TestDetect_JWTJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "jwt-auth.json")
	content := `{"alg":"RS256","typ":"JWT"}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := config.New().Detect(context.Background(), collector.FileRef{Path: "jwt-auth.json", AbsPath: path})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Primitive != "RSA" {
		t.Fatalf("got %+v", got)
	}
}
