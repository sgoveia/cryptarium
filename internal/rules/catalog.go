package rules

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
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
	var file catalogFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse library catalog %s: %w", path, err)
	}
	for i, lib := range file.Libraries {
		if lib.Name == "" {
			return nil, fmt.Errorf("library catalog %s: entry %d missing name", path, i)
		}
		if lib.Ecosystem == "" {
			return nil, fmt.Errorf("library catalog %s: %s missing ecosystem", path, lib.Name)
		}
		if len(lib.Primitives) == 0 {
			return nil, fmt.Errorf("library catalog %s: %s missing primitives", path, lib.Name)
		}
	}
	return file.Libraries, nil
}
