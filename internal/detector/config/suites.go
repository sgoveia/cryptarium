package config

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/sgoveia/cryptarium/internal/classify"
	"github.com/sgoveia/cryptarium/internal/model"
)

// suiteToken describes one cryptographic primitive extracted from a cipher-suite
// or algorithm-list token. One suite string may yield several tokens (DESIGN.md §4).
type suiteToken struct {
	Primitive  string
	Parameters map[string]any
	Functions  []model.CryptoFunction
}

var (
	aesSizeRe = regexp.MustCompile(`(?i)AES[_-]?(\d{3})`)
	shaSizeRe = regexp.MustCompile(`(?i)SHA[_-]?(\d{1,3})`)
)

// parseTLSCipherSuite splits an IANA TLS cipher suite name into primitives.
// Example: TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 → ECDH, RSA, AES-128, SHA-256.
func parseTLSCipherSuite(suite string) []suiteToken {
	s := strings.TrimSpace(suite)
	if s == "" {
		return nil
	}
	upper := strings.ToUpper(s)
	if !strings.HasPrefix(upper, "TLS_") && !strings.HasPrefix(upper, "SSL_") {
		return nil
	}

	var out []suiteToken
	seen := map[string]struct{}{}

	add := func(prim string, params map[string]any, fns ...model.CryptoFunction) {
		prim = classify.Canonical(prim)
		if prim == "" {
			return
		}
		key := prim
		if params != nil {
			if ks, ok := params["keySize"]; ok {
				key += ":" + stringify(ks)
			}
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		if params == nil {
			params = map[string]any{}
		}
		params["cipherSuite"] = s
		out = append(out, suiteToken{Primitive: prim, Parameters: params, Functions: fns})
	}

	// Key exchange / auth from the suite name body.
	body := upper
	switch {
	case strings.Contains(body, "_ECDHE_"):
		add("ECDH", nil, model.FunctionKeyExchange)
	case strings.Contains(body, "_ECDH_"):
		add("ECDH", nil, model.FunctionKeyExchange)
	case strings.Contains(body, "_DHE_"):
		add("DH", nil, model.FunctionKeyExchange)
	case strings.Contains(body, "_DH_"):
		add("DH", nil, model.FunctionKeyExchange)
	}

	// Authentication / certificate type in classical suites.
	switch {
	case strings.Contains(body, "_RSA_") || strings.HasSuffix(body, "_RSA") || strings.Contains(body, "RSA_WITH"):
		add("RSA", nil, model.FunctionSign, model.FunctionKeyExchange)
	case strings.Contains(body, "_ECDSA_"):
		add("ECDSA", nil, model.FunctionSign)
	case strings.Contains(body, "_DSS_") || strings.Contains(body, "_DSA_"):
		add("DSA", nil, model.FunctionSign)
	}

	if m := aesSizeRe.FindStringSubmatch(upper); len(m) == 2 {
		size, _ := strconv.Atoi(m[1])
		add("AES", map[string]any{"keySize": size, "mode": suiteMode(upper)}, model.FunctionEncrypt, model.FunctionDecrypt)
	}
	if strings.Contains(upper, "CHACHA20") {
		add("ChaCha20", map[string]any{"keySize": 256}, model.FunctionEncrypt, model.FunctionDecrypt)
	}
	if strings.Contains(upper, "3DES") || strings.Contains(upper, "DES_EDE") {
		add("3DES", nil, model.FunctionEncrypt, model.FunctionDecrypt)
	}

	if m := shaSizeRe.FindStringSubmatch(upper); len(m) == 2 {
		bits := m[1]
		name := "SHA-" + bits
		if bits == "1" {
			name = "SHA-1"
		}
		add(name, nil, model.FunctionDigest)
	}
	if strings.Contains(upper, "MD5") {
		add("MD5", nil, model.FunctionDigest)
	}

	return out
}

func suiteMode(upper string) string {
	switch {
	case strings.Contains(upper, "_GCM_"):
		return "GCM"
	case strings.Contains(upper, "_CCM_"):
		return "CCM"
	case strings.Contains(upper, "_CBC_"):
		return "CBC"
	default:
		return ""
	}
}

// parseSSHAlgToken maps OpenSSH algorithm identifiers to primitives.
func parseSSHAlgToken(token string) []suiteToken {
	t := strings.TrimSpace(token)
	if t == "" || strings.HasPrefix(t, "#") {
		return nil
	}
	lower := strings.ToLower(t)
	var out []suiteToken

	add := func(prim string, params map[string]any, fns ...model.CryptoFunction) {
		prim = classify.Canonical(prim)
		if prim == "" {
			return
		}
		if params == nil {
			params = map[string]any{}
		}
		params["sshAlgorithm"] = t
		out = append(out, suiteToken{Primitive: prim, Parameters: params, Functions: fns})
	}

	switch {
	case strings.Contains(lower, "curve25519") || strings.Contains(lower, "x25519"):
		add("ECDH", map[string]any{"curve": "X25519"}, model.FunctionKeyExchange)
	case strings.HasPrefix(lower, "ecdh-sha2-nistp256"):
		add("ECDH", map[string]any{"curve": "P-256"}, model.FunctionKeyExchange)
	case strings.HasPrefix(lower, "ecdh-sha2-nistp384"):
		add("ECDH", map[string]any{"curve": "P-384"}, model.FunctionKeyExchange)
	case strings.HasPrefix(lower, "ecdh-sha2-nistp521"):
		add("ECDH", map[string]any{"curve": "P-521"}, model.FunctionKeyExchange)
	case strings.HasPrefix(lower, "diffie-hellman"):
		add("DH", nil, model.FunctionKeyExchange)
	case strings.HasPrefix(lower, "ssh-rsa") || strings.HasPrefix(lower, "rsa-sha"):
		add("RSA", nil, model.FunctionSign)
	case strings.HasPrefix(lower, "ecdsa-sha2"):
		add("ECDSA", nil, model.FunctionSign)
	case strings.HasPrefix(lower, "ssh-ed25519") || strings.HasPrefix(lower, "ed25519"):
		add("EdDSA", map[string]any{"curve": "Ed25519"}, model.FunctionSign)
	case strings.HasPrefix(lower, "ssh-dss"):
		add("DSA", nil, model.FunctionSign)
	case strings.HasPrefix(lower, "aes"):
		if m := aesSizeRe.FindStringSubmatch(lower); len(m) == 2 {
			size, _ := strconv.Atoi(m[1])
			add("AES", map[string]any{"keySize": size}, model.FunctionEncrypt, model.FunctionDecrypt)
		}
	case strings.HasPrefix(lower, "chacha20"):
		add("ChaCha20", map[string]any{"keySize": 256}, model.FunctionEncrypt, model.FunctionDecrypt)
	case strings.Contains(lower, "sha1") || strings.HasSuffix(lower, "-sha1"):
		add("SHA-1", nil, model.FunctionDigest)
	case strings.Contains(lower, "sha256") || strings.HasSuffix(lower, "-sha256"):
		add("SHA-256", nil, model.FunctionDigest)
	case strings.Contains(lower, "sha512") || strings.HasSuffix(lower, "-sha512"):
		add("SHA-512", nil, model.FunctionDigest)
	case strings.Contains(lower, "md5"):
		add("MD5", nil, model.FunctionDigest)
	default:
		// Unrecognized SSH algorithm — emit unknown so inventories are not silent.
		add("UNKNOWN", map[string]any{"raw": t})
	}
	return out
}

// parseJWTAlg maps a JOSE "alg" value to primitives (RFC 7518).
func parseJWTAlg(alg string) []suiteToken {
	a := strings.TrimSpace(alg)
	if a == "" {
		return nil
	}
	upper := strings.ToUpper(a)
	params := map[string]any{"jwtAlg": a}

	switch upper {
	case "RS256", "RS384", "RS512", "PS256", "PS384", "PS512":
		return []suiteToken{{Primitive: "RSA", Parameters: params, Functions: []model.CryptoFunction{model.FunctionSign, model.FunctionVerify}}}
	case "ES256", "ES384", "ES512":
		return []suiteToken{{Primitive: "ECDSA", Parameters: params, Functions: []model.CryptoFunction{model.FunctionSign, model.FunctionVerify}}}
	case "EDDSA":
		return []suiteToken{{Primitive: "EdDSA", Parameters: params, Functions: []model.CryptoFunction{model.FunctionSign, model.FunctionVerify}}}
	case "HS256", "HS384", "HS512":
		// HMAC itself is not a classified public-key primitive; surface the hash
		// component only (DESIGN.md: never invent a claim we cannot cite).
		hash := map[string]string{"HS256": "SHA-256", "HS384": "SHA-384", "HS512": "SHA-512"}[upper]
		return []suiteToken{{Primitive: hash, Parameters: params, Functions: []model.CryptoFunction{model.FunctionDigest}}}
	case "NONE":
		return []suiteToken{{Primitive: "UNKNOWN", Parameters: params}}
	default:
		return []suiteToken{{Primitive: "UNKNOWN", Parameters: params}}
	}
}

func stringify(v any) string {
	switch x := v.(type) {
	case int:
		return strconv.Itoa(x)
	case string:
		return x
	default:
		return ""
	}
}
