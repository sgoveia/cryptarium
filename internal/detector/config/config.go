// Package config detects cryptographic settings in TLS, SSH, JWT, and similar configs.
package config

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/sgoveia/cryptarium/internal/collector"
	"github.com/sgoveia/cryptarium/internal/detector"
	"github.com/sgoveia/cryptarium/internal/model"
)

func init() {
	detector.Register(New())
}

// Detector extracts cryptographic primitives from configuration files.
type Detector struct{}

// New returns a configuration detector.
func New() *Detector {
	return &Detector{}
}

// Name returns the stable detector name.
func (*Detector) Name() string { return "config" }

var (
	nginxCipherDir = regexp.MustCompile(`(?i)^\s*ssl_ciphers\s+(.+?)\s*;`)
	sshdKV         = regexp.MustCompile(`(?i)^\s*(KexAlgorithms|HostKeyAlgorithms|Ciphers|MACs)\s+(.+)$`)
	jwtAlgJSON     = regexp.MustCompile(`(?i)"alg"\s*:\s*"([^"]+)"`)
	jwtAlgYAML     = regexp.MustCompile(`(?i)^\s*alg\s*:\s*["']?([A-Za-z0-9_-]+)["']?\s*$`)
	tlsSuiteToken  = regexp.MustCompile(`(?i)\b((?:TLS|SSL)_[A-Z0-9_]+)\b`)
)

// Handles reports whether f looks like a crypto-relevant config by name/extension.
func (*Detector) Handles(f collector.FileRef) bool {
	base := strings.ToLower(path.Base(f.Path))
	ext := strings.ToLower(collector.Ext(f))

	switch base {
	case "nginx.conf", "sshd_config", "ssh_config":
		return true
	}
	if strings.Contains(base, "nginx") && (ext == ".conf" || ext == "") {
		return true
	}
	if strings.HasPrefix(base, "sshd_config") || strings.HasPrefix(base, "ssh_config") {
		return true
	}
	switch ext {
	case ".conf":
		return true
	case ".json", ".yml", ".yaml":
		// Content-gated in Detect for JWT; accept common auth/jwt basenames cheaply.
		if strings.Contains(base, "jwt") || strings.Contains(base, "auth") ||
			strings.Contains(base, "security") || strings.Contains(base, "token") {
			return true
		}
	}
	return false
}

// Detect parses configuration directives and emits one finding per primitive.
func (d *Detector) Detect(ctx context.Context, f collector.FileRef) ([]model.CryptoFinding, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	data, err := os.ReadFile(f.AbsPath) //nolint:gosec // G304: AbsPath from collector walk
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", f.Path, err)
	}

	base := strings.ToLower(path.Base(f.Path))
	ext := strings.ToLower(collector.Ext(f))

	var findings []model.CryptoFinding
	switch {
	case base == "sshd_config" || strings.HasPrefix(base, "sshd_config") ||
		base == "ssh_config" || strings.HasPrefix(base, "ssh_config"):
		findings = append(findings, d.scanSSHD(f, data)...)
	case strings.Contains(base, "nginx") || ext == ".conf":
		findings = append(findings, d.scanNginxOrConf(f, data)...)
	case ext == ".json" || ext == ".yml" || ext == ".yaml":
		findings = append(findings, d.scanJWT(f, data)...)
	}

	return findings, nil
}

func (d *Detector) scanNginxOrConf(f collector.FileRef, data []byte) []model.CryptoFinding {
	var findings []model.CryptoFinding
	sc := bufio.NewScanner(bytes.NewReader(data))
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if m := nginxCipherDir.FindStringSubmatch(line); len(m) == 2 {
			findings = append(findings, d.emitCipherList(f, lineNum, "config.nginx.ssl_ciphers", m[1], true)...)
			continue
		}
		// Generic conf: bare TLS_* suite tokens (not in comments — already skipped).
		for _, m := range tlsSuiteToken.FindAllStringSubmatch(line, -1) {
			if len(m) < 2 {
				continue
			}
			for _, tok := range parseTLSCipherSuite(m[1]) {
				findings = append(findings, d.finding(f, lineNum, "config.tls.cipher_suite", tok, m[1]))
			}
		}
	}
	return findings
}

