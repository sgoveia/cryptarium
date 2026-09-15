package classify

import (
	"fmt"

	"github.com/sgoveia/cryptarium/internal/model"
)

// classifySymmetric handles AES and SHA-* where class depends on parameters.
// Grover's algorithm halves effective symmetric security (NIST IR 8413).
func classifySymmetric(name string, params map[string]any) (classification, bool) {
	switch name {
	case "AES":
		size := keySize(params)
		switch {
		case size == 128:
			// Grover reduces AES-128 to ~64-bit effective security (NIST IR 8413).
			return classification{
				class:     model.ClassWeakened,
				rationale: "AES-128 effective margin reduced by Grover's algorithm",
				migration: model.Migration{Target: "AES-256", Standard: "FIPS 197"},
			}, true
		case size >= 256:
			// AES-256 retains adequate margin under Grover (NIST IR 8413 / CNSA 2.0).
			return classification{
				class:     model.ClassSafe,
				rationale: "AES-256 retains adequate margin under Grover's algorithm",
				migration: model.Migration{Target: "AES-256 (already adequate)", Standard: "FIPS 197"},
			}, true
		case size == 192:
			return classification{
				class:     model.ClassWeakened,
				rationale: "AES-192; prefer AES-256 for quantum-resistant margin (CNSA 2.0)",
				migration: model.Migration{Target: "AES-256", Standard: "FIPS 197 / CNSA 2.0"},
			}, true
		default:
			return classification{
				class:     model.ClassUnknown,
				rationale: fmt.Sprintf("AES with unrecognized keySize %v", params["keySize"]),
			}, true
		}
	case "ChaCha20":
		// 256-bit key; treated like AES-256 margin under Grover (common guidance; RFC 8439).
		return classification{
			class:     model.ClassSafe,
			rationale: "ChaCha20 uses a 256-bit key; adequate margin under Grover (RFC 8439)",
			migration: model.Migration{Target: "ChaCha20-Poly1305 (already adequate)", Standard: "RFC 8439"},
		}, true
	case "SHA-256":
		// High-assurance contexts may prefer SHA-384/512 (CNSA 2.0).
		return classification{
			class:     model.ClassWeakened,
			rationale: "SHA-256; prefer SHA-384/SHA-512 in high-assurance contexts (CNSA 2.0)",
			migration: model.Migration{Target: "SHA-384 / SHA-512", Standard: "FIPS 180-4 / CNSA 2.0"},
		}, true
	case "SHA-384", "SHA-512":
		return classification{
			class:     model.ClassSafe,
			rationale: "SHA-384/SHA-512 meet CNSA 2.0 hash guidance",
			migration: model.Migration{Target: name + " (already adequate)", Standard: "FIPS 180-4 / CNSA 2.0"},
		}, true
	default:
		return classification{}, false
	}
}

func keySize(params map[string]any) int {
	if params == nil {
		return 0
	}
	switch v := params["keySize"].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}
