package rules

import (
	"fmt"
	"io/fs"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadRulePacksFS loads all *.yaml / *.yml rule packs under fsys, skipping
// the libraries/ directory (dependency catalog, not source packs).
func LoadRulePacksFS(fsys fs.FS) ([]*RulePack, error) {
	var packs []*RulePack
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if d.Name() == "libraries" {
				return fs.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			return nil
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("read rule pack %s: %w", path, err)
		}
		pack, err := parseRulePack(data, path)
		if err != nil {
			return err
		}
		packs = append(packs, pack)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return packs, nil
}

// LoadLibraryCatalogFS reads libraries/catalog.yaml (or path) from fsys.
func LoadLibraryCatalogFS(fsys fs.FS, path string) ([]LibraryEntry, error) {
	if path == "" {
		path = "libraries/catalog.yaml"
	}
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("read library catalog %s: %w", path, err)
	}
	return parseLibraryCatalog(data, path)
}

func parseRulePack(data []byte, path string) (*RulePack, error) {
	var pack RulePack
	if err := yaml.Unmarshal(data, &pack); err != nil {
		return nil, fmt.Errorf("parse rule pack %s: %w", path, err)
	}
	if pack.ID == "" {
		return nil, fmt.Errorf("rule pack %s: missing id", path)
	}
	if pack.Language == "" {
		return nil, fmt.Errorf("rule pack %s: missing language", path)
	}
	for i, r := range pack.Rules {
		if r.ID == "" {
			return nil, fmt.Errorf("rule pack %s: rule %d missing id", path, i)
		}
		if r.Primitive == "" {
			return nil, fmt.Errorf("rule pack %s: rule %s missing primitive", path, r.ID)
		}
		if r.Match.Kind == "" {
			return nil, fmt.Errorf("rule pack %s: rule %s missing match.kind", path, r.ID)
		}
	}
	return &pack, nil
}

func parseLibraryCatalog(data []byte, path string) ([]LibraryEntry, error) {
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
