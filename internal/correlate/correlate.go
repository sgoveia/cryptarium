// Package correlate links related CryptoFindings into clusters.
// Correlation raises confidence and merges evidence; it never creates findings
// and never invents a quantum classification (DESIGN.md §8).
package correlate

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/sgoveia/cryptarium/internal/classify"
	"github.com/sgoveia/cryptarium/internal/model"
)

// Cluster is a correlated set of findings describing the same underlying crypto.
type Cluster struct {
	Members []model.CryptoFinding // sorted by ID; length >= 1
}

// Link groups findings using DESIGN.md §8 v0.1 join keys:
//  1. primitive + normalized parameters
//  2. import/module → dependency
//  3. path proximity (certificate ↔ configuration)
//
// Unlinked findings become singleton clusters. Result order is deterministic.
func Link(findings []model.CryptoFinding) []Cluster {
	if len(findings) == 0 {
		return nil
	}
	n := len(findings)
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	find := func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	union := func(i, j int) {
		ri, rj := find(i), find(j)
		if ri == rj {
			return
		}
		if ri < rj {
			parent[rj] = ri
		} else {
			parent[ri] = rj
		}
	}

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if shouldLink(findings[i], findings[j]) {
				union(i, j)
			}
		}
	}

	groups := map[int][]model.CryptoFinding{}
	for i, f := range findings {
		r := find(i)
		groups[r] = append(groups[r], f)
	}

	roots := make([]int, 0, len(groups))
	for r := range groups {
		roots = append(roots, r)
	}
	sort.Ints(roots)

	out := make([]Cluster, 0, len(roots))
	for _, r := range roots {
		members := groups[r]
		sort.Slice(members, func(i, j int) bool {
			if members[i].ID != members[j].ID {
				return members[i].ID < members[j].ID
			}
			return members[i].Evidence.Path < members[j].Evidence.Path
		})
		out = append(out, Cluster{Members: members})
	}

	// Stable cluster order by canonical member path/line/id.
	sort.Slice(out, func(i, j int) bool {
		a, b := pickCanonical(out[i].Members), pickCanonical(out[j].Members)
		if a.Evidence.Path != b.Evidence.Path {
			return a.Evidence.Path < b.Evidence.Path
		}
		if a.Evidence.Line != b.Evidence.Line {
			return a.Evidence.Line < b.Evidence.Line
		}
		return a.ID < b.ID
	})
	return out
}

// Materialize classifies cluster members and emits one CryptoAsset for the
// cluster. Quantum class is taken from the strongest member; RelatedIDs list
// every other finding. Confidence is raised when a dependency links to source.
func Materialize(c Cluster) model.CryptoAsset {
	if len(c.Members) == 0 {
		return model.CryptoAsset{}
	}
	classified := make([]model.CryptoAsset, len(c.Members))
	for i, m := range c.Members {
		classified[i] = classify.Classify(m)
	}

	primary := pickCanonical(c.Members)
	var primaryAsset model.CryptoAsset
	for _, a := range classified {
		if a.ID == primary.ID {
			primaryAsset = a
			break
		}
	}

	strongest := classified[0]
	for _, a := range classified[1:] {
		if classRank(a.QuantumClass) > classRank(strongest.QuantumClass) {
			strongest = a
		}
	}
	primaryAsset.QuantumClass = strongest.QuantumClass
	primaryAsset.Rationale = strongest.Rationale
	primaryAsset.Recommendation = strongest.Recommendation
	if strongest.OID != "" {
		primaryAsset.OID = strongest.OID
	}

	related := make([]string, 0, len(c.Members)-1)
	for _, m := range c.Members {
		if m.ID != primary.ID {
			related = append(related, m.ID)
		}
	}
	sort.Strings(related)
	primaryAsset.RelatedIDs = related

	if shouldRaiseConfidence(c.Members) && primaryAsset.Evidence.Confidence != model.ConfidenceHigh {
		primaryAsset.Evidence.Confidence = model.ConfidenceHigh
	}
	return primaryAsset
}

// AllLinks materializes every cluster in order.
func AllLinks(findings []model.CryptoFinding) []model.CryptoAsset {
	clusters := Link(findings)
	out := make([]model.CryptoAsset, len(clusters))
	for i, c := range clusters {
		out[i] = Materialize(c)
	}
	return out
}

func shouldLink(a, b model.CryptoFinding) bool {
	if primitiveParamLink(a, b) {
		return true
	}
	if importDepLink(a, b) {
		return true
	}
	if pathProximityLink(a, b) {
		return true
	}
	return false
}

func primitiveParamLink(a, b model.CryptoFinding) bool {
	pa := classify.Canonical(a.Primitive)
	pb := classify.Canonical(b.Primitive)
	if pa == "" || pb == "" || pa != pb || pa == "UNKNOWN" {
		return false
	}
	fa, fb := fingerprint(a), fingerprint(b)
	if fa == fb {
		return true
	}
	// Deps/config often lack keySize; link bare primitive to parameterized peer.
	if fa == pa && strings.HasPrefix(fb, pa+"|") {
		return true
	}
	if fb == pa && strings.HasPrefix(fa, pa+"|") {
		return true
	}
	return false
}

