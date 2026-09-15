// Package model holds the pipeline's shared type contracts.
// It imports nothing from the rest of the project; every other package
// imports model. Changing these types is a breaking change.
package model

// Evidence records where a finding came from. Every finding has at least one.
type Evidence struct {
	Source     SourceKind `json:"source"`         // source-code | dependency | certificate | configuration
	Path       string     `json:"path"`           // repo-relative, always forward slashes
	Line       int        `json:"line,omitempty"` // 1-based; 0 when not line-addressable
	Column     int        `json:"column,omitempty"`
	Snippet    string     `json:"snippet,omitempty"` // redacted; never contains key material
	RuleID     string     `json:"ruleId,omitempty"`  // the rule that fired, for auditability
	Confidence Confidence `json:"confidence"`        // high | medium | low
}

// SourceKind identifies which detector family produced the evidence.
type SourceKind string

const (
	SourceCode          SourceKind = "source-code"
	SourceDependency    SourceKind = "dependency"
	SourceCertificate   SourceKind = "certificate"
	SourceConfiguration SourceKind = "configuration"
)

// Confidence is how strongly the match establishes the primitive and parameters.
type Confidence string

const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
)

// AssetType is the CycloneDX-aligned category of a cryptographic asset.
type AssetType string

const (
	AssetAlgorithm   AssetType = "algorithm"
	AssetCertificate AssetType = "certificate"
	AssetKey         AssetType = "key"
	AssetProtocol    AssetType = "protocol"
	AssetLibrary     AssetType = "library"
)

// CryptoFunction is a cryptographic operation the asset participates in.
type CryptoFunction string

const (
	FunctionKeygen      CryptoFunction = "keygen"
	FunctionEncrypt     CryptoFunction = "encrypt"
	FunctionDecrypt     CryptoFunction = "decrypt"
	FunctionSign        CryptoFunction = "sign"
	FunctionVerify      CryptoFunction = "verify"
	FunctionKeyExchange CryptoFunction = "keyexchange"
	FunctionDigest      CryptoFunction = "digest"
)

// CryptoFinding is what a detector emits. Raw, unclassified, unscored.
type CryptoFinding struct {
	ID         string           `json:"id"`         // deterministic; see FindingID
	AssetType  AssetType        `json:"assetType"`  // algorithm | certificate | key | protocol | library
	Primitive  string           `json:"primitive"`  // canonical name: "RSA", "ECDSA", "AES", "SHA-256"
	Parameters map[string]any   `json:"parameters"` // keySize, curve, mode, padding...
	Functions  []CryptoFunction `json:"functions"`  // keygen | encrypt | decrypt | sign | verify | keyexchange | digest
	Evidence   Evidence         `json:"evidence"`
}

// CryptoAsset is a classified, deduplicated finding, post-correlation.
type CryptoAsset struct {
	CryptoFinding
	QuantumClass   QuantumClass `json:"quantumClass"` // broken | weakened | safe | unknown
	Rationale      string       `json:"rationale"`    // why: "factoring; broken by Shor's algorithm"
	Recommendation Migration    `json:"recommendation"`
	RelatedIDs     []string     `json:"relatedIds"`    // correlated findings
	OID            string       `json:"oid,omitempty"` // for CBOM emission
}

// QuantumClass is the quantum-vulnerability classification of an asset.
type QuantumClass string

const (
	ClassBroken   QuantumClass = "broken"   // Shor: RSA, DH, ECDH, ECDSA, EdDSA, DSA
	ClassWeakened QuantumClass = "weakened" // Grover: symmetric ciphers and hashes below target margin
	ClassSafe     QuantumClass = "safe"     // standardized PQC, or symmetric/hash at adequate size
	ClassUnknown  QuantumClass = "unknown"  // detected but unresolvable — report, never guess
)

// Migration recommends a post-quantum (or otherwise stronger) target.
type Migration struct {
	Target   string `json:"target"`           // "ML-KEM-768"
	Standard string `json:"standard"`         // "FIPS 203"
	Hybrid   string `json:"hybrid,omitempty"` // "X25519+ML-KEM-768"
	Notes    string `json:"notes,omitempty"`
}

// ScoredAsset adds prioritization.
type ScoredAsset struct {
	CryptoAsset
	Risk RiskScore `json:"risk"`
}

// RiskScore is the four-axis prioritization result.
type RiskScore struct {
	Score         int      `json:"score"`         // 0-100
	Priority      Priority `json:"priority"`      // critical | high | medium | low
	Vulnerability int      `json:"vulnerability"` // axis subscores, 0-100 each
	Longevity     int      `json:"longevity"`
	Exposure      int      `json:"exposure"`
	Agility       int      `json:"agility"`
	Explanation   string   `json:"explanation"` // human-readable derivation
}

// Priority is the band derived from RiskScore.Score.
type Priority string

const (
	PriorityCritical Priority = "critical" // >= 80
	PriorityHigh     Priority = "high"     // >= 60
	PriorityMedium   Priority = "medium"   // >= 35
	PriorityLow      Priority = "low"      // below 35
)
