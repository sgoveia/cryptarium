// Package source implements rule-pack source-code scanning via gotreesitter
// (pure-Go tree-sitter runtime; CGO_ENABLED=0 — DESIGN.md §17).
package source

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"

	"github.com/sgoveia/cryptarium/internal/collector"
	"github.com/sgoveia/cryptarium/internal/detector"
	"github.com/sgoveia/cryptarium/internal/model"
	"github.com/sgoveia/cryptarium/internal/rules"
)

func init() {
	detector.Register(New())
}

// Detector matches YAML rule packs against parsed source ASTs.
type Detector struct {
	rulesDir  string
	extraDirs []string

	mu     sync.Mutex
	packs  []*rules.RulePack
	err    error
	loaded bool
}

// New returns a source detector that loads packs via LoadDefaultRulePacks
// (on-disk rules/ when present, otherwise packs embedded in the binary).
func New() *Detector {
	return &Detector{}
}

// NewWithRulesDir returns a detector that loads packs from dir.
func NewWithRulesDir(dir string) *Detector {
	return &Detector{rulesDir: dir}
}

// NewConfigured returns a detector that loads packs from rulesDir when
// non-empty, otherwise LoadDefaultRulePacks, then appends packs from extra.
func NewConfigured(rulesDir string, extra ...string) *Detector {
	return &Detector{
		rulesDir:  rulesDir,
		extraDirs: append([]string(nil), extra...),
	}
}

// Name returns the stable detector name.
func (*Detector) Name() string { return "source" }

var handledExt = map[string]string{
	".go":   "go",
	".py":   "python",
	".js":   "javascript",
	".mjs":  "javascript",
	".cjs":  "javascript",
	".ts":   "typescript",
	".tsx":  "typescript",
	".jsx":  "javascript",
	".java": "java",
	".c":    "c",
	".h":    "c",
	".cc":   "cpp",
	".cpp":  "cpp",
	".cxx":  "cpp",
	".hpp":  "cpp",
	".hh":   "cpp",
}

// Handles reports whether f is a supported source file by extension.
func (*Detector) Handles(f collector.FileRef) bool {
	_, ok := handledExt[collector.Ext(f)]
	return ok
}

var parseMu sync.Mutex // gotreesitter grammar/runtime is process-global; serialize parses

// Detect parses f and applies loaded rule packs for its language.
func (d *Detector) Detect(ctx context.Context, f collector.FileRef) ([]model.CryptoFinding, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if err := d.ensurePacks(); err != nil {
		return nil, err
	}

	langName, ok := handledExt[collector.Ext(f)]
	if !ok {
		return nil, nil
	}

	data, err := os.ReadFile(f.AbsPath) //nolint:gosec // G304: AbsPath comes from collector walk
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", f.Path, err)
	}

	parseMu.Lock()
	defer parseMu.Unlock()

	entry := grammars.DetectLanguage(f.Path)
	if entry == nil {
		return d.regexFallback(f, data, langName), nil
	}
	lang := entry.Language()
	if lang == nil {
		return d.regexFallback(f, data, langName), nil
	}

	parser := gotreesitter.NewParser(lang)
	tree, err := parser.Parse(data)
	if err != nil {
		return d.regexFallback(f, data, langName), fmt.Errorf("parse %s: %w", f.Path, err)
	}
	if tree == nil || tree.RootNode() == nil {
		return d.regexFallback(f, data, langName), nil
	}
	defer tree.Release()

	packs := d.packsFor(langName)
	var findings []model.CryptoFinding
	for _, pack := range packs {
		for _, rule := range pack.Rules {
			got, err := d.applyRule(f, data, lang, tree, rule)
			if err != nil {
				return findings, fmt.Errorf("rule %s on %s: %w", rule.ID, f.Path, err)
			}
			findings = append(findings, got...)
		}
	}
	return findings, nil
}

func (d *Detector) ensurePacks() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.loaded {
		return d.err
	}
	d.loaded = true
	var (
		packs []*rules.RulePack
		err   error
	)
	switch {
	case d.rulesDir != "":
		packs, err = rules.LoadRulePacksDir(d.rulesDir)
	default:
		packs, err = rules.LoadDefaultRulePacks()
	}
	if err != nil {
		d.err = err
		return err
	}
	for _, extra := range d.extraDirs {
		more, err := rules.LoadRulePacksDir(extra)
		if err != nil {
			d.err = fmt.Errorf("load additional rules %s: %w", extra, err)
			return d.err
		}
		packs = append(packs, more...)
	}
	d.packs = packs
	return nil
}