func fingerprint(f model.CryptoFinding) string {
	p := classify.Canonical(f.Primitive)
	if p == "" {
		p = f.Primitive
	}
	if ks, ok := intParam(f.Parameters, "keySize"); ok {
		return fmt.Sprintf("%s|ks=%d", p, ks)
	}
	if curve, ok := stringParam(f.Parameters, "curve"); ok && curve != "" {
		return fmt.Sprintf("%s|curve=%s", p, curve)
	}
	return p
}

func importDepLink(a, b model.CryptoFinding) bool {
	var src, dep model.CryptoFinding
	switch {
	case a.Evidence.Source == model.SourceCode && b.Evidence.Source == model.SourceDependency:
		src, dep = a, b
	case b.Evidence.Source == model.SourceCode && a.Evidence.Source == model.SourceDependency:
		src, dep = b, a
	default:
		return false
	}
	mod, _ := stringParam(dep.Parameters, "module")
	if mod == "" {
		return false
	}
	pkg := inferredPackage(src)
	if pkg != "" && (pkg == mod || strings.HasPrefix(pkg, mod+"/") || strings.HasPrefix(mod, pkg+"/")) {
		return true
	}
	// Same canonical primitive: source use validates a deps catalog hit (DESIGN §8 #2 intent).
	return classify.Canonical(src.Primitive) == classify.Canonical(dep.Primitive) &&
		classify.Canonical(src.Primitive) != "" &&
		classify.Canonical(src.Primitive) != "UNKNOWN"
}

func inferredPackage(src model.CryptoFinding) string {
	if p, ok := stringParam(src.Parameters, "package"); ok && p != "" {
		return p
	}
	// go.crypto.rsa.generatekey → crypto/rsa
	id := src.Evidence.RuleID
	if strings.HasPrefix(id, "go.crypto.") {
		rest := strings.TrimPrefix(id, "go.crypto.")
		parts := strings.Split(rest, ".")
		if len(parts) >= 1 && parts[0] != "" {
			return "crypto/" + parts[0]
		}
	}
	return ""
}

func pathProximityLink(a, b model.CryptoFinding) bool {
	// Certificate ↔ configuration in the same directory or immediate parent/child.
	sa, sb := a.Evidence.Source, b.Evidence.Source
	certConfig := (sa == model.SourceCertificate && sb == model.SourceConfiguration) ||
		(sb == model.SourceCertificate && sa == model.SourceConfiguration)
	if !certConfig {
		return false
	}
	if classify.Canonical(a.Primitive) != classify.Canonical(b.Primitive) {
		return false
	}
	da := path.Dir(a.Evidence.Path)
	db := path.Dir(b.Evidence.Path)
	if da == db {
		return true
	}
	if path.Dir(da) == db || path.Dir(db) == da {
		return true
	}
	return false
}

func pickCanonical(members []model.CryptoFinding) model.CryptoFinding {
	best := members[0]
	bestRank := sourceRank(best.Evidence.Source)
	for _, m := range members[1:] {
		r := sourceRank(m.Evidence.Source)
		if r > bestRank {
			best = m
			bestRank = r
			continue
		}
		if r == bestRank {
			if m.Evidence.Path < best.Evidence.Path ||
				(m.Evidence.Path == best.Evidence.Path && m.Evidence.Line < best.Evidence.Line) ||
				(m.Evidence.Path == best.Evidence.Path && m.Evidence.Line == best.Evidence.Line && m.ID < best.ID) {
				best = m
			}
		}
	}
	return best
}

func sourceRank(s model.SourceKind) int {
	switch s {
	case model.SourceCode:
		return 4
	case model.SourceCertificate:
		return 3
	case model.SourceConfiguration:
		return 2
	case model.SourceDependency:
		return 1
	default:
		return 0
	}
}

func classRank(c model.QuantumClass) int {
	switch c {
	case model.ClassBroken:
		return 4
	case model.ClassWeakened:
		return 3
	case model.ClassUnknown:
		return 2
	case model.ClassSafe:
		return 1
	default:
		return 0
	}
}

func shouldRaiseConfidence(members []model.CryptoFinding) bool {
	var hasSourceHigh, hasDeps bool
	for _, m := range members {
		if m.Evidence.Source == model.SourceCode && m.Evidence.Confidence == model.ConfidenceHigh {
			hasSourceHigh = true
		}
		if m.Evidence.Source == model.SourceDependency {
			hasDeps = true
		}
	}
	return hasSourceHigh && hasDeps
}

func intParam(params map[string]any, key string) (int, bool) {
	if params == nil {
		return 0, false
	}
	v, ok := params[key]
	if !ok {
		return 0, false
	}
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	case float64:
		return int(x), true
	case string:
		n, err := strconv.Atoi(x)
		return n, err == nil
	default:
		return 0, false
	}
}

func stringParam(params map[string]any, key string) (string, bool) {
	if params == nil {
		return "", false
	}
	v, ok := params[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}