func (d *Detector) scanSSHD(f collector.FileRef, data []byte) []model.CryptoFinding {
	var findings []model.CryptoFinding
	sc := bufio.NewScanner(bytes.NewReader(data))
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		m := sshdKV.FindStringSubmatch(line)
		if len(m) != 3 {
			continue
		}
		directive := strings.ToLower(m[1])
		ruleID := "config.sshd." + directive
		for _, part := range splitAlgoList(m[2]) {
			for _, tok := range parseSSHAlgToken(part) {
				findings = append(findings, d.finding(f, lineNum, ruleID, tok, part))
			}
		}
	}
	return findings
}

func (d *Detector) scanJWT(f collector.FileRef, data []byte) []model.CryptoFinding {
	var findings []model.CryptoFinding
	ext := strings.ToLower(collector.Ext(f))

	if ext == ".json" {
		// Prefer structured JSON when valid; fall back to regex.
		var root any
		if err := json.Unmarshal(data, &root); err == nil {
			findings = append(findings, d.walkJSONForAlg(f, root, "")...)
			if len(findings) > 0 {
				return findings
			}
		}
	}

	sc := bufio.NewScanner(bytes.NewReader(data))
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		var alg string
		if m := jwtAlgJSON.FindStringSubmatch(line); len(m) == 2 {
			alg = m[1]
		} else if m := jwtAlgYAML.FindStringSubmatch(line); len(m) == 2 {
			alg = m[1]
		}
		if alg == "" {
			continue
		}
		for _, tok := range parseJWTAlg(alg) {
			findings = append(findings, d.finding(f, lineNum, "config.jwt.alg", tok, alg))
		}
	}
	return findings
}

func (d *Detector) walkJSONForAlg(f collector.FileRef, v any, pathHint string) []model.CryptoFinding {
	var out []model.CryptoFinding
	switch x := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			child := x[k]
			if strings.EqualFold(k, "alg") {
				if s, ok := child.(string); ok {
					for _, tok := range parseJWTAlg(s) {
						out = append(out, d.finding(f, 0, "config.jwt.alg", tok, s))
					}
				}
				continue
			}
			out = append(out, d.walkJSONForAlg(f, child, pathHint+"."+k)...)
		}
	case []any:
		for _, child := range x {
			out = append(out, d.walkJSONForAlg(f, child, pathHint)...)
		}
	}
	return out
}

func (d *Detector) emitCipherList(f collector.FileRef, line int, ruleID, raw string, tls bool) []model.CryptoFinding {
	var findings []model.CryptoFinding
	for _, part := range splitAlgoList(raw) {
		part = strings.Trim(part, `"'`)
		var tokens []suiteToken
		if tls {
			tokens = parseTLSCipherSuite(part)
			if len(tokens) == 0 {
				// OpenSSL-style names sometimes appear; try suite token extract.
				if m := tlsSuiteToken.FindStringSubmatch(part); len(m) == 2 {
					tokens = parseTLSCipherSuite(m[1])
				}
			}
		}
		for _, tok := range tokens {
			findings = append(findings, d.finding(f, line, ruleID, tok, part))
		}
	}
	return findings
}

func (d *Detector) finding(f collector.FileRef, line int, ruleID string, tok suiteToken, snippetSrc string) model.CryptoFinding {
	params := tok.Parameters
	if params == nil {
		params = map[string]any{}
	}
	snippet := strings.TrimSpace(snippetSrc)
	if len(snippet) > 120 {
		snippet = snippet[:117] + "..."
	}
	return model.CryptoFinding{
		ID: model.FindingID(model.FindingIDInput{
			Detector:   d.Name(),
			RuleID:     ruleID,
			RelPath:    f.Path,
			Line:       line,
			Primitive:  tok.Primitive,
			Parameters: params,
		}),
		AssetType:  model.AssetProtocol,
		Primitive:  tok.Primitive,
		Parameters: params,
		Functions:  tok.Functions,
		Evidence: model.Evidence{
			Source:     model.SourceConfiguration,
			Path:       f.Path,
			Line:       line,
			Snippet:    snippet,
			RuleID:     ruleID,
			Confidence: model.ConfidenceHigh,
		},
	}
}

func splitAlgoList(raw string) []string {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, `"'`)
	raw = strings.TrimSuffix(raw, ";")
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ':' || r == ',' || r == ' ' || r == '\t'
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || p == "!" || strings.HasPrefix(p, "!") {
			// OpenSSL exclusion prefixes — skip negated tokens.
			continue
		}
		out = append(out, p)
	}
	return out
}
