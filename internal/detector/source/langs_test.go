package source_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sgoveia/cryptarium/internal/collector"
	"github.com/sgoveia/cryptarium/internal/detector/source"
)

func TestDetect_PythonHashlibMD5(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "h.py")
	code := "import hashlib\n\ndef f():\n    hashlib.md5(b'data')\n"
	if err := os.WriteFile(path, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	d := source.NewWithRulesDir(filepath.Join("..", "..", "..", "rules"))
	findings, err := d.Detect(context.Background(), collector.FileRef{Path: "h.py", AbsPath: path})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) < 1 {
		t.Fatalf("expected hashlib.md5 finding, got %+v", findings)
	}
}

func TestDetect_COpenSSL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "k.c")
	code := `#include <openssl/rsa.h>
int f(RSA *r, BIGNUM *e, BN_GENCB *cb) {
  return RSA_generate_key_ex(r, 2048, e, cb);
}
`
	if err := os.WriteFile(path, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	d := source.NewWithRulesDir(filepath.Join("..", "..", "..", "rules"))
	findings, err := d.Detect(context.Background(), collector.FileRef{Path: "k.c", AbsPath: path})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) < 1 {
		t.Fatalf("expected RSA_generate_key_ex finding, got %+v", findings)
	}
}
