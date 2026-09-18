package rules

import (
	"errors"
	"fmt"

	ruledata "github.com/sgoveia/cryptarium/rules"
)

// ErrNotFound means no rules/ tree was found walking up from the working directory.
var ErrNotFound = errors.New("rules not found on disk")

// LoadEmbeddedCatalog loads the library catalog baked into the binary.
func LoadEmbeddedCatalog() ([]LibraryEntry, error) {
	return LoadLibraryCatalogFS(ruledata.FS, "libraries/catalog.yaml")
}

// LoadEmbeddedRulePacks loads source rule packs baked into the binary.
func LoadEmbeddedRulePacks() ([]*RulePack, error) {
	return LoadRulePacksFS(ruledata.FS)
}

// LoadDefaultCatalog prefers a filesystem rules/libraries/catalog.yaml found
// by walking up from cwd (so local edits work without rebuild), then falls
// back to the embedded catalog so release binaries and go install work from
// any directory.
func LoadDefaultCatalog() ([]LibraryEntry, error) {
	path, err := FindDefaultCatalog()
	if err == nil {
		return LoadLibraryCatalog(path)
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	libs, err := LoadEmbeddedCatalog()
	if err != nil {
		return nil, fmt.Errorf("load embedded library catalog: %w", err)
	}
	return libs, nil
}

// LoadDefaultRulePacks prefers a filesystem rules/ directory, then embedded packs.
func LoadDefaultRulePacks() ([]*RulePack, error) {
	dir, err := FindDefaultRulesDir()
	if err == nil {
		return LoadRulePacksDir(dir)
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	packs, err := LoadEmbeddedRulePacks()
	if err != nil {
		return nil, fmt.Errorf("load embedded rule packs: %w", err)
	}
	return packs, nil
}
