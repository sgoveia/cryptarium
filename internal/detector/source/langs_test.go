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

func TestDetect_CLibsodiumPackageTag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.c")
	code := `#include <sodium.h>
int f(unsigned char *pk, unsigned char *sk) {
  return crypto_sign_keypair(pk, sk);
}
`
	if err := os.WriteFile(path, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	d := source.NewWithRulesDir(filepath.Join("..", "..", "..", "rules"))
	findings, err := d.Detect(context.Background(), collector.FileRef{Path: "s.c", AbsPath: path})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range findings {
		if f.Evidence.RuleID == "c.libsodium.crypto_sign_keypair" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected c.libsodium.crypto_sign_keypair finding, got %+v", findings)
	}
}

func TestDetect_CMbedtlsPackageTag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "m.c")
	code := `#include <mbedtls/rsa.h>
int f(mbedtls_rsa_context *ctx, int (*f_rng)(void *, unsigned char *, size_t), void *p_rng) {
  return mbedtls_rsa_gen_key(ctx, f_rng, p_rng, 2048, 65537);
}
`
	if err := os.WriteFile(path, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	d := source.NewWithRulesDir(filepath.Join("..", "..", "..", "rules"))
	findings, err := d.Detect(context.Background(), collector.FileRef{Path: "m.c", AbsPath: path})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range findings {
		if f.Evidence.RuleID == "c.mbedtls.rsa_gen_key" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected c.mbedtls.rsa_gen_key finding, got %+v", findings)
	}
}
