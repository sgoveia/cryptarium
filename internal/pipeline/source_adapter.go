package pipeline

import (
	"github.com/sgoveia/cryptarium/internal/detector"
	"github.com/sgoveia/cryptarium/internal/detector/source"
)

func newSourceConfigured(rulesDir string, extra []string) detector.Detector {
	return source.NewConfigured(rulesDir, extra...)
}
