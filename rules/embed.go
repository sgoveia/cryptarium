// Package ruledata embeds the default YAML rule packs and library catalog.
// The YAML files in this directory remain the community contribution surface;
// go embed ships them inside release binaries and go install builds so scans
// work without a source checkout beside the binary.
package ruledata

import "embed"

// FS is the embedded rules tree shipped with the cryptarium binary.
//
//go:embed go/*.yaml python/*.yaml javascript/*.yaml java/*.yaml c/*.yaml cpp/*.yaml libraries/*.yaml
var FS embed.FS
