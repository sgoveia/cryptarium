package main

import (
	"errors"
	"flag"
	"fmt"
	"strings"
)

var errHelp = errors.New("help")

// scanFlags holds CLI flags for `cryptarium scan`.
// Parsing only; no pipeline logic. See DESIGN.md §11.
type scanFlags struct {
	fs            *flag.FlagSet
	format        multiFlag
	output        string
	rules         multiFlag
	exclude       multiFlag
	includeTests  bool
	failOn        string
	policy        string
	concurrency   int
	deterministic bool
	verbose       bool
	target        string
}

func newScanFlags() *scanFlags {
	s := &scanFlags{
		fs: flag.NewFlagSet("scan", flag.ContinueOnError),
	}
	s.fs.SetOutput(ioDiscard{})
	s.fs.Var(&s.format, "format", "output format: json|cbom|sarif|markdown|html (repeatable)")
	s.fs.StringVar(&s.output, "output", "", "output path, or \"-\" for stdout")
	s.fs.Var(&s.rules, "rules", "additional rule-pack directory (repeatable)")
	s.fs.Var(&s.exclude, "exclude", "glob to exclude (repeatable)")
	s.fs.BoolVar(&s.includeTests, "include-tests", false, "include test files and fixtures at normal scoring")
	s.fs.StringVar(&s.failOn, "fail-on", "none", "fail when priority reaches: critical|high|medium|low|none")
	s.fs.StringVar(&s.policy, "policy", "", "path to a policy file")
	s.fs.IntVar(&s.concurrency, "concurrency", 0, "worker count (default: NumCPU)")
	s.fs.BoolVar(&s.deterministic, "deterministic", false, "suppress timestamps and machine-specific metadata")
	s.fs.BoolVar(&s.verbose, "verbose", false, "verbose logging")
	s.fs.BoolVar(&s.verbose, "v", false, "verbose logging (shorthand)")
	return s
}

func (s *scanFlags) parse(args []string) error {
	if err := s.fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return errHelp
		}
		return err
	}
	rest := s.fs.Args()
	if len(rest) > 1 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(rest[1:], " "))
	}
	if len(rest) == 1 {
		s.target = rest[0]
	}
	if len(s.format) == 0 {
		s.format = multiFlag{"markdown"}
	}
	return nil
}

func (s *scanFlags) usage() string {
	var b strings.Builder
	b.WriteString("Usage:\n  cryptarium scan <path|git-url> [flags]\n\nFlags:\n")
	s.fs.SetOutput(&b)
	s.fs.PrintDefaults()
	return b.String()
}

// multiFlag collects repeatable string flags.
type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

// ioDiscard is a tiny Writer used to silence flag.FlagSet's default error output
// so we can format errors ourselves.
type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
