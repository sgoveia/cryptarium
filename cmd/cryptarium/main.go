// Package main is the cryptarium CLI entrypoint.
// Flag parsing only lives here; pipeline logic belongs in internal packages.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/sgoveia/cryptarium/internal/collector"
	"github.com/sgoveia/cryptarium/internal/pipeline"
	"github.com/sgoveia/cryptarium/internal/report"
	"github.com/sgoveia/cryptarium/internal/rules"
	"github.com/sgoveia/cryptarium/internal/score"
)

// version is overridden at release build time via:
//
//	-ldflags "-X main.version=0.1.0"
//
// Dev builds keep the default. `go install @vX.Y.Z` picks up the module
// version via runtime/debug.ReadBuildInfo when ldflags were not set.
var version = "0.0.0-dev"

func resolvedVersion() string {
	if version != "" && version != "0.0.0-dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return version
}

// Exit codes per DESIGN.md §11:
//
//	0 — clean or below threshold
//	1 — policy / --fail-on threshold exceeded
//	2 — scan error (unreadable target, invalid rules)
const (
	exitOK        = 0
	exitPolicy    = 1
	exitScanError = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return exitScanError
	}

	switch args[0] {
	case "version", "--version", "-V":
		writef(stdout, "cryptarium %s\n", resolvedVersion())
		return exitOK
	case "scan":
		return runScan(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return exitOK
	default:
		writef(stderr, "unknown command %q\n\n", args[0])
		printUsage(stderr)
		return exitScanError
	}
}

func runScan(args []string, stdout, stderr io.Writer) int {
	fs := newScanFlags()
	if err := fs.parse(args); err != nil {
		if errors.Is(err, errHelp) {
			write(stderr, fs.usage())
			return exitOK
		}
		writef(stderr, "scan: %v\n", err)
		return exitScanError
	}
	if fs.target == "" {
		writeln(stderr, "scan: target path or git URL required")
		write(stderr, fs.usage())
		return exitScanError
	}

	resolved, err := collector.Resolve(context.Background(), fs.target)
	if err != nil {
		writef(stderr, "scan: %v\n", err)
		return exitScanError
	}
	defer resolved.Cleanup()

	catalog, err := rules.FindDefaultCatalog()
	if err != nil {
		writef(stderr, "scan: %v\n", err)
		return exitScanError
	}
	rulesDir, err := rules.FindDefaultRulesDir()
	if err != nil {
		writef(stderr, "scan: %v\n", err)
		return exitScanError
	}

	result, err := pipeline.Run(context.Background(), pipeline.Options{
		Root:        resolved.Root,
		Concurrency: fs.concurrency,
		CatalogPath: catalog,
		RulesDir:    rulesDir,
	})
	if err != nil {
		writef(stderr, "scan: %v\n", err)
		return exitScanError
	}

	meta := report.Meta{
		ToolVersion:   resolvedVersion(),
		Root:          resolved.Display,
		Deterministic: fs.deterministic,
	}

	for i, formatName := range fs.format {
		format := report.Format(formatName)
		out, closer, err := openOutput(fs.output, formatName, len(fs.format) > 1, i, stdout)
		if err != nil {
			writef(stderr, "scan: %v\n", err)
			return exitScanError
		}
		if err := report.Write(out, format, result, meta); err != nil {
			_ = closer()
			writef(stderr, "scan: write %s: %v\n", formatName, err)
			return exitScanError
		}
		if err := closer(); err != nil {
			writef(stderr, "scan: close output: %v\n", err)
			return exitScanError
		}
	}

	if fs.verbose {
		writef(stderr, "cryptarium: %d finding(s), %d warning(s)\n", len(result.Findings), len(result.Warnings))
	}

	for _, s := range result.Scored {
		if score.MeetsFailOn(s.Risk.Priority, fs.failOn) {
			writef(stderr, "cryptarium: fail-on %s triggered by %s (%s) at %s\n",
				fs.failOn, s.Primitive, s.Risk.Priority, s.Evidence.Path)
			return exitPolicy
		}
	}
	return exitOK
}

func openOutput(output, format string, multi bool, index int, stdout io.Writer) (io.Writer, func() error, error) {
	noop := func() error { return nil }
	if output == "" || output == "-" {
		if multi && index > 0 {
			return nil, noop, fmt.Errorf("multiple --format values require --output directory or file path")
		}
		return stdout, noop, nil
	}
	info, err := os.Stat(output)
	if err == nil && info.IsDir() {
		path := filepath.Join(output, defaultFileName(format))
		f, err := os.Create(filepath.Clean(path)) //nolint:gosec // G304: --output is an explicit user CLI path
		if err != nil {
			return nil, noop, err
		}
		return f, f.Close, nil
	}
	if multi {
		return nil, noop, fmt.Errorf("multiple --format values require --output to be a directory")
	}
	f, err := os.Create(filepath.Clean(output)) //nolint:gosec // G304: --output is an explicit user CLI path
	if err != nil {
		return nil, noop, err
	}
	return f, f.Close, nil
}

func defaultFileName(format string) string {
	switch format {
	case "json":
		return "cryptarium.json"
	case "markdown":
		return "CRYPTO-REPORT.md"
	case "cbom":
		return "cbom.json"
	case "sarif":
		return "results.sarif"
	default:
		return "cryptarium." + format
	}
}

func printUsage(w io.Writer) {
	write(w, `cryptarium — cryptographic discovery and CBOM generation

Usage:
  cryptarium <command> [arguments]

Commands:
  scan      Scan a local path or public HTTPS git URL
  version   Print version
  help      Show this help

`)
}

func write(w io.Writer, s string) {
	_, _ = io.WriteString(w, s)
}

func writeln(w io.Writer, s string) {
	writef(w, "%s\n", s)
}

func writef(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}
