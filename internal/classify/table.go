package classify

import "github.com/sgoveia/cryptarium/internal/model"

// quantumTable is the core knowledge claim for asymmetric and already-broken
// primitives. Citations are required for every entry (AGENT.md).
var quantumTable = map[string]classification{
	// Shor's algorithm breaks integer factorization and discrete log problems.
	// See NIST IR 8413; FIPS 203/204/205 standardize replacements.
	"RSA": {
		class:     model.ClassBroken,
		rationale: "integer factorization; broken by Shor's algorithm",
		// FIPS 203 (ML-KEM) for key establishment; FIPS 204 (ML-DSA) for signatures.
		migration: model.Migration{
			Target:   "ML-KEM-768 (key establishment) / ML-DSA-65 (signatures)",
			Standard: "FIPS 203 / FIPS 204",
			Hybrid:   "X25519+ML-KEM-768",
		},
		oid: "1.2.840.113549.1.1.1",
	},
	"DSA": {
		class:     model.ClassBroken,
		rationale: "finite-field discrete log; broken by Shor's algorithm",
		// FIPS 204 — Module-Lattice-Based Digital Signature Algorithm.
		migration: model.Migration{Target: "ML-DSA-65", Standard: "FIPS 204"},
	},
	"DH": {
		class:     model.ClassBroken,
		rationale: "finite-field discrete log; broken by Shor's algorithm",
		// FIPS 203 — ML-KEM.
		migration: model.Migration{
			Target:   "ML-KEM-768",
			Standard: "FIPS 203",
			Hybrid:   "X25519+ML-KEM-768",
		},
	},
	"ECDH": {
		class:     model.ClassBroken,
		rationale: "elliptic-curve discrete log; broken by Shor's algorithm",
		// FIPS 203; CNSA 2.0 recommends ML-KEM for key establishment.
		migration: model.Migration{
			Target:   "ML-KEM-768",
			Standard: "FIPS 203",
			Hybrid:   "X25519+ML-KEM-768",
		},
	},
	"ECDSA": {
		class:     model.ClassBroken,
		rationale: "elliptic-curve discrete log; broken by Shor's algorithm",
		// FIPS 204; FIPS 205 (SLH-DSA) where a conservative hash-based option is preferred.
		migration: model.Migration{
			Target:   "ML-DSA-65",
			Standard: "FIPS 204",
			Notes:    "SLH-DSA (FIPS 205) where a conservative hash-based option is preferred",
		},
	},
	"EdDSA": {
		class:     model.ClassBroken,
		rationale: "elliptic-curve discrete log; broken by Shor's algorithm",
		// FIPS 204.
		migration: model.Migration{Target: "ML-DSA-65", Standard: "FIPS 204"},
	},

	// Classical breaks — report as broken regardless of quantum considerations.
	// NIST SP 800-131A Rev. 2 disallows MD5 and SHA-1 for digital signatures.
	"MD5": {
		class:     model.ClassBroken,
		rationale: "classically broken collision resistance (NIST SP 800-131A)",
		migration: model.Migration{Target: "SHA-384 / SHA-512", Standard: "FIPS 180-4 / FIPS 202"},
	},
	"SHA-1": {
		class:     model.ClassBroken,
		rationale: "classically broken collision resistance (NIST SP 800-131A)",
		migration: model.Migration{Target: "SHA-384 / SHA-512", Standard: "FIPS 180-4"},
	},
	"3DES": {
		class:     model.ClassBroken,
		rationale: "classically deprecated; inadequate margin (NIST SP 800-131A)",
		migration: model.Migration{Target: "AES-256", Standard: "FIPS 197"},
	},
	"DES": {
		class:     model.ClassBroken,
		rationale: "classically broken key size",
		migration: model.Migration{Target: "AES-256", Standard: "FIPS 197"},
	},

	// Standardized PQC — FIPS 203/204/205.
	"ML-KEM": {
		class:     model.ClassSafe,
		rationale: "NIST standardized module-lattice KEM (FIPS 203)",
		migration: model.Migration{Target: "ML-KEM (already PQC)", Standard: "FIPS 203"},
	},
	"ML-DSA": {
		class:     model.ClassSafe,
		rationale: "NIST standardized module-lattice signature (FIPS 204)",
		migration: model.Migration{Target: "ML-DSA (already PQC)", Standard: "FIPS 204"},
	},
	"SLH-DSA": {
		class:     model.ClassSafe,
		rationale: "NIST standardized hash-based signature (FIPS 205)",
		migration: model.Migration{Target: "SLH-DSA (already PQC)", Standard: "FIPS 205"},
	},
}
