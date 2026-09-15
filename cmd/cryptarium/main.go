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
	"strings"

	"github.com/sgoveia/cryptarium/internal/pipeline"
	"github.com/sgoveia/cryptarium/internal/report"
	"github.com/sgoveia/cryptarium/internal/rules"
)

// version is hardcoded for Phase 0; release builds may override later.
const version = "0.0.0-dev"

// Exit codes per DESIGN.md §11:
//
//	0 — clean or below threshold
//	1 — policy / --fail-on threshold exceeded (reserved until Phase 3)
//	2 — scan error (unreadable target, invalid rules, unimplemented)
const (
	exitOK        = 0
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
		writef(stdout, "cryptarium %s\n", version)
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
	if looksLikeGitURL(fs.target) {
		writef(stderr, "scan: remote git URLs are not supported yet; pass a local path\n")
		return exitScanError
	}

	catalog, err := rules.FindDefaultCatalog()
	if err != nil {
		writef(stderr, "scan: %v\n", err)
		return exitScanError
	}

	result, err := pipeline.Run(context.Background(), pipeline.Options{
		Root:        fs.target,
		Concurrency: fs.concurrency,
		CatalogPath: catalog,
	})
	if err != nil {
		writef(stderr, "scan: %v\n", err)
		return exitScanError
	}

	meta := report.Meta{
		ToolVersion:   version,
		Root:          fs.target,
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
	default:
		return "cryptarium." + format
	}
}

func looksLikeGitURL(target string) bool {
	return strings.HasPrefix(target, "http://") ||
		strings.HasPrefix(target, "https://") ||
		strings.HasPrefix(target, "git@") ||
		strings.HasPrefix(target, "ssh://")
}

func printUsage(w io.Writer) {
	write(w, `cryptarium — cryptographic discovery and CBOM generation

Usage:
  cryptarium <command> [arguments]

Commands:
  scan      Scan a local repository path
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