func (d *Detector) packsFor(lang string) []*rules.RulePack {
	want := languageAliases(lang)
	var out []*rules.RulePack
	for _, p := range d.packs {
		for _, w := range want {
			if p.Language == w {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

func languageAliases(lang string) []string {
	switch lang {
	case "typescript":
		return []string{"typescript", "javascript"}
	case "cpp":
		return []string{"cpp", "c"}
	default:
		return []string{lang}
	}
}

func (d *Detector) applyRule(f collector.FileRef, src []byte, lang *gotreesitter.Language, tree *gotreesitter.Tree, rule rules.Rule) ([]model.CryptoFinding, error) {
	switch rule.Match.Kind {
	case "call":
		return d.matchCalls(f, src, lang, tree, rule)
	case "identifier":
		return d.matchIdentifiers(f, src, lang, tree, rule)
	default:
		return nil, fmt.Errorf("unsupported match.kind %q", rule.Match.Kind)
	}
}

func (d *Detector) matchCalls(f collector.FileRef, src []byte, lang *gotreesitter.Language, tree *gotreesitter.Tree, rule rules.Rule) ([]model.CryptoFinding, error) {
	query, ok := callQueryFor(langNameOf(f))
	if !ok {
		return nil, nil
	}
	q, err := gotreesitter.NewQuery(query, lang)
	if err != nil {
		return nil, err
	}
	imports := collectImports(langNameOf(f), src, lang, tree)

	var findings []model.CryptoFinding
	for _, m := range q.Execute(tree) {
		caps := captureMap(m)
		symNode, callNode := caps["sym"], caps["call"]
		if symNode == nil || callNode == nil {
			continue
		}
		pkgNode := caps["pkg"]
		pkgText := ""
		if pkgNode != nil {
			pkgText = pkgNode.Text(src)
		}
		symText := symNode.Text(src)
		if symText != rule.Match.Symbol {
			continue
		}
		if !packageMatches(rule.Match.Package, pkgText, imports) {
			continue
		}
		params := resolveParams(rule, src, callNode, lang)
		findings = append(findings, d.finding(f, rule, callNode, params, src))
	}
	return findings, nil
}

func (d *Detector) matchIdentifiers(f collector.FileRef, src []byte, lang *gotreesitter.Language, tree *gotreesitter.Tree, rule rules.Rule) ([]model.CryptoFinding, error) {
	if rule.Match.Pattern == "" {
		return nil, fmt.Errorf("identifier rule %s missing pattern", rule.ID)
	}
	re, err := regexp.Compile(rule.Match.Pattern)
	if err != nil {
		return nil, err
	}
	q, err := gotreesitter.NewQuery(`(identifier) @id`, lang)
	if err != nil {
		// Some grammars use different node names; fall back to line scan of non-comment is hard — use regex on source with low confidence already on rule.
		return d.regexIdentifier(f, src, rule, re), nil
	}
	var findings []model.CryptoFinding
	for _, m := range q.Execute(tree) {
		for _, cap := range m.Captures {
			if cap.Name != "id" || cap.Node == nil {
				continue
			}
			text := cap.Node.Text(src)
			if !re.MatchString(text) {
				continue
			}
			params := resolveParams(rule, src, cap.Node, lang)
			findings = append(findings, d.finding(f, rule, cap.Node, params, src))
		}
	}
	return findings, nil
}

func (d *Detector) regexIdentifier(f collector.FileRef, src []byte, rule rules.Rule, re *regexp.Regexp) []model.CryptoFinding {
	var findings []model.CryptoFinding
	lines := strings.Split(string(src), "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "//") || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if loc := re.FindStringIndex(line); loc != nil {
			params := staticParams(rule)
			findings = append(findings, model.CryptoFinding{
				ID: model.FindingID(model.FindingIDInput{
					Detector:   d.Name(),
					RuleID:     rule.ID,
					RelPath:    f.Path,
					Line:       i + 1,
					Primitive:  rule.Primitive,
					Parameters: params,
				}),
				AssetType:  model.AssetType(rule.AssetType),
				Primitive:  rule.Primitive,
				Parameters: params,
				Functions:  toFunctions(rule.Functions),
				Evidence: model.Evidence{
					Source:     model.SourceCode,
					Path:       f.Path,
					Line:       i + 1,
					Column:     loc[0] + 1,
					Snippet:    truncate(strings.TrimSpace(line), 120),
					RuleID:     rule.ID,
					Confidence: model.ConfidenceLow, // regex fallback
				},
			})
		}
	}
	return findings
}

func (d *Detector) regexFallback(f collector.FileRef, src []byte, lang string) []model.CryptoFinding {
	var findings []model.CryptoFinding
	for _, pack := range d.packsFor(lang) {
		for _, rule := range pack.Rules {
			if rule.Match.Kind != "call" {
				continue
			}
			// Lowered confidence regex for symbol( when grammar unavailable.
			re := regexp.MustCompile(`\b` + regexp.QuoteMeta(rule.Match.Symbol) + `\s*\(`)
			findings = append(findings, d.regexIdentifier(f, src, rule, re)...)
		}
	}
	return findings
}

func (d *Detector) finding(f collector.FileRef, rule rules.Rule, node *gotreesitter.Node, params map[string]any, src []byte) model.CryptoFinding {
	line := int(node.StartPoint().Row) + 1
	col := int(node.StartPoint().Column) + 1
	snippet := truncate(strings.TrimSpace(node.Text(src)), 120)
	conf := parseConfidence(rule.Confidence)
	return model.CryptoFinding{
		ID: model.FindingID(model.FindingIDInput{
			Detector:   d.Name(),
			RuleID:     rule.ID,
			RelPath:    f.Path,
			Line:       line,
			Primitive:  rule.Primitive,
			Parameters: params,
		}),
		AssetType:  model.AssetType(orDefault(rule.AssetType, string(model.AssetAlgorithm))),
		Primitive:  rule.Primitive,
		Parameters: params,
		Functions:  toFunctions(rule.Functions),
		Evidence: model.Evidence{
			Source:     model.SourceCode,
			Path:       f.Path,
			Line:       line,
			Column:     col,
			Snippet:    snippet,
			RuleID:     rule.ID,
			Confidence: conf,
		},
	}
}

func langNameOf(f collector.FileRef) string {
	return handledExt[collector.Ext(f)]
}

func callQueryFor(lang string) (string, bool) {
	switch lang {
	case "go":
		return `(call_expression
  function: (selector_expression
    operand: (identifier) @pkg
    field: (field_identifier) @sym)
) @call`, true
	case "python":
		return `(call
  function: (attribute
    object: (identifier) @pkg
    attribute: (identifier) @sym)
) @call`, true
	case "javascript", "typescript":
		return `(call_expression
  function: (member_expression
    object: (identifier) @pkg
    property: (property_identifier) @sym)
) @call`, true
	case "java":
		return `(method_invocation
  object: (identifier) @pkg
  name: (identifier) @sym
) @call`, true
	case "c", "cpp":
		return `(call_expression
  function: (identifier) @sym
) @call`, true
	default:
		return "", false
	}
}

func captureMap(m gotreesitter.QueryMatch) map[string]*gotreesitter.Node {
	out := make(map[string]*gotreesitter.Node, len(m.Captures))
	for _, c := range m.Captures {
		out[c.Name] = c.Node
	}
	return out
}

func packageMatches(wantPackage, pkgIdent string, imports map[string]string) bool {
	if wantPackage == "" {
		return true
	}
	// C/OpenSSL: no package qualifier; symbol-only match with package tag "openssl".
	if wantPackage == "openssl" && pkgIdent == "" {
		return true
	}
	if path, ok := imports[pkgIdent]; ok {
		return path == wantPackage || strings.HasSuffix(path, "/"+filepath.Base(wantPackage)) || path == filepath.Base(wantPackage)
	}
	// Fallback: last path segment equals identifier (rsa from crypto/rsa).
	base := filepath.Base(wantPackage)
	if strings.Contains(wantPackage, ".") {
		parts := strings.Split(wantPackage, ".")
		base = parts[len(parts)-1]
	}
	return pkgIdent == base || pkgIdent == wantPackage
}

func collectImports(lang string, src []byte, gLang *gotreesitter.Language, tree *gotreesitter.Tree) map[string]string {
	out := map[string]string{}
	switch lang {
	case "go":
		q, err := gotreesitter.NewQuery(`(import_spec path: (interpreted_string_literal) @path) @spec`, gLang)
		if err != nil {
			return out
		}
		for _, m := range q.Execute(tree) {
			caps := captureMap(m)
			pathNode := caps["path"]
			if pathNode == nil {
				continue
			}
			p := strings.Trim(pathNode.Text(src), `"`)
			alias := filepath.Base(p)
			// import rsa "crypto/rsa" — check for name identifier sibling via parent text heuristic
			spec := caps["spec"]
			if spec != nil {
				text := spec.Text(src)
				fields := strings.Fields(text)
				if len(fields) == 2 && !strings.HasPrefix(fields[0], `"`) {
					alias = fields[0]
				}
			}
			out[alias] = p
		}
	case "python":
		// import hashlib / from X import y as z — keep simple
		q, err := gotreesitter.NewQuery(`(import_statement name: (dotted_name) @name)`, gLang)
		if err == nil {
			for _, m := range q.Execute(tree) {
				caps := captureMap(m)
				if n := caps["name"]; n != nil {
					name := n.Text(src)
					out[filepath.Base(strings.ReplaceAll(name, ".", "/"))] = name
					out[name] = name
				}
			}
		}
	case "javascript", "typescript":
		out["crypto"] = "crypto"
		out["subtle"] = "crypto.subtle"
	case "java":
		q, err := gotreesitter.NewQuery(`(import_declaration (scoped_identifier) @name)`, gLang)
		if err == nil {
			for _, m := range q.Execute(tree) {
				caps := captureMap(m)
				if n := caps["name"]; n != nil {
					name := n.Text(src)
					parts := strings.Split(name, ".")
					out[parts[len(parts)-1]] = name
				}
			}
		}
	case "c", "cpp":
		// symbol-only; treat empty pkg as openssl when rule says so
		out[""] = "openssl"
	}
	return out
}

func resolveParams(rule rules.Rule, src []byte, callNode *gotreesitter.Node, lang *gotreesitter.Language) map[string]any {
	params := staticParams(rule)
	if rule.Parameters == nil {
		return params
	}
	for key, raw := range rule.Parameters {
		spec, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		from, _ := spec["from"].(string)
		switch from {
		case "argument":
			idx := toInt(spec["index"])
			if v, ok := nthArgument(src, callNode, lang, idx, spec); ok {
				params[key] = v
			}
		case "value":
			params[key] = spec["value"]
		}
	}
	return params
}

func staticParams(rule rules.Rule) map[string]any {
	params := map[string]any{}
	if rule.Parameters == nil {
		return params
	}
	for key, raw := range rule.Parameters {
		if m, ok := raw.(map[string]any); ok {
			if v, ok := m["value"]; ok {
				params[key] = v
			}
			continue
		}
		params[key] = raw
	}
	return params
}

func nthArgument(src []byte, callNode *gotreesitter.Node, lang *gotreesitter.Language, index int, spec map[string]any) (any, bool) {
	if callNode == nil || index < 0 {
		return nil, false
	}
	// Find argument_list / arguments child
	var args *gotreesitter.Node
	for i := 0; i < callNode.ChildCount(); i++ {
		ch := callNode.Child(i)
		if ch == nil {
			continue
		}
		t := ch.Type(lang)
		if t == "argument_list" || t == "arguments" || t == "argument_list_expression" {
			args = ch
			break
		}
	}
	if args == nil {
		return nil, false
	}
	named := 0
	for i := 0; i < args.NamedChildCount(); i++ {
		ch := args.NamedChild(i)
		if ch == nil {
			continue
		}
		if named == index {
			text := strings.TrimSpace(ch.Text(src))
			switch typ, _ := spec["type"].(string); typ {
			case "int":
				n, err := strconv.Atoi(text)
				if err != nil {
					return text, true
				}
				return n, true
			case "string":
				return strings.Trim(text, `"'`), true
			default:
				return text, true
			}
		}
		named++
	}
	return nil, false
}

func parseConfidence(s string) model.Confidence {
	switch strings.ToLower(s) {
	case "high":
		return model.ConfidenceHigh
	case "low":
		return model.ConfidenceLow
	default:
		return model.ConfidenceMedium
	}
}

func toFunctions(fns []string) []model.CryptoFunction {
	out := make([]model.CryptoFunction, 0, len(fns))
	for _, f := range fns {
		out = append(out, model.CryptoFunction(f))
	}
	return out
}

func toInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	case string:
		n, _ := strconv.Atoi(x)
		return n
	default:
		return 0
	}
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
