package detector

import (
	"context"

	"github.com/sgoveia/cryptarium/internal/collector"
	"github.com/sgoveia/cryptarium/internal/model"
)

// Detector parses one class of files and emits unclassified findings.
// Implementations must be goroutine-safe and free of mutable global state.
type Detector interface {
	// Name is stable and appears in output. Lowercase, hyphenated.
	Name() string

	// Handles reports whether this detector wants the file. Cheap:
	// extension and basename checks only, no file reads.
	Handles(f collector.FileRef) bool

	// Detect parses one file and emits findings. Must be pure with respect
	// to the file contents. A parse failure returns findings-so-far plus an
	// error; it never panics and never aborts the whole scan.
	Detect(ctx context.Context, f collector.FileRef) ([]model.CryptoFinding, error)
}
