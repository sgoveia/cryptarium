package rules

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindDefaultCatalog walks up from the working directory looking for
// rules/libraries/catalog.yaml (the repo-root contribution surface).
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
	return "", fmt.Errorf("rules/libraries/catalog.yaml not found from %s", mustGetwd())
}

func mustGetwd() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}
