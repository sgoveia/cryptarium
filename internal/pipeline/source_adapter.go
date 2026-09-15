package pipeline

import (
	"github.com/sgoveia/cryptarium/internal/detector"
	"github.com/sgoveia/cryptarium/internal/detector/source"
)

func newSourceWithRules(dir string) detector.Detector {
	return source.NewWithRulesDir(dir)
}
