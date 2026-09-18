package rules

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindDefaultCatalog walks up from the working directory looking for
// rules/libraries/catalog.yaml (the repo-root contribution surface).
// Returns ErrNotFound when no on-disk catalog exists; callers that need a
// catalog should then use LoadDefaultCatalog / LoadEmbeddedCatalog.
func FindDefaultCatalog() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}
	for {
		candidate := filepath.Join(dir, "rules", "libraries", "catalog.yaml")
		fi, err := os.Stat(candidate)
		if err == nil && !fi.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("%w: rules/libraries/catalog.yaml not found from %s", ErrNotFound, mustGetwd())
}

func mustGetwd() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}
