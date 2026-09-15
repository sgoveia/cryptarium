// Package main is the cryptarium CLI entrypoint.
// Flag parsing only lives here; pipeline logic belongs in internal packages.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// version is hardcoded for Phase 0; release builds may override later.
const version = "0.0.0-dev"

// Exit codes per DESIGN.md §11:
//
//	0 — clean or below threshold
//	1 — policy / --fail-on threshold exceeded (reserved until Phase 1+)
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
	_ = stdout
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

	// Phase 0: CLI surface only. Detectors land in Phase 1.
	writef(stderr, "cryptarium: scan is not implemented yet (phase 0 scaffold)\n")
	return exitScanError
}

func printUsage(w io.Writer) {
	write(w, `cryptarium — cryptographic discovery and CBOM generation

Usage:
  cryptarium <command> [arguments]

Commands:
  scan      Scan a repository path or git URL (not yet implemented)
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
