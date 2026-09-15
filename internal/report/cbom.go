package report

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/sgoveia/cryptarium/internal/model"
)

// WriteCBOM emits a CycloneDX 1.6 document with cryptographic-asset components.
// Hand-marshaled against the CycloneDX 1.6 cryptographic-asset shape (DESIGN.md §10)
// because we only need that component type for Phase 2.
func WriteCBOM(w io.Writer, assets []model.CryptoAsset, meta Meta) error {
	doc := cbomDocument{
		BOMFormat:    "CycloneDX",
		SpecVersion:  "1.6",
		Version:      1,
		SerialNumber: "urn:uuid:00000000-0000-0000-0000-000000000000",
		Metadata: cbomMetadata{
			Tools: cbomTools{Components: []cbomToolComponent{{
				Type:    "application",
				Name:    "cryptarium",
				Version: meta.ToolVersion,
			}}},
			Component: &cbomComponent{
				Type: "application",
				Name: "scanned-target",
			},
		},
		Components: make([]cbomComponent, 0, len(assets)),
	}
	if !meta.Deterministic {
		doc.Metadata.Timestamp = time.Now().UTC().Format(time.RFC3339)
		doc.SerialNumber = ""
	}
	if meta.Root != "" && !meta.Deterministic {
		doc.Metadata.Component.Name = meta.Root
	}

	sorted := append([]model.CryptoAsset(nil), assets...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Evidence.Path != sorted[j].Evidence.Path {
			return sorted[i].Evidence.Path < sorted[j].Evidence.Path
		}
		if sorted[i].Evidence.Line != sorted[j].Evidence.Line {
			return sorted[i].Evidence.Line < sorted[j].Evidence.Line
		}
		return sorted[i].ID < sorted[j].ID
	})

	for _, a := range sorted {
		doc.Components = append(doc.Components, assetToComponent(a))
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(doc)
}

func assetToComponent(a model.CryptoAsset) cbomComponent {
	name := a.Primitive
	if size, ok := a.Parameters["keySize"]; ok {
		name = fmt.Sprintf("%s-%v", a.Primitive, size)
	} else if curve, ok := a.Parameters["curve"].(string); ok && curve != "" {
		name = fmt.Sprintf("%s-%s", a.Primitive, curve)
	}
	ref := "crypto/" + strings.ToLower(string(a.AssetType)) + "/" + a.ID
	if a.AssetType == "" {
		ref = "crypto/algorithm/" + a.ID
	}

	level := nistLevel(a.QuantumClass)
	props := &cbomCryptoProperties{
		AssetType: string(a.AssetType),
		AlgorithmProperties: &cbomAlgorithmProperties{
			Primitive:                primitiveKind(a),
			ParameterSetIdentifier:   paramSet(a),
			CryptoFunctions:          cryptoFuncs(a),
			NISTQuantumSecurityLevel: level,
		},
		OID: a.OID,
	}
	comp := cbomComponent{
		Type:             "cryptographic-asset",
		BOMRef:           ref,
		Name:             name,
		CryptoProperties: props,
	}
	return comp
}

func nistLevel(c model.QuantumClass) int {
	switch c {
	case model.ClassSafe:
		return 1 // conservative placeholder; specific ML-KEM/ML-DSA levels refine later
	case model.ClassBroken, model.ClassWeakened:
		return 0
	default:
		return 0
	}
}

func primitiveKind(a model.CryptoAsset) string {
	switch a.Primitive {
	case "AES", "ChaCha20", "3DES", "DES":
		return "ae"
	case "SHA-1", "SHA-256", "SHA-384", "SHA-512", "MD5":
		return "hash"
	case "RSA":
		return "pke"
	case "ECDSA", "EdDSA", "DSA", "ML-DSA", "SLH-DSA":
		return "signature"
	case "DH", "ECDH", "ML-KEM":
		return "kem"
	default:
		return "other"
	}
}

func paramSet(a model.CryptoAsset) string {
	if size, ok := a.Parameters["keySize"]; ok {
		return fmt.Sprintf("%v", size)
	}
	if curve, ok := a.Parameters["curve"].(string); ok {
		return curve
	}
	return ""
}

func cryptoFuncs(a model.CryptoAsset) []string {
	out := make([]string, 0, len(a.Functions))
	for _, f := range a.Functions {
		out = append(out, string(f))
	}
	return out
}

type cbomDocument struct {
	BOMFormat    string          `json:"bomFormat"`
	SpecVersion  string          `json:"specVersion"`
	SerialNumber string          `json:"serialNumber,omitempty"`
	Version      int             `json:"version"`
	Metadata     cbomMetadata    `json:"metadata"`
	Components   []cbomComponent `json:"components"`
}

type cbomMetadata struct {
	Timestamp string         `json:"timestamp,omitempty"`
	Tools     cbomTools      `json:"tools"`
	Component *cbomComponent `json:"component,omitempty"`
}

type cbomTools struct {
	Components []cbomToolComponent `json:"components"`
}

type cbomToolComponent struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type cbomComponent struct {
	Type             string                `json:"type"`
	BOMRef           string                `json:"bom-ref,omitempty"`
	Name             string                `json:"name"`
	CryptoProperties *cbomCryptoProperties `json:"cryptoProperties,omitempty"`
}

type cbomCryptoProperties struct {
	AssetType           string                   `json:"assetType"`
	AlgorithmProperties *cbomAlgorithmProperties `json:"algorithmProperties,omitempty"`
	OID                 string                   `json:"oid,omitempty"`
}

type cbomAlgorithmProperties struct {
	Primitive                string   `json:"primitive,omitempty"`
	ParameterSetIdentifier   string   `json:"parameterSetIdentifier,omitempty"`
	CryptoFunctions          []string `json:"cryptoFunctions,omitempty"`
	NISTQuantumSecurityLevel int      `json:"nistQuantumSecurityLevel"`
}
