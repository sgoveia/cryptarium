package classify

import "strings"

// Canonical returns the canonical primitive name used in the classification table.
// Rule packs must emit these names (DESIGN.md §5).
func Canonical(name string) string {
	n := strings.TrimSpace(name)
	if n == "" {
		return ""
	}
	if c, ok := aliases[strings.ToUpper(n)]; ok {
		return c
	}
	// Preserve mixed forms like SHA-256.
	if c, ok := aliases[n]; ok {
		return c
	}
	return n
}

var aliases = map[string]string{
	"RSA":      "RSA",
	"ECDSA":    "ECDSA",
	"EDDSA":    "EdDSA",
	"ED25519":  "EdDSA",
	"DSA":      "DSA",
	"DH":       "DH",
	"ECDH":     "ECDH",
	"X25519":   "ECDH",
	"AES":      "AES",
	"CHACHA20": "ChaCha20",
	"SHA-1":    "SHA-1",
	"SHA1":     "SHA-1",
	"SHA-256":  "SHA-256",
	"SHA256":   "SHA-256",
	"SHA-384":  "SHA-384",
	"SHA384":   "SHA-384",
	"SHA-512":  "SHA-512",
	"SHA512":   "SHA-512",
	"MD5":      "MD5",
	"3DES":     "3DES",
	"DES":      "DES",
	"ML-KEM":   "ML-KEM",
	"ML-DSA":   "ML-DSA",
	"SLH-DSA":  "SLH-DSA",
	"UNKNOWN":  "UNKNOWN",
}
