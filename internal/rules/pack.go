package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RulePack is a YAML rule pack for source detection (DESIGN.md §5).
type RulePack struct {
	ID          string `yaml:"id"`
	Version     int    `yaml:"version"`
	Language    string `yaml:"language"`
	Description string `yaml:"description"`
	Rules       []Rule `yaml:"rules"`
}

// Rule is one detection pattern.
type Rule struct {
	ID         string         `yaml:"id"`
	Primitive  string         `yaml:"primitive"`
	AssetType  string         `yaml:"assetType"`
	Functions  []string       `yaml:"functions"`
	Confidence string         `yaml:"confidence"`
	Match      RuleMatch      `yaml:"match"`
	Parameters map[string]any `yaml:"parameters"`
	References []string       `yaml:"references"`
}

// RuleMatch describes how to match source.
type RuleMatch struct {
	Kind    string `yaml:"kind"` // call | identifier
	Package string `yaml:"package,omitempty"`
	Symbol  string `yaml:"symbol,omitempty"`
	Pattern string `yaml:"pattern,omitempty"`
}

// LoadRulePack reads one YAML rule pack file.
func LoadRulePack(path string) (*RulePack, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: rule pack path is tool-resolved
	if err != nil {
		return nil, fmt.Errorf("read rule pack %s: %w", path, err)
	}
	return parseRulePack(data, path)
}

// LoadRulePacksDir loads all *.yaml rule packs under dir (non-recursive for
// language dirs) and recursively under language subdirectories, skipping libraries/.
func LoadRulePacksDir(dir string) ([]*RulePack, error) {
	var packs []*RulePack
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if d.Name() == "libraries" {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			return nil
		}
		pack, err := LoadRulePack(path)
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

// FindDefaultRulesDir walks up from cwd looking for rules/.
// Returns ErrNotFound when no on-disk rules/ exists; callers that need packs
// should then use LoadDefaultRulePacks / LoadEmbeddedRulePacks.
func FindDefaultRulesDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}
	for {
		candidate := filepath.Join(dir, "rules")
		fi, err := os.Stat(candidate)
		if err == nil && fi.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("%w: rules/ directory not found from %s", ErrNotFound, mustGetwd())
}
