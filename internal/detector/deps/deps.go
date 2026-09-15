package deps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"

	"github.com/sgoveia/cryptarium/internal/collector"
	"github.com/sgoveia/cryptarium/internal/detector"
	"github.com/sgoveia/cryptarium/internal/model"
	"github.com/sgoveia/cryptarium/internal/rules"
)

func init() {
	detector.Register(New())
}

// Detector matches dependency manifests against the known-library catalog.
// A hit is Confidence medium at most — presence is not proof of use.
type Detector struct {
	catalogPath string
	catalog     []rules.LibraryEntry
	catalogErr  error
	loaded      bool
}

// New returns a deps detector using the default catalog path relative to the
// process working directory (rules/libraries/catalog.yaml). Prefer
// NewWithCatalog in tests and when the scan root differs from the module root.
func New() *Detector {
	return &Detector{catalogPath: filepath.Join("rules", "libraries", "catalog.yaml")}
}

// NewWithCatalog returns a deps detector that loads libraries from catalogPath.
func NewWithCatalog(catalogPath string) *Detector {
	return &Detector{catalogPath: catalogPath}
}

// Name returns the stable detector name.
func (*Detector) Name() string { return "deps" }

// Handles reports whether f is a supported dependency manifest.
func (*Detector) Handles(f collector.FileRef) bool {
	base := filepath.Base(f.Path)
	switch base {
	case "go.mod":
		return true
	default:
		return false
	}
}

// Detect parses a manifest and emits one finding per matched catalog library.
func (d *Detector) Detect(ctx context.Context, f collector.FileRef) ([]model.CryptoFinding, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if err := d.ensureCatalog(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(f.AbsPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", f.Path, err)
	}

	switch filepath.Base(f.Path) {
	case "go.mod":
		return d.detectGoMod(f, data)
	default:
		return nil, fmt.Errorf("unsupported manifest %s", f.Path)
	}
}

func (d *Detector) ensureCatalog() error {
	if d.loaded {
		return d.catalogErr
	}
	d.loaded = true
	libs, err := rules.LoadLibraryCatalog(d.catalogPath)
	if err != nil {
		d.catalogErr = err
		return err
	}
	var goLibs []rules.LibraryEntry
	for _, lib := range libs {
		if lib.Ecosystem == "go" {
			goLibs = append(goLibs, lib)
		}
	}
	d.catalog = goLibs
	return nil
}

func (d *Detector) detectGoMod(f collector.FileRef, data []byte) ([]model.CryptoFinding, error) {
	mf, err := modfile.Parse(f.Path, data, nil)
	if err != nil {
		return nil, fmt.Errorf("parse go.mod %s: %w", f.Path, err)
	}

	type req struct {
		path    string
		version string
		line    int
	}
	var reqs []req
	for _, r := range mf.Require {
		if r == nil || r.Mod.Path == "" {
			continue
		}
		line := 0
		if r.Syntax != nil {
			line = r.Syntax.Start.Line
		}
		reqs = append(reqs, req{path: r.Mod.Path, version: r.Mod.Version, line: line})
	}

	var findings []model.CryptoFinding
	for _, lib := range d.catalog {
		for _, r := range reqs {
			if !moduleMatches(r.path, lib.Name) {
				continue
			}
			findings = append(findings, d.finding(f, lib, r.path, r.version, r.line))
		}
	}
	return findings, nil
}

// moduleMatches reports whether a go.mod module path is the catalog entry or a subpackage path.
func moduleMatches(modulePath, catalogName string) bool {
	if modulePath == catalogName {
		return true
	}
	return strings.HasPrefix(modulePath, catalogName+"/")
}

func (d *Detector) finding(f collector.FileRef, lib rules.LibraryEntry, modulePath, version string, line int) model.CryptoFinding {
	// One finding per catalog hit; primitive is the first listed (inventory signal).
	// Full primitive list is retained in parameters for correlation later.
	primitive := "UNKNOWN"
	if len(lib.Primitives) > 0 {
		primitive = lib.Primitives[0]
	}
	params := map[string]any{
		"module":     modulePath,
		"version":    version,
		"primitives": append([]string(nil), lib.Primitives...),
	}
	if lib.Notes != "" {
		params["notes"] = lib.Notes
	}
	conf := model.ConfidenceMedium
	switch strings.ToLower(lib.Confidence) {
	case "high":
		conf = model.ConfidenceHigh
	case "low":
		conf = model.ConfidenceLow
	case "medium", "":
		conf = model.ConfidenceMedium
	}

	ruleID := "deps.go." + sanitizeRuleID(lib.Name)
	return model.CryptoFinding{
		ID: model.FindingID(model.FindingIDInput{
			Detector:   d.Name(),
			RuleID:     ruleID,
			RelPath:    f.Path,
			Line:       line,
			Primitive:  primitive,
			Parameters: params,
		}),
		AssetType:  model.AssetLibrary,
		Primitive:  primitive,
		Parameters: params,
		Evidence: model.Evidence{
			Source:     model.SourceDependency,
			Path:       f.Path,
			Line:       line,
			Snippet:    fmt.Sprintf("require %s %s", modulePath, version),
			RuleID:     ruleID,
			Confidence: conf,
		},
	}
}

func sanitizeRuleID(name string) string {
	r := strings.NewReplacer("/", ".", "@", ".", ":", ".")
	return r.Replace(name)
}
