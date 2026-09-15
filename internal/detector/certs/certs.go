package certs

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"

	// crypto/dsa is deprecated for key generation (FIPS 186-5). We still parse
	// DSA public keys so inventories do not silently omit legacy material.
	"crypto/dsa" //nolint:staticcheck // SA1019: detect legacy DSA; never generate.

	"github.com/sgoveia/cryptarium/internal/collector"
	"github.com/sgoveia/cryptarium/internal/detector"
	"github.com/sgoveia/cryptarium/internal/model"
)

func init() {
	detector.Register(New())
}

// Detector extracts cryptographic metadata from certificates and keys.
// Private key bytes are never included in findings, snippets, or errors.
type Detector struct{}

// New returns a certificate/key detector.
func New() *Detector {
	return &Detector{}
}

// Name returns the stable detector name.
func (*Detector) Name() string { return "certs" }

var handledExt = map[string]struct{}{
	".pem": {},
	".crt": {},
	".cer": {},
	".der": {},
	".key": {},
}

// Handles reports whether f looks like a certificate or key by extension.
func (*Detector) Handles(f collector.FileRef) bool {
	_, ok := handledExt[collector.Ext(f)]
	return ok
}

// Detect parses certificates and keys in f and emits metadata-only findings.
func (d *Detector) Detect(ctx context.Context, f collector.FileRef) ([]model.CryptoFinding, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	data, err := os.ReadFile(f.AbsPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", f.Path, err)
	}

	if looksLikePEM(data) {
		return d.parsePEM(f, data)
	}
	return d.parseDER(f, data)
}

func looksLikePEM(data []byte) bool {
	return strings.Contains(string(data), "-----BEGIN ")
}

func (d *Detector) parsePEM(f collector.FileRef, data []byte) ([]model.CryptoFinding, error) {
	var findings []model.CryptoFinding
	rest := data
	var parseErrors []string

	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		// Drop any headers that might carry bag attributes; never log block.Bytes.
		switch block.Type {
		case "CERTIFICATE":
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				parseErrors = append(parseErrors, fmt.Sprintf("certificate: %v", err))
				continue
			}
			findings = append(findings, d.findingFromCert(f, cert))
		case "PRIVATE KEY", "RSA PRIVATE KEY", "EC PRIVATE KEY":
			finding, err := d.findingFromPrivateKey(f, block.Type, block.Bytes)
			if err != nil {
				parseErrors = append(parseErrors, fmt.Sprintf("private key: %v", err))
				continue
			}
			findings = append(findings, finding)
		case "PUBLIC KEY", "RSA PUBLIC KEY":
			finding, err := d.findingFromPublicKey(f, block.Bytes)
			if err != nil {
				parseErrors = append(parseErrors, fmt.Sprintf("public key: %v", err))
				continue
			}
			findings = append(findings, finding)
		default:
			// Unrecognized PEM type — report as unknown rather than silent skip.
			findings = append(findings, d.unknownPEM(f, block.Type))
		}
	}

	if len(findings) == 0 && len(parseErrors) > 0 {
		return nil, fmt.Errorf("parse %s: %s", f.Path, strings.Join(parseErrors, "; "))
	}
	if len(findings) == 0 {
		return nil, fmt.Errorf("parse %s: no PEM blocks found", f.Path)
	}
	if len(parseErrors) > 0 {
		return findings, fmt.Errorf("parse %s: %s", f.Path, strings.Join(parseErrors, "; "))
	}
	return findings, nil
}

func (d *Detector) parseDER(f collector.FileRef, data []byte) ([]model.CryptoFinding, error) {
	cert, err := x509.ParseCertificate(data)
	if err != nil {
		return nil, fmt.Errorf("parse DER certificate %s: %w", f.Path, err)
	}
	return []model.CryptoFinding{d.findingFromCert(f, cert)}, nil
}

func (d *Detector) findingFromCert(f collector.FileRef, cert *x509.Certificate) model.CryptoFinding {
	primitive, params := publicKeyInfo(cert.PublicKey)
	if params == nil {
		params = map[string]any{}
	}
	params["signatureAlgorithm"] = signatureAlgorithmName(cert.SignatureAlgorithm)
	if cn := cert.Subject.CommonName; cn != "" {
		params["subjectCN"] = cn
	}

	return model.CryptoFinding{
		ID:         d.id(f, "certs.x509.certificate", primitive, params),
		AssetType:  model.AssetCertificate,
		Primitive:  primitive,
		Parameters: params,
		Functions:  []model.CryptoFunction{model.FunctionVerify},
		Evidence: model.Evidence{
			Source:     model.SourceCertificate,
			Path:       f.Path,
			Snippet:    certSnippet(cert, primitive, params),
			RuleID:     "certs.x509.certificate",
			Confidence: model.ConfidenceHigh,
		},
	}
}

func (d *Detector) findingFromPrivateKey(f collector.FileRef, pemType string, der []byte) (model.CryptoFinding, error) {
	key, err := parsePrivateKey(pemType, der)
	if err != nil {
		return model.CryptoFinding{}, err
	}
	primitive, params := publicKeyInfo(publicFromPrivate(key))
	if primitive == "UNKNOWN" {
		return model.CryptoFinding{}, fmt.Errorf("unrecognized private key type")
	}
	return model.CryptoFinding{
		ID:         d.id(f, "certs.key.private", primitive, params),
		AssetType:  model.AssetKey,
		Primitive:  primitive,
		Parameters: params,
		Functions:  []model.CryptoFunction{model.FunctionSign},
		Evidence: model.Evidence{
			Source:     model.SourceCertificate,
			Path:       f.Path,
			Snippet:    fmt.Sprintf("private key metadata; %s", describe(primitive, params)),
			RuleID:     "certs.key.private",
			Confidence: model.ConfidenceHigh,
		},
	}, nil
}

