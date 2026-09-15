package pipeline

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"sync"

	"github.com/sgoveia/cryptarium/internal/classify"
	"github.com/sgoveia/cryptarium/internal/collector"
	"github.com/sgoveia/cryptarium/internal/detector"
	"github.com/sgoveia/cryptarium/internal/detector/deps"
	"github.com/sgoveia/cryptarium/internal/model"

	// Register built-in detectors at init.
	_ "github.com/sgoveia/cryptarium/internal/detector/certs"
	_ "github.com/sgoveia/cryptarium/internal/detector/source"
)

// Options controls a scan run.
type Options struct {
	Root        string
	Concurrency int
	// CatalogPath overrides the default rules/libraries/catalog.yaml for deps.
	CatalogPath string
	// RulesDir overrides the default rules/ directory for source packs.
	RulesDir string
}

// Result is the deterministic scan output before reporting.
type Result struct {
	Findings []model.CryptoFinding
	Assets   []model.CryptoAsset
	Warnings []string
}

// Run walks root and runs all registered detectors. Findings are sorted by
// (path, line, column, ruleID) before return (DESIGN.md §9), then classified.
func Run(ctx context.Context, opt Options) (*Result, error) {
	if opt.Root == "" {
		return nil, fmt.Errorf("pipeline: root is required")
	}
	workers := opt.Concurrency
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	files, err := collector.Walk(ctx, collector.Target{Root: opt.Root})
	if err != nil {
		return nil, err
	}

	detectors := activeDetectors(opt)

	type job struct {
		file collector.FileRef
		det  detector.Detector
	}
	var jobs []job
	for _, f := range files {
		for _, d := range detectors {
			if d.Handles(f) {
				jobs = append(jobs, job{file: f, det: d})
			}
		}
	}

	var (
		mu       sync.Mutex
		findings []model.CryptoFinding
		warnings []string
		wg       sync.WaitGroup
	)
	sem := make(chan struct{}, workers)

	for _, j := range jobs {
		j := j
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			got, err := j.det.Detect(ctx, j.file)
			mu.Lock()
			defer mu.Unlock()
			findings = append(findings, got...)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("%s: %v", j.file.Path, err))
			}
		}()
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	sortFindings(findings)
	sort.Strings(warnings)
	assets := classify.All(findings)
	return &Result{Findings: findings, Assets: assets, Warnings: warnings}, nil
}

func activeDetectors(opt Options) []detector.Detector {
	all := detector.All()
	out := make([]detector.Detector, 0, len(all))
	seenDeps := false
	seenSource := false
	for _, d := range all {
		switch d.Name() {
		case "deps":
			if opt.CatalogPath != "" {
				out = append(out, deps.NewWithCatalog(opt.CatalogPath))
			} else {
				out = append(out, d)
			}
			seenDeps = true
		case "source":
			if opt.RulesDir != "" {
				out = append(out, newSourceWithRules(opt.RulesDir))
			} else {
				out = append(out, d)
			}
			seenSource = true
		default:
			out = append(out, d)
		}
	}
	if opt.CatalogPath != "" && !seenDeps {
		out = append(out, deps.NewWithCatalog(opt.CatalogPath))
	}
	if opt.RulesDir != "" && !seenSource {
		out = append(out, newSourceWithRules(opt.RulesDir))
	}
	return out
}

func sortFindings(findings []model.CryptoFinding) {
	sort.Slice(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.Evidence.Path != b.Evidence.Path {
			return a.Evidence.Path < b.Evidence.Path
		}
		if a.Evidence.Line != b.Evidence.Line {
			return a.Evidence.Line < b.Evidence.Line
		}
		if a.Evidence.Column != b.Evidence.Column {
			return a.Evidence.Column < b.Evidence.Column
		}
		if a.Evidence.RuleID != b.Evidence.RuleID {
			return a.Evidence.RuleID < b.Evidence.RuleID
		}
		return a.ID < b.ID
	})
}
