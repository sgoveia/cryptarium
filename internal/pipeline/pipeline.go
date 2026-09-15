package pipeline

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"sync"

	"github.com/sgoveia/cryptarium/internal/collector"
	"github.com/sgoveia/cryptarium/internal/detector"
	"github.com/sgoveia/cryptarium/internal/detector/deps"
	"github.com/sgoveia/cryptarium/internal/model"

	// Register built-in detectors at init.
	_ "github.com/sgoveia/cryptarium/internal/detector/certs"
)

// Options controls a scan run.
type Options struct {
	Root        string
	Concurrency int
	// CatalogPath overrides the default rules/libraries/catalog.yaml for deps.
	CatalogPath string
}

// Result is the deterministic scan output before reporting.
type Result struct {
	Findings []model.CryptoFinding
	Warnings []string
}

// Run walks root and runs all registered detectors. Findings are sorted by
// (path, line, column, ruleID) before return (DESIGN.md §9).
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

	detectors := activeDetectors(opt.CatalogPath)

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
	return &Result{Findings: findings, Warnings: warnings}, nil
}

func activeDetectors(catalogPath string) []detector.Detector {
	all := detector.All()
	if catalogPath == "" {
		return all
	}
	out := make([]detector.Detector, 0, len(all))
	seenDeps := false
	for _, d := range all {
		if d.Name() == "deps" {
			out = append(out, deps.NewWithCatalog(catalogPath))
			seenDeps = true
			continue
		}
		out = append(out, d)
	}
	if !seenDeps {
		out = append(out, deps.NewWithCatalog(catalogPath))
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
