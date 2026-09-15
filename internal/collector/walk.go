package collector

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// defaultSkipDirs are never descended into. vendor/ is deferred (DESIGN §17).
var defaultSkipDirs = map[string]struct{}{
	".git":         {},
	".hg":          {},
	".svn":         {},
	"node_modules": {},
	"bin":          {},
}

// Walk lists files under target.Root. It skips defaultSkipDirs, honors
// cancellation, and returns FileRefs sorted by Path for determinism.
func Walk(ctx context.Context, target Target) ([]FileRef, error) {
	root, err := filepath.Abs(target.Root)
	if err != nil {
		return nil, fmt.Errorf("resolve root %s: %w", target.Root, err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat root %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root %s is not a directory", root)
	}

	var out []FileRef
	err = filepath.WalkDir(root, func(abs string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		name := d.Name()
		if d.IsDir() {
			if _, skip := defaultSkipDirs[name]; skip && abs != root {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}

		rel, err := filepath.Rel(root, abs)
		if err != nil {
			return fmt.Errorf("rel %s: %w", abs, err)
		}
		fi, err := d.Info()
		if err != nil {
			return fmt.Errorf("info %s: %w", abs, err)
		}
		out = append(out, FileRef{
			Path:    filepath.ToSlash(rel),
			AbsPath: abs,
			Size:    fi.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", root, err)
	}

	// filepath.WalkDir is lexical; normalize anyway for clarity.
	sortFileRefs(out)
	return out, nil
}

func sortFileRefs(refs []FileRef) {
	sort.Slice(refs, func(i, j int) bool { return refs[i].Path < refs[j].Path })
}

// Ext returns the lower-case extension of f.Path including the leading dot
// (e.g. ".pem"), or "" if none.
func Ext(f FileRef) string {
	return strings.ToLower(filepath.Ext(f.Path))
}
