package certs_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sgoveia/cryptarium/internal/collector"
	"github.com/sgoveia/cryptarium/internal/detector/certs"
	"github.com/sgoveia/cryptarium/internal/model"
)

func TestHandles(t *testing.T) {
	d := certs.New()
	cases := []struct {
		path string
		want bool
	}{
		{"a.pem", true},
		{"a.crt", true},
		{"a.cer", true},
		{"a.der", true},
		{"a.key", true},
		{"a.go", false},
		{"readme.txt", false},
	}
	for _, tc := range cases {
		got := d.Handles(collector.FileRef{Path: tc.path})
		if got != tc.want {
			t.Fatalf("Handles(%q)=%v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestDetect_RSA2048PEM(t *testing.T) {
	f := fixture(t, "rsa2048.pem")
	d := certs.New()
	findings, err := d.Detect(context.Background(), f)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("len=%d, want 1: %+v", len(findings), findings)
	}
	got := findings[0]
	if got.Primitive != "RSA" {
		t.Fatalf("primitive=%q", got.Primitive)
	}
	if got.AssetType != model.AssetCertificate {
		t.Fatalf("assetType=%q", got.AssetType)
	}
	if got.Parameters["keySize"] != 2048 {
		t.Fatalf("keySize=%v", got.Parameters["keySize"])
	}
	if got.Evidence.Confidence != model.ConfidenceHigh {
		t.Fatalf("confidence=%q", got.Evidence.Confidence)
	}
	assertNoKeyMaterial(t, got)
	// Deterministic ID
	again, err := d.Detect(context.Background(), f)
	if err != nil {
		t.Fatal(err)
	}
	if again[0].ID != got.ID {
		t.Fatalf("ID nondeterministic: %q vs %q", got.ID, again[0].ID)
	}
}

func TestDetect_ECDSAP256PEM(t *testing.T) {
	f := fixture(t, "ecdsa-p256.pem")
	findings, err := certs.New().Detect(context.Background(), f)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("len=%d", len(findings))
	}
	got := findings[0]
	if got.Primitive != "ECDSA" {
		t.Fatalf("primitive=%q", got.Primitive)
	}
	if got.Parameters["curve"] != "P-256" {
		t.Fatalf("curve=%v", got.Parameters["curve"])
	}
	assertNoKeyMaterial(t, got)
}

func TestDetect_RSA2048DER(t *testing.T) {
	f := fixture(t, "rsa2048.der")
	findings, err := certs.New().Detect(context.Background(), f)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(findings) != 1 || findings[0].Primitive != "RSA" {
		t.Fatalf("unexpected: %+v", findings)
	}
}

func TestDetect_PrivateKeyMetadataOnly(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "ephemeral.key")
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}

	f := collector.FileRef{Path: "ephemeral.key", AbsPath: keyPath}
	got, err := certs.New().Detect(context.Background(), f)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].AssetType != model.AssetKey || got[0].Primitive != "RSA" {
		t.Fatalf("unexpected: %+v", got[0])
	}
	assertNoKeyMaterial(t, got[0])
	// Snippet must not contain PEM armor or modulus hex.
	if strings.Contains(got[0].Evidence.Snippet, "BEGIN") || strings.Contains(got[0].Evidence.Snippet, "PRIVATE") {
		t.Fatalf("snippet leaked key framing: %q", got[0].Evidence.Snippet)
	}
}

func TestDetect_NegativeGarbagePEM(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "garbage.pem")
	body := "-----BEGIN CERTIFICATE-----\nnot-valid-base64@@@\n-----END CERTIFICATE-----\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	f := collector.FileRef{Path: "garbage.pem", AbsPath: path}
	findings, err := certs.New().Detect(context.Background(), f)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %+v", findings)
	}
}

func TestDetect_NegativeNonCertExtension(t *testing.T) {
	d := certs.New()
	if d.Handles(collector.FileRef{Path: "main.go"}) {
		t.Fatal("must not handle .go")
	}
}

func TestDetect_SyntheticECDSA(t *testing.T) {
	// Ensures we do not depend solely on checked-in fixtures.
	dir := t.TempDir()
	path := filepath.Join(dir, "synth.pem")
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := certs.New().Detect(context.Background(), collector.FileRef{Path: "synth.pem", AbsPath: path})
	if err != nil {
		t.Fatal(err)
	}
	if findings[0].Parameters["curve"] != "P-256" {
		t.Fatalf("curve=%v", findings[0].Parameters["curve"])
	}
}

func fixture(t *testing.T, name string) collector.FileRef {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", "..", "testdata", "certs", name))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("fixture %s missing (run .devcontainer/post-create.sh?): %v", abs, err)
	}
	return collector.FileRef{
		Path:    "testdata/certs/" + name,
		AbsPath: abs,
	}
}

func assertNoKeyMaterial(t *testing.T, f model.CryptoFinding) {
	t.Helper()
	raw, err := jsonLike(f)
	if err != nil {
		t.Fatal(err)
	}
	lower := strings.ToLower(raw)
	for _, needle := range []string{
		"begin rsa private",
		"begin ec private",
		"begin private",
		"modulus",
	} {
		if strings.Contains(lower, needle) {
			t.Fatalf("finding appears to contain key material (%q): %s", needle, raw)
		}
	}
}

func jsonLike(f model.CryptoFinding) (string, error) {
	var b bytes.Buffer
	b.WriteString(f.Evidence.Snippet)
	b.WriteString(f.ID)
	b.WriteString(f.Primitive)
	for k, v := range f.Parameters {
		b.WriteString(k)
		b.WriteString(strings.ToLower(stringify(v)))
	}
	return b.String(), nil
}

func stringify(v any) string {
	switch x := v.(type) {
	case string:
		return x
	default:
		return ""
	}
}
