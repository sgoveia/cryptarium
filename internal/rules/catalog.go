package rules

import (
	"fmt"
	"os"
)

// LibraryEntry describes a known cryptographic dependency from the catalog.
type LibraryEntry struct {
	Name       string   `yaml:"name"`
	Ecosystem  string   `yaml:"ecosystem"`
	Primitives []string `yaml:"primitives"`
	Notes      string   `yaml:"notes,omitempty"`
	Confidence string   `yaml:"confidence"`
}

type catalogFile struct {
	Libraries []LibraryEntry `yaml:"libraries"`
}

// LoadLibraryCatalog reads rules/libraries/catalog.yaml (or an explicit path).
func LoadLibraryCatalog(path string) ([]LibraryEntry, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: catalog path is tool-resolved or an explicit test override
	if err != nil {
		return nil, fmt.Errorf("read library catalog %s: %w", path, err)
	}
	return parseLibraryCatalog(data, path)
}