func (d *Detector) findingFromPublicKey(f collector.FileRef, der []byte) (model.CryptoFinding, error) {
	key, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return model.CryptoFinding{}, err
	}
	primitive, params := publicKeyInfo(key)
	return model.CryptoFinding{
		ID:         d.id(f, "certs.key.public", primitive, params),
		AssetType:  model.AssetKey,
		Primitive:  primitive,
		Parameters: params,
		Functions:  []model.CryptoFunction{model.FunctionVerify},
		Evidence: model.Evidence{
			Source:     model.SourceCertificate,
			Path:       f.Path,
			Snippet:    fmt.Sprintf("public key metadata; %s", describe(primitive, params)),
			RuleID:     "certs.key.public",
			Confidence: model.ConfidenceHigh,
		},
	}, nil
}

func (d *Detector) unknownPEM(f collector.FileRef, pemType string) model.CryptoFinding {
	params := map[string]any{"pemType": pemType}
	primitive := "UNKNOWN"
	return model.CryptoFinding{
		ID:         d.id(f, "certs.pem.unknown", primitive, params),
		AssetType:  model.AssetKey,
		Primitive:  primitive,
		Parameters: params,
		Evidence: model.Evidence{
			Source:     model.SourceCertificate,
			Path:       f.Path,
			Snippet:    fmt.Sprintf("unrecognized PEM type %q", pemType),
			RuleID:     "certs.pem.unknown",
			Confidence: model.ConfidenceLow,
		},
	}
}

func (d *Detector) id(f collector.FileRef, ruleID, primitive string, params map[string]any) string {
	return model.FindingID(model.FindingIDInput{
		Detector:   d.Name(),
		RuleID:     ruleID,
		RelPath:    f.Path,
		Line:       0,
		Primitive:  primitive,
		Parameters: params,
	})
}

func parsePrivateKey(pemType string, der []byte) (any, error) {
	switch pemType {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(der)
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(der)
	default:
		if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
			return key, nil
		}
		if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
			return key, nil
		}
		if key, err := x509.ParseECPrivateKey(der); err == nil {
			return key, nil
		}
		return nil, fmt.Errorf("unable to parse private key")
	}
}

func publicFromPrivate(key any) any {
	switch k := key.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	case ed25519.PrivateKey:
		return k.Public()
	case *dsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func publicKeyInfo(pub any) (string, map[string]any) {
	params := map[string]any{}
	switch k := pub.(type) {
	case *rsa.PublicKey:
		params["keySize"] = k.N.BitLen()
		return "RSA", params
	case *ecdsa.PublicKey:
		params["curve"] = curveName(k.Curve)
		params["keySize"] = k.Curve.Params().BitSize
		return "ECDSA", params
	case ed25519.PublicKey:
		params["curve"] = "Ed25519"
		params["keySize"] = 256
		return "EdDSA", params
	case *dsa.PublicKey:
		if k.P != nil {
			params["keySize"] = k.P.BitLen()
		}
		return "DSA", params
	default:
		return "UNKNOWN", params
	}
}

func curveName(c elliptic.Curve) string {
	switch c {
	case elliptic.P256():
		return "P-256"
	case elliptic.P384():
		return "P-384"
	case elliptic.P521():
		return "P-521"
	default:
		if c == nil || c.Params() == nil {
			return "unknown"
		}
		name := c.Params().Name
		if name == "" {
			return "unknown"
		}
		return name
	}
}

func signatureAlgorithmName(alg x509.SignatureAlgorithm) string {
	// Canonical display names; classifier will interpret the public-key primitive.
	switch alg {
	case x509.MD5WithRSA:
		return "MD5-RSA"
	case x509.SHA1WithRSA:
		return "SHA1-RSA"
	case x509.SHA256WithRSA:
		return "SHA256-RSA"
	case x509.SHA384WithRSA:
		return "SHA384-RSA"
	case x509.SHA512WithRSA:
		return "SHA512-RSA"
	case x509.ECDSAWithSHA1:
		return "ECDSA-SHA1"
	case x509.ECDSAWithSHA256:
		return "ECDSA-SHA256"
	case x509.ECDSAWithSHA384:
		return "ECDSA-SHA384"
	case x509.ECDSAWithSHA512:
		return "ECDSA-SHA512"
	case x509.PureEd25519:
		return "Ed25519"
	case x509.SHA256WithRSAPSS:
		return "SHA256-RSAPSS"
	case x509.SHA384WithRSAPSS:
		return "SHA384-RSAPSS"
	case x509.SHA512WithRSAPSS:
		return "SHA512-RSAPSS"
	default:
		return alg.String()
	}
}

func certSnippet(cert *x509.Certificate, primitive string, params map[string]any) string {
	cn := cert.Subject.CommonName
	if cn == "" {
		cn = "unnamed"
	}
	return fmt.Sprintf("x509 certificate CN=%s; %s", cn, describe(primitive, params))
}

func describe(primitive string, params map[string]any) string {
	if curve, ok := params["curve"].(string); ok && curve != "" {
		return fmt.Sprintf("%s %s", primitive, curve)
	}
	if size, ok := params["keySize"]; ok {
		return fmt.Sprintf("%s-%v", primitive, size)
	}
	return primitive
}
