package score

import (
	"fmt"
	"path"
	"strings"

	"github.com/sgoveia/cryptarium/internal/model"
)

// Score applies the four-axis formula from DESIGN.md §7:
//
//	score = 0.40·vulnerability + 0.25·longevity + 0.25·exposure + 0.10·agility
//
// Heuristics that fire are recorded in Explanation. Missing signals default to 50.
func Score(a model.CryptoAsset) model.ScoredAsset {
	v, vWhy := vulnerabilityAxis(a.QuantumClass)
	l, lWhy := longevityAxis(a)
	e, eWhy := exposureAxis(a)
	ag, aWhy := agilityAxis(a)

	// Integer arithmetic for determinism across platforms.
	total := (40*v + 25*l + 25*e + 10*ag) / 100
	priority := priorityBand(total)
	explanation := fmt.Sprintf(
		"score=%d (%s): vulnerability=%d (%s); longevity=%d (%s); exposure=%d (%s); agility=%d (%s)",
		total, priority, v, vWhy, l, lWhy, e, eWhy, ag, aWhy,
	)
	return model.ScoredAsset{
		CryptoAsset: a,
		Risk: model.RiskScore{
			Score:         total,
			Priority:      priority,
			Vulnerability: v,
			Longevity:     l,
			Exposure:      e,
			Agility:       ag,
			Explanation:   explanation,
		},
	}
}

// All scores each asset in order.
func All(assets []model.CryptoAsset) []model.ScoredAsset {
	out := make([]model.ScoredAsset, len(assets))
	for i, a := range assets {
		out[i] = Score(a)
	}
	return out
}

func vulnerabilityAxis(c model.QuantumClass) (int, string) {
	switch c {
	case model.ClassBroken:
		return 100, "quantumClass=broken"
	case model.ClassWeakened:
		return 50, "quantumClass=weakened"
	case model.ClassSafe:
		return 0, "quantumClass=safe"
	default:
		return 50, "quantumClass=unknown; defaulted to 50"
	}
}

func longevityAxis(a model.CryptoAsset) (int, string) {
	p := strings.ToLower(a.Evidence.Path)
	switch {
	case strings.Contains(p, "testdata/") || strings.Contains(p, "/fixtures/") || strings.HasSuffix(p, "_test.go"):
		return 20, "test/fixture path suggests short-lived material"
	case a.AssetType == model.AssetCertificate:
		// Certificates often back long-lived trust; raise above default.
		return 75, "certificate asset; long-lived trust anchor heuristic"
	case hasFunction(a, model.FunctionSign) || hasFunction(a, model.FunctionKeygen):
		return 70, "keygen/sign function; longer shelf-life heuristic"
	default:
		return 50, "no longevity signal; defaulted to 50"
	}
}

func exposureAxis(a model.CryptoAsset) (int, string) {
	p := strings.ToLower(a.Evidence.Path)
	base := path.Base(p)
	switch {
	case strings.Contains(p, "testdata/") || strings.Contains(p, "/example") || strings.Contains(p, "/examples/") ||
		strings.Contains(p, "/fixtures/") || strings.HasSuffix(base, "_test.go") || strings.Contains(p, "/vendor/"):
		return 10, "test/example/vendor path; low exposure"
	case strings.Contains(p, "nginx") || strings.Contains(p, "ingress") || strings.Contains(p, "/deploy/") ||
		strings.Contains(p, "/helm/") || strings.Contains(base, "dockerfile"):
		return 90, "deploy/ingress/nginx path; high exposure"
	case a.Evidence.Source == model.SourceConfiguration:
		return 80, "configuration source; likely runtime-facing"
	case a.Evidence.Source == model.SourceCertificate:
		return 70, "certificate in repo; often production-facing"
	default:
		return 50, "no exposure signal; defaulted to 50"
	}
}

func agilityAxis(a model.CryptoAsset) (int, string) {
	switch a.Evidence.Source {
	case model.SourceCode:
		return 80, "source call site; hardcoded primitive heuristic"
	case model.SourceCertificate, model.SourceConfiguration:
		return 70, "cert/config binding; limited agility heuristic"
	case model.SourceDependency:
		return 50, "dependency presence; medium agility cost"
	default:
		return 50, "no agility signal; defaulted to 50"
	}
}

func hasFunction(a model.CryptoAsset, fn model.CryptoFunction) bool {
	for _, f := range a.Functions {
		if f == fn {
			return true
		}
	}
	return false
}

func priorityBand(score int) model.Priority {
	switch {
	case score >= 80:
		return model.PriorityCritical
	case score >= 60:
		return model.PriorityHigh
	case score >= 35:
		return model.PriorityMedium
	default:
		return model.PriorityLow
	}
}

// MeetsFailOn reports whether priority meets or exceeds the --fail-on threshold.
func MeetsFailOn(priority model.Priority, failOn string) bool {
	failOn = strings.ToLower(strings.TrimSpace(failOn))
	if failOn == "" || failOn == "none" {
		return false
	}
	order := map[model.Priority]int{
		model.PriorityLow:      1,
		model.PriorityMedium:   2,
		model.PriorityHigh:     3,
		model.PriorityCritical: 4,
	}
	threshold := map[string]int{
		"low": 1, "medium": 2, "high": 3, "critical": 4,
	}
	t, ok := threshold[failOn]
	if !ok {
		return false
	}
	return order[priority] >= t
}
