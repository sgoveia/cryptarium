// Package classify maps CryptoFindings to quantum-vulnerability classes and
// migration recommendations. Every table entry cites FIPS, NIST SP, CNSA 2.0, or an RFC.
package classify

import (
	"fmt"
	"strings"

	"github.com/sgoveia/cryptarium/internal/model"
)

// Classify turns a raw finding into a classified asset. Unrecognized primitives
// become ClassUnknown — never guessed (DESIGN.md §6, AGENT.md hard rules).
func Classify(f model.CryptoFinding) model.CryptoAsset {
	asset := model.CryptoAsset{CryptoFinding: f}
	name := Canonical(f.Primitive)
	if name == "" {
		name = strings.TrimSpace(f.Primitive)
	}

	if c, ok := classifySymmetric(name, f.Parameters); ok {
		asset.QuantumClass = c.class
		asset.Rationale = c.rationale
		asset.Recommendation = c.migration
		asset.OID = c.oid
		return asset
	}

	if c, ok := quantumTable[name]; ok {
		asset.QuantumClass = c.class
		asset.Rationale = c.rationale
		asset.Recommendation = c.migration
		asset.OID = c.oid
		return asset
	}

	asset.QuantumClass = model.ClassUnknown
	asset.Rationale = fmt.Sprintf("primitive %q is not in the classification table; reported as unknown", f.Primitive)
	return asset
}

// All classifies each finding in order.
func All(findings []model.CryptoFinding) []model.CryptoAsset {
	out := make([]model.CryptoAsset, len(findings))
	for i, f := range findings {
		out[i] = Classify(f)
	}
	return out
}

type classification struct {
	class     model.QuantumClass
	rationale string
	migration model.Migration
	oid       string
}
